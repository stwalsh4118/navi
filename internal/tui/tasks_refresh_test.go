package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stwalsh4118/navi/internal/task"
)

func TestTaskRefreshCmd_AllCachedSkipsProviderExecution(t *testing.T) {
	cache := task.NewResultCache()
	globalConfig := &task.GlobalConfig{}

	projectA := t.TempDir()
	projectB := t.TempDir()

	cache.Set(projectA, &task.ProviderResult{Tasks: []task.Task{{ID: "a-1", Title: "Cached A", Status: "todo"}}}, nil)
	cache.Set(projectB, &task.ProviderResult{Tasks: []task.Task{{ID: "b-1", Title: "Cached B", Status: "todo"}}}, nil)

	configs := []task.ProjectConfig{
		{ProjectDir: projectA, Tasks: task.ProjectTaskConfig{Provider: "missing-provider.sh"}},
		{ProjectDir: projectB, Tasks: task.ProjectTaskConfig{Provider: "missing-provider.sh"}},
	}

	msg := taskRefreshCmd(configs, cache, globalConfig, time.Second)().(tasksMsg)

	if len(msg.errors) != 0 {
		t.Fatalf("expected no errors when all projects are cached, got %d", len(msg.errors))
	}
	if len(msg.resultsByProject) != 2 {
		t.Fatalf("expected cached results for both projects, got %d", len(msg.resultsByProject))
	}
	if len(msg.groupsByProject[projectA]) == 0 || len(msg.groupsByProject[projectB]) == 0 {
		t.Fatalf("expected normalized groups for cached results")
	}
}

func TestTaskRefreshCmd_MixedCachedAndUncachedIsolation(t *testing.T) {
	cache := task.NewResultCache()
	globalConfig := &task.GlobalConfig{}

	cachedProject := t.TempDir()
	cache.Set(cachedProject, &task.ProviderResult{Tasks: []task.Task{{ID: "cached-1", Title: "Cached", Status: "todo"}}}, nil)

	successProject := t.TempDir()
	writeExecutableScript(t, successProject, "provider.sh", `#!/bin/sh
printf '{"tasks":[{"id":"success-1","title":"Success","status":"todo"}]}'
`)

	errorProject := t.TempDir()
	writeExecutableScript(t, errorProject, "provider.sh", `#!/bin/sh
echo "provider failed" 1>&2
exit 1
`)

	configs := []task.ProjectConfig{
		{ProjectDir: cachedProject, Tasks: task.ProjectTaskConfig{Provider: "missing-provider.sh"}},
		{ProjectDir: successProject, Tasks: task.ProjectTaskConfig{Provider: "provider.sh"}},
		{ProjectDir: errorProject, Tasks: task.ProjectTaskConfig{Provider: "provider.sh"}},
	}

	msg := taskRefreshCmd(configs, cache, globalConfig, 3*time.Second)().(tasksMsg)

	if len(msg.resultsByProject) != 2 {
		t.Fatalf("expected 2 successful project results, got %d", len(msg.resultsByProject))
	}
	if _, ok := msg.resultsByProject[cachedProject]; !ok {
		t.Fatalf("expected cached project result to be present")
	}
	successResult, ok := msg.resultsByProject[successProject]
	if !ok {
		t.Fatalf("expected successful uncached project result to be present")
	}
	if len(successResult.Tasks) != 1 || successResult.Tasks[0].ID != "success-1" {
		t.Fatalf("unexpected success project tasks: %#v", successResult.Tasks)
	}

	if len(msg.errors) != 1 {
		t.Fatalf("expected exactly 1 project error, got %d", len(msg.errors))
	}
	if _, ok := msg.errors[errorProject]; !ok {
		t.Fatalf("expected failing project error to be isolated to its project")
	}
}

func TestTaskRefreshCmd_ConcurrentExecutionReducesLatency(t *testing.T) {
	cache := task.NewResultCache()
	globalConfig := &task.GlobalConfig{}

	projectCount := 6
	configs := make([]task.ProjectConfig, 0, projectCount)
	for i := 0; i < projectCount; i++ {
		projectDir := t.TempDir()
		script := `#!/bin/sh
sleep 0.5
printf '{"tasks":[{"id":"task-` + fmt.Sprintf("%d", i) + `","title":"Task","status":"todo"}]}'
`
		writeExecutableScript(t, projectDir, "provider.sh", script)
		configs = append(configs, task.ProjectConfig{ProjectDir: projectDir, Tasks: task.ProjectTaskConfig{Provider: "provider.sh"}})
	}

	sequentialStart := time.Now()
	for _, cfg := range configs {
		if _, err := task.ExecuteProvider(cfg, 3*time.Second); err != nil {
			t.Fatalf("sequential provider execution failed for %s: %v", cfg.ProjectDir, err)
		}
	}
	sequentialElapsed := time.Since(sequentialStart)

	start := time.Now()
	msg := taskRefreshCmd(configs, cache, globalConfig, 3*time.Second)().(tasksMsg)
	elapsed := time.Since(start)

	if len(msg.errors) != 0 {
		t.Fatalf("expected no provider errors, got %d", len(msg.errors))
	}
	if len(msg.resultsByProject) != projectCount {
		t.Fatalf("expected results for all projects, got %d", len(msg.resultsByProject))
	}

	if elapsed >= sequentialElapsed {
		t.Fatalf("expected concurrent execution (%s) to be faster than sequential (%s)", elapsed, sequentialElapsed)
	}
}

func TestTaskRefreshCmd_DuplicateProjectRunsOnce(t *testing.T) {
	cache := task.NewResultCache()
	projectDir := t.TempDir()
	writeExecutableScript(t, projectDir, "provider.sh", `#!/bin/sh
echo "run" >> invocations.log
printf '{"tasks":[{"id":"dup-1","title":"Duplicate","status":"todo"}]}'
`)

	config := task.ProjectConfig{ProjectDir: projectDir, Tasks: task.ProjectTaskConfig{Provider: "provider.sh"}}
	configs := []task.ProjectConfig{config, config}

	msg := taskRefreshCmd(configs, cache, &task.GlobalConfig{}, 2*time.Second)().(tasksMsg)
	if len(msg.errors) != 0 {
		t.Fatalf("expected duplicate project refresh to succeed, got %d errors", len(msg.errors))
	}

	data, err := os.ReadFile(filepath.Join(projectDir, "invocations.log"))
	if err != nil {
		t.Fatalf("failed reading invocation log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected provider to execute once for duplicate project configs, got %d", len(lines))
	}
}

func writeExecutableScript(t *testing.T, dir string, name string, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("failed to write script %s: %v", path, err)
	}
	return path
}
