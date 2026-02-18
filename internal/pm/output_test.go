package pm

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestParseOutputStructuredOutputObject(t *testing.T) {
	raw := []byte(`{"session_id":"sid-1","structured_output":{"summary":"hello","projects":[],"attention_items":[],"breadcrumbs":[]}}`)

	briefing, err := ParseOutput(raw)
	if err != nil {
		t.Fatalf("ParseOutput failed: %v", err)
	}
	if briefing.Summary != "hello" {
		t.Fatalf("summary = %q, want hello", briefing.Summary)
	}
}

func TestParseOutputStructuredOutputString(t *testing.T) {
	raw := []byte(`{"structured_output":"{\"summary\":\"from-string\",\"projects\":[],\"attention_items\":[],\"breadcrumbs\":[]}"}`)

	briefing, err := ParseOutput(raw)
	if err != nil {
		t.Fatalf("ParseOutput failed: %v", err)
	}
	if briefing.Summary != "from-string" {
		t.Fatalf("summary = %q, want from-string", briefing.Summary)
	}
}

func TestParseOutputFromResultField(t *testing.T) {
	raw := []byte(`{"result":{"summary":"from-result","projects":[],"attention_items":[],"breadcrumbs":[]}}`)

	briefing, err := ParseOutput(raw)
	if err != nil {
		t.Fatalf("ParseOutput failed: %v", err)
	}
	if briefing.Summary != "from-result" {
		t.Fatalf("summary = %q, want from-result", briefing.Summary)
	}
}

func TestParseOutputMalformedJSON(t *testing.T) {
	_, err := ParseOutput([]byte("{bad-json}"))
	if err == nil {
		t.Fatal("expected ParseOutput error for malformed JSON")
	}
}

func TestParseOutputMissingFields(t *testing.T) {
	raw := []byte(`{"structured_output":{"summary":"partial"}}`)

	briefing, err := ParseOutput(raw)
	if err != nil {
		t.Fatalf("ParseOutput failed: %v", err)
	}
	if briefing.Summary != "partial" {
		t.Fatalf("summary = %q, want partial", briefing.Summary)
	}
	if len(briefing.Projects) != 0 || len(briefing.AttentionItems) != 0 || len(briefing.Breadcrumbs) != 0 {
		t.Fatal("expected missing fields to default to zero values")
	}
}

func TestParseOutputResultStringContainingJSON(t *testing.T) {
	// Simulates stream-json result where the result field is a JSON-encoded string
	// containing the briefing JSON (double-encoded).
	raw := []byte(`{"type":"result","result":"{\"summary\":\"stream result\",\"projects\":[],\"attention_items\":[],\"breadcrumbs\":[{\"timestamp\":\"2026-02-18T17:44:00Z\",\"summary\":\"test\"}]}","session_id":"sid-2"}`)

	briefing, err := ParseOutput(raw)
	if err != nil {
		t.Fatalf("ParseOutput failed: %v", err)
	}
	if briefing.Summary != "stream result" {
		t.Fatalf("summary = %q, want stream result", briefing.Summary)
	}
	if len(briefing.Breadcrumbs) != 1 || briefing.Breadcrumbs[0].Timestamp != "2026-02-18T17:44:00Z" {
		t.Fatalf("breadcrumbs = %+v, want 1 entry with timestamp", briefing.Breadcrumbs)
	}
}

func TestParseOutputResultMarkdownWithEmbeddedJSON(t *testing.T) {
	// Simulates Claude returning prose with JSON embedded in it.
	raw := []byte(`{"type":"result","result":"Here is the output:\n{\"summary\":\"embedded\",\"projects\":[],\"attention_items\":[],\"breadcrumbs\":[]}"}`)

	briefing, err := ParseOutput(raw)
	if err != nil {
		t.Fatalf("ParseOutput failed: %v", err)
	}
	if briefing.Summary != "embedded" {
		t.Fatalf("summary = %q, want embedded", briefing.Summary)
	}
}

func TestParseOutputDirectJSONBriefing(t *testing.T) {
	// Simulates candidateJSON fallback: raw JSON that is directly a briefing.
	raw := []byte(`{"summary":"direct","projects":[{"name":"proj","status":"active","current_work":"work","recent_activity":"recent"}],"attention_items":[],"breadcrumbs":[]}`)

	briefing, err := ParseOutput(raw)
	if err != nil {
		t.Fatalf("ParseOutput failed: %v", err)
	}
	if briefing.Summary != "direct" {
		t.Fatalf("summary = %q, want direct", briefing.Summary)
	}
	if len(briefing.Projects) != 1 || briefing.Projects[0].Name != "proj" {
		t.Fatalf("projects = %+v, want 1 project named proj", briefing.Projects)
	}
}

func TestCacheOutputRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	briefing := &PMBriefing{Summary: "cached"}
	if err := CacheOutput(briefing); err != nil {
		t.Fatalf("CacheOutput failed: %v", err)
	}

	cached, err := LoadCachedOutput()
	if err != nil {
		t.Fatalf("LoadCachedOutput failed: %v", err)
	}
	if cached == nil {
		t.Fatal("expected cached output, got nil")
	}
	if cached.Briefing == nil {
		t.Fatal("expected briefing in cache")
	}
	if cached.Briefing.Summary != "cached" {
		t.Fatalf("cached summary = %q, want cached", cached.Briefing.Summary)
	}
	if cached.CachedAt.IsZero() {
		t.Fatal("cached_at should be set")
	}
}

func TestLoadCachedOutputMissingFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cached, err := LoadCachedOutput()
	if err != nil {
		t.Fatalf("LoadCachedOutput failed: %v", err)
	}
	if cached != nil {
		t.Fatal("expected nil cached output when file is missing")
	}
}

func TestCacheOutputAtomicWrites(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	const writerCount = 20
	var wg sync.WaitGroup
	for i := 0; i < writerCount; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = CacheOutput(&PMBriefing{Summary: "run"})
		}(i)
	}
	wg.Wait()

	cached, err := LoadCachedOutput()
	if err != nil {
		t.Fatalf("LoadCachedOutput failed: %v", err)
	}
	if cached == nil || cached.Briefing == nil {
		t.Fatal("expected valid cached output after concurrent writes")
	}

	dir := filepath.Dir(resolveStoragePath(lastOutputFile))
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "last-output-") && strings.HasSuffix(entry.Name(), ".tmp") {
			t.Fatalf("temporary cache file left behind: %s", entry.Name())
		}
	}
}
