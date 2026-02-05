package main

import (
	"os"
	"strings"
	"testing"
	"time"
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
			result := shortenPath(tt.path)
			if result != tt.want {
				t.Errorf("shortenPath(%q) = %q, want %q", tt.path, result, tt.want)
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

	session := SessionInfo{
		TmuxSession: "test-session",
		Status:      "working",
		Message:     "Processing files...",
		CWD:         "/tmp/test",
		Timestamp:   time.Now().Unix(),
	}

	// Test unselected
	result := m.renderSession(session, false, 80)
	if strings.Contains(result, selectedMarker) {
		t.Error("unselected row should not contain selection marker")
	}
	if !strings.Contains(result, "test-session") {
		t.Error("row should contain session name")
	}
	if !strings.Contains(result, "/tmp/test") {
		t.Error("row should contain working directory")
	}
	if !strings.Contains(result, "Processing files...") {
		t.Error("row should contain message")
	}

	// Test selected
	result = m.renderSession(session, true, 80)
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
			m.sessions = make([]SessionInfo, tt.sessionCount)

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
	// Use wider terminal to fit all keybindings
	m := Model{width: 120, height: 24}
	result := m.renderFooter()

	// Check that all keybindings are present
	expectedKeys := []string{"navigate", "attach", "dismiss", "quit", "refresh"}
	for _, key := range expectedKeys {
		if !strings.Contains(result, key) {
			t.Errorf("footer should contain keybinding %q", key)
		}
	}
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

	t.Run("shows session name in header", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
				{TmuxSession: "test-session"},
			},
			cursor:         0,
			previewVisible: true,
			previewContent: "some content",
		}

		result := m.renderPreview(40, 20)
		if !strings.Contains(result, "test-session") {
			t.Error("preview should show session name in header")
		}
	})

	t.Run("shows empty message when no content", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []SessionInfo{
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
			sessions: []SessionInfo{
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
			sessions: []SessionInfo{
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
