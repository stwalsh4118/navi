package main

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

// Remote configuration constants
const (
	// remotesConfigPath is the default location for the remotes configuration file
	remotesConfigPath = "~/.config/navi/remotes.yaml"

	// defaultSessionsDir is the default directory for session status files on remote machines
	defaultSessionsDir = "~/.claude-sessions"
)

// Validation error messages for remote configuration
var (
	errRemoteNameRequired = errors.New("remote name is required")
	errRemoteHostRequired = errors.New("remote host is required")
	errRemoteUserRequired = errors.New("remote user is required")
	errRemoteKeyRequired  = errors.New("remote SSH key path is required")
)

// RemoteConfig represents a single remote machine configuration.
// This struct matches the YAML schema defined in the PRD.
type RemoteConfig struct {
	Name        string `yaml:"name"`                   // Unique identifier for this remote
	Host        string `yaml:"host"`                   // SSH hostname or IP
	User        string `yaml:"user"`                   // SSH username
	Key         string `yaml:"key"`                    // Path to SSH private key
	SessionsDir string `yaml:"sessions_dir,omitempty"` // Path to session status directory (default: ~/.claude-sessions)
	JumpHost    string `yaml:"jump_host,omitempty"`    // Optional bastion/jump host
}

// RemotesConfig is the root structure for the remotes YAML configuration file.
type RemotesConfig struct {
	Remotes []RemoteConfig `yaml:"remotes"`
}

// loadRemotesConfig loads the remote machine configuration from the YAML file.
// Returns an empty slice if the configuration file does not exist.
// Returns an error if the file exists but cannot be parsed or has invalid content.
func loadRemotesConfig() ([]RemoteConfig, error) {
	configPath := expandPath(remotesConfigPath)

	// Check if file exists - missing file is not an error
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return []RemoteConfig{}, nil
	}

	// Read the configuration file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Parse YAML
	var config RemotesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Validate and apply defaults to each remote
	for i := range config.Remotes {
		if err := validateRemoteConfig(&config.Remotes[i]); err != nil {
			return nil, err
		}
		applyRemoteDefaults(&config.Remotes[i])
	}

	return config.Remotes, nil
}

// validateRemoteConfig validates that all required fields are present in a RemoteConfig.
func validateRemoteConfig(rc *RemoteConfig) error {
	if rc.Name == "" {
		return errRemoteNameRequired
	}
	if rc.Host == "" {
		return errRemoteHostRequired
	}
	if rc.User == "" {
		return errRemoteUserRequired
	}
	if rc.Key == "" {
		return errRemoteKeyRequired
	}
	return nil
}

// applyRemoteDefaults applies default values to optional fields in a RemoteConfig.
func applyRemoteDefaults(rc *RemoteConfig) {
	if rc.SessionsDir == "" {
		rc.SessionsDir = defaultSessionsDir
	}
	// Expand ~ in the SSH key path
	rc.Key = expandPath(rc.Key)
}

// getRemoteByName finds a remote configuration by its name.
// Returns nil if no remote with the given name exists.
func getRemoteByName(remotes []RemoteConfig, name string) *RemoteConfig {
	for i := range remotes {
		if remotes[i].Name == name {
			return &remotes[i]
		}
	}
	return nil
}

// expandConfigPath expands the remotes configuration file path.
// Exported for use in initialization.
func expandConfigPath() string {
	return expandPath(remotesConfigPath)
}

// parseRemotesConfigData parses YAML data into a RemotesConfig struct.
// This is useful for testing configuration parsing without file I/O.
func parseRemotesConfigData(data []byte) (RemotesConfig, error) {
	var config RemotesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return config, err
	}
	return config, nil
}
