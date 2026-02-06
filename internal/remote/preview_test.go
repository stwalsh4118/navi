package remote

import "testing"

func TestStripANSI(t *testing.T) {
	t.Run("removes color codes", func(t *testing.T) {
		input := "\x1b[31mred text\x1b[0m"
		result := stripANSI(input)
		expected := "red text"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("removes multiple color codes", func(t *testing.T) {
		input := "\x1b[1;31mbold red\x1b[0m normal \x1b[32mgreen\x1b[0m"
		result := stripANSI(input)
		expected := "bold red normal green"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("removes OSC sequences", func(t *testing.T) {
		input := "\x1b]0;Window Title\x07some text"
		result := stripANSI(input)
		expected := "some text"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("handles empty string", func(t *testing.T) {
		result := stripANSI("")
		if result != "" {
			t.Errorf("empty string should remain empty, got %q", result)
		}
	})

	t.Run("preserves newlines and tabs", func(t *testing.T) {
		input := "line1\nline2\tindented"
		result := stripANSI(input)
		if result != input {
			t.Errorf("newlines and tabs should be preserved, got %q", result)
		}
	})

	t.Run("removes control characters", func(t *testing.T) {
		input := "text\x00with\x1fcontrol\x08chars"
		result := stripANSI(input)
		expected := "textwithcontrolchars"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

func TestCapturePane_emptyOutput(t *testing.T) {
	// CapturePane requires an SSHPool, but we can test the post-processing
	// logic by verifying that stripANSI + TrimRight on empty input gives empty output.
	input := ""
	cleaned := stripANSI(input)
	result := cleaned // TrimRight of "" is ""
	if result != "" {
		t.Errorf("empty output should return empty string, got %q", result)
	}
}

func TestCapturePane_trailingWhitespace(t *testing.T) {
	// Simulate what CapturePane does after getting output
	input := "some output\n\n\n   \n"
	cleaned := stripANSI(input)
	// Mimic the TrimRight from CapturePane
	result := trimPreviewOutput(cleaned)
	expected := "some output"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestCapturePane_ansiAndTrailingWhitespace(t *testing.T) {
	input := "\x1b[32m✓ Done\x1b[0m\n\n  \n"
	cleaned := stripANSI(input)
	result := trimPreviewOutput(cleaned)
	expected := "✓ Done"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// trimPreviewOutput mirrors the TrimRight logic from CapturePane for testing.
func trimPreviewOutput(s string) string {
	// This duplicates the strings.TrimRight call in CapturePane
	// so we can test it without needing a real SSH connection.
	return trimRight(s)
}

func trimRight(s string) string {
	// Same cutset as CapturePane
	i := len(s)
	for i > 0 {
		r := s[i-1]
		if r == '\n' || r == '\t' || r == ' ' {
			i--
		} else {
			break
		}
	}
	return s[:i]
}
