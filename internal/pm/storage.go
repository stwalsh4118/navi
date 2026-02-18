package pm

import (
	"errors"
	"os"

	"github.com/stwalsh4118/navi/internal/pathutil"
)

const (
	pmDir             = "~/.config/navi/pm"
	memoryDir         = "~/.config/navi/pm/memory"
	projectsMemoryDir = "~/.config/navi/pm/memory/projects"
	snapshotsDir      = "~/.config/navi/pm/snapshots"
	systemPromptFile  = "~/.config/navi/pm/system-prompt.md"
	outputSchemaFile  = "~/.config/navi/pm/output-schema.json"
	lastOutputFile    = "~/.config/navi/pm/last-output.json"
	shortTermMemFile  = "~/.config/navi/pm/memory/short-term.md"
	longTermMemFile   = "~/.config/navi/pm/memory/long-term.md"
)

const (
	shortTermMemoryTemplate = "# Short-term PM memory\n\n"
	longTermMemoryTemplate  = "# Long-term PM memory\n\n"
)

func EnsureStorageLayout() error {
	for _, dir := range []string{pmDir, memoryDir, projectsMemoryDir, snapshotsDir} {
		if err := os.MkdirAll(resolveStoragePath(dir), 0755); err != nil {
			return err
		}
	}

	if err := seedFileIfMissing(resolveStoragePath(shortTermMemFile), shortTermMemoryTemplate); err != nil {
		return err
	}
	if err := seedFileIfMissing(resolveStoragePath(longTermMemFile), longTermMemoryTemplate); err != nil {
		return err
	}

	if err := os.WriteFile(resolveStoragePath(systemPromptFile), []byte(SystemPromptTemplate), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(resolveStoragePath(outputSchemaFile), []byte(OutputSchemaTemplate), 0644); err != nil {
		return err
	}

	return nil
}

func seedFileIfMissing(path, content string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return os.WriteFile(path, []byte(content), 0644)
}

func resolveStoragePath(path string) string {
	return pathutil.ExpandPath(path)
}
