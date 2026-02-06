package pathutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Could not get home directory")
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"~/Documents", filepath.Join(home, "Documents")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"", ""},
	}

	for _, tc := range tests {
		result := ExpandPath(tc.input)
		if result != tc.expected {
			t.Errorf("ExpandPath(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestShortenPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Could not get home directory")
	}

	tests := []struct {
		input    string
		expected string
	}{
		{filepath.Join(home, "Documents"), "~/Documents"},
		{"/other/path", "/other/path"},
		{"", ""},
	}

	for _, tc := range tests {
		result := ShortenPath(tc.input)
		if result != tc.expected {
			t.Errorf("ShortenPath(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}
