package pm

import (
	"os"
	"testing"
)

func TestEnsureStorageLayoutCreatesDirectoriesAndFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := EnsureStorageLayout(); err != nil {
		t.Fatalf("EnsureStorageLayout failed: %v", err)
	}

	for _, dir := range []string{pmDir, memoryDir, projectsMemoryDir, snapshotsDir} {
		resolved := resolveStoragePath(dir)
		info, err := os.Stat(resolved)
		if err != nil {
			t.Fatalf("expected dir %q to exist: %v", resolved, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %q to be directory", resolved)
		}
	}

	for _, file := range []string{shortTermMemFile, longTermMemFile, systemPromptFile, outputSchemaFile} {
		resolved := resolveStoragePath(file)
		content, err := os.ReadFile(resolved)
		if err != nil {
			t.Fatalf("expected file %q to exist: %v", resolved, err)
		}
		if len(content) == 0 {
			t.Fatalf("expected file %q to be non-empty", resolved)
		}
	}
}

func TestEnsureStorageLayoutIsIdempotentAndPreservesMemoryFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := EnsureStorageLayout(); err != nil {
		t.Fatalf("first EnsureStorageLayout failed: %v", err)
	}

	shortTermPath := resolveStoragePath(shortTermMemFile)
	longTermPath := resolveStoragePath(longTermMemFile)

	if err := os.WriteFile(shortTermPath, []byte("custom short\n"), 0644); err != nil {
		t.Fatalf("write short-term memory failed: %v", err)
	}
	if err := os.WriteFile(longTermPath, []byte("custom long\n"), 0644); err != nil {
		t.Fatalf("write long-term memory failed: %v", err)
	}

	if err := EnsureStorageLayout(); err != nil {
		t.Fatalf("second EnsureStorageLayout failed: %v", err)
	}

	shortAfter, err := os.ReadFile(shortTermPath)
	if err != nil {
		t.Fatalf("read short-term memory failed: %v", err)
	}
	if string(shortAfter) != "custom short\n" {
		t.Fatalf("short-term memory was overwritten: %q", string(shortAfter))
	}

	longAfter, err := os.ReadFile(longTermPath)
	if err != nil {
		t.Fatalf("read long-term memory failed: %v", err)
	}
	if string(longAfter) != "custom long\n" {
		t.Fatalf("long-term memory was overwritten: %q", string(longAfter))
	}
}
