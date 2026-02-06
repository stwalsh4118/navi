package tui

import (
	"testing"

	"github.com/stwalsh4118/navi/internal/session"
)

func TestValidateSessionName(t *testing.T) {
	existingSessions := []session.Info{
		{TmuxSession: "existing-session"},
		{TmuxSession: "another-session"},
	}

	testCases := []struct {
		name     string
		input    string
		sessions []session.Info
		wantErr  error
	}{
		{
			name:     "valid name",
			input:    "my-new-session",
			sessions: existingSessions,
			wantErr:  nil,
		},
		{
			name:     "empty name",
			input:    "",
			sessions: existingSessions,
			wantErr:  errEmptyName,
		},
		{
			name:     "whitespace only",
			input:    "   ",
			sessions: existingSessions,
			wantErr:  errEmptyName,
		},
		{
			name:     "contains period",
			input:    "my.session",
			sessions: existingSessions,
			wantErr:  errInvalidChars,
		},
		{
			name:     "contains colon",
			input:    "my:session",
			sessions: existingSessions,
			wantErr:  errInvalidChars,
		},
		{
			name:     "contains both invalid chars",
			input:    "my.session:test",
			sessions: existingSessions,
			wantErr:  errInvalidChars,
		},
		{
			name:     "duplicate name",
			input:    "existing-session",
			sessions: existingSessions,
			wantErr:  errNameExists,
		},
		{
			name:     "name with spaces",
			input:    "session with spaces",
			sessions: existingSessions,
			wantErr:  nil, // Spaces are allowed in tmux s names
		},
		{
			name:     "name with numbers",
			input:    "session-123",
			sessions: existingSessions,
			wantErr:  nil,
		},
		{
			name:     "name with underscores",
			input:    "my_session_name",
			sessions: existingSessions,
			wantErr:  nil,
		},
		{
			name:     "no existing sessions",
			input:    "any-name",
			sessions: []session.Info{},
			wantErr:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSessionName(tc.input, tc.sessions)
			if err != tc.wantErr {
				t.Errorf("validateSessionName(%q) = %v, want %v", tc.input, err, tc.wantErr)
			}
		})
	}
}

func TestInitNameInput(t *testing.T) {
	ti := initNameInput()

	if ti.Placeholder != "Session name" {
		t.Errorf("placeholder = %q, want %q", ti.Placeholder, "Session name")
	}

	if ti.CharLimit != inputNameCharLimit {
		t.Errorf("CharLimit = %d, want %d", ti.CharLimit, inputNameCharLimit)
	}

	if ti.Width != inputWidth {
		t.Errorf("Width = %d, want %d", ti.Width, inputWidth)
	}

	if !ti.Focused() {
		t.Error("name input should be focused by default")
	}
}

func TestInitDirInput(t *testing.T) {
	ti := initDirInput()

	if ti.Placeholder != "Working directory" {
		t.Errorf("placeholder = %q, want %q", ti.Placeholder, "Working directory")
	}

	if ti.CharLimit != inputDirCharLimit {
		t.Errorf("CharLimit = %d, want %d", ti.CharLimit, inputDirCharLimit)
	}

	if ti.Width != inputWidth {
		t.Errorf("Width = %d, want %d", ti.Width, inputWidth)
	}

	if ti.Focused() {
		t.Error("dir input should not be focused by default")
	}
}

func TestGetDefaultDirectory(t *testing.T) {
	dir := getDefaultDirectory()
	if dir == "" {
		t.Error("default directory should not be empty")
	}
}

func TestGetDefaultSessionName(t *testing.T) {
	name := getDefaultSessionName()
	if name == "" {
		t.Error("default s name should not be empty")
	}
}

func TestValidateDirectory(t *testing.T) {
	t.Run("empty path is valid", func(t *testing.T) {
		err := validateDirectory("")
		if err != nil {
			t.Errorf("empty path should be valid, got %v", err)
		}
	})

	t.Run("home directory is valid", func(t *testing.T) {
		err := validateDirectory("~/")
		if err != nil {
			t.Errorf("home directory should be valid, got %v", err)
		}
	})

	t.Run("non-existent path is invalid", func(t *testing.T) {
		err := validateDirectory("/nonexistent/path/12345")
		if err != errInvalidDir {
			t.Errorf("non-existent path should return errInvalidDir, got %v", err)
		}
	})

	t.Run("root directory is valid", func(t *testing.T) {
		err := validateDirectory("/tmp")
		if err != nil {
			t.Errorf("/tmp should be valid, got %v", err)
		}
	})
}
