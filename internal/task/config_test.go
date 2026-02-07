package task

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindProjectConfig_CurrentDir(t *testing.T) {
	dir := t.TempDir()
	configContent := `tasks:
  provider: "github-issues"
  args:
    repo: "owner/repo"
  interval: "30s"
`
	if err := os.WriteFile(filepath.Join(dir, ProjectConfigFile), []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := FindProjectConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.Tasks.Provider != "github-issues" {
		t.Errorf("expected provider %q, got %q", "github-issues", cfg.Tasks.Provider)
	}
	if cfg.Tasks.Args["repo"] != "owner/repo" {
		t.Errorf("expected args[repo] %q, got %q", "owner/repo", cfg.Tasks.Args["repo"])
	}
	if cfg.Tasks.Interval.Duration != 30*time.Second {
		t.Errorf("expected interval 30s, got %v", cfg.Tasks.Interval.Duration)
	}
	if cfg.ProjectDir != dir {
		t.Errorf("expected ProjectDir %q, got %q", dir, cfg.ProjectDir)
	}
}

func TestFindProjectConfig_ParentDir(t *testing.T) {
	root := t.TempDir()
	configContent := `tasks:
  provider: "jira"
`
	if err := os.WriteFile(filepath.Join(root, ProjectConfigFile), []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	child := filepath.Join(root, "sub", "deep")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg, err := FindProjectConfig(child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.Tasks.Provider != "jira" {
		t.Errorf("expected provider %q, got %q", "jira", cfg.Tasks.Provider)
	}
	if cfg.ProjectDir != root {
		t.Errorf("expected ProjectDir %q, got %q", root, cfg.ProjectDir)
	}
}

func TestFindProjectConfig_NotFound(t *testing.T) {
	dir := t.TempDir()

	cfg, err := FindProjectConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Errorf("expected nil config, got %+v", cfg)
	}
}

func TestLoadGlobalConfig_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, ".navi")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}

	configContent := `tasks:
  default_provider: "github-issues"
  interval: "120s"
  status_map:
    open: "todo"
    closed: "done"
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Override HOME so LoadGlobalConfig finds our test file.
	originalHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer func() { os.Setenv("HOME", originalHome) }()

	cfg, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Tasks.DefaultProvider != "github-issues" {
		t.Errorf("expected default_provider %q, got %q", "github-issues", cfg.Tasks.DefaultProvider)
	}
	if cfg.Tasks.Interval.Duration != 120*time.Second {
		t.Errorf("expected interval 120s, got %v", cfg.Tasks.Interval.Duration)
	}
	if cfg.Tasks.StatusMap["open"] != "todo" {
		t.Errorf("expected status_map[open] %q, got %q", "todo", cfg.Tasks.StatusMap["open"])
	}
	if cfg.Tasks.StatusMap["closed"] != "done" {
		t.Errorf("expected status_map[closed] %q, got %q", "done", cfg.Tasks.StatusMap["closed"])
	}
}

func TestLoadGlobalConfig_MissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected default config, got nil")
	}
	if cfg.Tasks.DefaultProvider != "" {
		t.Errorf("expected empty default_provider, got %q", cfg.Tasks.DefaultProvider)
	}
}

func TestMergeConfig_AppliesDefaultProvider(t *testing.T) {
	project := &ProjectConfig{}
	global := &GlobalConfig{
		Tasks: GlobalTaskConfig{
			DefaultProvider: "github-issues",
		},
	}

	result := MergeConfig(project, global)
	if result.Tasks.Provider != "github-issues" {
		t.Errorf("expected provider %q, got %q", "github-issues", result.Tasks.Provider)
	}
}

func TestMergeConfig_DoesNotOverrideExistingProvider(t *testing.T) {
	project := &ProjectConfig{
		Tasks: ProjectTaskConfig{
			Provider: "jira",
		},
	}
	global := &GlobalConfig{
		Tasks: GlobalTaskConfig{
			DefaultProvider: "github-issues",
		},
	}

	result := MergeConfig(project, global)
	if result.Tasks.Provider != "jira" {
		t.Errorf("expected provider %q, got %q", "jira", result.Tasks.Provider)
	}
}

func TestMergeConfig_AppliesInterval(t *testing.T) {
	project := &ProjectConfig{}
	global := &GlobalConfig{
		Tasks: GlobalTaskConfig{
			Interval: Duration{Duration: 90 * time.Second},
		},
	}

	result := MergeConfig(project, global)
	if result.Tasks.Interval.Duration != 90*time.Second {
		t.Errorf("expected interval 90s, got %v", result.Tasks.Interval.Duration)
	}
}

func TestMergeConfig_DoesNotOverrideExistingInterval(t *testing.T) {
	project := &ProjectConfig{
		Tasks: ProjectTaskConfig{
			Interval: Duration{Duration: 30 * time.Second},
		},
	}
	global := &GlobalConfig{
		Tasks: GlobalTaskConfig{
			Interval: Duration{Duration: 90 * time.Second},
		},
	}

	result := MergeConfig(project, global)
	if result.Tasks.Interval.Duration != 30*time.Second {
		t.Errorf("expected interval 30s, got %v", result.Tasks.Interval.Duration)
	}
}

func TestNormalizeStatus_MapsKnown(t *testing.T) {
	statusMap := map[string]string{
		"open":        "todo",
		"in_progress": "active",
		"closed":      "done",
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"open", "todo"},
		{"in_progress", "active"},
		{"closed", "done"},
	}

	for _, tt := range tests {
		result := NormalizeStatus(tt.input, statusMap)
		if result != tt.expected {
			t.Errorf("NormalizeStatus(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNormalizeStatus_PassesThrough(t *testing.T) {
	statusMap := map[string]string{
		"open": "todo",
	}

	result := NormalizeStatus("unknown", statusMap)
	if result != "unknown" {
		t.Errorf("expected %q, got %q", "unknown", result)
	}
}

func TestNormalizeStatus_NilMap(t *testing.T) {
	result := NormalizeStatus("open", nil)
	if result != "open" {
		t.Errorf("expected %q, got %q", "open", result)
	}
}

func TestDiscoverProjects_DeduplicatesCWDs(t *testing.T) {
	root := t.TempDir()
	configContent := `tasks:
  provider: "github-issues"
`
	if err := os.WriteFile(filepath.Join(root, ProjectConfigFile), []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	subA := filepath.Join(root, "src")
	subB := filepath.Join(root, "tests")
	if err := os.MkdirAll(subA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(subB, 0o755); err != nil {
		t.Fatal(err)
	}

	global := &GlobalConfig{}
	configs := DiscoverProjects([]string{subA, subB, root}, global)

	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}
	if configs[0].Tasks.Provider != "github-issues" {
		t.Errorf("expected provider %q, got %q", "github-issues", configs[0].Tasks.Provider)
	}
	if configs[0].ProjectDir != root {
		t.Errorf("expected ProjectDir %q, got %q", root, configs[0].ProjectDir)
	}
}

func TestDiscoverProjects_SkipsDirsWithoutConfig(t *testing.T) {
	dirWithConfig := t.TempDir()
	dirWithoutConfig := t.TempDir()

	configContent := `tasks:
  provider: "jira"
`
	if err := os.WriteFile(filepath.Join(dirWithConfig, ProjectConfigFile), []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	global := &GlobalConfig{}
	configs := DiscoverProjects([]string{dirWithoutConfig, dirWithConfig}, global)

	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}
	if configs[0].Tasks.Provider != "jira" {
		t.Errorf("expected provider %q, got %q", "jira", configs[0].Tasks.Provider)
	}
}

func TestDiscoverProjects_MergesGlobalDefaults(t *testing.T) {
	root := t.TempDir()
	configContent := `tasks:
  args:
    repo: "owner/repo"
`
	if err := os.WriteFile(filepath.Join(root, ProjectConfigFile), []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	global := &GlobalConfig{
		Tasks: GlobalTaskConfig{
			DefaultProvider: "github-issues",
			Interval:        Duration{Duration: 120 * time.Second},
		},
	}

	configs := DiscoverProjects([]string{root}, global)

	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}
	if configs[0].Tasks.Provider != "github-issues" {
		t.Errorf("expected provider %q from global default, got %q", "github-issues", configs[0].Tasks.Provider)
	}
	if configs[0].Tasks.Interval.Duration != 120*time.Second {
		t.Errorf("expected interval 120s from global default, got %v", configs[0].Tasks.Interval.Duration)
	}
}

func TestDiscoverProjects_EmptyCWDs(t *testing.T) {
	global := &GlobalConfig{}
	configs := DiscoverProjects([]string{}, global)

	if len(configs) != 0 {
		t.Errorf("expected 0 configs, got %d", len(configs))
	}
}
