package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/audio"
	"github.com/stwalsh4118/navi/internal/git"
	"github.com/stwalsh4118/navi/internal/metrics"
	"github.com/stwalsh4118/navi/internal/monitor"
	"github.com/stwalsh4118/navi/internal/pathutil"
	"github.com/stwalsh4118/navi/internal/pm"
	"github.com/stwalsh4118/navi/internal/remote"
	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/task"
)

// Model is the Bubble Tea application state for navi.
type Model struct {
	sessions            []session.Info
	cursor              int
	sessionScrollOffset int // First visible session index for session list scrolling
	width               int
	height              int
	err                 error
	lastSelectedSession string // Used to preserve cursor position after attach/detach

	// Dialog state
	dialogMode  DialogMode // Which dialog is currently open (DialogNone if none)
	dialogError string     // Error message to display in dialog

	// Text inputs for dialogs
	nameInput       textinput.Model // Session name input
	dirInput        textinput.Model // Working directory input
	focusedInput    int             // Which input is focused (0 = name, 1 = dir, 2 = skipPerms)
	skipPermissions bool            // Whether to start claude with --dangerously-skip-permissions
	sessionToModify *session.Info   // Session being killed or renamed

	// Preview pane state
	previewVisible      bool          // Whether preview pane is shown
	previewUserEnabled  bool          // User's intended state (for restore after terminal resize)
	previewContent      string        // Cached captured output
	previewLayout       PreviewLayout // Current layout mode (default: PreviewLayoutSide)
	previewWidth        int           // Width of preview pane in columns (side layout)
	previewHeight       int           // Height of preview pane in rows (bottom layout)
	previewWrap         bool          // Whether to wrap long lines (true) or truncate (false)
	previewScrollOffset int           // First visible line in preview pane
	previewAutoScroll   bool          // Auto-scroll to bottom on new content (default true)
	previewFocused      bool          // Whether keyboard focus is in preview pane
	previewLastCapture  time.Time     // Last capture timestamp for debouncing
	previewLastCursor   int           // Last cursor position for detecting cursor changes

	// Git info cache
	gitCache map[string]*git.Info // Cache of git info by session working directory

	// Resource metrics cache (RSS bytes by session name)
	resourceCache map[string]int64

	// Remote session support
	Remotes             []remote.Config    // Configured remote machines
	SSHPool             *remote.SSHPool    // SSH connection pool for remotes
	filterMode          session.FilterMode // Current session filter mode
	audioNotifier       *audio.Notifier    // Audio notification manager
	lastSessionStates   map[string]string  // Last known status by session name
	lastAgentStates     map[string]map[string]string
	attachMonitor       *monitor.AttachMonitor
	attachMonitorCancel context.CancelFunc
	audioNotifyFn       func(string, string) // Test hook; defaults to notifier.Notify

	// Search and filter state
	searchQuery     string          // Current search text
	searchMode      bool            // Whether search input is active
	searchInput     textinput.Model // Text input for search
	searchMatches   []int           // Indices of matching sessions in the filtered list
	currentMatchIdx int             // Position within searchMatches (which match is "current")
	statusFilter    string          // Active status filter (empty = show all)
	hideOffline     bool            // Whether to hide offline/done sessions (default false = show all)
	sortMode        SortMode        // Active sort mode

	// Task panel state
	taskPanelVisible     bool                        // Whether task panel is shown below session list
	taskPanelUserEnabled bool                        // User's intended state (for restore after terminal resize)
	taskPanelFocused     bool                        // Whether keyboard focus is inside the task panel
	taskPanelHeight      int                         // Height of task panel in rows
	taskCursor           int                         // Cursor position in flat task item list
	taskScrollOffset     int                         // First visible item index for task panel scrolling
	taskExpandedGroups   map[string]bool             // Which groups are expanded (collapsed by default)
	taskSearchMode       bool                        // Whether task search input is active
	taskSearchQuery      string                      // Current task search text
	taskSearchInput      textinput.Model             // Text input for task search
	taskSearchMatches    []int                       // Indices of matching task items in visible list
	taskCurrentMatchIdx  int                         // Position within taskSearchMatches
	taskGroups           []task.TaskGroup            // Displayed task groups (for focused project)
	taskGroupsByProject  map[string][]task.TaskGroup // All task groups keyed by project dir
	taskFocusedProject   string                      // Project dir whose tasks are displayed
	taskErrors           map[string]error            // Provider errors keyed by project dir
	taskProjectConfigs   []task.ProjectConfig        // Discovered project configs
	taskCache            *task.ResultCache           // Cache for provider results
	taskGlobalConfig     *task.GlobalConfig          // Global task config
	taskLastCWDs         []string                    // Last seen session CWDs (for change detection)
	taskSortMode         taskSortMode                // Current sort mode for task groups
	taskSortReversed     bool                        // Whether to reverse the sort direction
	taskFilterMode       taskFilterMode              // Current filter mode for task groups
	taskAccordionMode    bool                        // Accordion mode: expanding one group collapses others
	taskRefreshing       bool                        // Whether a manual refresh is in progress

	// PR auto-refresh state
	prAutoRefreshActive bool // Whether auto-refresh ticker is active for pending checks

	// PM engine state
	pmEngine              *pm.Engine
	pmOutput              *pm.PMOutput
	pmTaskResults         map[string]*task.ProviderResult
	pmRunInFlight         bool
	pmViewVisible         bool
	pmZoneFocus           int
	pmInvoker             *pm.Invoker
	pmBriefing            *pm.PMBriefing
	pmBriefingStale       bool
	pmInvokeInFlight      bool
	pmInvokeStatus        string
	pmProjectCursor       int
	pmProjectScrollOffset int
	pmEventScrollOffset   int
	pmExpandedProjects    map[string]bool
	pmProjectFilterDir    string
	pmLastError           string

	// Sound pack picker state
	soundPacks           []audio.PackInfo // Loaded pack list
	soundPackCursor      int              // Current selection index
	soundPackScrollOffset int             // Viewport scroll offset
	activeSoundPack      string           // Currently active pack name

	// Content viewer state
	contentViewerTitle      string      // Title displayed in the content viewer header
	contentViewerLines      []string    // Content split into lines for scrolling
	contentViewerScroll     int         // Current scroll offset (line index of first visible line)
	contentViewerMode       ContentMode // Content type (plain text or diff)
	contentViewerPrevDialog DialogMode  // Dialog to return to when closing (DialogNone if standalone)
}

// Message types for Bubble Tea communication.
type tickMsg time.Time
type sessionsMsg []session.Info
type attachDoneMsg struct{}

// createSessionResultMsg is returned after attempting to create a new session.
type createSessionResultMsg struct {
	err error
}

// killSessionResultMsg is returned after attempting to kill a session.
type killSessionResultMsg struct {
	err error
}

// renameSessionResultMsg is returned after attempting to rename a session.
type renameSessionResultMsg struct {
	err     error
	newName string
}

// previewContentMsg is returned after capturing preview content.
type previewContentMsg struct {
	content string
	err     error
}

// previewTickMsg is sent to trigger periodic preview refresh.
type previewTickMsg time.Time

// previewDebounceMsg is sent after cursor movement debounce delay.
type previewDebounceMsg struct{}

// pmTickMsg is sent to trigger periodic PM engine refresh.
type pmTickMsg time.Time

// pmOutputMsg delivers PM engine results back to the model.
type pmOutputMsg struct {
	output *pm.PMOutput
	err    error
}

// pmInvokeMsg delivers Claude PM invoker results back to the model.
type pmInvokeMsg struct {
	briefing *pm.PMBriefing
	isStale  bool
	err      error
}

// pmStreamMsg delivers an intermediate streaming status update from the PM invoker.
type pmStreamMsg struct {
	status   string
	streamCh <-chan pm.StreamEvent
	resultCh <-chan pmInvokeMsg
}

// soundPacksMsg is returned after loading available sound packs.
type soundPacksMsg struct {
	packs []audio.PackInfo
	err   error
}

// resourceTickMsg is sent to trigger periodic resource usage polling.
type resourceTickMsg time.Time

// resourcePollMsg carries polled RSS data keyed by session name.
type resourcePollMsg map[string]int64

// gitTickMsg is sent to trigger periodic git info refresh.
type gitTickMsg time.Time

// prAutoRefreshTickMsg is sent to trigger periodic PR detail refresh when checks are pending.
type prAutoRefreshTickMsg time.Time

// prAutoRefreshInterval is the interval between auto-refreshes of PR data when checks are pending.
const prAutoRefreshInterval = 30 * time.Second

// gitInfoMsg is returned after polling git info for all sessions.
type gitInfoMsg struct {
	cache map[string]*git.Info // Map of CWD to git.Info
}

// gitPRMsg is returned after fetching PR info for a specific directory.
type gitPRMsg struct {
	cwd      string
	prNum    int
	prDetail *git.PRDetail
}

// gitPRCommentsMsg carries fetched PR comments.
type gitPRCommentsMsg struct {
	comments []git.PRComment
	err      error
}

// remoteSessionsMsg is the Bubble Tea message for remote session polling results.
type remoteSessionsMsg struct {
	sessions []session.Info
}

// remoteGitInfoMsg is returned after fetching git info from a remote session via SSH.
type remoteGitInfoMsg struct {
	cwd  string
	info *git.Info
	err  error
}

// remoteDismissResultMsg is returned after dismissing a remote session via SSH.
type remoteDismissResultMsg struct {
	err error
}

// Init implements tea.Model.
// Starts the tick command and performs initial poll.
func (m Model) Init() tea.Cmd {
	// Initialize git cache if nil
	if m.gitCache == nil {
		m.gitCache = make(map[string]*git.Info)
	}

	cmds := []tea.Cmd{tickCmd(), pollSessions, gitTickCmd(), pmTickCmd(), resourceTickCmd()}

	// Start task refresh tick
	interval := taskDefaultRefreshInterval
	if m.taskGlobalConfig != nil && m.taskGlobalConfig.Tasks.Interval.Duration > 0 {
		interval = m.taskGlobalConfig.Tasks.Interval.Duration
	}
	cmds = append(cmds, taskTickCmd(interval))

	return tea.Batch(cmds...)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle dialog mode first - block main keybindings when dialog is open
		if m.dialogMode != DialogNone {
			return m.updateDialog(msg)
		}

		// Handle PM view mode before any other focused panel handlers.
		if m.pmViewVisible {
			return m.updatePMView(msg)
		}

		// Handle task panel focus mode - route to task panel keybindings
		if m.taskPanelFocused {
			return m.updateTaskPanelFocus(msg)
		}

		// Handle preview focus mode - route to preview scroll keybindings
		if m.previewFocused {
			return m.updatePreviewFocus(msg)
		}

		// Handle search mode - forward most keys to search input
		if m.searchMode {
			return m.updateSearchMode(msg)
		}

		// Main keybindings (only when no dialog is open and not in search mode)
		switch msg.String() {
		case "up", "k":
			filteredSessions := m.getFilteredSessions()
			if len(filteredSessions) > 0 {
				oldCursor := m.cursor
				if m.cursor > 0 {
					m.cursor--
				} else {
					m.cursor = len(filteredSessions) - 1 // wrap to bottom
				}
				m.ensureSessionCursorVisible(m.sessionListMaxVisible())
				if m.cursor != oldCursor {
					// Trigger debounced preview capture if preview visible
					if m.previewVisible {
						m.previewLastCursor = m.cursor
						return m, previewDebounceCmd()
					}
					// Update task panel if visible
					if m.taskPanelVisible {
						m.updateTaskPanelForCursor()
					}
				}
			}
			return m, nil

		case "down", "j":
			filteredSessions := m.getFilteredSessions()
			if len(filteredSessions) > 0 {
				oldCursor := m.cursor
				if m.cursor < len(filteredSessions)-1 {
					m.cursor++
				} else {
					m.cursor = 0 // wrap to top
				}
				m.ensureSessionCursorVisible(m.sessionListMaxVisible())
				if m.cursor != oldCursor {
					// Trigger debounced preview capture if preview visible
					if m.previewVisible {
						m.previewLastCursor = m.cursor
						return m, previewDebounceCmd()
					}
					// Update task panel if visible
					if m.taskPanelVisible {
						m.updateTaskPanelForCursor()
					}
				}
			}
			return m, nil

		case "enter":
			filteredSessions := m.getFilteredSessions()
			if len(filteredSessions) > 0 && m.cursor < len(filteredSessions) {
				s := filteredSessions[m.cursor]
				m.lastSelectedSession = s.TmuxSession

				// Check if this is a remote session
				if s.Remote != "" && m.SSHPool != nil {
					// Get the remote config for this session
					r := m.SSHPool.GetRemoteConfig(s.Remote)
					if r != nil {
						m.startAttachMonitor()
						return m, attachRemoteSession(r, s.TmuxSession)
					}
					// Fallback to local attach if remote config not found
				}

				m.startAttachMonitor()
				return m, attachSession(s.TmuxSession)
			}
			return m, nil

		case "d":
			filteredSessions := m.getFilteredSessions()
			if len(filteredSessions) > 0 && m.cursor < len(filteredSessions) {
				s := filteredSessions[m.cursor]
				// Remote dismiss via SSH
				if s.Remote != "" && m.SSHPool != nil {
					return m, dismissRemoteSessionCmd(m.SSHPool, s.Remote, s.TmuxSession)
				}
				_ = dismissSession(s) // Ignore error, poll will update view
				return m, pollSessions
			}
			return m, nil

		case "r":
			return m, pollSessions

		case "n":
			// When search query is active, jump to next match
			if m.searchQuery != "" {
				m.nextMatch()
				return m, nil
			}
			// Open new session dialog
			m.dialogMode = DialogNewSession
			m.dialogError = ""
			m.nameInput = initNameInput()
			m.dirInput = initDirInput()
			m.dirInput.SetValue(getDefaultDirectory())
			m.focusedInput = focusName
			m.skipPermissions = false
			return m, nil

		case "N":
			// When search query is active, jump to previous match
			if m.searchQuery != "" {
				m.prevMatch()
				return m, nil
			}
			return m, nil

		case "x":
			// Open kill confirmation dialog
			filteredSessions := m.getFilteredSessions()
			if len(filteredSessions) > 0 && m.cursor < len(filteredSessions) {
				s := filteredSessions[m.cursor]
				m.sessionToModify = &s
				m.dialogMode = DialogKillConfirm
				m.dialogError = ""
				return m, nil
			}
			return m, nil

		case "R":
			// Open rename dialog
			filteredSessions := m.getFilteredSessions()
			if len(filteredSessions) > 0 && m.cursor < len(filteredSessions) {
				s := filteredSessions[m.cursor]
				m.sessionToModify = &s
				m.dialogMode = DialogRename
				m.dialogError = ""
				m.nameInput = initNameInput()
				m.nameInput.SetValue(s.TmuxSession)
				m.nameInput.CursorEnd() // Put cursor at end
				return m, nil
			}
			return m, nil

		case "tab":
			// Tab enters task panel focus when task panel is visible
			if m.taskPanelVisible {
				m.taskPanelFocused = true
				m.taskCursor = 0
				m.taskScrollOffset = 0
				return m, nil
			}
			// Tab enters preview focus when preview is visible
			if m.previewVisible {
				m.previewFocused = true
				return m, nil
			}
			// Otherwise toggle preview (same as p)
			return m.togglePreview()

		case "p":
			// Toggle preview pane visibility (mutually exclusive with task panel)
			return m.togglePreview()

		case "[":
			// Shrink active panel
			if m.taskPanelVisible {
				currentHeight := m.getTaskPanelHeight()
				newHeight := currentHeight - previewResizeStep
				if newHeight < previewMinHeight {
					newHeight = previewMinHeight
				}
				m.taskPanelHeight = newHeight
			} else if m.previewVisible {
				if m.previewLayout == PreviewLayoutBottom {
					currentHeight := m.getPreviewHeight()
					newHeight := currentHeight - previewResizeStep
					if newHeight < previewMinHeight {
						newHeight = previewMinHeight
					}
					m.previewHeight = newHeight
				} else {
					currentWidth := m.getPreviewWidth()
					newWidth := currentWidth - previewResizeStep
					if newWidth < previewMinWidth {
						newWidth = previewMinWidth
					}
					m.previewWidth = newWidth
				}
			}
			return m, nil

		case "]":
			// Expand active panel
			if m.taskPanelVisible {
				contentHeight := m.height - 8
				maxHeight := contentHeight - sessionListMinHeight
				currentHeight := m.getTaskPanelHeight()
				newHeight := currentHeight + previewResizeStep
				if newHeight > maxHeight {
					newHeight = maxHeight
				}
				m.taskPanelHeight = newHeight
			} else if m.previewVisible {
				if m.previewLayout == PreviewLayoutBottom {
					contentHeight := m.height - 8
					maxHeight := contentHeight - sessionListMinHeight
					currentHeight := m.getPreviewHeight()
					newHeight := currentHeight + previewResizeStep
					if newHeight > maxHeight {
						newHeight = maxHeight
					}
					m.previewHeight = newHeight
				} else {
					currentWidth := m.getPreviewWidth()
					maxWidth := m.width - sessionListMinWidth - 1
					newWidth := currentWidth + previewResizeStep
					if newWidth > maxWidth {
						newWidth = maxWidth
					}
					m.previewWidth = newWidth
				}
			}
			return m, nil

		case "L":
			// Toggle preview layout between side and bottom
			if m.previewVisible {
				if m.previewLayout == PreviewLayoutSide {
					m.previewLayout = PreviewLayoutBottom
				} else {
					m.previewLayout = PreviewLayoutSide
				}
			}
			return m, nil

		case "W":
			// Toggle preview wrap mode
			if m.previewVisible {
				m.previewWrap = !m.previewWrap
			}
			return m, nil

		case "G":
			// Open git detail view for selected session
			filteredSessions := m.getFilteredSessions()
			if len(filteredSessions) > 0 && m.cursor < len(filteredSessions) {
				s := filteredSessions[m.cursor]
				m.sessionToModify = &s
				m.dialogMode = DialogGitDetail
				m.dialogError = ""

				// For remote sessions, check cache or fetch via SSH
				if s.Remote != "" && m.SSHPool != nil {
					if cached, ok := m.gitCache[s.CWD]; ok && !cached.IsStale() {
						m.sessionToModify.Git = cached
						return m, fetchRemotePRCmd(s.CWD, cached.Branch, cached.Remote)
					}
					return m, fetchRemoteGitCmd(m.SSHPool, s.Remote, s.CWD)
				}

				// For local sessions, lazily fetch PR info
				if s.CWD != "" && s.Git != nil {
					return m, fetchPRCmd(s.CWD)
				}
				return m, nil
			}
			return m, nil

		case "i":
			// Open metrics detail view for selected session
			filteredSessions := m.getFilteredSessions()
			if len(filteredSessions) > 0 && m.cursor < len(filteredSessions) {
				s := filteredSessions[m.cursor]
				m.sessionToModify = &s
				m.dialogMode = DialogMetricsDetail
				m.dialogError = ""
				return m, nil
			}
			return m, nil

		case "/":
			// Enter search mode
			m.searchMode = true
			m.searchInput.SetValue("")
			m.searchInput.Focus()
			return m, nil

		case "esc":
			// Clear search state first if search is active (persisted after Enter)
			if m.searchQuery != "" {
				m.clearSearchState()
				return m, nil
			}
			if m.pmProjectFilterDir != "" {
				m.pmProjectFilterDir = ""
				m.cursor = 0
				m.sessionScrollOffset = 0
				return m, nil
			}
			// Clear active filters
			if m.statusFilter != "" || m.hideOffline {
				m.statusFilter = ""
				m.hideOffline = false
				m.cursor = 0
				return m, nil
			}
			return m, nil

		case "s":
			// Cycle sort mode: Priority -> Name -> Age -> Status -> Directory -> Priority
			m.sortMode = (m.sortMode + 1) % SortMode(sortModeCount)
			if m.searchQuery != "" {
				m.computeSearchMatches()
			}
			return m, nil

		case "o":
			// Toggle offline session visibility
			selectedSession := m.selectedSessionName()
			m.hideOffline = !m.hideOffline
			m.preserveCursor(selectedSession)
			if m.searchQuery != "" {
				m.computeSearchMatches()
			}
			return m, nil

		case "0":
			// Clear status and project filters
			selectedSession := m.selectedSessionName()
			m.statusFilter = ""
			m.pmProjectFilterDir = ""
			m.preserveCursor(selectedSession)
			if m.searchQuery != "" {
				m.computeSearchMatches()
			}
			return m, nil

		case "1", "2", "3", "4", "5":
			// Toggle status filter by number key
			selectedSession := m.selectedSessionName()
			targetStatus := statusFilterKeys[msg.String()]
			if m.statusFilter == targetStatus {
				m.statusFilter = "" // Toggle off if same key pressed
			} else {
				m.statusFilter = targetStatus
			}
			m.preserveCursor(selectedSession)
			if m.searchQuery != "" {
				m.computeSearchMatches()
			}
			return m, nil

		case "f":
			// Cycle filter mode: All -> Local -> Remote -> All
			if len(m.Remotes) > 0 {
				selectedSession := m.selectedSessionName()
				switch m.filterMode {
				case session.FilterAll:
					m.filterMode = session.FilterLocal
				case session.FilterLocal:
					m.filterMode = session.FilterRemote
				case session.FilterRemote:
					m.filterMode = session.FilterAll
				}
				m.preserveCursor(selectedSession)
				if m.searchQuery != "" {
					m.computeSearchMatches()
				}
			}
			return m, nil

		case "T":
			// Toggle task panel (mutually exclusive with preview)
			m.taskPanelVisible = !m.taskPanelVisible
			m.taskPanelUserEnabled = m.taskPanelVisible
			if m.taskPanelVisible {
				// Close preview when opening task panel
				m.previewVisible = false
				m.previewUserEnabled = false
				m.previewContent = ""

				// Update displayed tasks for current cursor position
				m.updateTaskPanelForCursor()

				// If we have a config for this project but no data yet, trigger a refresh
				if len(m.taskGroups) == 0 && m.taskFocusedProject != "" {
					return m, taskRefreshCmd(m.taskProjectConfigs, m.taskCache, m.taskGlobalConfig, task.DefaultProviderTimeout)
				}
			} else {
				// Clear focus and search when hiding
				m.taskPanelFocused = false
				m.clearTaskSearchState()
			}
			return m, nil

		case "P":
			return m, m.togglePMView()

		case "m":
			if m.audioNotifier != nil {
				m.audioNotifier.SetMuted(!m.audioNotifier.IsMuted())
			}
			return m, nil

		case "S":
			// Open sound pack picker dialog (only when audio is configured)
			if m.audioNotifier != nil {
				m.dialogMode = DialogSoundPackPicker
				m.dialogError = ""
				m.soundPackCursor = 0
				m.soundPackScrollOffset = 0
				return m, loadSoundPacksCmd()
			}
			return m, nil

		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case taskTickMsg:
		// Periodic task refresh
		interval := taskDefaultRefreshInterval
		if m.taskGlobalConfig != nil && m.taskGlobalConfig.Tasks.Interval.Duration > 0 {
			interval = m.taskGlobalConfig.Tasks.Interval.Duration
		}
		if len(m.taskProjectConfigs) > 0 {
			return m, tea.Batch(
				taskRefreshCmd(m.taskProjectConfigs, m.taskCache, m.taskGlobalConfig, task.DefaultProviderTimeout),
				taskTickCmd(interval),
			)
		}
		return m, taskTickCmd(interval)

	case soundPacksMsg:
		if msg.err != nil {
			m.dialogError = msg.err.Error()
		} else {
			m.soundPacks = msg.packs
		}
		if m.audioNotifier != nil {
			m.activeSoundPack = m.audioNotifier.ActivePack()
		}
		return m, nil

	case tasksMsg:
		m.taskGroupsByProject = msg.groupsByProject
		m.taskErrors = msg.errors
		m.pmTaskResults = msg.resultsByProject
		m.taskRefreshing = false // Clear refreshing indicator
		// Update displayed groups if task panel is visible
		if m.taskPanelVisible {
			m.updateTaskPanelForCursor()
		}
		return m, nil

	case pmTickMsg:
		if m.pmEngine == nil {
			return m, pmTickCmd()
		}
		if m.pmRunInFlight {
			return m, pmTickCmd()
		}
		m.pmRunInFlight = true
		return m, tea.Batch(pmRunCmd(m.pmEngine, m.sessions, m.pmTaskResults), pmTickCmd())

	case pmOutputMsg:
		m.pmRunInFlight = false
		if msg.err == nil {
			m.pmOutput = msg.output
			m.pmLastError = ""
			// Scan events for PM invocation triggers.
			if m.pmInvoker != nil && !m.pmInvokeInFlight && msg.output != nil {
				if trigger := pmCheckTrigger(msg.output.Events); trigger != "" {
					m.pmInvokeInFlight = true
					return m, pmInvokeCmd(m.pmInvoker, trigger, msg.output.Snapshots, msg.output.Events)
				}
			}
			return m, nil
		}
		m.pmLastError = msg.err.Error()
		return m, nil

	case pmStreamMsg:
		m.pmInvokeStatus = msg.status
		return m, pmStreamReadCmd(msg.streamCh, msg.resultCh)

	case pmInvokeMsg:
		m.pmInvokeInFlight = false
		m.pmInvokeStatus = ""
		if msg.err == nil {
			m.pmBriefing = msg.briefing
			m.pmBriefingStale = msg.isStale
			m.pmLastError = ""
			return m, nil
		}
		m.pmLastError = msg.err.Error()
		return m, nil

	case taskConfigsMsg:
		m.taskProjectConfigs = msg.configs
		// Trigger task refresh if we have new configs
		if len(msg.configs) > 0 {
			return m, taskRefreshCmd(msg.configs, m.taskCache, m.taskGlobalConfig, task.DefaultProviderTimeout)
		}
		return m, nil

	case tickMsg:
		// On tick, poll sessions and schedule next tick
		// Also poll remote sessions if configured
		cmds := []tea.Cmd{pollSessions, tickCmd()}
		if m.SSHPool != nil && len(m.Remotes) > 0 {
			cmds = append(cmds, func() tea.Msg {
				return remoteSessionsMsg{sessions: remote.PollSessions(m.SSHPool, m.Remotes)}
			})
		}
		return m, tea.Batch(cmds...)

	case sessionsMsg:
		// Update local sessions while preserving remote sessions
		var remoteSessions []session.Info
		for _, s := range m.sessions {
			if s.Remote != "" {
				remoteSessions = append(remoteSessions, s)
			}
		}

		// Combine new local sessions with preserved remote sessions
		allSessions := append([]session.Info{}, msg...)
		allSessions = append(allSessions, remoteSessions...)
		session.SortSessions(allSessions)
		m.sessions = allSessions

		// Merge cached git info into sessions
		if m.gitCache != nil {
			for i := range m.sessions {
				if info, ok := m.gitCache[m.sessions[i].CWD]; ok {
					m.sessions[i].Git = info
				}
			}
		}
		// Merge cached resource metrics into sessions
		m.mergeResourceCache()
		m.detectStatusChanges(m.sessions)

		// Trigger immediate git poll if cache is empty and we have sessions
		// This makes git info appear quickly on startup instead of waiting for gitPollInterval
		needsGitPoll := len(m.gitCache) == 0 && len(m.sessions) > 0

		// Extract CWDs for task config discovery
		var currentCWDs []sessionCWD
		for _, s := range m.sessions {
			if s.CWD != "" {
				currentCWDs = append(currentCWDs, sessionCWD{cwd: s.CWD})
			}
		}
		cwdStrings := extractSessionCWDs(currentCWDs)
		cwdsChanged := !stringSlicesEqual(cwdStrings, m.taskLastCWDs)
		if cwdsChanged {
			m.taskLastCWDs = cwdStrings
		}

		var cmds []tea.Cmd

		// Try to restore cursor to last selected session if set
		filteredSessions := m.getFilteredSessions()
		if m.lastSelectedSession != "" {
			for i, s := range filteredSessions {
				if s.TmuxSession == m.lastSelectedSession {
					m.cursor = i
					m.lastSelectedSession = "" // Clear after restoring
					break
				}
			}
			if m.lastSelectedSession != "" {
				m.lastSelectedSession = "" // Session no longer exists, clear it
			}
		}

		// Clamp cursor if filtered sessions list shrunk
		if m.cursor >= len(filteredSessions) && len(filteredSessions) > 0 {
			m.cursor = len(filteredSessions) - 1
		} else if len(filteredSessions) == 0 {
			m.cursor = 0
		}

		// Recompute search matches when session list changes
		if m.searchQuery != "" {
			m.computeSearchMatches()
		}

		if needsGitPoll {
			cmds = append(cmds, m.pollAllGitInfoCmd())
		}

		// Trigger task config discovery if CWDs changed
		if cwdsChanged && len(currentCWDs) > 0 {
			cmds = append(cmds, discoverTaskConfigsCmd(currentCWDs, m.taskGlobalConfig))
		}

		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}

	case remoteSessionsMsg:
		// Merge remote sessions with existing local sessions
		if len(msg.sessions) > 0 {
			// Keep local sessions (Remote == ""), add remote sessions
			var localSessions []session.Info
			for _, s := range m.sessions {
				if s.Remote == "" {
					localSessions = append(localSessions, s)
				}
			}
			// Add remote sessions
			allSessions := append(localSessions, msg.sessions...)
			// Re-sort the combined list
			session.SortSessions(allSessions)
			m.sessions = allSessions

			// Merge cached git info into sessions
			if m.gitCache != nil {
				for i := range m.sessions {
					if info, ok := m.gitCache[m.sessions[i].CWD]; ok {
						m.sessions[i].Git = info
					}
				}
			}
			// Merge cached resource metrics into sessions
			m.mergeResourceCache()
			m.detectStatusChanges(m.sessions)

			// Clamp cursor for filtered sessions
			filteredSessions := m.getFilteredSessions()
			if m.cursor >= len(filteredSessions) && len(filteredSessions) > 0 {
				m.cursor = len(filteredSessions) - 1
			}

			// Poll remote git info after session list is loaded.
			// This runs after remote session polling completes, avoiding SSH mutex contention.
			if m.SSHPool != nil {
				if cmd := pollRemoteGitInfoCmd(m.SSHPool, m.sessions); cmd != nil {
					return m, cmd
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Handle preview visibility based on terminal width
		if m.width < previewMinTerminalWidth {
			// Auto-hide panels when terminal too narrow
			m.previewVisible = false
			m.taskPanelVisible = false
		} else if m.previewUserEnabled && !m.previewVisible {
			// Restore preview if user had it enabled and space now available
			m.previewVisible = true
			// Trigger capture if we have sessions
			filteredSessions := m.getFilteredSessions()
			if len(filteredSessions) > 0 && m.cursor < len(filteredSessions) {
				return m, tea.Batch(
					m.capturePreviewForSession(filteredSessions[m.cursor]),
					previewTickCmd(),
				)
			}
		} else if m.taskPanelUserEnabled && !m.taskPanelVisible {
			// Restore task panel if user had it enabled and space now available
			m.taskPanelVisible = true
			m.updateTaskPanelForCursor()
		}

	case attachDoneMsg:
		m.stopAttachMonitor()
		// After returning from tmux, trigger immediate refresh
		return m, pollSessions

	case createSessionResultMsg:
		if msg.err != nil {
			// Show error in dialog
			m.dialogError = "Failed to create session: " + msg.err.Error()
			return m, nil
		}
		// Success - close dialog and refresh
		m.dialogMode = DialogNone
		m.dialogError = ""
		return m, pollSessions

	case killSessionResultMsg:
		if msg.err != nil {
			// Show error in dialog
			m.dialogError = "Failed to kill session: " + msg.err.Error()
			return m, nil
		}
		// Success - close dialog and refresh
		m.dialogMode = DialogNone
		m.dialogError = ""
		m.sessionToModify = nil
		return m, pollSessions

	case renameSessionResultMsg:
		if msg.err != nil {
			// Show error in dialog
			m.dialogError = "Failed to rename session: " + msg.err.Error()
			return m, nil
		}
		// Success - close dialog and set lastSelectedSession for cursor preservation
		m.dialogMode = DialogNone
		m.dialogError = ""
		m.sessionToModify = nil
		m.lastSelectedSession = msg.newName // Preserve cursor position on renamed session
		return m, pollSessions

	case previewContentMsg:
		if msg.err == nil {
			m.previewContent = msg.content
			m.previewLastCapture = time.Now()
			// Auto-scroll is handled in renderPreview - when previewAutoScroll is true,
			// the render function always shows the bottom of content
		}
		// Silently ignore errors - preview just won't update
		return m, nil

	case previewTickMsg:
		// Periodic preview refresh
		filteredSessions := m.getFilteredSessions()
		if !m.previewVisible || len(filteredSessions) == 0 {
			// Don't continue polling if preview hidden or no sessions
			return m, nil
		}
		// Capture current session and schedule next tick
		if m.cursor < len(filteredSessions) {
			return m, tea.Batch(
				m.capturePreviewForSession(filteredSessions[m.cursor]),
				previewTickCmd(),
			)
		}
		return m, previewTickCmd()

	case previewDebounceMsg:
		// Debounced capture after cursor movement - reset preview scroll state
		m.previewScrollOffset = 0
		m.previewAutoScroll = true
		m.previewFocused = false
		filteredSessions := m.getFilteredSessions()
		if !m.previewVisible || len(filteredSessions) == 0 {
			return m, nil
		}
		if m.cursor < len(filteredSessions) {
			return m, m.capturePreviewForSession(filteredSessions[m.cursor])
		}
		return m, nil

	case prAutoRefreshTickMsg:
		// Only refresh if git detail is open and auto-refresh is active
		if !m.prAutoRefreshActive || m.dialogMode != DialogGitDetail {
			m.prAutoRefreshActive = false
			return m, nil
		}
		// Fetch updated PR data
		if m.sessionToModify != nil && m.sessionToModify.Git != nil {
			var fetchCmd tea.Cmd
			if m.sessionToModify.Remote != "" {
				fetchCmd = fetchRemotePRCmd(m.sessionToModify.CWD, m.sessionToModify.Git.Branch, m.sessionToModify.Git.Remote)
			} else if m.sessionToModify.CWD != "" {
				fetchCmd = fetchPRCmd(m.sessionToModify.CWD)
			}
			if fetchCmd != nil {
				return m, tea.Batch(fetchCmd, prAutoRefreshTickCmd())
			}
		}
		return m, prAutoRefreshTickCmd()

	case resourceTickMsg:
		// Periodic resource usage poll (30s interval)
		if len(m.sessions) == 0 {
			return m, resourceTickCmd()
		}
		// Copy sessions for the poll goroutine
		sessionsCopy := make([]session.Info, len(m.sessions))
		copy(sessionsCopy, m.sessions)
		return m, tea.Batch(pollResourceMetricsCmd(sessionsCopy), resourceTickCmd())

	case resourcePollMsg:
		// Update resource cache with new poll data
		if m.resourceCache == nil {
			m.resourceCache = make(map[string]int64)
		}
		for name, rss := range msg {
			m.resourceCache[name] = rss
		}
		// Merge cached resource data onto sessions
		m.mergeResourceCache()
		return m, nil

	case gitTickMsg:
		// Periodic git info refresh
		if len(m.sessions) == 0 {
			// No sessions, just schedule next tick
			return m, gitTickCmd()
		}
		// Poll git info for all sessions and schedule next tick
		return m, tea.Batch(m.pollAllGitInfoCmd(), gitTickCmd())

	case gitInfoMsg:
		// Initialize cache if nil
		if m.gitCache == nil {
			m.gitCache = make(map[string]*git.Info)
		}
		// Update git cache with new data
		for cwd, info := range msg.cache {
			m.gitCache[cwd] = info
		}
		// Update sessions with cached git info
		for i := range m.sessions {
			if info, ok := m.gitCache[m.sessions[i].CWD]; ok {
				m.sessions[i].Git = info
			}
		}
		return m, nil

	case gitPRMsg:
		// Update PR number and detail for the session being viewed (lazy-loaded)
		if m.sessionToModify != nil && m.sessionToModify.CWD == msg.cwd && m.sessionToModify.Git != nil {
			m.sessionToModify.Git.PRNum = msg.prNum
			m.sessionToModify.Git.PRDetail = msg.prDetail
		}
		// Also update the cache so it persists
		if m.gitCache != nil {
			if info, ok := m.gitCache[msg.cwd]; ok {
				info.PRNum = msg.prNum
				info.PRDetail = msg.prDetail
			}
		}
		// Start or stop auto-refresh based on check status
		if m.dialogMode == DialogGitDetail && msg.prDetail != nil {
			if msg.prDetail.CheckSummary.IsPending() && !m.prAutoRefreshActive {
				m.prAutoRefreshActive = true
				return m, prAutoRefreshTickCmd()
			} else if !msg.prDetail.CheckSummary.IsPending() {
				m.prAutoRefreshActive = false
			}
		}
		return m, nil

	case gitPRCommentsMsg:
		if msg.err != nil {
			m.dialogError = "Failed to fetch comments: " + msg.err.Error()
			return m, nil
		}
		// Open comments in content viewer
		if len(msg.comments) == 0 {
			m.dialogError = "No comments on this PR"
			return m, nil
		}
		var commentContent strings.Builder
		for _, c := range msg.comments {
			commentContent.WriteString(fmt.Sprintf("--- %s (%s) ---\n", c.Author, c.CreatedAt))
			if c.Type == git.CommentTypeReview && c.FilePath != "" {
				commentContent.WriteString(fmt.Sprintf("[%s:%d]\n", c.FilePath, c.Line))
			}
			commentContent.WriteString(c.Body)
			commentContent.WriteString("\n\n")
		}
		title := fmt.Sprintf("PR Comments (%d)", len(msg.comments))
		m.openContentViewerFrom(title, commentContent.String(), ContentModePlain, DialogGitDetail)
		return m, nil

	case remoteDismissResultMsg:
		// Remote dismiss completed - refresh sessions regardless of error
		// (errors are silent, same as local dismiss behavior)
		return m, pollSessions

	case remoteGitInfoMsg:
		if msg.err != nil {
			m.dialogError = "SSH error: " + msg.err.Error()
			return m, nil
		}
		if msg.info != nil {
			// Cache the remote git info
			if m.gitCache == nil {
				m.gitCache = make(map[string]*git.Info)
			}
			m.gitCache[msg.cwd] = msg.info

			// Update the session being viewed
			if m.sessionToModify != nil && m.sessionToModify.CWD == msg.cwd {
				m.sessionToModify.Git = msg.info
			}

			// Also update any sessions with matching CWD
			for i := range m.sessions {
				if m.sessions[i].CWD == msg.cwd && m.sessions[i].Remote != "" {
					m.sessions[i].Git = msg.info
				}
			}

			// Trigger PR fetch using the branch and remote URL via gh -R flag
			return m, fetchRemotePRCmd(msg.cwd, msg.info.Branch, msg.info.Remote)
		}
		return m, nil
	}
	return m, nil
}

// Empty state message constant
// pollAllGitInfoCmd returns a batched command that polls git info for local sessions
// and triggers remote git polling if remote sessions exist.
// Remote git info is fetched separately to avoid mutex contention with remote session polling.
func (m Model) pollAllGitInfoCmd() tea.Cmd {
	return pollGitInfoCmd(m.sessions)
}

const noSessionsMessage = "  No active sessions"

// getPreviewWidth returns the width to use for the preview pane (side layout).
func (m Model) getPreviewWidth() int {
	if m.previewWidth > 0 {
		return m.previewWidth
	}
	// Default to percentage of terminal width
	return m.width * previewDefaultWidthPercent / 100
}

// updateTaskPanelForCursor updates the task panel's focused project and displayed groups
// based on the currently selected session.
func (m *Model) updateTaskPanelForCursor() {
	filteredSessions := m.getFilteredSessions()
	oldProject := m.taskFocusedProject
	if m.cursor < len(filteredSessions) {
		m.taskFocusedProject = findProjectForCWD(filteredSessions[m.cursor].CWD, m.taskProjectConfigs)
	} else {
		m.taskFocusedProject = ""
	}

	if m.taskFocusedProject != "" {
		m.taskGroups = m.taskGroupsByProject[m.taskFocusedProject]
	} else {
		m.taskGroups = nil
	}

	// Reset scroll offset when project changes
	if m.taskFocusedProject != oldProject {
		m.taskScrollOffset = 0
	}
}

// getTaskPanelHeight returns the height to use for the task panel.
func (m Model) getTaskPanelHeight() int {
	if m.taskPanelHeight > 0 {
		return m.taskPanelHeight
	}
	// Default to percentage of available content height
	contentHeight := m.height - 8
	return contentHeight * previewDefaultHeightPercent / 100
}

// togglePreview toggles the preview pane, closing the task panel if needed.
func (m Model) togglePreview() (tea.Model, tea.Cmd) {
	m.previewVisible = !m.previewVisible
	m.previewUserEnabled = m.previewVisible
	if m.previewVisible {
		// Close task panel when opening preview
		m.taskPanelVisible = false
		m.taskPanelUserEnabled = false
		m.taskPanelFocused = false
		m.taskSearchMode = false
		m.taskSearchQuery = ""
		// Reset preview scroll state
		m.previewScrollOffset = 0
		m.previewAutoScroll = true
		m.previewFocused = false
	} else {
		m.previewFocused = false
	}
	filteredSessions := m.getFilteredSessions()
	if m.previewVisible && len(filteredSessions) > 0 && m.cursor < len(filteredSessions) {
		m.previewWrap = true
		m.previewLayout = PreviewLayoutBottom
		m.previewLastCursor = m.cursor
		return m, tea.Batch(
			m.capturePreviewForSession(filteredSessions[m.cursor]),
			previewTickCmd(),
		)
	}
	m.previewContent = ""
	return m, nil
}

// updatePreviewFocus handles key messages when the preview pane has focus.
func (m Model) updatePreviewFocus(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "esc":
		m.previewFocused = false
		return m, nil

	case "down", "j":
		m.previewScrollOffset++
		m.previewAutoScroll = false
		return m, nil

	case "up", "k":
		if m.previewScrollOffset > 0 {
			m.previewScrollOffset--
		}
		m.previewAutoScroll = false
		return m, nil

	case "pgdown":
		m.previewScrollOffset += previewPageScrollAmt
		m.previewAutoScroll = false
		return m, nil

	case "pgup":
		m.previewScrollOffset -= previewPageScrollAmt
		if m.previewScrollOffset < 0 {
			m.previewScrollOffset = 0
		}
		m.previewAutoScroll = false
		return m, nil

	case "g":
		m.previewScrollOffset = 0
		m.previewAutoScroll = false
		return m, nil

	case "G":
		// Jump to bottom and re-enable auto-scroll
		m.previewAutoScroll = true
		return m, nil

	case "q", "ctrl+c":
		return m, tea.Quit
	}

	return m, nil
}

// updateTaskPanelFocus handles key messages when the task panel has focus.
func (m Model) updateTaskPanelFocus(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Route to task search mode if active
	if m.taskSearchMode {
		return m.updateTaskSearchMode(msg)
	}

	switch msg.String() {
	case "tab":
		// Return focus to session list
		m.taskPanelFocused = false
		return m, nil

	case "esc":
		// Clear task search first if active, then return focus
		if m.taskSearchQuery != "" {
			m.clearTaskSearchState()
			m.taskCursor = 0
			return m, nil
		}
		m.taskPanelFocused = false
		return m, nil

	case "T":
		// Close task panel entirely
		m.taskPanelVisible = false
		m.taskPanelUserEnabled = false
		m.taskPanelFocused = false
		return m, nil

	case "n":
		// Jump to next task match when search query is active
		if m.taskSearchQuery != "" {
			m.nextTaskMatch()
			m.ensureTaskCursorVisible(m.taskPanelViewportLines())
			return m, nil
		}
		return m, nil

	case "N":
		// Jump to previous task match when search query is active
		if m.taskSearchQuery != "" {
			m.prevTaskMatch()
			m.ensureTaskCursorVisible(m.taskPanelViewportLines())
			return m, nil
		}
		return m, nil

	case "up", "k":
		m.moveTaskCursor(-1)
		m.ensureTaskCursorVisible(m.taskPanelViewportLines())
		return m, nil

	case "down", "j":
		m.moveTaskCursor(1)
		m.ensureTaskCursorVisible(m.taskPanelViewportLines())
		return m, nil

	case "pgup":
		maxLines := m.taskPanelViewportLines()
		m.taskScrollOffset -= taskPanelPageScrollAmt
		if m.taskScrollOffset < 0 {
			m.taskScrollOffset = 0
		}
		m.taskCursor = m.taskScrollOffset
		m.ensureTaskCursorVisible(maxLines)
		return m, nil

	case "pgdown":
		maxLines := m.taskPanelViewportLines()
		items := m.getVisibleTaskItems()
		m.taskScrollOffset += taskPanelPageScrollAmt
		maxScroll := taskPanelMaxScroll(len(items), maxLines)
		if m.taskScrollOffset > maxScroll {
			m.taskScrollOffset = maxScroll
		}
		m.taskCursor = m.taskScrollOffset
		m.ensureTaskCursorVisible(maxLines)
		return m, nil

	case "g":
		// Jump to top
		m.taskCursor = 0
		m.taskScrollOffset = 0
		return m, nil

	case "G":
		// Jump to bottom
		items := m.getVisibleTaskItems()
		if len(items) > 0 {
			m.taskCursor = len(items) - 1
			maxLines := m.taskPanelViewportLines()
			m.ensureTaskCursorVisible(maxLines)
		}
		return m, nil

	case "enter":
		// Open task detail or toggle group
		item := m.getSelectedTaskItem()
		if item == nil {
			return m, nil
		}
		if item.isGroup {
			// Toggle group expansion (same as space)
			m.toggleGroupExpansion(item.groupID)
			return m, nil
		}
		// Task item: open URL externally or file in content viewer
		return m.openTaskDetail(item)

	case " ":
		// Toggle group expansion
		item := m.getSelectedTaskItem()
		if item != nil && item.isGroup {
			m.toggleGroupExpansion(item.groupID)
		}
		return m, nil

	case "s":
		// Cycle sort mode: source → status → name → progress → source
		currentItem := m.getSelectedTaskItem()
		m.taskSortMode = nextTaskSortMode(m.taskSortMode)
		// Cursor stability: try to keep cursor on the same item
		m.preserveTaskCursor(currentItem)
		return m, nil

	case "S":
		// Toggle sort direction (ascending/descending)
		m.taskSortReversed = !m.taskSortReversed
		return m, nil

	case "f":
		// Cycle filter mode: all → active → incomplete → all
		currentItem := m.getSelectedTaskItem()
		m.taskFilterMode = nextTaskFilterMode(m.taskFilterMode)
		// Cursor stability: try to keep cursor on the same item
		m.preserveTaskCursor(currentItem)
		// Recompute search matches within new filtered set
		if m.taskSearchQuery != "" {
			m.computeTaskSearchMatches()
		}
		return m, nil

	case "J":
		// Jump to next group header
		m.jumpToNextGroup()
		m.ensureTaskCursorVisible(m.taskPanelViewportLines())
		return m, nil

	case "K":
		// Jump to previous group header
		m.jumpToPrevGroup()
		m.ensureTaskCursorVisible(m.taskPanelViewportLines())
		return m, nil

	case "e":
		// Toggle expand/collapse all groups
		m.toggleExpandCollapseAll()
		m.clampTaskScrollOffset(m.taskPanelViewportLines())
		m.ensureTaskCursorVisible(m.taskPanelViewportLines())
		return m, nil

	case "a":
		// Toggle accordion mode
		m.taskAccordionMode = !m.taskAccordionMode
		return m, nil

	case "r":
		// Manual refresh: invalidate cache and re-execute provider
		if m.taskRefreshing {
			return m, nil // Ignore if already refreshing
		}
		m.taskRefreshing = true
		// Invalidate cache for current project
		if m.taskFocusedProject != "" && m.taskCache != nil {
			m.taskCache.Invalidate(m.taskFocusedProject)
		}
		return m, taskRefreshCmd(m.taskProjectConfigs, m.taskCache, m.taskGlobalConfig, task.DefaultProviderTimeout)

	case "/":
		// Enter task search mode
		m.taskSearchMode = true
		m.taskSearchInput.SetValue("")
		m.taskSearchQuery = ""
		m.taskSearchInput.Focus()
		return m, nil

	case "q", "ctrl+c":
		return m, tea.Quit
	}

	return m, nil
}

// openTaskDetail handles Enter on a task item: opens URL externally or file in content viewer.
func (m Model) openTaskDetail(item *taskItem) (tea.Model, tea.Cmd) {
	// Tasks with a URL: open externally
	if item.url != "" {
		if err := git.OpenURL(item.url); err != nil {
			m.dialogError = "Failed to open URL: " + err.Error()
		}
		return m, nil
	}

	// Local markdown tasks: derive file path from project dir and task ID
	if m.taskFocusedProject == "" {
		return m, nil
	}

	// Task file path: <projectDir>/docs/delivery/<pbi-num>/<taskID>.md
	// Extract PBI number from group ID (e.g. "PBI-29" -> "29")
	pbiNum := strings.TrimPrefix(item.groupID, "PBI-")
	taskFilePath := filepath.Join(m.taskFocusedProject, "docs", "delivery", pbiNum, item.taskID+".md")

	content, err := os.ReadFile(taskFilePath)
	if err != nil {
		// Show error in content viewer
		m.openContentViewer("Error", fmt.Sprintf("Could not read file:\n%s\n\nError: %s", taskFilePath, err.Error()), ContentModePlain)
		return m, nil
	}

	m.openContentViewer(item.title, string(content), ContentModePlain)
	return m, nil
}

// updateTaskSearchMode handles key messages when task search is active within the focused panel.
func (m Model) updateTaskSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Clear all task search state (stay in focus mode)
		m.clearTaskSearchState()
		m.taskCursor = 0
		m.taskScrollOffset = 0
		return m, nil

	case "up", "k":
		m.moveTaskCursor(-1)
		m.ensureTaskCursorVisible(m.taskPanelViewportLines())
		return m, nil

	case "down", "j":
		m.moveTaskCursor(1)
		m.ensureTaskCursorVisible(m.taskPanelViewportLines())
		return m, nil

	case "enter":
		// Exit input mode but keep search state (search persists like vim)
		m.taskSearchMode = false
		m.taskSearchInput.Blur()
		m.computeTaskSearchMatches()
		return m, nil

	case " ":
		// Toggle group expansion even during search
		item := m.getSelectedTaskItem()
		if item != nil && item.isGroup {
			m.toggleGroupExpansion(item.groupID)
		}
		return m, nil
	}

	// Forward all other keys to search input
	var cmd tea.Cmd
	m.taskSearchInput, cmd = m.taskSearchInput.Update(msg)
	m.taskSearchQuery = m.taskSearchInput.Value()

	// Recompute matches after query change
	m.computeTaskSearchMatches()

	// Jump cursor to first match if matches exist
	if len(m.taskSearchMatches) > 0 {
		m.taskCurrentMatchIdx = 0
		m.taskCursor = m.taskSearchMatches[0]
	}

	// Clamp cursor after query change
	items := m.getVisibleTaskItems()
	if m.taskCursor >= len(items) {
		if len(items) > 0 {
			m.taskCursor = len(items) - 1
		} else {
			m.taskCursor = 0
		}
	}

	m.ensureTaskCursorVisible(m.taskPanelViewportLines())

	return m, cmd
}

// taskItem represents a navigable item in the task panel (group header or task).
type taskItem struct {
	isGroup bool
	groupID string
	title   string
	status  string
	taskID  string
	url     string // URL for external tasks (empty for local markdown tasks)
	number  int    // Sequential group number (1-based, 0 for tasks)
}

// getVisibleTaskItems returns the flat list of navigable items based on expand state.
// Applies the current filter and sort pipeline before building the item list.
// When a task search query is active, all items remain visible but groups containing
// matches are auto-expanded. Match highlighting is handled separately via taskSearchMatches.
func (m Model) getVisibleTaskItems() []taskItem {
	var items []taskItem
	groupNum := 0

	// Apply filter and sort pipeline
	groups := m.getSortedAndFilteredTaskGroups()

	if m.taskSearchQuery != "" {
		// Search mode: show all groups, auto-expand groups that contain matches
		for _, g := range groups {
			groupMatches := exactMatch(m.taskSearchQuery, g.Title)

			// Check if any tasks in this group match
			hasMatchingTasks := false
			for _, t := range g.Tasks {
				if exactMatch(m.taskSearchQuery, t.Title) || exactMatch(m.taskSearchQuery, t.ID) {
					hasMatchingTasks = true
					break
				}
			}

			groupNum++
			items = append(items, taskItem{
				isGroup: true,
				groupID: g.ID,
				title:   g.Title,
				status:  g.Status,
				url:     g.URL,
				number:  groupNum,
			})

			// Auto-expand groups that contain matches or match themselves
			shouldExpand := groupMatches || hasMatchingTasks || m.taskExpandedGroups[g.ID]
			if shouldExpand {
				for _, t := range g.Tasks {
					items = append(items, taskItem{
						isGroup: false,
						groupID: g.ID,
						taskID:  t.ID,
						title:   t.Title,
						status:  t.Status,
						url:     t.URL,
					})
				}
			}
		}
		return items
	}

	// Normal mode: respect expand/collapse state
	for _, g := range groups {
		groupNum++
		items = append(items, taskItem{
			isGroup: true,
			groupID: g.ID,
			title:   g.Title,
			status:  g.Status,
			url:     g.URL,
			number:  groupNum,
		})
		if m.taskExpandedGroups[g.ID] {
			for _, t := range g.Tasks {
				items = append(items, taskItem{
					isGroup: false,
					groupID: g.ID,
					taskID:  t.ID,
					title:   t.Title,
					status:  t.Status,
					url:     t.URL,
				})
			}
		}
	}
	return items
}

// computeTaskSearchMatches recomputes the list of matching task item indices.
func (m *Model) computeTaskSearchMatches() {
	items := m.getVisibleTaskItems()
	m.taskSearchMatches = nil
	if m.taskSearchQuery == "" {
		m.taskCurrentMatchIdx = 0
		return
	}
	for i, item := range items {
		if item.isGroup {
			if exactMatch(m.taskSearchQuery, item.title) {
				m.taskSearchMatches = append(m.taskSearchMatches, i)
			}
		} else {
			if exactMatch(m.taskSearchQuery, item.title) || exactMatch(m.taskSearchQuery, item.taskID) {
				m.taskSearchMatches = append(m.taskSearchMatches, i)
			}
		}
	}
	if len(m.taskSearchMatches) == 0 {
		m.taskCurrentMatchIdx = 0
	} else if m.taskCurrentMatchIdx >= len(m.taskSearchMatches) {
		m.taskCurrentMatchIdx = 0
	}
}

// nextTaskMatch moves the task cursor to the next search match, wrapping around.
func (m *Model) nextTaskMatch() {
	if len(m.taskSearchMatches) == 0 {
		return
	}
	m.taskCurrentMatchIdx = (m.taskCurrentMatchIdx + 1) % len(m.taskSearchMatches)
	m.taskCursor = m.taskSearchMatches[m.taskCurrentMatchIdx]
}

// prevTaskMatch moves the task cursor to the previous search match, wrapping around.
func (m *Model) prevTaskMatch() {
	if len(m.taskSearchMatches) == 0 {
		return
	}
	m.taskCurrentMatchIdx--
	if m.taskCurrentMatchIdx < 0 {
		m.taskCurrentMatchIdx = len(m.taskSearchMatches) - 1
	}
	m.taskCursor = m.taskSearchMatches[m.taskCurrentMatchIdx]
}

// clearTaskSearchState resets all task search-related fields.
func (m *Model) clearTaskSearchState() {
	m.taskSearchMode = false
	m.taskSearchQuery = ""
	m.taskSearchInput.SetValue("")
	m.taskSearchInput.Blur()
	m.taskSearchMatches = nil
	m.taskCurrentMatchIdx = 0
}

// getSelectedTaskItem returns the currently selected task item, or nil.
func (m Model) getSelectedTaskItem() *taskItem {
	items := m.getVisibleTaskItems()
	if m.taskCursor >= 0 && m.taskCursor < len(items) {
		return &items[m.taskCursor]
	}
	return nil
}

// jumpToNextGroup moves the task cursor to the next group header, wrapping at bounds.
func (m *Model) jumpToNextGroup() {
	items := m.getVisibleTaskItems()
	if len(items) == 0 {
		return
	}
	// Search forward from current position for next group header
	for i := m.taskCursor + 1; i < len(items); i++ {
		if items[i].isGroup {
			m.taskCursor = i
			return
		}
	}
	// Wrap to first group
	for i := 0; i <= m.taskCursor; i++ {
		if items[i].isGroup {
			m.taskCursor = i
			return
		}
	}
}

// jumpToPrevGroup moves the task cursor to the previous group header, wrapping at bounds.
func (m *Model) jumpToPrevGroup() {
	items := m.getVisibleTaskItems()
	if len(items) == 0 {
		return
	}
	// Search backward from current position for previous group header
	for i := m.taskCursor - 1; i >= 0; i-- {
		if items[i].isGroup {
			m.taskCursor = i
			return
		}
	}
	// Wrap to last group
	for i := len(items) - 1; i >= m.taskCursor; i-- {
		if items[i].isGroup {
			m.taskCursor = i
			return
		}
	}
}

// toggleExpandCollapseAll expands all groups if any are collapsed, or collapses all if all are expanded.
func (m *Model) toggleExpandCollapseAll() {
	groups := m.getSortedAndFilteredTaskGroups()
	if len(groups) == 0 {
		return
	}

	// Check if any group is collapsed
	anyCollapsed := false
	for _, g := range groups {
		if !m.taskExpandedGroups[g.ID] {
			anyCollapsed = true
			break
		}
	}

	if anyCollapsed {
		// Expand all
		for _, g := range groups {
			m.taskExpandedGroups[g.ID] = true
		}
	} else {
		// Collapse all
		for _, g := range groups {
			delete(m.taskExpandedGroups, g.ID)
		}
	}
}

// toggleGroupExpansion toggles a group's expanded state, applying accordion mode if active.
// Also clamps cursor and scroll offset after the change.
func (m *Model) toggleGroupExpansion(groupID string) {
	if m.taskExpandedGroups[groupID] {
		// Collapsing
		delete(m.taskExpandedGroups, groupID)
	} else {
		// Expanding
		if m.taskAccordionMode {
			m.applyAccordion(groupID)
		} else {
			m.taskExpandedGroups[groupID] = true
		}
	}
	// Clamp cursor and scroll after expansion change
	items := m.getVisibleTaskItems()
	if m.taskCursor >= len(items) && len(items) > 0 {
		m.taskCursor = len(items) - 1
	}
	m.clampTaskScrollOffset(m.taskPanelViewportLines())
	m.ensureTaskCursorVisible(m.taskPanelViewportLines())
}

// applyAccordion collapses all groups except the specified one.
func (m *Model) applyAccordion(expandGroupID string) {
	groups := m.getSortedAndFilteredTaskGroups()
	for _, g := range groups {
		if g.ID == expandGroupID {
			m.taskExpandedGroups[g.ID] = true
		} else {
			delete(m.taskExpandedGroups, g.ID)
		}
	}
}

// preserveTaskCursor attempts to keep the cursor on the same item after a sort/filter change.
// If the item is no longer visible, moves to the nearest valid position.
func (m *Model) preserveTaskCursor(prevItem *taskItem) {
	items := m.getVisibleTaskItems()
	if len(items) == 0 {
		m.taskCursor = 0
		m.taskScrollOffset = 0
		return
	}

	if prevItem != nil {
		// Try to find the same item in the new list
		for i, item := range items {
			if item.isGroup == prevItem.isGroup && item.groupID == prevItem.groupID && item.taskID == prevItem.taskID {
				m.taskCursor = i
				m.ensureTaskCursorVisible(m.taskPanelViewportLines())
				return
			}
		}
	}

	// Item not found — clamp cursor
	if m.taskCursor >= len(items) {
		m.taskCursor = len(items) - 1
	}
	m.ensureTaskCursorVisible(m.taskPanelViewportLines())
}

// moveTaskCursor moves the task cursor by delta, wrapping at bounds.
// After moving, it ensures the cursor is visible within the scroll viewport.
func (m *Model) moveTaskCursor(delta int) {
	items := m.getVisibleTaskItems()
	if len(items) == 0 {
		m.taskCursor = 0
		m.taskScrollOffset = 0
		return
	}
	m.taskCursor += delta
	if m.taskCursor < 0 {
		m.taskCursor = len(items) - 1
	} else if m.taskCursor >= len(items) {
		m.taskCursor = 0
	}
}

// taskPanelMaxScroll returns the maximum valid scroll offset for the task panel.
// When scrolling is active, the top scroll indicator takes 1 line from content,
// so at maximum scroll only maxLines-1 items are visible.
func taskPanelMaxScroll(totalItems, maxLines int) int {
	if totalItems <= maxLines {
		return 0
	}
	// At max scroll, the top indicator is shown, reducing visible items by 1.
	// To show the last item: maxScroll + (maxLines - 1) >= totalItems
	// So maxScroll = totalItems - maxLines + 1
	return totalItems - maxLines + 1
}

// ensureTaskCursorVisible adjusts taskScrollOffset so the cursor is within the visible viewport.
// Uses a two-pass approach to account for scroll indicators reducing the effective viewport.
func (m *Model) ensureTaskCursorVisible(maxLines int) {
	if maxLines <= 0 {
		return
	}
	items := m.getVisibleTaskItems()

	// If cursor is above viewport, scroll up
	if m.taskCursor < m.taskScrollOffset {
		m.taskScrollOffset = m.taskCursor
	}
	// If cursor is below viewport, scroll down (first pass with raw maxLines)
	if m.taskCursor >= m.taskScrollOffset+maxLines {
		m.taskScrollOffset = m.taskCursor - maxLines + 1
	}

	// Second pass: account for scroll indicators reducing visible content.
	// When scrollOffset > 0, the top indicator takes 1 line; when items extend
	// below the visible area, the bottom indicator takes 1 line.
	if len(items) > maxLines {
		effective := maxLines
		if m.taskScrollOffset > 0 {
			effective--
		}
		if m.taskScrollOffset+effective < len(items) {
			effective--
		}
		if effective < 1 {
			effective = 1
		}
		if m.taskCursor >= m.taskScrollOffset+effective {
			m.taskScrollOffset = m.taskCursor - effective + 1
		}
	}

	// Clamp scroll offset
	maxScroll := taskPanelMaxScroll(len(items), maxLines)
	if m.taskScrollOffset > maxScroll {
		m.taskScrollOffset = maxScroll
	}
	if m.taskScrollOffset < 0 {
		m.taskScrollOffset = 0
	}
}

// clampTaskScrollOffset clamps the task scroll offset after item count changes.
func (m *Model) clampTaskScrollOffset(maxLines int) {
	items := m.getVisibleTaskItems()
	maxScroll := taskPanelMaxScroll(len(items), maxLines)
	if m.taskScrollOffset > maxScroll {
		m.taskScrollOffset = maxScroll
	}
	if m.taskScrollOffset < 0 {
		m.taskScrollOffset = 0
	}
}

// taskPanelHeaderLines returns the number of lines used by the task panel header.
// Returns 2 when the summary stats line is shown, 1 otherwise.
func (m Model) taskPanelHeaderLines() int {
	height := m.getTaskPanelHeight()
	contentLines := height - 3 // base header(1) + borders(2)
	if contentLines >= 4 && len(m.taskGroups) > 0 {
		return 2 // header + summary line
	}
	return 1
}

// taskPanelViewportLines returns the number of content lines available in the task panel.
func (m Model) taskPanelViewportLines() int {
	height := m.getTaskPanelHeight()
	headerLines := m.taskPanelHeaderLines()
	maxLines := height - headerLines - 2 // header(N) + borders(2)
	if m.taskSearchMode || m.taskSearchQuery != "" {
		maxLines-- // search bar takes 1 line
	}
	if maxLines < 1 {
		maxLines = 1
	}
	return maxLines
}

// getPreviewHeight returns the height to use for the preview pane (bottom layout).
func (m Model) getPreviewHeight() int {
	if m.previewHeight > 0 {
		return m.previewHeight
	}
	// Default to percentage of available content height
	contentHeight := m.height - 8
	return contentHeight * previewDefaultHeightPercent / 100
}

func (m *Model) startAttachMonitor() {
	m.stopAttachMonitor()

	ctx, cancel := context.WithCancel(context.Background())
	mon := monitor.New(m.audioNotifier, pathutil.ExpandPath(session.StatusDir), session.PollInterval)
	mon.Start(ctx, m.lastSessionStates, m.lastAgentStates)

	m.attachMonitor = mon
	m.attachMonitorCancel = cancel
}

func (m *Model) stopAttachMonitor() {
	if m.attachMonitorCancel != nil {
		m.attachMonitorCancel()
	}
	if m.attachMonitor != nil {
		m.lastSessionStates = m.attachMonitor.States()
		m.lastAgentStates = m.attachMonitor.AgentStates()
	}
	m.attachMonitor = nil
	m.attachMonitorCancel = nil
}

// loadSoundPacksCmd returns a tea.Cmd that loads available sound packs.
func loadSoundPacksCmd() tea.Cmd {
	return func() tea.Msg {
		packs, err := audio.ListPacks()
		return soundPacksMsg{packs: packs, err: err}
	}
}

// soundPackPickerMaxVisible is the maximum number of pack rows visible in the picker viewport.
const soundPackPickerMaxVisible = 10

// updateSoundPackPicker handles keyboard input for the sound pack picker dialog.
func (m Model) updateSoundPackPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.dialogMode = DialogNone
		m.dialogError = ""
		return m, nil

	case "up", "k":
		if len(m.soundPacks) > 0 && m.soundPackCursor > 0 {
			m.soundPackCursor--
			if m.soundPackCursor < m.soundPackScrollOffset {
				m.soundPackScrollOffset = m.soundPackCursor
			}
		}
		return m, nil

	case "down", "j":
		if len(m.soundPacks) > 0 && m.soundPackCursor < len(m.soundPacks)-1 {
			m.soundPackCursor++
			if m.soundPackCursor >= m.soundPackScrollOffset+soundPackPickerMaxVisible {
				m.soundPackScrollOffset = m.soundPackCursor - soundPackPickerMaxVisible + 1
			}
		}
		return m, nil

	case "enter":
		if len(m.soundPacks) == 0 {
			return m, nil
		}
		selected := m.soundPacks[m.soundPackCursor]
		if m.audioNotifier != nil {
			if err := m.audioNotifier.SetPack(selected.Name); err != nil {
				m.dialogError = fmt.Sprintf("Failed to switch pack: %v", err)
				return m, nil
			}
		}
		configPath := pathutil.ExpandPath(audio.DefaultConfigPath)
		if err := audio.SavePackSelection(configPath, selected.Name); err != nil {
			m.dialogError = fmt.Sprintf("Pack applied but failed to save: %v", err)
			return m, nil
		}
		m.activeSoundPack = selected.Name
		m.dialogMode = DialogNone
		m.dialogError = ""
		return m, nil

	case "p", " ":
		// Preview: play a sample sound from the highlighted pack
		if len(m.soundPacks) == 0 || m.audioNotifier == nil {
			return m, nil
		}
		selected := m.soundPacks[m.soundPackCursor]
		return m, previewSoundPackCmd(selected.Name)
	}

	return m, nil
}

// previewSoundPackCmd returns a tea.Cmd that plays a sample sound from the given pack.
func previewSoundPackCmd(packName string) tea.Cmd {
	return func() tea.Msg {
		files, err := audio.ResolveSoundFiles(packName)
		if err != nil || len(files) == 0 {
			return nil
		}
		// Pick the first available event's first file
		for _, paths := range files {
			if len(paths) > 0 {
				p := audio.NewPlayer("")
				if p.Available() {
					_ = p.Play(paths[0], 100)
				}
				break
			}
		}
		return nil
	}
}

// attachSession returns a command that attaches to a local tmux session.
// Uses tea.ExecProcess to hand off terminal control to tmux.
func attachSession(name string) tea.Cmd {
	c := exec.Command("tmux", "attach-session", "-t", name)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return attachDoneMsg{}
	})
}

// attachRemoteSession returns a command that attaches to a remote tmux session via SSH.
// Uses tea.ExecProcess to hand off terminal control to SSH with tmux.
func attachRemoteSession(r *remote.Config, sessionName string) tea.Cmd {
	args := remote.BuildSSHAttachCommand(r, sessionName)
	// First arg is "ssh", rest are arguments
	c := exec.Command(args[0], args[1:]...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return attachDoneMsg{}
	})
}

// capturePreviewCmd returns a command that captures preview content from a tmux session.
func capturePreviewCmd(sessionName string) tea.Cmd {
	return func() tea.Msg {
		content, err := capturePane(sessionName, previewDefaultLines)
		return previewContentMsg{content: content, err: err}
	}
}

// updateDialog handles key messages when a dialog is open.
func (m Model) updateDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Route content viewer keys to its own handler
	if m.dialogMode == DialogContentViewer {
		return m.updateContentViewer(msg)
	}

	// Route sound pack picker keys to its own handler
	if m.dialogMode == DialogSoundPackPicker {
		return m.updateSoundPackPicker(msg)
	}

	switch msg.String() {
	case "esc":
		// Close any dialog and reset state
		m.dialogMode = DialogNone
		m.dialogError = ""
		m.sessionToModify = nil
		m.prAutoRefreshActive = false // Stop auto-refresh when closing dialog
		return m, nil

	case "tab":
		// Switch focus between inputs in new session dialog
		if m.dialogMode == DialogNewSession {
			m.focusedInput = (m.focusedInput + 1) % 3
			m.nameInput.Blur()
			m.dirInput.Blur()
			switch m.focusedInput {
			case focusName:
				m.nameInput.Focus()
			case focusDir:
				m.dirInput.Focus()
				// focusSkipPerms - no text input to focus
			}
			return m, nil
		}

	case " ":
		// Toggle skip permissions checkbox
		if m.dialogMode == DialogNewSession && m.focusedInput == focusSkipPerms {
			m.skipPermissions = !m.skipPermissions
			return m, nil
		}

	case "enter", "o":
		// Handle submission based on dialog type
		switch m.dialogMode {
		case DialogNewSession:
			if msg.String() == "enter" {
				return m.submitNewSession()
			}
		case DialogRename:
			if msg.String() == "enter" {
				return m.submitRename()
			}
		case DialogGitDetail:
			// Open PR/issue link if available
			return m.openGitLink()
		}

	case "y":
		// Confirm kill
		if m.dialogMode == DialogKillConfirm && m.sessionToModify != nil {
			// Remote kill via SSH
			if m.sessionToModify.Remote != "" && m.SSHPool != nil {
				return m, killRemoteSessionCmd(m.SSHPool, m.sessionToModify.Remote, m.sessionToModify.TmuxSession)
			}
			return m, killSessionCmd(m.sessionToModify.TmuxSession)
		}

	case "n":
		// Cancel kill (same as escape)
		if m.dialogMode == DialogKillConfirm {
			m.dialogMode = DialogNone
			m.dialogError = ""
			m.sessionToModify = nil
			return m, nil
		}

	case "d":
		// Show diff in content viewer from git detail view
		if m.dialogMode == DialogGitDetail && m.sessionToModify != nil && m.sessionToModify.Git != nil {
			dir := pathutil.ExpandPath(m.sessionToModify.CWD)
			diff := git.GetDiff(dir)
			if diff == "" {
				if m.sessionToModify.Git.Dirty {
					diff = "No unstaged changes (changes may be staged)"
				} else {
					diff = "Working tree clean - no changes"
				}
			}
			title := fmt.Sprintf("Git Diff: %s", m.sessionToModify.Git.Branch)
			m.openContentViewerFrom(title, diff, ContentModeDiff, DialogGitDetail)
			return m, nil
		}

	case "r":
		// Refresh PR data from git detail view
		if m.dialogMode == DialogGitDetail && m.sessionToModify != nil && m.sessionToModify.Git != nil {
			// Reset auto-refresh timer (next auto-tick will be after full interval)
			m.prAutoRefreshActive = false
			if m.sessionToModify.Remote != "" {
				return m, fetchRemotePRCmd(m.sessionToModify.CWD, m.sessionToModify.Git.Branch, m.sessionToModify.Git.Remote)
			}
			if m.sessionToModify.CWD != "" {
				return m, fetchPRCmd(m.sessionToModify.CWD)
			}
		}

	case "c":
		// Fetch and show PR comments
		if m.dialogMode == DialogGitDetail && m.sessionToModify != nil && m.sessionToModify.Git != nil && m.sessionToModify.Git.PRNum > 0 {
			m.dialogError = "Loading comments..."
			if m.sessionToModify.Remote != "" {
				ghInfo := git.ParseGitHubRemote(m.sessionToModify.Git.Remote)
				if ghInfo != nil {
					return m, fetchRemotePRCommentsCmd(ghInfo.Owner, ghInfo.Repo, m.sessionToModify.Git.PRNum)
				}
			} else {
				return m, fetchPRCommentsCmd(m.sessionToModify.CWD, m.sessionToModify.Git.PRNum)
			}
		}
	}

	// Update the focused text input for dialogs that use text input
	var cmd tea.Cmd
	switch m.dialogMode {
	case DialogNewSession:
		if m.focusedInput == focusName {
			m.nameInput, cmd = m.nameInput.Update(msg)
		} else {
			m.dirInput, cmd = m.dirInput.Update(msg)
		}
	case DialogRename:
		m.nameInput, cmd = m.nameInput.Update(msg)
	}

	return m, cmd
}

// submitNewSession validates and creates a new tmux session.
func (m Model) submitNewSession() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.nameInput.Value())
	dir := strings.TrimSpace(m.dirInput.Value())

	// Use default name if empty
	if name == "" {
		name = getDefaultSessionName()
	}

	// Validate session name
	if err := validateSessionName(name, m.sessions); err != nil {
		m.dialogError = err.Error()
		return m, nil
	}

	// Validate directory
	if err := validateDirectory(dir); err != nil {
		m.dialogError = err.Error()
		return m, nil
	}

	// Use default directory if empty
	if dir == "" {
		dir = getDefaultDirectory()
	}

	// Expand home directory
	dir = pathutil.ExpandPath(dir)

	// Create the session
	return m, createSessionCmd(name, dir, m.skipPermissions)
}

// submitRename validates and renames a tmux session.
func (m Model) submitRename() (tea.Model, tea.Cmd) {
	if m.sessionToModify == nil {
		m.dialogError = "No session selected"
		return m, nil
	}

	oldName := m.sessionToModify.TmuxSession
	newName := strings.TrimSpace(m.nameInput.Value())

	// If same name, just close dialog
	if newName == oldName {
		m.dialogMode = DialogNone
		m.dialogError = ""
		m.sessionToModify = nil
		return m, nil
	}

	// Validate session name (exclude current session from duplicate check)
	sessionsWithoutCurrent := make([]session.Info, 0, len(m.sessions)-1)
	for _, s := range m.sessions {
		if s.TmuxSession != oldName {
			sessionsWithoutCurrent = append(sessionsWithoutCurrent, s)
		}
	}

	if err := validateSessionName(newName, sessionsWithoutCurrent); err != nil {
		m.dialogError = err.Error()
		return m, nil
	}

	// Remote rename via SSH
	if m.sessionToModify.Remote != "" && m.SSHPool != nil {
		return m, renameRemoteSessionCmd(m.SSHPool, m.sessionToModify.Remote, oldName, newName)
	}

	// Rename the local session
	return m, renameSessionCmd(oldName, newName)
}

// openGitLink opens the GitHub PR link in the system browser.
func (m Model) openGitLink() (tea.Model, tea.Cmd) {
	if m.sessionToModify == nil || m.sessionToModify.Git == nil {
		m.dialogError = "No git information available"
		return m, nil
	}

	g := m.sessionToModify.Git

	// Check if we have a PR number and a GitHub remote
	if g.PRNum == 0 {
		m.dialogError = "No PR found for this branch"
		return m, nil
	}

	if g.Remote == "" {
		m.dialogError = "No remote URL configured"
		return m, nil
	}

	ghInfo := git.ParseGitHubRemote(g.Remote)
	if ghInfo == nil {
		m.dialogError = "Remote is not a GitHub repository"
		return m, nil
	}

	// Construct the PR URL
	url := ghInfo.PRURL(g.PRNum)

	// Open the URL in the browser
	if err := git.OpenURL(url); err != nil {
		m.dialogError = "Failed to open browser: " + err.Error()
		return m, nil
	}

	// Close the dialog after successful open
	m.dialogMode = DialogNone
	m.dialogError = ""
	m.sessionToModify = nil
	return m, nil
}

// computeSearchMatches recomputes the list of matching session indices
// based on the current search query and filtered session list.
func (m *Model) computeSearchMatches() {
	filteredSessions := m.getFilteredSessions()
	m.searchMatches = findMatches(filteredSessions, m.searchQuery)
	// Clamp currentMatchIdx
	if len(m.searchMatches) == 0 {
		m.currentMatchIdx = 0
	} else if m.currentMatchIdx >= len(m.searchMatches) {
		m.currentMatchIdx = 0
	}
}

// nextMatch moves the cursor to the next search match, wrapping around.
func (m *Model) nextMatch() {
	if len(m.searchMatches) == 0 {
		return
	}
	m.currentMatchIdx = (m.currentMatchIdx + 1) % len(m.searchMatches)
	m.cursor = m.searchMatches[m.currentMatchIdx]
	m.ensureSessionCursorVisible(m.sessionListMaxVisible())
}

// prevMatch moves the cursor to the previous search match, wrapping around.
func (m *Model) prevMatch() {
	if len(m.searchMatches) == 0 {
		return
	}
	m.currentMatchIdx--
	if m.currentMatchIdx < 0 {
		m.currentMatchIdx = len(m.searchMatches) - 1
	}
	m.cursor = m.searchMatches[m.currentMatchIdx]
	m.ensureSessionCursorVisible(m.sessionListMaxVisible())
}

// clearSearchState resets all search-related fields.
func (m *Model) clearSearchState() {
	m.searchMode = false
	m.searchQuery = ""
	m.searchInput.SetValue("")
	m.searchInput.Blur()
	m.searchMatches = nil
	m.currentMatchIdx = 0
}

// updateSearchMode handles key messages when search mode is active.
// Navigation keys (up/down/enter) still work; other keys go to the search input.
func (m Model) updateSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Clear all search state and exit search mode
		m.clearSearchState()
		m.sessionScrollOffset = 0
		return m, nil

	case "up", "k":
		filteredSessions := m.getFilteredSessions()
		if len(filteredSessions) > 0 {
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(filteredSessions) - 1
			}
			m.ensureSessionCursorVisible(m.sessionListMaxVisible())
		}
		return m, nil

	case "down", "j":
		filteredSessions := m.getFilteredSessions()
		if len(filteredSessions) > 0 {
			if m.cursor < len(filteredSessions)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
			m.ensureSessionCursorVisible(m.sessionListMaxVisible())
		}
		return m, nil

	case "enter":
		// Exit input mode but keep search state (search persists like vim)
		m.searchMode = false
		m.searchInput.Blur()
		// Recompute matches so they persist after exiting input mode
		m.computeSearchMatches()

		filteredSessions := m.getFilteredSessions()
		if len(filteredSessions) > 0 && m.cursor < len(filteredSessions) {
			s := filteredSessions[m.cursor]
			m.lastSelectedSession = s.TmuxSession

			if s.Remote != "" && m.SSHPool != nil {
				r := m.SSHPool.GetRemoteConfig(s.Remote)
				if r != nil {
					m.startAttachMonitor()
					return m, attachRemoteSession(r, s.TmuxSession)
				}
			}
			m.startAttachMonitor()
			return m, attachSession(s.TmuxSession)
		}
		return m, nil
	}

	// Forward all other keys to search input
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.searchQuery = m.searchInput.Value()

	// Recompute matches after query change
	m.computeSearchMatches()

	// Jump cursor to first match if matches exist
	if len(m.searchMatches) > 0 {
		m.currentMatchIdx = 0
		m.cursor = m.searchMatches[0]
		m.ensureSessionCursorVisible(m.sessionListMaxVisible())
	}

	return m, cmd
}

// ensureSessionCursorVisible adjusts sessionScrollOffset so the cursor is within the visible viewport.
// Since session rows have variable height, this uses session index rather than line count.
// The maxSessions parameter is the maximum number of sessions that can fit in the viewport.
func (m *Model) ensureSessionCursorVisible(maxSessions int) {
	if maxSessions <= 0 {
		return
	}
	filteredSessions := m.getFilteredSessions()
	if len(filteredSessions) == 0 {
		m.sessionScrollOffset = 0
		return
	}
	// If cursor is above viewport, scroll up
	if m.cursor < m.sessionScrollOffset {
		m.sessionScrollOffset = m.cursor
	}
	// If cursor is below viewport, scroll down
	if m.cursor >= m.sessionScrollOffset+maxSessions {
		m.sessionScrollOffset = m.cursor - maxSessions + 1
	}
	// Clamp scroll offset
	maxScroll := len(filteredSessions) - maxSessions
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.sessionScrollOffset > maxScroll {
		m.sessionScrollOffset = maxScroll
	}
	if m.sessionScrollOffset < 0 {
		m.sessionScrollOffset = 0
	}
}

// sessionRowEstimatedHeight is the estimated number of lines per session row.
// Sessions have 2-5 lines depending on git info, metrics, and message.
const sessionRowEstimatedHeight = 3

// sessionListMaxVisible returns the estimated number of sessions that fit in the available height.
func (m Model) sessionListMaxVisible() int {
	contentHeight := m.height - 8 // Header(3) + Footer(3) + spacing
	if contentHeight < 5 {
		contentHeight = 5
	}

	sessionListHeight := contentHeight
	if m.taskPanelVisible && m.width >= previewMinTerminalWidth {
		panelHeight := m.getTaskPanelHeight()
		sessionListHeight = contentHeight - panelHeight - 1
	} else if m.previewVisible && m.width >= previewMinTerminalWidth {
		if m.previewLayout == PreviewLayoutBottom {
			previewHeight := m.getPreviewHeight()
			sessionListHeight = contentHeight - previewHeight - 1
		}
		// Side layout doesn't reduce session list height
	}

	if m.searchMode || m.searchQuery != "" {
		sessionListHeight-- // Search bar takes 1 line
	}

	maxSessions := sessionListHeight / sessionRowEstimatedHeight
	if maxSessions < 1 {
		maxSessions = 1
	}
	return maxSessions
}

// selectedSessionName returns the name of the currently selected session, or empty string.
func (m Model) selectedSessionName() string {
	filtered := m.getFilteredSessions()
	if m.cursor < len(filtered) {
		return filtered[m.cursor].TmuxSession
	}
	return ""
}

// preserveCursor attempts to keep the cursor on the same session after a filter change.
// If the session is no longer in the filtered list, cursor resets to 0.
func (m *Model) preserveCursor(sessionName string) {
	if sessionName == "" {
		m.cursor = 0
		m.sessionScrollOffset = 0
		return
	}
	filtered := m.getFilteredSessions()
	for i, s := range filtered {
		if s.TmuxSession == sessionName {
			m.cursor = i
			m.ensureSessionCursorVisible(m.sessionListMaxVisible())
			return
		}
	}
	// Session not found in new filtered list
	if len(filtered) > 0 && m.cursor >= len(filtered) {
		m.cursor = len(filtered) - 1
	} else if len(filtered) == 0 {
		m.cursor = 0
	}
	m.sessionScrollOffset = 0
}

// closeDialog resets dialog state.
func (m *Model) closeDialog() {
	m.dialogMode = DialogNone
	m.dialogError = ""
}

// getFilteredSessions returns sessions filtered and sorted by all active filters.
// Pipeline: local/remote filter → status filter → offline filter → fuzzy search → sort.
func (m Model) getFilteredSessions() []session.Info {
	// Start with all sessions
	result := m.sessions

	// Step 1: Local/Remote filter
	if m.filterMode != session.FilterAll {
		var filtered []session.Info
		for _, s := range result {
			switch m.filterMode {
			case session.FilterLocal:
				if s.Remote == "" {
					filtered = append(filtered, s)
				}
			case session.FilterRemote:
				if s.Remote != "" {
					filtered = append(filtered, s)
				}
			}
		}
		result = filtered
	}

	// Step 2: Status filter
	if m.statusFilter != "" {
		result = filterByStatus(result, m.statusFilter)
	}

	// Step 3: Offline filter
	if m.hideOffline {
		result = filterOffline(result)
	}

	// Step 4: Project directory filter (PM view Enter jump target).
	if m.pmProjectFilterDir != "" {
		var filtered []session.Info
		for _, s := range result {
			if strings.TrimSpace(s.CWD) == "" {
				continue
			}
			expanded := pathutil.ExpandPath(s.CWD)
			abs, err := filepath.Abs(expanded)
			if err != nil {
				abs = expanded
			}
			if abs == m.pmProjectFilterDir || strings.HasPrefix(abs, m.pmProjectFilterDir+string(filepath.Separator)) {
				filtered = append(filtered, s)
			}
		}
		result = filtered
	}

	// Step 5: Search no longer filters — all sessions remain visible.
	// Match indices are computed separately via computeSearchMatches().

	// Step 6: Sort
	result = sortSessions(result, m.sortMode)

	return result
}

// filterModeString returns a display string for the current filter mode.
func (m Model) filterModeString() string {
	switch m.filterMode {
	case session.FilterLocal:
		return "local"
	case session.FilterRemote:
		return "remote"
	default:
		return "all"
	}
}

// InitialModel creates the initial Model for the application.
func InitialModel() Model {
	// Load remote configuration (errors are logged but not fatal)
	remotes, err := remote.LoadConfig()
	if err != nil {
		// Log error but continue - remotes are optional
		fmt.Fprintf(os.Stderr, "Warning: failed to load remotes config: %v\n", err)
		remotes = []remote.Config{}
	}

	// Initialize SSH pool if remotes are configured
	var sshPool *remote.SSHPool
	if len(remotes) > 0 {
		sshPool = remote.NewSSHPool(remotes)
	}

	// Load global task config (errors are logged but not fatal)
	globalTaskConfig, err := task.LoadGlobalConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load task config: %v\n", err)
		globalTaskConfig = &task.GlobalConfig{}
	}

	audioConfig, err := audio.LoadConfig(audio.DefaultConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load audio config: %v\n", err)
		audioConfig = audio.DefaultConfig()
	}
	audioNotifier := audio.NewNotifier(audioConfig)

	m := Model{
		sessions:            []session.Info{},
		cursor:              0,
		width:               80,
		height:              24,
		Remotes:             remotes,
		SSHPool:             sshPool,
		sortMode:            SortPriority,
		searchInput:         initSearchInput(),
		taskSearchInput:     initTaskSearchInput(),
		taskExpandedGroups:  make(map[string]bool),
		taskGroupsByProject: make(map[string][]task.TaskGroup),
		taskCache:           task.NewResultCache(),
		taskGlobalConfig:    globalTaskConfig,
		taskSortMode:        taskSortSource,
		taskFilterMode:      taskFilterAll,
		previewAutoScroll:   true,
		audioNotifier:       audioNotifier,
		activeSoundPack:     audioConfig.Pack,
		lastSessionStates:   make(map[string]string),
		lastAgentStates:     make(map[string]map[string]string),
		pmEngine:            pm.NewEngine(),
		pmInvoker:           initPMInvoker(),
		pmTaskResults:       make(map[string]*task.ProviderResult),
		pmExpandedProjects:  make(map[string]bool),
	}

	// Load cached briefing from last session if available.
	if cached := initPMCachedBriefing(); cached != nil {
		m.pmBriefing = cached
		m.pmBriefingStale = true
	}

	return m
}

// initPMInvoker attempts to create a PM invoker. Returns nil if storage
// initialization fails (PM invocation is optional).
func initPMInvoker() *pm.Invoker {
	invoker, _ := pm.NewInvoker()
	return invoker
}

// initPMCachedBriefing loads the last cached PM briefing from disk.
// Returns nil if no cache exists.
func initPMCachedBriefing() *pm.PMBriefing {
	cached, err := pm.LoadCachedOutput()
	if err != nil || cached == nil {
		return nil
	}
	return cached.Briefing
}

// mergeResourceCache re-applies cached resource metrics onto the current sessions list.
// Called after sessionsMsg replaces the sessions slice to preserve resource data.
func (m *Model) mergeResourceCache() {
	if len(m.resourceCache) == 0 {
		return
	}
	for i := range m.sessions {
		if rss, ok := m.resourceCache[m.sessions[i].TmuxSession]; ok {
			if m.sessions[i].Metrics == nil {
				m.sessions[i].Metrics = &metrics.Metrics{}
			}
			m.sessions[i].Metrics.Resource = &metrics.ResourceMetrics{RSSBytes: rss}
		}
	}
}

func (m *Model) detectStatusChanges(current []session.Info) {
	if m.audioNotifier == nil {
		return
	}
	if m.lastSessionStates == nil {
		m.lastSessionStates = make(map[string]string)
	}
	if m.lastAgentStates == nil {
		m.lastAgentStates = make(map[string]map[string]string)
	}

	currentStates := make(map[string]string, len(current))
	currentAgentStates := make(map[string]map[string]string)
	for _, s := range current {
		currentStates[s.TmuxSession] = s.Status
		if len(s.Agents) == 0 {
			continue
		}

		agentStates := make(map[string]string, len(s.Agents))
		for agentType, agent := range s.Agents {
			agentStates[agentType] = agent.Status
		}
		currentAgentStates[s.TmuxSession] = agentStates
	}

	// First poll should only initialize state to avoid startup noise.
	if len(m.lastSessionStates) == 0 && len(m.lastAgentStates) == 0 {
		m.lastSessionStates = currentStates
		m.lastAgentStates = currentAgentStates
		return
	}

	for sessionName, newStatus := range currentStates {
		if oldStatus, ok := m.lastSessionStates[sessionName]; !ok {
			continue
		} else if oldStatus != newStatus {
			m.notifyStatusChange(sessionName, newStatus)
		}
	}

	for sessionName, agentStates := range currentAgentStates {
		lastSessionAgents, ok := m.lastAgentStates[sessionName]
		if !ok {
			continue
		}

		for agentType, newStatus := range agentStates {
			oldStatus, ok := lastSessionAgents[agentType]
			if !ok {
				continue
			}
			if oldStatus != newStatus {
				m.notifyAgentStatusChange(sessionName, agentType, newStatus)
			}
		}
	}

	m.lastSessionStates = currentStates
	m.lastAgentStates = currentAgentStates
}

func (m *Model) notifyStatusChange(sessionName, newStatus string) {
	if m.audioNotifyFn != nil {
		m.audioNotifyFn(sessionName, newStatus)
		return
	}
	if m.audioNotifier != nil {
		m.audioNotifier.Notify(sessionName, newStatus)
	}
}

func (m *Model) notifyAgentStatusChange(sessionName, agentType, newStatus string) {
	m.notifyStatusChange(sessionName+":"+agentType, newStatus)
}

// stringSlicesEqual returns true if two string slices have the same elements (order-independent).
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	setA := make(map[string]bool, len(a))
	for _, s := range a {
		setA[s] = true
	}
	for _, s := range b {
		if !setA[s] {
			return false
		}
	}
	return true
}
