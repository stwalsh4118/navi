package task

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/stwalsh4118/navi/internal/pathutil"
)

// FindProjectConfig walks up from dir to find .navi.yaml.
// Returns nil, nil if no config file is found (not an error).
func FindProjectConfig(dir string) (*ProjectConfig, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	for {
		configPath := filepath.Join(dir, ProjectConfigFile)

		if _, err := os.Stat(configPath); err == nil {
			data, err := os.ReadFile(configPath)
			if err != nil {
				return nil, err
			}

			var cfg ProjectConfig
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return nil, err
			}
			cfg.ProjectDir = dir
			return &cfg, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding config.
			return nil, nil
		}
		dir = parent
	}
}

// LoadGlobalConfig loads from ~/.navi/config.yaml.
// Returns a default GlobalConfig if the file doesn't exist (not an error).
func LoadGlobalConfig() (*GlobalConfig, error) {
	configPath := pathutil.ExpandPath(GlobalConfigPath)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &GlobalConfig{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg GlobalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// MergeConfig applies global defaults to a project config where project fields are empty.
func MergeConfig(project *ProjectConfig, global *GlobalConfig) *ProjectConfig {
	if project.Tasks.Provider == "" && global.Tasks.DefaultProvider != "" {
		project.Tasks.Provider = global.Tasks.DefaultProvider
	}
	if project.Tasks.Interval.Duration == 0 && global.Tasks.Interval.Duration != 0 {
		project.Tasks.Interval = global.Tasks.Interval
	}
	return project
}

// NormalizeStatus maps a provider status string to a display status using the status map.
// Returns the original status if no mapping exists.
func NormalizeStatus(status string, statusMap map[string]string) string {
	if mapped, ok := statusMap[status]; ok {
		return mapped
	}
	return status
}
