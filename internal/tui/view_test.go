package tui

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/git"
	"github.com/stwalsh4118/navi/internal/pathutil"
	"github.com/stwalsh4118/navi/internal/session"
)

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name       string
		timestamp  int64
		wantSuffix string
	}{
		{
			name:       "seconds ago",
			timestamp:  time.Now().Unix() - 30,
			wantSuffix: "s ago",
		},
		{
			name:       "minutes ago",
			timestamp:  time.Now().Unix() - 120,
			wantSuffix: "m ago",
		},
		{
			name:       "hours ago",
			timestamp:  time.Now().Unix() - 7200,
			wantSuffix: "h ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAge(tt.timestamp)
			if !strings.HasSuffix(result, tt.wantSuffix) {
				t.Errorf("formatAge() = %q, want suffix %q", result, tt.wantSuffix)
			}
		})
	}
}

func TestShortenPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get home directory")
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "home directory",
			path: home + "/projects/test",
			want: "~/projects/test",
		},
		{
			name: "non-home path",
			path: "/var/log/test",
			want: "/var/log/test",
		},
		{
			name: "home directory itself",
			path: home,
			want: "~",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pathutil.ShortenPath(tt.path)
			if result != tt.want {
				t.Errorf("pathutil.ShortenPath(%q) = %q, want %q", tt.path, result, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{
			name:   "no truncation needed",
			s:      "short",
			maxLen: 10,
			want:   "short",
		},
		{
			name:   "truncation with ellipsis",
			s:      "this is a very long string",
			maxLen: 10,
			want:   "this is...",
		},
		{
			name:   "exact length",
			s:      "exact",
			maxLen: 5,
			want:   "exact",
		},
		{
			name:   "very short maxLen",
			s:      "test string",
			maxLen: 2,
			want:   "t...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.s, tt.maxLen)
			if result != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, result, tt.want)
			}
		})
	}
}

func TestRenderSession(t *testing.T) {
	m := Model{width: 80, height: 24}

	s := session.Info{
		TmuxSession: "test-session",
		Status:      "working",
		Message:     "Processing files...",
		CWD:         "/tmp/test",
		Timestamp:   time.Now().Unix(),
	}

	// Test unselected
	result := m.renderSession(s, false, 80)
	if strings.Contains(result, selectedMarker) {
		t.Error("unselected row should not contain selection marker")
	}
	if !strings.Contains(result, "test-session") {
		t.Error("row should contain s name")
	}
	if !strings.Contains(result, "/tmp/test") {
		t.Error("row should contain working directory")
	}
	if !strings.Contains(result, "Processing files...") {
		t.Error("row should contain message")
	}

	// Test selected
	result = m.renderSession(s, true, 80)
	if !strings.Contains(result, selectedMarker) {
		t.Error("selected row should contain selection marker")
	}
}

func TestRenderHeader(t *testing.T) {
	tests := []struct {
		name         string
		sessionCount int
		wantTitle    bool
		wantCount    string
	}{
		{
			name:         "no sessions",
			sessionCount: 0,
			wantTitle:    true,
			wantCount:    "0 active",
		},
		{
			name:         "multiple sessions",
			sessionCount: 3,
			wantTitle:    true,
			wantCount:    "3 active",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{width: 80, height: 24}
			m.sessions = make([]session.Info, tt.sessionCount)

			result := m.renderHeader()
			if !strings.Contains(result, headerTitle) {
				t.Errorf("header should contain title %q", headerTitle)
			}
			if !strings.Contains(result, tt.wantCount) {
				t.Errorf("header should contain count %q", tt.wantCount)
			}
		})
	}
}

func TestRenderFooter(t *testing.T) {
	t.Run("shows base keys when preview hidden", func(t *testing.T) {
		m := Model{width: 120, height: 24, previewVisible: false}
		result := m.renderFooter()

		// Check base keybindings are present
		expectedKeys := []string{"nav", "attach", "preview", "dismiss", "quit", "refresh"}
		for _, key := range expectedKeys {
			if !strings.Contains(result, key) {
				t.Errorf("footer should contain keybinding %q", key)
			}
		}

		// Preview-specific keys should NOT be present
		unexpectedKeys := []string{"layout", "wrap", "resize"}
		for _, key := range unexpectedKeys {
			if strings.Contains(result, key) {
				t.Errorf("footer should NOT contain %q when preview hidden", key)
			}
		}
	})

	t.Run("shows preview keys when preview visible", func(t *testing.T) {
		m := Model{width: 120, height: 24, previewVisible: true}
		result := m.renderFooter()

		// Check all keybindings including preview-specific ones
		expectedKeys := []string{"nav", "attach", "preview", "layout", "wrap", "resize", "dismiss", "quit"}
		for _, key := range expectedKeys {
			if !strings.Contains(result, key) {
				t.Errorf("footer should contain keybinding %q when preview visible", key)
			}
		}
	})
}

func TestRenderPreview(t *testing.T) {
	t.Run("returns empty string when preview not visible", func(t *testing.T) {
		m := Model{
			width:          80,
			height:         24,
			previewVisible: false,
		}

		result := m.renderPreview(40, 20)
		if result != "" {
			t.Error("renderPreview should return empty string when not visible")
		}
	})

	t.Run("shows s name in header", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []session.Info{
				{TmuxSession: "test-session"},
			},
			cursor:         0,
			previewVisible: true,
			previewContent: "some content",
		}

		result := m.renderPreview(40, 20)
		if !strings.Contains(result, "test-session") {
			t.Error("preview should show s name in header")
		}
	})

	t.Run("shows empty message when no content", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []session.Info{
				{TmuxSession: "test-session"},
			},
			cursor:         0,
			previewVisible: true,
			previewContent: "",
		}

		result := m.renderPreview(40, 20)
		if !strings.Contains(result, "No preview available") {
			t.Error("preview should show empty message when no content")
		}
	})

	t.Run("displays preview content", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []session.Info{
				{TmuxSession: "test-session"},
			},
			cursor:         0,
			previewVisible: true,
			previewContent: "line 1\nline 2\nline 3",
		}

		result := m.renderPreview(40, 20)
		if !strings.Contains(result, "line 1") {
			t.Error("preview should display content lines")
		}
	})
}

func TestViewWithPreview(t *testing.T) {
	t.Run("view includes preview when visible", func(t *testing.T) {
		m := Model{
			width:  120,
			height: 30,
			sessions: []session.Info{
				{TmuxSession: "test-session", Status: "working", CWD: "/tmp", Timestamp: time.Now().Unix()},
			},
			cursor:         0,
			previewVisible: true,
			previewContent: "preview content here",
		}

		result := m.View()
		if !strings.Contains(result, "preview content here") {
			t.Error("view should include preview content when visible")
		}
	})
}

func TestRenderGitInfo(t *testing.T) {
	t.Run("returns empty string for nil git info", func(t *testing.T) {
		result := renderGitInfo(nil, 80)
		if result != "" {
			t.Errorf("renderGitInfo(nil) = %q, want empty string", result)
		}
	})

	t.Run("shows branch name", func(t *testing.T) {
		g := &git.Info{Branch: "main"}
		result := renderGitInfo(g, 80)
		if !strings.Contains(result, "main") {
			t.Errorf("renderGitInfo should contain branch name, got %q", result)
		}
	})

	t.Run("shows dirty indicator when dirty", func(t *testing.T) {
		g := &git.Info{Branch: "main", Dirty: true}
		result := renderGitInfo(g, 80)
		if !strings.Contains(result, git.DirtyIndicator) {
			t.Errorf("renderGitInfo should contain dirty indicator, got %q", result)
		}
	})

	t.Run("hides dirty indicator when clean", func(t *testing.T) {
		g := &git.Info{Branch: "main", Dirty: false}
		result := renderGitInfo(g, 80)
		if strings.Contains(result, git.DirtyIndicator) {
			t.Errorf("renderGitInfo should not contain dirty indicator when clean, got %q", result)
		}
	})

	t.Run("shows ahead count when ahead", func(t *testing.T) {
		g := &git.Info{Branch: "main", Ahead: 3}
		result := renderGitInfo(g, 80)
		if !strings.Contains(result, "+3") {
			t.Errorf("renderGitInfo should contain ahead count, got %q", result)
		}
	})

	t.Run("shows behind count when behind", func(t *testing.T) {
		g := &git.Info{Branch: "main", Behind: 2}
		result := renderGitInfo(g, 80)
		if !strings.Contains(result, "-2") {
			t.Errorf("renderGitInfo should contain behind count, got %q", result)
		}
	})

	t.Run("shows PR number when detected", func(t *testing.T) {
		g := &git.Info{Branch: "feature/add-auth", PRNum: 42}
		result := renderGitInfo(g, 80)
		if !strings.Contains(result, "[PR#42]") {
			t.Errorf("renderGitInfo should contain PR number, got %q", result)
		}
	})

	t.Run("truncates long branch names", func(t *testing.T) {
		g := &git.Info{Branch: "feature/this-is-a-very-long-branch-name-that-should-be-truncated"}
		result := renderGitInfo(g, 80)
		// The branch should be truncated to git.MaxBranchLength (30) characters
		if strings.Contains(result, "that-should-be-truncated") {
			t.Errorf("renderGitInfo should truncate long branch names, got %q", result)
		}
		if !strings.Contains(result, "...") {
			t.Errorf("renderGitInfo should show ellipsis for truncated branch, got %q", result)
		}
	})
}

func TestRenderGitDetailView(t *testing.T) {
	t.Run("shows not a git repo message when no git info", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogGitDetail,
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/tmp",
				Git:         nil,
			},
		}

		result := m.renderGitDetailView()
		if !strings.Contains(result, "Not a git repository") {
			t.Error("should show 'Not a git repository' message")
		}
	})

	t.Run("shows branch name", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogGitDetail,
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/tmp",
				Git: &git.Info{
					Branch: "feature/auth",
				},
			},
		}

		result := m.renderGitDetailView()
		if !strings.Contains(result, "feature/auth") {
			t.Error("should show branch name")
		}
	})

	t.Run("shows dirty status", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogGitDetail,
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/tmp",
				Git: &git.Info{
					Branch: "main",
					Dirty:  true,
				},
			},
		}

		result := m.renderGitDetailView()
		if !strings.Contains(result, "uncommitted changes") {
			t.Error("should show dirty status message")
		}
	})

	t.Run("shows clean status", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogGitDetail,
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/tmp",
				Git: &git.Info{
					Branch: "main",
					Dirty:  false,
				},
			},
		}

		result := m.renderGitDetailView()
		if !strings.Contains(result, "(clean)") {
			t.Error("should show clean status")
		}
	})

	t.Run("shows ahead/behind counts", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogGitDetail,
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/tmp",
				Git: &git.Info{
					Branch: "main",
					Ahead:  3,
					Behind: 2,
				},
			},
		}

		result := m.renderGitDetailView()
		if !strings.Contains(result, "3 ahead") {
			t.Error("should show ahead count")
		}
		if !strings.Contains(result, "2 behind") {
			t.Error("should show behind count")
		}
	})

	t.Run("shows PR number and link", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogGitDetail,
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/tmp",
				Git: &git.Info{
					Branch: "feature/auth",
					PRNum:  123,
					Remote: "https://github.com/user/repo.git",
				},
			},
		}

		result := m.renderGitDetailView()
		if !strings.Contains(result, "#123") {
			t.Error("should show PR number")
		}
		if !strings.Contains(result, "github.com") {
			t.Error("should show GitHub link")
		}
	})

	t.Run("shows keybindings", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogGitDetail,
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/tmp",
				Git: &git.Info{
					Branch: "main",
				},
			},
		}

		result := m.renderGitDetailView()
		if !strings.Contains(result, "Esc: close") {
			t.Error("should show Esc keybinding")
		}
		if !strings.Contains(result, "d: diff") {
			t.Error("should show diff keybinding")
		}
	})
}

func TestRenderSessionWithGitInfo(t *testing.T) {
	m := Model{width: 80, height: 24}

	t.Run("session without git info has no git line", func(t *testing.T) {
		s := session.Info{
			TmuxSession: "test",
			Status:      "working",
			CWD:         "/tmp",
			Timestamp:   time.Now().Unix(),
			Git:         nil,
		}

		result := m.renderSession(s, false, 80)
		lines := strings.Split(result, "\n")

		// Should have 2 lines: name+status, cwd
		if len(lines) != 2 {
			t.Errorf("session without git should have 2 lines, got %d: %v", len(lines), lines)
		}
	})

	t.Run("session with git info shows git line", func(t *testing.T) {
		s := session.Info{
			TmuxSession: "test",
			Status:      "working",
			CWD:         "/tmp",
			Timestamp:   time.Now().Unix(),
			Git: &git.Info{
				Branch: "main",
				Dirty:  true,
			},
		}

		result := m.renderSession(s, false, 80)
		if !strings.Contains(result, "main") {
			t.Error("session with git info should show branch name")
		}
		if !strings.Contains(result, git.DirtyIndicator) {
			t.Error("session with dirty git should show dirty indicator")
		}
	})

	t.Run("session with git info and message shows both", func(t *testing.T) {
		s := session.Info{
			TmuxSession: "test",
			Status:      "working",
			CWD:         "/tmp",
			Message:     "Processing...",
			Timestamp:   time.Now().Unix(),
			Git: &git.Info{
				Branch: "feature/auth",
				Dirty:  false,
				Ahead:  2,
			},
		}

		result := m.renderSession(s, false, 80)
		lines := strings.Split(result, "\n")

		// Should have 4 lines: name+status, cwd, git, message
		if len(lines) != 4 {
			t.Errorf("session with git and message should have 4 lines, got %d", len(lines))
		}
		if !strings.Contains(result, "feature/auth") {
			t.Error("should show branch name")
		}
		if !strings.Contains(result, "+2") {
			t.Error("should show ahead count")
		}
		if !strings.Contains(result, "Processing...") {
			t.Error("should show message")
		}
	})
}

func TestRenderGitDiffViaContentViewer(t *testing.T) {
	t.Run("d key from git detail opens content viewer with diff", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogGitDetail,
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/tmp",
				Git:         &git.Info{Branch: "feature/test", Dirty: true},
			},
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
		updatedModel, _ := m.Update(msg)
		newM := updatedModel.(Model)

		if newM.dialogMode != DialogContentViewer {
			t.Errorf("expected DialogContentViewer, got %d", newM.dialogMode)
		}
		if !strings.Contains(newM.contentViewerTitle, "feature/test") {
			t.Error("content viewer title should contain branch name")
		}
	})

	t.Run("content viewer renders diff with branch in title", func(t *testing.T) {
		m := Model{
			width:              80,
			height:             24,
			dialogMode:         DialogContentViewer,
			contentViewerTitle: "Git Diff: main",
			contentViewerLines: []string{"+added", "-removed", " context"},
			contentViewerMode:  ContentModeDiff,
		}

		result := m.renderContentViewer()
		if !strings.Contains(result, "Git Diff: main") {
			t.Error("should show title with branch name")
		}
		if !strings.Contains(result, "added") {
			t.Error("should show diff content")
		}
	})

	t.Run("Esc from diff content viewer returns to git detail", func(t *testing.T) {
		m := Model{
			width:                   80,
			height:                  24,
			dialogMode:              DialogContentViewer,
			contentViewerPrevDialog: DialogGitDetail,
			contentViewerLines:      []string{"some diff"},
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/tmp",
				Git:         &git.Info{Branch: "main"},
			},
		}

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updatedModel, _ := m.Update(msg)
		newM := updatedModel.(Model)

		if newM.dialogMode != DialogGitDetail {
			t.Errorf("expected return to DialogGitDetail, got %d", newM.dialogMode)
		}
	})

	t.Run("content viewer shows Esc close keybinding", func(t *testing.T) {
		m := Model{
			width:              80,
			height:             24,
			dialogMode:         DialogContentViewer,
			contentViewerTitle: "Git Diff: main",
			contentViewerLines: []string{"content"},
			contentViewerMode:  ContentModeDiff,
		}

		result := m.renderContentViewer()
		if !strings.Contains(result, "Esc close") {
			t.Error("should show Esc close keybinding")
		}
	})
}

func TestStatusIconIdle(t *testing.T) {
	result := StatusIcon("idle")
	if !strings.Contains(result, iconIdle) {
		t.Errorf("StatusIcon(idle) should contain idle icon, got %q", result)
	}
}

func TestStatusIconStopped(t *testing.T) {
	result := StatusIcon("stopped")
	if !strings.Contains(result, iconStopped) {
		t.Errorf("StatusIcon(stopped) should contain stopped icon, got %q", result)
	}
}

func TestRenderSessionWithTeam(t *testing.T) {
	m := Model{width: 80, height: 24}

	t.Run("session with team shows agent count badge", func(t *testing.T) {
		s := session.Info{
			TmuxSession: "test-session",
			Status:      "working",
			CWD:         "/tmp/test",
			Timestamp:   time.Now().Unix(),
			Team: &session.TeamInfo{
				Name: "my-project",
				Agents: []session.AgentInfo{
					{Name: "researcher", Status: "working", Timestamp: time.Now().Unix()},
					{Name: "implementer", Status: "idle", Timestamp: time.Now().Unix()},
				},
			},
		}

		result := m.renderSession(s, false, 80)
		if !strings.Contains(result, "[2 agents]") {
			t.Error("session with team should show agent count badge")
		}
	})

	t.Run("session with team shows agent status icons and names", func(t *testing.T) {
		s := session.Info{
			TmuxSession: "test-session",
			Status:      "working",
			CWD:         "/tmp/test",
			Timestamp:   time.Now().Unix(),
			Team: &session.TeamInfo{
				Name: "my-project",
				Agents: []session.AgentInfo{
					{Name: "researcher", Status: "working", Timestamp: time.Now().Unix()},
					{Name: "tester", Status: "done", Timestamp: time.Now().Unix()},
				},
			},
		}

		result := m.renderSession(s, false, 80)
		if !strings.Contains(result, "researcher") {
			t.Error("should show agent name 'researcher'")
		}
		if !strings.Contains(result, "tester") {
			t.Error("should show agent name 'tester'")
		}
	})

	t.Run("session without team has no badge or agents line", func(t *testing.T) {
		s := session.Info{
			TmuxSession: "test-session",
			Status:      "working",
			CWD:         "/tmp/test",
			Timestamp:   time.Now().Unix(),
			Team:        nil,
		}

		result := m.renderSession(s, false, 80)
		if strings.Contains(result, "agents]") {
			t.Error("session without team should not show agent badge")
		}
	})

	t.Run("session with stopped agents excludes them from badge count", func(t *testing.T) {
		s := session.Info{
			TmuxSession: "test-session",
			Status:      "working",
			CWD:         "/tmp/test",
			Timestamp:   time.Now().Unix(),
			Team: &session.TeamInfo{
				Name: "my-project",
				Agents: []session.AgentInfo{
					{Name: "researcher", Status: "working", Timestamp: time.Now().Unix()},
					{Name: "implementer", Status: session.StatusStopped, Timestamp: time.Now().Unix()},
					{Name: "tester", Status: "idle", Timestamp: time.Now().Unix()},
				},
			},
		}

		result := m.renderSession(s, false, 80)
		if !strings.Contains(result, "[2 agents]") {
			t.Error("badge should show 2 agents (excluding stopped)")
		}
		if strings.Contains(result, "[3 agents]") {
			t.Error("badge should NOT count stopped agents")
		}
	})

	t.Run("session with all agents stopped shows team done", func(t *testing.T) {
		s := session.Info{
			TmuxSession: "test-session",
			Status:      "working",
			CWD:         "/tmp/test",
			Timestamp:   time.Now().Unix(),
			Team: &session.TeamInfo{
				Name: "my-project",
				Agents: []session.AgentInfo{
					{Name: "researcher", Status: session.StatusStopped, Timestamp: time.Now().Unix()},
					{Name: "implementer", Status: session.StatusStopped, Timestamp: time.Now().Unix()},
				},
			},
		}

		result := m.renderSession(s, false, 80)
		if !strings.Contains(result, "[team done]") {
			t.Error("should show [team done] when all agents are stopped")
		}
		if strings.Contains(result, "agents]") {
			t.Error("should NOT show agent count when all stopped")
		}
	})

	t.Run("session with empty team agents has no badge", func(t *testing.T) {
		s := session.Info{
			TmuxSession: "test-session",
			Status:      "working",
			CWD:         "/tmp/test",
			Timestamp:   time.Now().Unix(),
			Team: &session.TeamInfo{
				Name:   "empty-team",
				Agents: []session.AgentInfo{},
			},
		}

		result := m.renderSession(s, false, 80)
		if strings.Contains(result, "agents]") {
			t.Error("session with empty team should not show agent badge")
		}
	})
}

func TestRenderTeamAgents(t *testing.T) {
	t.Run("renders agents with status icons", func(t *testing.T) {
		agents := []session.AgentInfo{
			{Name: "researcher", Status: "working"},
			{Name: "implementer", Status: "idle"},
		}
		result := renderTeamAgents(agents, 80, len(rowIndent))
		if !strings.Contains(result, "researcher") {
			t.Error("should contain researcher name")
		}
		if !strings.Contains(result, "implementer") {
			t.Error("should contain implementer name")
		}
	})

	t.Run("wraps to next line when agents overflow width", func(t *testing.T) {
		agents := []session.AgentInfo{
			{Name: "agent-one", Status: "working"},
			{Name: "agent-two", Status: "idle"},
			{Name: "agent-three", Status: "done"},
			{Name: "agent-four", Status: "permission"},
			{Name: "agent-five", Status: "working"},
		}
		// Use narrow width to force wrapping
		result := renderTeamAgents(agents, 40, len(rowIndent))
		lines := strings.Split(result, "\n")
		if len(lines) < 2 {
			t.Errorf("expected multiple lines for narrow width, got %d lines", len(lines))
		}
	})

	t.Run("single agent fits on one line", func(t *testing.T) {
		agents := []session.AgentInfo{
			{Name: "solo", Status: "working"},
		}
		result := renderTeamAgents(agents, 80, len(rowIndent))
		lines := strings.Split(result, "\n")
		if len(lines) != 1 {
			t.Errorf("single agent should fit on one line, got %d lines", len(lines))
		}
	})
}
