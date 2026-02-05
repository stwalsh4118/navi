package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// claudeProjectsDir is the base directory for Claude project data
const claudeProjectsDir = ".claude/projects"

// cwdToProjectPath converts a working directory to Claude's project folder format.
// Example: "/home/sean/workspace/navi" -> "-home-sean-workspace-navi"
func cwdToProjectPath(cwd string) string {
	// Expand ~ if present
	if strings.HasPrefix(cwd, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			cwd = home + cwd[1:]
		}
	}

	// Clean the path and replace / with -
	cwd = filepath.Clean(cwd)
	return strings.ReplaceAll(cwd, "/", "-")
}

// findSessionTranscript finds the most recently modified .jsonl file in a project folder.
// Returns the full path to the transcript file, or an error if not found.
func findSessionTranscript(projectPath string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	projectDir := filepath.Join(home, claudeProjectsDir, projectPath)

	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return "", err
	}

	var newestFile string
	var newestTime int64

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		modTime := info.ModTime().Unix()
		if modTime > newestTime {
			newestTime = modTime
			newestFile = filepath.Join(projectDir, entry.Name())
		}
	}

	if newestFile == "" {
		return "", os.ErrNotExist
	}

	return newestFile, nil
}

// transcriptMessage represents the structure of a message in the transcript JSONL
type transcriptMessage struct {
	Type    string `json:"type"`
	Message struct {
		Usage struct {
			InputTokens              int64 `json:"input_tokens"`
			OutputTokens             int64 `json:"output_tokens"`
			CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

// ParseTranscriptTokens parses a .jsonl transcript file and returns aggregated token counts.
// Returns nil if the file cannot be parsed or contains no token data.
func ParseTranscriptTokens(transcriptPath string) (*TokenMetrics, error) {
	file, err := os.Open(transcriptPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var totalInput, totalOutput, totalCacheRead, totalCacheCreation int64

	scanner := bufio.NewScanner(file)
	// Increase buffer size for large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // 1MB max line size

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg transcriptMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			// Skip malformed lines
			continue
		}

		// Only process assistant messages with usage data
		if msg.Type != "assistant" {
			continue
		}

		usage := msg.Message.Usage
		totalInput += usage.InputTokens
		totalOutput += usage.OutputTokens
		totalCacheRead += usage.CacheReadInputTokens
		totalCacheCreation += usage.CacheCreationInputTokens
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Calculate totals
	// Input includes base input plus cache tokens
	input := totalInput + totalCacheRead + totalCacheCreation
	output := totalOutput
	total := input + output

	// Return nil if no tokens found
	if total == 0 {
		return nil, nil
	}

	return &TokenMetrics{
		Input:  input,
		Output: output,
		Total:  total,
	}, nil
}

// GetSessionTokens retrieves token metrics for a session based on its working directory.
// Returns nil if tokens cannot be determined.
func GetSessionTokens(cwd string) *TokenMetrics {
	if cwd == "" {
		return nil
	}

	projectPath := cwdToProjectPath(cwd)
	transcriptPath, err := findSessionTranscript(projectPath)
	if err != nil {
		return nil
	}

	tokens, err := ParseTranscriptTokens(transcriptPath)
	if err != nil {
		return nil
	}

	return tokens
}
