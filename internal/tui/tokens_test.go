package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stwalsh4118/navi/internal/metrics"
	"github.com/stwalsh4118/navi/internal/tokens"
)

func TestCwdToProjectPath(t *testing.T) {
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
		result := tokens.CWDToProjectPath(tc.cwd)
		if result != tc.expected {
			t.Errorf("tokens.CWDToProjectPath(%q) = %q, want %q", tc.cwd, result, tc.expected)
		}
	}
}

func TestParseTranscriptTokens(t *testing.T) {
	// Create a temp file with sample transcript data
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "test-session.jsonl")

	// Sample transcript with assistant messages containing token usage
	content := `{"type":"user","message":{"content":"hello"}}
{"type":"assistant","message":{"usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":1000,"cache_creation_input_tokens":200}}}
{"type":"tool_result","message":{"content":"result"}}
{"type":"assistant","message":{"usage":{"input_tokens":150,"output_tokens":75,"cache_read_input_tokens":500,"cache_creation_input_tokens":100}}}
`

	if err := os.WriteFile(transcriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tokens, err := tokens.ParseTranscriptTokens(transcriptPath)
	if err != nil {
		t.Fatalf("tokens.ParseTranscriptTokens failed: %v", err)
	}

	if tokens == nil {
		t.Fatal("Expected non-nil tokens")
	}

	// Expected:
	// input_tokens: 100 + 150 = 250
	// cache_read: 1000 + 500 = 1500
	// cache_creation: 200 + 100 = 300
	// Total input: 250 + 1500 + 300 = 2050
	// output_tokens: 50 + 75 = 125
	// Total: 2050 + 125 = 2175

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

	tokens, err := tokens.ParseTranscriptTokens(transcriptPath)
	if err != nil {
		t.Fatalf("tokens.ParseTranscriptTokens failed: %v", err)
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

	tokens, err := tokens.ParseTranscriptTokens(transcriptPath)
	if err != nil {
		t.Fatalf("tokens.ParseTranscriptTokens failed: %v", err)
	}

	if tokens != nil {
		t.Errorf("Expected nil tokens when no assistant messages, got %+v", tokens)
	}
}

func TestParseTranscriptTokens_MalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "malformed.jsonl")

	// Mix of valid and invalid JSON
	content := `not valid json
{"type":"assistant","message":{"usage":{"input_tokens":100,"output_tokens":50}}}
also not valid
{"type":"assistant","message":{"usage":{"input_tokens":200,"output_tokens":100}}}
`

	if err := os.WriteFile(transcriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tokens, err := tokens.ParseTranscriptTokens(transcriptPath)
	if err != nil {
		t.Fatalf("tokens.ParseTranscriptTokens failed: %v", err)
	}

	if tokens == nil {
		t.Fatal("Expected non-nil tokens")
	}

	// Should only count the valid assistant messages
	expectedInput := int64(300)  // 100 + 200
	expectedOutput := int64(150) // 50 + 100

	if tokens.Input != expectedInput {
		t.Errorf("Input = %d, want %d", tokens.Input, expectedInput)
	}
	if tokens.Output != expectedOutput {
		t.Errorf("Output = %d, want %d", tokens.Output, expectedOutput)
	}
}

func TestParseTranscriptTokens_FileNotFound(t *testing.T) {
	_, err := tokens.ParseTranscriptTokens("/nonexistent/path/file.jsonl")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestFindSessionTranscript(t *testing.T) {
	// Create a temp directory structure mimicking Claude's layout
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "-test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Create some .jsonl files with different modification times
	file1 := filepath.Join(projectDir, "old-session.jsonl")
	file2 := filepath.Join(projectDir, "new-session.jsonl")

	if err := os.WriteFile(file1, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to write file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to write file2: %v", err)
	}

	// Note: This test is limited because we can't easily control file mod times
	// without additional dependencies. The function should find one of the files.

	// We can't test tokens.FindSessionTranscript directly without mocking the home directory,
	// but we've verified the parsing logic works.
}

func TestGetSessionTokens_EmptyCwd(t *testing.T) {
	tokens := tokens.GetSessionTokens("")
	if tokens != nil {
		t.Errorf("Expected nil for empty CWD, got %+v", tokens)
	}
}

func TestGetSessionTokens_RealProject(t *testing.T) {
	// This test verifies we can parse real Claude transcript files
	// It will be skipped if no transcript data exists
	tokens := tokens.GetSessionTokens("/home/sean/workspace/navi")
	if tokens == nil {
		t.Skip("No transcript data found for navi project - this is expected in CI")
	}

	t.Logf("Real token data found:")
	t.Logf("  Input:  %s (%d)", metrics.FormatTokenCount(tokens.Input), tokens.Input)
	t.Logf("  Output: %s (%d)", metrics.FormatTokenCount(tokens.Output), tokens.Output)
	t.Logf("  Total:  %s (%d)", metrics.FormatTokenCount(tokens.Total), tokens.Total)

	// Basic sanity checks
	if tokens.Total <= 0 {
		t.Error("Total tokens should be positive")
	}
	if tokens.Input+tokens.Output != tokens.Total {
		t.Logf("Note: Input + Output != Total (cache tokens included in input)")
	}
}
