package pm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/stwalsh4118/navi/internal/debug"
)

type CachedOutput struct {
	Briefing *PMBriefing `json:"briefing"`
	CachedAt time.Time   `json:"cached_at"`
	IsStale  bool        `json:"-"`
}

func ParseOutput(raw []byte) (*PMBriefing, error) {
	if len(raw) == 0 {
		return nil, errors.New("empty output")
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		debug.Log("pm: parse envelope failed, raw prefix: %s", truncateBytes(raw, 500))
		return nil, fmt.Errorf("parse output envelope: %w", err)
	}

	// Log available keys for debugging.
	keys := make([]string, 0, len(envelope))
	for k := range envelope {
		keys = append(keys, k)
	}
	debug.Log("pm: envelope keys: %v", keys)

	if structuredRaw, ok := envelope["structured_output"]; ok {
		briefing, err := parseBriefingValue(structuredRaw)
		if err != nil {
			debug.Log("pm: structured_output parse failed: %v, raw prefix: %s", err, truncateBytes(structuredRaw, 300))
			return nil, fmt.Errorf("parse structured_output: %w", err)
		}
		return briefing, nil
	}

	if resultRaw, ok := envelope["result"]; ok {
		briefing, err := parseBriefingValue(resultRaw)
		if err != nil {
			debug.Log("pm: result parse failed: %v, raw prefix: %s", err, truncateBytes(resultRaw, 300))
			return nil, fmt.Errorf("parse result: %w", err)
		}
		return briefing, nil
	}

	briefing, err := parseBriefingValue(raw)
	if err != nil {
		return nil, fmt.Errorf("no structured output found: %w", err)
	}
	return briefing, nil
}

func CacheOutput(briefing *PMBriefing) error {
	if briefing == nil {
		return errors.New("briefing is nil")
	}
	if err := EnsureStorageLayout(); err != nil {
		return err
	}

	path := resolveStoragePath(lastOutputFile)
	dir := filepath.Dir(path)

	payload := CachedOutput{
		Briefing: briefing,
		CachedAt: time.Now().UTC(),
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(dir, "last-output-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(encoded); err != nil {
		tmpFile.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	return nil
}

func LoadCachedOutput() (*CachedOutput, error) {
	path := resolveStoragePath(lastOutputFile)
	encoded, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var cached CachedOutput
	if err := json.Unmarshal(encoded, &cached); err != nil {
		return nil, err
	}

	return &cached, nil
}

func parseBriefingValue(raw json.RawMessage) (*PMBriefing, error) {
	if len(raw) == 0 {
		return nil, errors.New("empty value")
	}

	// If it's a JSON string, unquote and recurse.
	if raw[0] == '"' {
		var encoded string
		if err := json.Unmarshal(raw, &encoded); err != nil {
			return nil, err
		}
		return parseBriefingValue([]byte(encoded))
	}

	// If it's a JSON object, check for nested structured_output/result.
	var nested map[string]json.RawMessage
	if err := json.Unmarshal(raw, &nested); err == nil {
		if structuredRaw, ok := nested["structured_output"]; ok {
			return parseBriefingValue(structuredRaw)
		}
		if resultRaw, ok := nested["result"]; ok {
			return parseBriefingValue(resultRaw)
		}
	}

	// Try parsing directly as a PMBriefing.
	var briefing PMBriefing
	if err := json.Unmarshal(raw, &briefing); err != nil {
		debug.Log("pm: direct unmarshal to PMBriefing failed: %v", err)
	} else if briefing.Summary != "" {
		return &briefing, nil
	} else {
		debug.Log("pm: unmarshaled PMBriefing has empty summary, treating as failed")
	}

	// Last resort: try extracting an embedded JSON object from text.
	// Claude sometimes wraps JSON in prose like "Here is the output:\n{...}"
	if extracted := extractJSON(raw); extracted != nil {
		var extractedBriefing PMBriefing
		if err := json.Unmarshal(extracted, &extractedBriefing); err != nil {
			debug.Log("pm: extracted JSON unmarshal failed: %v", err)
		} else if extractedBriefing.Summary != "" {
			debug.Log("pm: extracted embedded JSON from text response")
			return &extractedBriefing, nil
		}
	}

	return nil, fmt.Errorf("could not parse as briefing: %s", truncateBytes(raw, 200))
}

// extractJSON finds the first top-level JSON object in text.
func extractJSON(text []byte) []byte {
	start := bytes.IndexByte(text, '{')
	if start < 0 {
		return nil
	}
	// Find matching closing brace by counting depth.
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(text); i++ {
		if escaped {
			escaped = false
			continue
		}
		b := text[i]
		if b == '\\' && inString {
			escaped = true
			continue
		}
		if b == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if b == '{' {
			depth++
		} else if b == '}' {
			depth--
			if depth == 0 {
				return text[start : i+1]
			}
		}
	}
	return nil
}

func truncateBytes(b []byte, max int) string {
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "..."
}
