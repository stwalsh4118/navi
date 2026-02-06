package tokens

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/stwalsh4118/navi/internal/metrics"
)

// ClaudeProjectsDir is the base directory for Claude project data
const ClaudeProjectsDir = ".claude/projects"

// CWDToProjectPath converts a working directory to Claude's project folder format.
// Example: "/home/sean/workspace/navi" -> "-home-sean-workspace-navi"
func CWDToProjectPath(cwd string) string {
	if strings.HasPrefix(cwd, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			cwd = home + cwd[1:]
		}
	}

	cwd = filepath.Clean(cwd)
	return strings.ReplaceAll(cwd, "/", "-")
}

// FindSessionTranscript finds the most recently modified .jsonl file in a project folder.
func FindSessionTranscript(projectPath string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	projectDir := filepath.Join(home, ClaudeProjectsDir, projectPath)

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
func ParseTranscriptTokens(transcriptPath string) (*metrics.TokenMetrics, error) {
	file, err := os.Open(transcriptPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var totalInput, totalOutput, totalCacheRead, totalCacheCreation int64

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // 1MB max line size

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg transcriptMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

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

	input := totalInput + totalCacheRead + totalCacheCreation
	output := totalOutput
	total := input + output

	if total == 0 {
		return nil, nil
	}

	return &metrics.TokenMetrics{
		Input:  input,
		Output: output,
		Total:  total,
	}, nil
}

// GetSessionTokens retrieves token metrics for a session based on its working directory.
func GetSessionTokens(cwd string) *metrics.TokenMetrics {
	if cwd == "" {
		return nil
	}

	projectPath := CWDToProjectPath(cwd)
	transcriptPath, err := FindSessionTranscript(projectPath)
	if err != nil {
		return nil
	}

	t, err := ParseTranscriptTokens(transcriptPath)
	if err != nil {
		return nil
	}

	return t
}
