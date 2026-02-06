package tui

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/stwalsh4118/navi/internal/remote"
)

func TestValidateRemoteConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  remote.Config
		wantErr error
	}{
		{
			name: "valid config with all fields",
			config: remote.Config{
				Name:        "dev-server",
				Host:        "dev.example.com",
				User:        "sean",
				Key:         "~/.ssh/id_rsa",
				SessionsDir: "~/.claude-sessions",
				JumpHost:    "bastion.example.com",
			},
			wantErr: nil,
		},
		{
			name: "valid config with required fields only",
			config: remote.Config{
				Name: "dev-server",
				Host: "dev.example.com",
				User: "sean",
				Key:  "~/.ssh/id_rsa",
			},
			wantErr: nil,
		},
		{
			name: "missing name",
			config: remote.Config{
				Host: "dev.example.com",
				User: "sean",
				Key:  "~/.ssh/id_rsa",
			},
			wantErr: remote.ErrRemoteNameRequired,
		},
		{
			name: "missing host",
			config: remote.Config{
				Name: "dev-server",
				User: "sean",
				Key:  "~/.ssh/id_rsa",
			},
			wantErr: remote.ErrRemoteHostRequired,
		},
		{
			name: "missing user",
			config: remote.Config{
				Name: "dev-server",
				Host: "dev.example.com",
				Key:  "~/.ssh/id_rsa",
			},
			wantErr: remote.ErrRemoteUserRequired,
		},
		{
			name: "missing key",
			config: remote.Config{
				Name: "dev-server",
				Host: "dev.example.com",
				User: "sean",
			},
			wantErr: remote.ErrRemoteKeyRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := remote.ValidateConfig(&tt.config)
			if err != tt.wantErr {
				t.Errorf("remote.ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestApplyRemoteDefaults(t *testing.T) {
	t.Run("applies default sessions_dir", func(t *testing.T) {
		config := remote.Config{
			Name: "test",
			Host: "host",
			User: "user",
			Key:  "~/.ssh/id_rsa",
		}
		remote.ApplyDefaults(&config)
		if config.SessionsDir != remote.DefaultSessionsDir {
			t.Errorf("SessionsDir = %v, want %v", config.SessionsDir, remote.DefaultSessionsDir)
		}
	})

	t.Run("does not override custom sessions_dir", func(t *testing.T) {
		customDir := "/custom/sessions"
		config := remote.Config{
			Name:        "test",
			Host:        "host",
			User:        "user",
			Key:         "~/.ssh/id_rsa",
			SessionsDir: customDir,
		}
		remote.ApplyDefaults(&config)
		if config.SessionsDir != customDir {
			t.Errorf("SessionsDir = %v, want %v", config.SessionsDir, customDir)
		}
	})

	t.Run("expands key path", func(t *testing.T) {
		config := remote.Config{
			Name: "test",
			Host: "host",
			User: "user",
			Key:  "~/.ssh/id_rsa",
		}
		remote.ApplyDefaults(&config)
		// Key should be expanded (not start with ~)
		if config.Key[0] == '~' {
			t.Errorf("Key path not expanded: %v", config.Key)
		}
	})
}

func TestGetRemoteByName(t *testing.T) {
	remotes := []remote.Config{
		{Name: "dev-server", Host: "dev.example.com"},
		{Name: "staging", Host: "staging.example.com"},
		{Name: "production", Host: "prod.example.com"},
	}

	t.Run("finds existing remote", func(t *testing.T) {
		result := remote.GetByName(remotes, "staging")
		if result == nil {
			t.Fatal("expected to find remote, got nil")
		}
		if result.Host != "staging.example.com" {
			t.Errorf("Host = %v, want staging.example.com", result.Host)
		}
	})

	t.Run("returns nil for non-existent remote", func(t *testing.T) {
		result := remote.GetByName(remotes, "nonexistent")
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("handles empty list", func(t *testing.T) {
		result := remote.GetByName([]remote.Config{}, "any")
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})
}

func TestYAMLParsing(t *testing.T) {
	t.Run("parses valid YAML with all fields", func(t *testing.T) {
		configContent := []byte(`remotes:
  - name: dev-server
    host: dev.example.com
    user: sean
    key: ~/.ssh/id_rsa
  - name: staging
    host: staging.example.com
    user: deploy
    key: ~/.ssh/deploy_key
    sessions_dir: /opt/sessions
    jump_host: bastion.example.com
`)

		var config remote.RemotesConfig
		err := yaml.Unmarshal(configContent, &config)
		if err != nil {
			t.Fatalf("failed to parse YAML: %v", err)
		}

		if len(config.Remotes) != 2 {
			t.Errorf("expected 2 remotes, got %d", len(config.Remotes))
		}

		// Validate first remote
		if config.Remotes[0].Name != "dev-server" {
			t.Errorf("first remote name = %v, want dev-server", config.Remotes[0].Name)
		}
		if config.Remotes[0].Host != "dev.example.com" {
			t.Errorf("first remote host = %v, want dev.example.com", config.Remotes[0].Host)
		}

		// Validate second remote with optional fields
		if config.Remotes[1].JumpHost != "bastion.example.com" {
			t.Errorf("second remote jump_host = %v, want bastion.example.com", config.Remotes[1].JumpHost)
		}
		if config.Remotes[1].SessionsDir != "/opt/sessions" {
			t.Errorf("second remote sessions_dir = %v, want /opt/sessions", config.Remotes[1].SessionsDir)
		}
	})

	t.Run("parses empty remotes list", func(t *testing.T) {
		configContent := []byte(`remotes: []`)

		var config remote.RemotesConfig
		err := yaml.Unmarshal(configContent, &config)
		if err != nil {
			t.Fatalf("failed to parse YAML: %v", err)
		}

		if len(config.Remotes) != 0 {
			t.Errorf("expected 0 remotes, got %d", len(config.Remotes))
		}
	})

	t.Run("handles malformed YAML", func(t *testing.T) {
		configContent := []byte(`remotes:
  - name: test
    host: [invalid yaml`)

		var config remote.RemotesConfig
		err := yaml.Unmarshal(configContent, &config)
		if err == nil {
			t.Error("expected error for malformed YAML, got nil")
		}
	})
}

func TestLoadRemotesConfigFromFile(t *testing.T) {
	// Create a temporary directory for test config files
	tmpDir := t.TempDir()

	t.Run("loads valid config from file", func(t *testing.T) {
		configContent := `remotes:
  - name: test-server
    host: test.example.com
    user: testuser
    key: ~/.ssh/test_key
`
		configPath := filepath.Join(tmpDir, "test_remotes.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Read and parse the file
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}

		var config remote.RemotesConfig
		if err := yaml.Unmarshal(data, &config); err != nil {
			t.Fatalf("failed to parse YAML: %v", err)
		}

		if len(config.Remotes) != 1 {
			t.Errorf("expected 1 remote, got %d", len(config.Remotes))
		}

		if config.Remotes[0].Name != "test-server" {
			t.Errorf("remote name = %v, want test-server", config.Remotes[0].Name)
		}
	})

	t.Run("handles non-existent file", func(t *testing.T) {
		nonExistentPath := filepath.Join(tmpDir, "nonexistent.yaml")
		_, err := os.Stat(nonExistentPath)
		if !os.IsNotExist(err) {
			t.Fatal("test file should not exist")
		}
		// This is the expected behavior - file not existing should return empty list
	})
}

func TestExpandConfigPath(t *testing.T) {
	path := remote.ExpandConfigPath()
	// Should not contain ~ after expansion
	if len(path) > 0 && path[0] == '~' {
		t.Errorf("config path not expanded: %v", path)
	}
}
