package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ReadStatusFiles reads all JSON status files from the specified directory
// and parses them into Info structs.
// Returns an empty slice if directory doesn't exist.
// Malformed JSON files are skipped silently.
func ReadStatusFiles(dir string) ([]Info, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []Info
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var s Info
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}

		sessions = append(sessions, s)
	}

	return sessions, nil
}
