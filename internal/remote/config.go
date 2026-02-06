package remote

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/stwalsh4118/navi/internal/pathutil"
)

// Configuration constants
const (
	ConfigPath        = "~/.config/navi/remotes.yaml"
	DefaultSessionsDir = "~/.claude-sessions"
)

// Validation error messages
var (
	ErrRemoteNameRequired = errors.New("remote name is required")
	ErrRemoteHostRequired = errors.New("remote host is required")
	ErrRemoteUserRequired = errors.New("remote user is required")
	ErrRemoteKeyRequired  = errors.New("remote SSH key path is required")
)

// Config represents a single remote machine configuration.
type Config struct {
	Name        string `yaml:"name"`
	Host        string `yaml:"host"`
	User        string `yaml:"user"`
	Key         string `yaml:"key"`
	SessionsDir string `yaml:"sessions_dir,omitempty"`
	JumpHost    string `yaml:"jump_host,omitempty"`
}

// RemotesConfig is the root structure for the remotes YAML configuration file.
type RemotesConfig struct {
	Remotes []Config `yaml:"remotes"`
}

// LoadConfig loads the remote machine configuration from the YAML file.
func LoadConfig() ([]Config, error) {
	configPath := pathutil.ExpandPath(ConfigPath)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return []Config{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config RemotesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	for i := range config.Remotes {
		if err := ValidateConfig(&config.Remotes[i]); err != nil {
			return nil, err
		}
		ApplyDefaults(&config.Remotes[i])
	}

	return config.Remotes, nil
}

// ValidateConfig validates that all required fields are present in a Config.
func ValidateConfig(rc *Config) error {
	if rc.Name == "" {
		return ErrRemoteNameRequired
	}
	if rc.Host == "" {
		return ErrRemoteHostRequired
	}
	if rc.User == "" {
		return ErrRemoteUserRequired
	}
	if rc.Key == "" {
		return ErrRemoteKeyRequired
	}
	return nil
}

// ApplyDefaults applies default values to optional fields in a Config.
func ApplyDefaults(rc *Config) {
	if rc.SessionsDir == "" {
		rc.SessionsDir = DefaultSessionsDir
	}
	rc.Key = pathutil.ExpandPath(rc.Key)
}

// GetByName finds a remote configuration by its name.
func GetByName(remotes []Config, name string) *Config {
	for i := range remotes {
		if remotes[i].Name == name {
			return &remotes[i]
		}
	}
	return nil
}

// ExpandConfigPath expands the remotes configuration file path.
func ExpandConfigPath() string {
	return pathutil.ExpandPath(ConfigPath)
}

// ParseConfigData parses YAML data into a RemotesConfig struct.
func ParseConfigData(data []byte) (RemotesConfig, error) {
	var config RemotesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return config, err
	}
	return config, nil
}
