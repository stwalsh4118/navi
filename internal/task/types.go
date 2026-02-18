package task

import (
	"encoding/json"
	"time"
)

// Task represents an individual task item from a provider.
type Task struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Status   string    `json:"status"`
	Assignee string    `json:"assignee,omitempty"`
	Labels   []string  `json:"labels,omitempty"`
	Priority int       `json:"priority,omitempty"`
	URL      string    `json:"url,omitempty"`
	Created  time.Time `json:"created,omitempty"`
	Updated  time.Time `json:"updated,omitempty"`
}

// TaskGroup represents a group of related tasks (e.g., PBI, epic, milestone).
type TaskGroup struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status,omitempty"`
	URL       string `json:"url,omitempty"`
	IsCurrent bool   `json:"is_current,omitempty"`
	Tasks     []Task `json:"tasks"`
}

// ProviderResult is the top-level JSON output from a provider script.
// It supports both grouped format (groups array) and flat format (tasks array).
type ProviderResult struct {
	CurrentPBIID    string      `json:"current_pbi_id,omitempty"`
	CurrentPBITitle string      `json:"current_pbi_title,omitempty"`
	Groups          []TaskGroup `json:"groups,omitempty"`
	Tasks           []Task      `json:"tasks,omitempty"`
}

// ParseProviderOutput parses raw JSON bytes from a provider script into a ProviderResult.
// It handles both grouped format ({"groups": [...]}) and flat format ({"tasks": [...]}).
func ParseProviderOutput(data []byte) (*ProviderResult, error) {
	var result ProviderResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// AllTasks returns all tasks from the result, flattening groups if present.
func (r *ProviderResult) AllTasks() []Task {
	if len(r.Groups) > 0 {
		var all []Task
		for _, g := range r.Groups {
			all = append(all, g.Tasks...)
		}
		return all
	}
	return r.Tasks
}

// ProjectConfig represents per-project configuration from .navi.yaml.
type ProjectConfig struct {
	Tasks ProjectTaskConfig `yaml:"tasks"`
	// ProjectDir is the directory where .navi.yaml was found (set during discovery, not from YAML).
	ProjectDir string `yaml:"-"`
}

// ProjectTaskConfig holds task-specific settings from .navi.yaml.
type ProjectTaskConfig struct {
	Provider string            `yaml:"provider"`
	Args     map[string]string `yaml:"args,omitempty"`
	Interval Duration          `yaml:"interval,omitempty"`
}

// GlobalConfig represents the global configuration from ~/.navi/config.yaml.
type GlobalConfig struct {
	Tasks GlobalTaskConfig `yaml:"tasks"`
}

// GlobalTaskConfig holds global task settings from ~/.navi/config.yaml.
type GlobalTaskConfig struct {
	DefaultProvider string            `yaml:"default_provider,omitempty"`
	Interval        Duration          `yaml:"interval,omitempty"`
	StatusMap       map[string]string `yaml:"status_map,omitempty"`
}

// Duration is a wrapper around time.Duration that supports YAML/JSON string parsing (e.g., "60s", "5m").
type Duration struct {
	time.Duration
}

// UnmarshalYAML implements yaml.Unmarshaler for Duration.
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	if s == "" {
		d.Duration = 0
		return nil
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = dur
	return nil
}

// Default configuration values.
const (
	DefaultRefreshInterval = 60 * time.Second
	DefaultProviderTimeout = 30 * time.Second
)

// ProjectConfigFile is the per-project config filename.
const ProjectConfigFile = ".navi.yaml"

// GlobalConfigPath is the path to the global config file.
const GlobalConfigPath = "~/.navi/config.yaml"
