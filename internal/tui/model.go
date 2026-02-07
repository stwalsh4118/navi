package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/git"
	"github.com/stwalsh4118/navi/internal/pathutil"
	"github.com/stwalsh4118/navi/internal/remote"
	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/task"
)

// Model is the Bubble Tea application state for navi.
type Model struct {
	sessions            []session.Info
	cursor              int
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
	previewVisible     bool          // Whether preview pane is shown
	previewUserEnabled bool          // User's intended state (for restore after terminal resize)
	previewContent     string        // Cached captured output
	previewLayout      PreviewLayout // Current layout mode (default: PreviewLayoutSide)
	previewWidth       int           // Width of preview pane in columns (side layout)
	previewHeight      int           // Height of preview pane in rows (bottom layout)
	previewWrap        bool          // Whether to wrap long lines (true) or truncate (false)
	previewLastCapture time.Time     // Last capture timestamp for debouncing
	previewLastCursor  int           // Last cursor position for detecting cursor changes

	// Git info cache
	gitCache map[string]*git.Info // Cache of git info by session working directory

	// Remote session support
	Remotes    []remote.Config // Configured remote machines
	SSHPool    *remote.SSHPool // SSH connection pool for remotes
	filterMode session.FilterMode // Current session filter mode

	// Search and filter state
	searchQuery  string          // Current search text
	searchMode   bool            // Whether search input is active
	searchInput  textinput.Model // Text input for search
	statusFilter string          // Active status filter (empty = show all)
	hideOffline  bool            // Whether to hide offline/done sessions (default false = show all)
	sortMode     SortMode        // Active sort mode

	// Task panel state
	taskPanelVisible    bool                            // Whether task panel is shown below session list
	taskPanelUserEnabled bool                           // User's intended state (for restore after terminal resize)
	taskPanelFocused    bool                            // Whether keyboard focus is inside the task panel
	taskPanelHeight     int                             // Height of task panel in rows
	taskCursor          int                             // Cursor position in flat task item list
	taskExpandedGroups  map[string]bool                 // Which groups are expanded (collapsed by default)
	taskSearchMode      bool                            // Whether task search input is active
	taskSearchQuery     string                          // Current task search text
	taskSearchInput     textinput.Model                 // Text input for task search
	taskGroups          []task.TaskGroup                // Displayed task groups (for focused project)
	taskGroupsByProject map[string][]task.TaskGroup     // All task groups keyed by project dir
	taskFocusedProject  string                          // Project dir whose tasks are displayed
	taskErrors          map[string]error                // Provider errors keyed by project dir
	taskProjectConfigs  []task.ProjectConfig            // Discovered project configs
	taskCache           *task.ResultCache               // Cache for provider results
	taskGlobalConfig    *task.GlobalConfig              // Global task config
	taskLastCWDs        []string                        // Last seen session CWDs (for change detection)
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

// gitTickMsg is sent to trigger periodic git info refresh.
type gitTickMsg time.Time

// gitInfoMsg is returned after polling git info for all sessions.
type gitInfoMsg struct {
	cache map[string]*git.Info // Map of CWD to git.Info
}

// gitPRMsg is returned after fetching PR info for a specific directory.
type gitPRMsg struct {
	cwd   string
	prNum int
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

	cmds := []tea.Cmd{tickCmd(), pollSessions, gitTickCmd()}

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

		// Handle task panel focus mode - route to task panel keybindings
		if m.taskPanelFocused {
			return m.updateTaskPanelFocus(msg)
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
						return m, attachRemoteSession(r, s.TmuxSession)
					}
					// Fallback to local attach if remote config not found
				}

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
			// Open new session dialog
			m.dialogMode = DialogNewSession
			m.dialogError = ""
			m.nameInput = initNameInput()
			m.dirInput = initDirInput()
			m.dirInput.SetValue(getDefaultDirectory())
			m.focusedInput = focusName
			m.skipPermissions = false
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
			return m, nil

		case "o":
			// Toggle offline session visibility
			selectedSession := m.selectedSessionName()
			m.hideOffline = !m.hideOffline
			m.preserveCursor(selectedSession)
			return m, nil

		case "0":
			// Clear status filter
			selectedSession := m.selectedSessionName()
			m.statusFilter = ""
			m.preserveCursor(selectedSession)
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
				m.taskSearchMode = false
				m.taskSearchQuery = ""
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

	case tasksMsg:
		m.taskGroupsByProject = msg.groupsByProject
		m.taskErrors = msg.errors
		// Update displayed groups if task panel is visible
		if m.taskPanelVisible {
			m.updateTaskPanelForCursor()
		}
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
		// Debounced capture after cursor movement
		filteredSessions := m.getFilteredSessions()
		if !m.previewVisible || len(filteredSessions) == 0 {
			return m, nil
		}
		if m.cursor < len(filteredSessions) {
			return m, m.capturePreviewForSession(filteredSessions[m.cursor])
		}
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
		// Update PR number for the session being viewed (lazy-loaded)
		if m.sessionToModify != nil && m.sessionToModify.CWD == msg.cwd && m.sessionToModify.Git != nil {
			m.sessionToModify.Git.PRNum = msg.prNum
		}
		// Also update the cache so it persists
		if m.gitCache != nil {
			if info, ok := m.gitCache[msg.cwd]; ok {
				info.PRNum = msg.prNum
			}
		}
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

// updateTaskPanelFocus handles key messages when the task panel has focus.
func (m Model) updateTaskPanelFocus(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Route to task search mode if active
	if m.taskSearchMode {
		return m.updateTaskSearchMode(msg)
	}

	switch msg.String() {
	case "tab", "esc":
		// Return focus to session list
		m.taskPanelFocused = false
		return m, nil

	case "T":
		// Close task panel entirely
		m.taskPanelVisible = false
		m.taskPanelUserEnabled = false
		m.taskPanelFocused = false
		return m, nil

	case "up", "k":
		m.moveTaskCursor(-1)
		return m, nil

	case "down", "j":
		m.moveTaskCursor(1)
		return m, nil

	case " ":
		// Toggle group expansion
		item := m.getSelectedTaskItem()
		if item != nil && item.isGroup {
			if m.taskExpandedGroups[item.groupID] {
				delete(m.taskExpandedGroups, item.groupID)
			} else {
				m.taskExpandedGroups[item.groupID] = true
			}
			// Clamp cursor after expansion change
			items := m.getVisibleTaskItems()
			if m.taskCursor >= len(items) && len(items) > 0 {
				m.taskCursor = len(items) - 1
			}
		}
		return m, nil

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

// updateTaskSearchMode handles key messages when task search is active within the focused panel.
func (m Model) updateTaskSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Clear search and exit search mode (stay in focus mode)
		m.taskSearchMode = false
		m.taskSearchQuery = ""
		m.taskSearchInput.SetValue("")
		m.taskSearchInput.Blur()
		m.taskCursor = 0
		return m, nil

	case "up", "k":
		m.moveTaskCursor(-1)
		return m, nil

	case "down", "j":
		m.moveTaskCursor(1)
		return m, nil

	case "enter":
		// Accept search and return to normal focus navigation
		m.taskSearchMode = false
		m.taskSearchInput.Blur()
		return m, nil

	case " ":
		// Toggle group expansion even during search
		item := m.getSelectedTaskItem()
		if item != nil && item.isGroup {
			if m.taskExpandedGroups[item.groupID] {
				delete(m.taskExpandedGroups, item.groupID)
			} else {
				m.taskExpandedGroups[item.groupID] = true
			}
			items := m.getVisibleTaskItems()
			if m.taskCursor >= len(items) && len(items) > 0 {
				m.taskCursor = len(items) - 1
			}
		}
		return m, nil
	}

	// Forward all other keys to search input
	var cmd tea.Cmd
	m.taskSearchInput, cmd = m.taskSearchInput.Update(msg)
	m.taskSearchQuery = m.taskSearchInput.Value()

	// Clamp cursor after query change
	items := m.getVisibleTaskItems()
	if m.taskCursor >= len(items) {
		if len(items) > 0 {
			m.taskCursor = len(items) - 1
		} else {
			m.taskCursor = 0
		}
	}

	return m, cmd
}

// taskItem represents a navigable item in the task panel (group header or task).
type taskItem struct {
	isGroup bool
	groupID string
	title   string
	status  string
	taskID  string
	number  int // Sequential group number (1-based, 0 for tasks)
}

// getVisibleTaskItems returns the flat list of navigable items based on expand state.
// When a task search query is active, filters groups/tasks by fuzzy match and
// auto-expands groups that contain matching tasks.
func (m Model) getVisibleTaskItems() []taskItem {
	var items []taskItem
	groupNum := 0

	if m.taskSearchQuery != "" {
		// Search mode: filter and auto-expand matching groups
		for _, g := range m.taskGroups {
			groupMatches, _ := fuzzyMatch(m.taskSearchQuery, g.Title)

			// Find matching tasks in this group
			var matchingTasks []taskItem
			for _, t := range g.Tasks {
				titleMatch, _ := fuzzyMatch(m.taskSearchQuery, t.Title)
				idMatch, _ := fuzzyMatch(m.taskSearchQuery, t.ID)
				if titleMatch || idMatch {
					matchingTasks = append(matchingTasks, taskItem{
						isGroup: false,
						groupID: g.ID,
						taskID:  t.ID,
						title:   t.Title,
						status:  t.Status,
					})
				}
			}

			// Show group if it matches or has matching tasks
			if groupMatches || len(matchingTasks) > 0 {
				groupNum++
				items = append(items, taskItem{
					isGroup: true,
					groupID: g.ID,
					title:   g.Title,
					status:  g.Status,
					number:  groupNum,
				})
				// Auto-expand: show matching tasks (or all tasks if group title matched)
				if groupMatches && len(matchingTasks) == 0 {
					// Group title matched but no individual tasks - show all tasks
					for _, t := range g.Tasks {
						items = append(items, taskItem{
							isGroup: false,
							groupID: g.ID,
							taskID:  t.ID,
							title:   t.Title,
							status:  t.Status,
						})
					}
				} else {
					items = append(items, matchingTasks...)
				}
			}
		}
		return items
	}

	// Normal mode: respect expand/collapse state
	for _, g := range m.taskGroups {
		groupNum++
		items = append(items, taskItem{
			isGroup: true,
			groupID: g.ID,
			title:   g.Title,
			status:  g.Status,
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
				})
			}
		}
	}
	return items
}

// getSelectedTaskItem returns the currently selected task item, or nil.
func (m Model) getSelectedTaskItem() *taskItem {
	items := m.getVisibleTaskItems()
	if m.taskCursor >= 0 && m.taskCursor < len(items) {
		return &items[m.taskCursor]
	}
	return nil
}

// moveTaskCursor moves the task cursor by delta, wrapping at bounds.
func (m *Model) moveTaskCursor(delta int) {
	items := m.getVisibleTaskItems()
	if len(items) == 0 {
		m.taskCursor = 0
		return
	}
	m.taskCursor += delta
	if m.taskCursor < 0 {
		m.taskCursor = len(items) - 1
	} else if m.taskCursor >= len(items) {
		m.taskCursor = 0
	}
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
	switch msg.String() {
	case "esc":
		// Handle escape based on current dialog mode
		if m.dialogMode == DialogGitDiff {
			// Return to git detail view from diff view
			m.dialogMode = DialogGitDetail
			return m, nil
		}
		// Close any dialog and reset state
		m.dialogMode = DialogNone
		m.dialogError = ""
		m.sessionToModify = nil
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
		// Show diff view from git detail view
		if m.dialogMode == DialogGitDetail && m.sessionToModify != nil && m.sessionToModify.Git != nil {
			m.dialogMode = DialogGitDiff
			return m, nil
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

// updateSearchMode handles key messages when search mode is active.
// Navigation keys (up/down/enter) still work; other keys go to the search input.
func (m Model) updateSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Clear search and exit search mode
		m.searchMode = false
		m.searchQuery = ""
		m.searchInput.SetValue("")
		m.searchInput.Blur()
		m.cursor = 0
		return m, nil

	case "up", "k":
		filteredSessions := m.getFilteredSessions()
		if len(filteredSessions) > 0 {
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(filteredSessions) - 1
			}
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
		}
		return m, nil

	case "enter":
		filteredSessions := m.getFilteredSessions()
		if len(filteredSessions) > 0 && m.cursor < len(filteredSessions) {
			s := filteredSessions[m.cursor]
			m.lastSelectedSession = s.TmuxSession

			if s.Remote != "" && m.SSHPool != nil {
				r := m.SSHPool.GetRemoteConfig(s.Remote)
				if r != nil {
					return m, attachRemoteSession(r, s.TmuxSession)
				}
			}
			return m, attachSession(s.TmuxSession)
		}
		return m, nil
	}

	// Forward all other keys to search input
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.searchQuery = m.searchInput.Value()

	// Clamp cursor after search query change
	filteredSessions := m.getFilteredSessions()
	if m.cursor >= len(filteredSessions) {
		if len(filteredSessions) > 0 {
			m.cursor = len(filteredSessions) - 1
		} else {
			m.cursor = 0
		}
	}

	return m, cmd
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
		return
	}
	filtered := m.getFilteredSessions()
	for i, s := range filtered {
		if s.TmuxSession == sessionName {
			m.cursor = i
			return
		}
	}
	// Session not found in new filtered list
	if len(filtered) > 0 && m.cursor >= len(filtered) {
		m.cursor = len(filtered) - 1
	} else if len(filtered) == 0 {
		m.cursor = 0
	}
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

	// Step 4: Fuzzy search
	if m.searchQuery != "" {
		result = fuzzyFilter(result, m.searchQuery)
	}

	// Step 5: Sort
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

	return Model{
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
	}
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
