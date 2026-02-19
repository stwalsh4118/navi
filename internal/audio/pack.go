package audio

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/stwalsh4118/navi/internal/pathutil"
)

const defaultPackDir = "~/.config/navi/soundpacks"

// packBaseDir is the resolved base directory for sound packs.
// Package-level var allows test override.
var packBaseDir = func() string {
	return pathutil.ExpandPath(defaultPackDir)
}

var (
	supportedExtensions = map[string]bool{
		".wav":  true,
		".mp3":  true,
		".ogg":  true,
		".flac": true,
	}

	// Matches event-N pattern (e.g., "waiting-2").
	multiSoundPattern = regexp.MustCompile(`^(.+)-(\d+)$`)
)

// PackInfo describes an installed sound pack.
type PackInfo struct {
	Name       string
	EventCount int
	FileCount  int
}

// ScanPack reads a pack directory and returns a map of event name → file paths.
// Files are matched by supported extensions and grouped by event name.
// Multi-sound variants (e.g., waiting-1.wav, waiting-2.wav) are grouped together.
func ScanPack(packDir string) (map[string][]string, error) {
	entries, err := os.ReadDir(packDir)
	if err != nil {
		return nil, fmt.Errorf("read pack directory: %w", err)
	}

	result := make(map[string][]string)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if !supportedExtensions[ext] {
			continue
		}

		base := strings.TrimSuffix(name, ext)
		event := base

		if matches := multiSoundPattern.FindStringSubmatch(base); matches != nil {
			event = matches[1]
		}

		fullPath := filepath.Join(packDir, name)
		result[event] = append(result[event], fullPath)
	}

	for event := range result {
		sort.Strings(result[event])
	}

	return result, nil
}

// ResolveSoundFiles resolves sound files from the soundpacks directory for a given pack name.
func ResolveSoundFiles(packName string) (map[string][]string, error) {
	if packName == "" {
		return nil, fmt.Errorf("pack name is required")
	}

	packDir := filepath.Join(packBaseDir(), packName)

	if _, err := os.Stat(packDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("sound pack %q not found: %s", packName, packDir)
	}

	return ScanPack(packDir)
}

// ListPacks enumerates available sound packs from the soundpacks directory.
// Returns an empty slice (not error) if the directory does not exist.
func ListPacks() ([]PackInfo, error) {
	baseDir := packBaseDir()

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []PackInfo{}, nil
		}
		return nil, fmt.Errorf("read soundpacks directory: %w", err)
	}

	var packs []PackInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		packDir := filepath.Join(baseDir, entry.Name())
		files, err := ScanPack(packDir)
		if err != nil {
			continue
		}

		fileCount := 0
		for _, paths := range files {
			fileCount += len(paths)
		}

		packs = append(packs, PackInfo{
			Name:       entry.Name(),
			EventCount: len(files),
			FileCount:  fileCount,
		})
	}

	if packs == nil {
		packs = []PackInfo{}
	}

	return packs, nil
}
