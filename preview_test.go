package main

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

	t.Run("removes cursor movement codes", func(t *testing.T) {
		input := "\x1b[2Amove up\x1b[3Bmove down"
		result := stripANSI(input)
		expected := "move upmove down"
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

	t.Run("preserves plain text", func(t *testing.T) {
		input := "Hello, World!"
		result := stripANSI(input)
		if result != input {
			t.Errorf("plain text should be unchanged, got %q", result)
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

	t.Run("handles empty string", func(t *testing.T) {
		result := stripANSI("")
		if result != "" {
			t.Errorf("empty string should remain empty, got %q", result)
		}
	})

	t.Run("handles realistic Claude output", func(t *testing.T) {
		// Simulate typical Claude Code output with ANSI codes
		input := "\x1b[1m● Read\x1b[0m 1 file (ctrl+o to expand)\n\n\x1b[32m✓\x1b[0m Done"
		result := stripANSI(input)
		expected := "● Read 1 file (ctrl+o to expand)\n\n✓ Done"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}
