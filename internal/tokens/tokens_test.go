package tokens

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stwalsh4118/navi/internal/metrics"
)

func TestCWDToProjectPath(t *testing.T) {
	tests := []struct {
		cwd      string
		expected string
	}{
		{"/home/sean/workspace/navi", "-home-sean-workspace-navi"},
		{"/home/user/projects/my-app", "-home-user-projects-my-app"},
		{"/tmp", "-tmp"},
		{"/", "-"},
	}

	for _, tc := range tests {
		result := CWDToProjectPath(tc.cwd)
		if result != tc.expected {
			t.Errorf("CWDToProjectPath(%q) = %q, want %q", tc.cwd, result, tc.expected)
		}
	}
}

func TestParseTranscriptTokens(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "test-session.jsonl")

	content := `{"type":"user","message":{"content":"hello"}}
{"type":"assistant","message":{"usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":1000,"cache_creation_input_tokens":200}}}
{"type":"tool_result","message":{"content":"result"}}
{"type":"assistant","message":{"usage":{"input_tokens":150,"output_tokens":75,"cache_read_input_tokens":500,"cache_creation_input_tokens":100}}}
`

	if err := os.WriteFile(transcriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tokens, err := ParseTranscriptTokens(transcriptPath)
	if err != nil {
		t.Fatalf("ParseTranscriptTokens failed: %v", err)
	}

	if tokens == nil {
		t.Fatal("Expected non-nil tokens")
	}

	expectedInput := int64(2050)
	expectedOutput := int64(125)
	expectedTotal := int64(2175)

	if tokens.Input != expectedInput {
		t.Errorf("Input = %d, want %d", tokens.Input, expectedInput)
	}
	if tokens.Output != expectedOutput {
		t.Errorf("Output = %d, want %d", tokens.Output, expectedOutput)
	}
	if tokens.Total != expectedTotal {
		t.Errorf("Total = %d, want %d", tokens.Total, expectedTotal)
	}
}

func TestParseTranscriptTokens_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "empty.jsonl")

	if err := os.WriteFile(transcriptPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tokens, err := ParseTranscriptTokens(transcriptPath)
	if err != nil {
		t.Fatalf("ParseTranscriptTokens failed: %v", err)
	}

	if tokens != nil {
		t.Errorf("Expected nil tokens for empty file, got %+v", tokens)
	}
}

func TestParseTranscriptTokens_NoAssistantMessages(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "no-assistant.jsonl")

	content := `{"type":"user","message":{"content":"hello"}}
{"type":"tool_result","message":{"content":"result"}}
`

	if err := os.WriteFile(transcriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tokens, err := ParseTranscriptTokens(transcriptPath)
	if err != nil {
		t.Fatalf("ParseTranscriptTokens failed: %v", err)
	}

	if tokens != nil {
		t.Errorf("Expected nil tokens when no assistant messages, got %+v", tokens)
	}
}

func TestParseTranscriptTokens_MalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "malformed.jsonl")

	content := `not valid json
{"type":"assistant","message":{"usage":{"input_tokens":100,"output_tokens":50}}}
also not valid
{"type":"assistant","message":{"usage":{"input_tokens":200,"output_tokens":100}}}
`

	if err := os.WriteFile(transcriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tokens, err := ParseTranscriptTokens(transcriptPath)
	if err != nil {
		t.Fatalf("ParseTranscriptTokens failed: %v", err)
	}

	if tokens == nil {
		t.Fatal("Expected non-nil tokens")
	}

	expectedInput := int64(300)
	expectedOutput := int64(150)

	if tokens.Input != expectedInput {
		t.Errorf("Input = %d, want %d", tokens.Input, expectedInput)
	}
	if tokens.Output != expectedOutput {
		t.Errorf("Output = %d, want %d", tokens.Output, expectedOutput)
	}
}

func TestParseTranscriptTokens_FileNotFound(t *testing.T) {
	_, err := ParseTranscriptTokens("/nonexistent/path/file.jsonl")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestGetSessionTokens_EmptyCwd(t *testing.T) {
	tokens := GetSessionTokens("")
	if tokens != nil {
		t.Errorf("Expected nil for empty CWD, got %+v", tokens)
	}
}

func TestGetSessionTokens_RealProject(t *testing.T) {
	tokens := GetSessionTokens("/home/sean/workspace/navi")
	if tokens == nil {
		t.Skip("No transcript data found for navi project - this is expected in CI")
	}

	t.Logf("Real token data found:")
	t.Logf("  Input:  %s (%d)", metrics.FormatTokenCount(tokens.Input), tokens.Input)
	t.Logf("  Output: %s (%d)", metrics.FormatTokenCount(tokens.Output), tokens.Output)
	t.Logf("  Total:  %s (%d)", metrics.FormatTokenCount(tokens.Total), tokens.Total)

	if tokens.Total <= 0 {
		t.Error("Total tokens should be positive")
	}
}
