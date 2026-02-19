package audio

import (
	"os"
	"path/filepath"
	"testing"
)

func createTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
		t.Fatalf("write test file %s: %v", path, err)
	}
}

func withPackBaseDir(t *testing.T, dir string) {
	t.Helper()
	old := packBaseDir
	packBaseDir = func() string { return dir }
	t.Cleanup(func() { packBaseDir = old })
}

func TestScanPackSingleFiles(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, filepath.Join(dir, "done.wav"))
	createTestFile(t, filepath.Join(dir, "error.mp3"))

	result, err := ScanPack(dir)
	if err != nil {
		t.Fatalf("ScanPack error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 events, got %d", len(result))
	}
	if len(result["done"]) != 1 {
		t.Fatalf("expected 1 file for done, got %d", len(result["done"]))
	}
	if len(result["error"]) != 1 {
		t.Fatalf("expected 1 file for error, got %d", len(result["error"]))
	}
}

func TestScanPackMultiFiles(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, filepath.Join(dir, "waiting-1.wav"))
	createTestFile(t, filepath.Join(dir, "waiting-2.wav"))
	createTestFile(t, filepath.Join(dir, "waiting-3.mp3"))

	result, err := ScanPack(dir)
	if err != nil {
		t.Fatalf("ScanPack error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 event, got %d", len(result))
	}
	if len(result["waiting"]) != 3 {
		t.Fatalf("expected 3 files for waiting, got %d", len(result["waiting"]))
	}
}

func TestScanPackMixed(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, filepath.Join(dir, "done.wav"))
	createTestFile(t, filepath.Join(dir, "error-1.ogg"))
	createTestFile(t, filepath.Join(dir, "error-2.ogg"))

	result, err := ScanPack(dir)
	if err != nil {
		t.Fatalf("ScanPack error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 events, got %d", len(result))
	}
	if len(result["done"]) != 1 {
		t.Fatalf("expected 1 file for done, got %d", len(result["done"]))
	}
	if len(result["error"]) != 2 {
		t.Fatalf("expected 2 files for error, got %d", len(result["error"]))
	}
}

func TestScanPackExtensionDetection(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, filepath.Join(dir, "done.wav"))
	createTestFile(t, filepath.Join(dir, "error.mp3"))
	createTestFile(t, filepath.Join(dir, "waiting.ogg"))
	createTestFile(t, filepath.Join(dir, "permission.flac"))
	createTestFile(t, filepath.Join(dir, "readme.txt"))
	createTestFile(t, filepath.Join(dir, "notes.md"))

	result, err := ScanPack(dir)
	if err != nil {
		t.Fatalf("ScanPack error: %v", err)
	}

	if len(result) != 4 {
		t.Fatalf("expected 4 events (txt/md ignored), got %d", len(result))
	}
}

func TestScanPackEmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	result, err := ScanPack(dir)
	if err != nil {
		t.Fatalf("ScanPack error: %v", err)
	}

	if len(result) != 0 {
		t.Fatalf("expected empty map, got %d events", len(result))
	}
}

func TestScanPackMissingDirectory(t *testing.T) {
	_, err := ScanPack(filepath.Join(t.TempDir(), "nonexistent"))
	if err == nil {
		t.Fatalf("expected error for missing directory")
	}
}

func TestScanPackHyphenatedEventNames(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, filepath.Join(dir, "my-event-1.wav"))
	createTestFile(t, filepath.Join(dir, "my-event-2.wav"))

	result, err := ScanPack(dir)
	if err != nil {
		t.Fatalf("ScanPack error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 event, got %d: %v", len(result), result)
	}
	if len(result["my-event"]) != 2 {
		t.Fatalf("expected 2 files for my-event, got %d", len(result["my-event"]))
	}
}

func TestScanPackSortedOutput(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, filepath.Join(dir, "waiting-3.wav"))
	createTestFile(t, filepath.Join(dir, "waiting-1.wav"))
	createTestFile(t, filepath.Join(dir, "waiting-2.wav"))

	result, err := ScanPack(dir)
	if err != nil {
		t.Fatalf("ScanPack error: %v", err)
	}

	paths := result["waiting"]
	if len(paths) != 3 {
		t.Fatalf("expected 3 files, got %d", len(paths))
	}

	for i := 1; i < len(paths); i++ {
		if paths[i] < paths[i-1] {
			t.Fatalf("files not sorted: %v", paths)
		}
	}
}

func TestScanPackIgnoresSubdirectories(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, filepath.Join(dir, "done.wav"))
	subDir := filepath.Join(dir, "subdir")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	createTestFile(t, filepath.Join(subDir, "hidden.wav"))

	result, err := ScanPack(dir)
	if err != nil {
		t.Fatalf("ScanPack error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 event (subdirs ignored), got %d", len(result))
	}
}

func TestResolveSoundFilesValid(t *testing.T) {
	baseDir := t.TempDir()
	withPackBaseDir(t, baseDir)

	packDir := filepath.Join(baseDir, "starcraft")
	if err := os.Mkdir(packDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	createTestFile(t, filepath.Join(packDir, "done.wav"))
	createTestFile(t, filepath.Join(packDir, "waiting-1.wav"))
	createTestFile(t, filepath.Join(packDir, "waiting-2.wav"))

	result, err := ResolveSoundFiles("starcraft")
	if err != nil {
		t.Fatalf("ResolveSoundFiles error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 events, got %d", len(result))
	}
	if len(result["waiting"]) != 2 {
		t.Fatalf("expected 2 waiting files, got %d", len(result["waiting"]))
	}
}

func TestResolveSoundFilesEmptyName(t *testing.T) {
	_, err := ResolveSoundFiles("")
	if err == nil {
		t.Fatalf("expected error for empty pack name")
	}
}

func TestResolveSoundFilesMissingPack(t *testing.T) {
	baseDir := t.TempDir()
	withPackBaseDir(t, baseDir)

	_, err := ResolveSoundFiles("nonexistent")
	if err == nil {
		t.Fatalf("expected error for missing pack")
	}
}

func TestListPacksMultiplePacks(t *testing.T) {
	baseDir := t.TempDir()
	withPackBaseDir(t, baseDir)

	pack1 := filepath.Join(baseDir, "starcraft")
	pack2 := filepath.Join(baseDir, "retro")
	if err := os.Mkdir(pack1, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Mkdir(pack2, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	createTestFile(t, filepath.Join(pack1, "done.wav"))
	createTestFile(t, filepath.Join(pack1, "error.wav"))
	createTestFile(t, filepath.Join(pack1, "waiting-1.wav"))
	createTestFile(t, filepath.Join(pack1, "waiting-2.wav"))

	createTestFile(t, filepath.Join(pack2, "done.mp3"))

	packs, err := ListPacks()
	if err != nil {
		t.Fatalf("ListPacks error: %v", err)
	}

	if len(packs) != 2 {
		t.Fatalf("expected 2 packs, got %d", len(packs))
	}

	packMap := make(map[string]PackInfo)
	for _, p := range packs {
		packMap[p.Name] = p
	}

	sc := packMap["starcraft"]
	if sc.EventCount != 3 {
		t.Fatalf("expected 3 events in starcraft, got %d", sc.EventCount)
	}
	if sc.FileCount != 4 {
		t.Fatalf("expected 4 files in starcraft, got %d", sc.FileCount)
	}

	retro := packMap["retro"]
	if retro.EventCount != 1 {
		t.Fatalf("expected 1 event in retro, got %d", retro.EventCount)
	}
	if retro.FileCount != 1 {
		t.Fatalf("expected 1 file in retro, got %d", retro.FileCount)
	}
}

func TestListPacksMissingBaseDir(t *testing.T) {
	withPackBaseDir(t, filepath.Join(t.TempDir(), "nonexistent"))

	packs, err := ListPacks()
	if err != nil {
		t.Fatalf("expected no error for missing base dir, got %v", err)
	}
	if packs == nil {
		t.Fatalf("expected non-nil empty slice")
	}
	if len(packs) != 0 {
		t.Fatalf("expected empty slice, got %d", len(packs))
	}
}

func TestListPacksEmptyBaseDir(t *testing.T) {
	baseDir := t.TempDir()
	withPackBaseDir(t, baseDir)

	packs, err := ListPacks()
	if err != nil {
		t.Fatalf("ListPacks error: %v", err)
	}
	if len(packs) != 0 {
		t.Fatalf("expected empty slice, got %d", len(packs))
	}
}
