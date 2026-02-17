package pm

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/stwalsh4118/navi/internal/pathutil"
)

const EventLogPath = "~/.config/navi/pm/events.jsonl"

var eventLogPath = EventLogPath

// AppendEvents prunes old events, then appends new events as JSONL.
func AppendEvents(events []Event) error {
	if err := PruneEvents(); err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	path := resolveEventLogPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, event := range events {
		line, err := json.Marshal(event)
		if err != nil {
			return err
		}
		if _, err := file.Write(append(line, '\n')); err != nil {
			return err
		}
	}

	return nil
}

// ReadEvents reads event log lines and returns valid events.
func ReadEvents() ([]Event, error) {
	path := resolveEventLogPath()
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Event{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var events []Event
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var event Event
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}
		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// PruneEvents removes events older than 24 hours.
func PruneEvents() error {
	path := resolveEventLogPath()
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	events, err := ReadEvents()
	if err != nil {
		return err
	}

	cutoff := time.Now().UTC().Add(-24 * time.Hour)
	retained := make([]Event, 0, len(events))
	for _, event := range events {
		if event.Timestamp.IsZero() || !event.Timestamp.Before(cutoff) {
			retained = append(retained, event)
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(path), "events-*.jsonl")
	if err != nil {
		return err
	}

	tmpPath := tmpFile.Name()
	for _, event := range retained {
		line, err := json.Marshal(event)
		if err != nil {
			tmpFile.Close()
			_ = os.Remove(tmpPath)
			return err
		}
		if _, err := tmpFile.Write(append(line, '\n')); err != nil {
			tmpFile.Close()
			_ = os.Remove(tmpPath)
			return err
		}
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

func resolveEventLogPath() string {
	return pathutil.ExpandPath(eventLogPath)
}
