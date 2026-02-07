package task

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// writeScript creates an executable script file in dir with the given content.
// On non-Unix systems, it wraps the content in a shell-compatible format.
func writeScript(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("failed to write script %s: %v", name, err)
	}
	return path
}

func TestBuildEnvVars(t *testing.T) {
	tests := []struct {
		name string
		args map[string]string
		want map[string]string
	}{
		{
			name: "single arg",
			args: map[string]string{"repo": "owner/repo"},
			want: map[string]string{"NAVI_TASK_ARG_REPO": "owner/repo"},
		},
		{
			name: "multiple args",
			args: map[string]string{"repo": "owner/repo", "token": "abc123"},
			want: map[string]string{
				"NAVI_TASK_ARG_REPO":  "owner/repo",
				"NAVI_TASK_ARG_TOKEN": "abc123",
			},
		},
		{
			name: "mixed case keys uppercased",
			args: map[string]string{"myKey": "value"},
			want: map[string]string{"NAVI_TASK_ARG_MYKEY": "value"},
		},
		{
			name: "nil args",
			args: nil,
			want: nil,
		},
		{
			name: "empty args",
			args: map[string]string{},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildEnvVars(tt.args)
			if tt.want == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			got := make(map[string]string)
			for _, env := range result {
				parts := splitEnvVar(env)
				got[parts[0]] = parts[1]
			}

			for k, v := range tt.want {
				if gotV, ok := got[k]; !ok {
					t.Errorf("missing env var %s", k)
				} else if gotV != v {
					t.Errorf("env var %s: got %q, want %q", k, gotV, v)
				}
			}

			if len(got) != len(tt.want) {
				t.Errorf("got %d env vars, want %d", len(got), len(tt.want))
			}
		})
	}
}

// splitEnvVar splits "KEY=VALUE" into [KEY, VALUE].
func splitEnvVar(s string) [2]string {
	for i := range s {
		if s[i] == '=' {
			return [2]string{s[:i], s[i+1:]}
		}
	}
	return [2]string{s, ""}
}

func TestResolveProvider_AbsolutePath(t *testing.T) {
	dir := t.TempDir()
	scriptPath := writeScript(t, dir, "myprovider.sh", "#!/bin/sh\necho '{}'\n")

	resolved, err := ResolveProvider(scriptPath, "/some/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != scriptPath {
		t.Errorf("expected %q, got %q", scriptPath, resolved)
	}
}

func TestResolveProvider_RelativePath(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "custom-provider.sh", "#!/bin/sh\necho '{}'\n")

	resolved, err := ResolveProvider("custom-provider.sh", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(dir, "custom-provider.sh")
	if resolved != expected {
		t.Errorf("expected %q, got %q", expected, resolved)
	}
}

func TestResolveProvider_BuiltinName(t *testing.T) {
	dir := t.TempDir()
	providersDir := filepath.Join(dir, "providers")
	if err := os.MkdirAll(providersDir, 0o755); err != nil {
		t.Fatalf("failed to create providers dir: %v", err)
	}
	writeScript(t, providersDir, "github-issues.sh", "#!/bin/sh\necho '{}'\n")

	origDir := ProvidersDir
	ProvidersDir = providersDir
	t.Cleanup(func() { ProvidersDir = origDir })

	resolved, err := ResolveProvider("github-issues", "/some/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(providersDir, "github-issues.sh")
	if resolved != expected {
		t.Errorf("expected %q, got %q", expected, resolved)
	}
}

func TestResolveProvider_UnknownBuiltin(t *testing.T) {
	dir := t.TempDir()

	origDir := ProvidersDir
	ProvidersDir = dir
	t.Cleanup(func() { ProvidersDir = origDir })

	_, err := ResolveProvider("nonexistent-provider", "/some/project")
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}

	var provErr *ProviderError
	if !errors.As(err, &provErr) {
		t.Fatalf("expected ProviderError, got %T: %v", err, err)
	}
	if provErr.Type != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", provErr.Type)
	}
}

func TestResolveProvider_NotFound(t *testing.T) {
	_, err := ResolveProvider("/nonexistent/path/provider.sh", "/some/project")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var provErr *ProviderError
	if !errors.As(err, &provErr) {
		t.Fatalf("expected ProviderError, got %T: %v", err, err)
	}
	if provErr.Type != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", provErr.Type)
	}
}

func TestExecuteProvider_ValidJSON(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell script test on Windows")
	}

	dir := t.TempDir()
	writeScript(t, dir, "provider.sh", `#!/bin/sh
echo '{"tasks": [{"id": "1", "title": "Test task", "status": "open"}]}'
`)

	config := ProjectConfig{
		Tasks: ProjectTaskConfig{
			Provider: filepath.Join(dir, "provider.sh"),
		},
		ProjectDir: dir,
	}

	result, err := ExecuteProvider(config, DefaultProviderTimeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(result.Tasks))
	}
	if result.Tasks[0].ID != "1" {
		t.Errorf("expected task ID '1', got %q", result.Tasks[0].ID)
	}
	if result.Tasks[0].Title != "Test task" {
		t.Errorf("expected title 'Test task', got %q", result.Tasks[0].Title)
	}
	if result.Tasks[0].Status != "open" {
		t.Errorf("expected status 'open', got %q", result.Tasks[0].Status)
	}
}

func TestExecuteProvider_GroupedJSON(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell script test on Windows")
	}

	dir := t.TempDir()
	writeScript(t, dir, "provider.sh", `#!/bin/sh
echo '{"groups": [{"id": "g1", "title": "Group 1", "tasks": [{"id": "1", "title": "Task 1", "status": "open"}]}]}'
`)

	config := ProjectConfig{
		Tasks: ProjectTaskConfig{
			Provider: filepath.Join(dir, "provider.sh"),
		},
		ProjectDir: dir,
	}

	result, err := ExecuteProvider(config, DefaultProviderTimeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(result.Groups))
	}
	if result.Groups[0].ID != "g1" {
		t.Errorf("expected group ID 'g1', got %q", result.Groups[0].ID)
	}
	if len(result.Groups[0].Tasks) != 1 {
		t.Fatalf("expected 1 task in group, got %d", len(result.Groups[0].Tasks))
	}
}

func TestExecuteProvider_Timeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell script test on Windows")
	}

	dir := t.TempDir()
	writeScript(t, dir, "slow-provider.sh", `#!/bin/sh
sleep 10
echo '{}'
`)

	config := ProjectConfig{
		Tasks: ProjectTaskConfig{
			Provider: filepath.Join(dir, "slow-provider.sh"),
		},
		ProjectDir: dir,
	}

	timeout := 100 * time.Millisecond
	_, err := ExecuteProvider(config, timeout)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	var provErr *ProviderError
	if !errors.As(err, &provErr) {
		t.Fatalf("expected ProviderError, got %T: %v", err, err)
	}
	if provErr.Type != ErrTimeout {
		t.Errorf("expected ErrTimeout, got %v", provErr.Type)
	}
}

func TestExecuteProvider_NonZeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell script test on Windows")
	}

	dir := t.TempDir()
	writeScript(t, dir, "failing-provider.sh", `#!/bin/sh
echo "something went wrong" >&2
exit 1
`)

	config := ProjectConfig{
		Tasks: ProjectTaskConfig{
			Provider: filepath.Join(dir, "failing-provider.sh"),
		},
		ProjectDir: dir,
	}

	_, err := ExecuteProvider(config, DefaultProviderTimeout)
	if err == nil {
		t.Fatal("expected error for non-zero exit, got nil")
	}

	var provErr *ProviderError
	if !errors.As(err, &provErr) {
		t.Fatalf("expected ProviderError, got %T: %v", err, err)
	}
	if provErr.Type != ErrExec {
		t.Errorf("expected ErrExec, got %v", provErr.Type)
	}
	if provErr.Stderr == "" {
		t.Error("expected non-empty stderr")
	}
	if provErr.Stderr != "something went wrong\n" {
		t.Errorf("expected stderr 'something went wrong\\n', got %q", provErr.Stderr)
	}
}

func TestExecuteProvider_MalformedJSON(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell script test on Windows")
	}

	dir := t.TempDir()
	writeScript(t, dir, "bad-json-provider.sh", `#!/bin/sh
echo "not valid json at all"
`)

	config := ProjectConfig{
		Tasks: ProjectTaskConfig{
			Provider: filepath.Join(dir, "bad-json-provider.sh"),
		},
		ProjectDir: dir,
	}

	_, err := ExecuteProvider(config, DefaultProviderTimeout)
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}

	var provErr *ProviderError
	if !errors.As(err, &provErr) {
		t.Fatalf("expected ProviderError, got %T: %v", err, err)
	}
	if provErr.Type != ErrParse {
		t.Errorf("expected ErrParse, got %v", provErr.Type)
	}
}

func TestExecuteProvider_EnvVarsPassed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell script test on Windows")
	}

	dir := t.TempDir()
	// Script that reads env vars and outputs them as task titles.
	writeScript(t, dir, "env-provider.sh", `#!/bin/sh
echo "{\"tasks\": [{\"id\": \"1\", \"title\": \"$NAVI_TASK_ARG_REPO\", \"status\": \"$NAVI_TASK_ARG_LABEL\"}]}"
`)

	config := ProjectConfig{
		Tasks: ProjectTaskConfig{
			Provider: filepath.Join(dir, "env-provider.sh"),
			Args: map[string]string{
				"repo":  "owner/repo",
				"label": "bug",
			},
		},
		ProjectDir: dir,
	}

	result, err := ExecuteProvider(config, DefaultProviderTimeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(result.Tasks))
	}
	if result.Tasks[0].Title != "owner/repo" {
		t.Errorf("expected title 'owner/repo' (from NAVI_TASK_ARG_REPO), got %q", result.Tasks[0].Title)
	}
	if result.Tasks[0].Status != "bug" {
		t.Errorf("expected status 'bug' (from NAVI_TASK_ARG_LABEL), got %q", result.Tasks[0].Status)
	}
}

func TestExecuteProvider_ScriptNotFound(t *testing.T) {
	config := ProjectConfig{
		Tasks: ProjectTaskConfig{
			Provider: "/nonexistent/provider.sh",
		},
		ProjectDir: "/tmp",
	}

	_, err := ExecuteProvider(config, DefaultProviderTimeout)
	if err == nil {
		t.Fatal("expected error for missing script, got nil")
	}

	var provErr *ProviderError
	if !errors.As(err, &provErr) {
		t.Fatalf("expected ProviderError, got %T: %v", err, err)
	}
	if provErr.Type != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", provErr.Type)
	}
}
