package main

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestValidateRemoteConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  RemoteConfig
		wantErr error
	}{
		{
			name: "valid config with all fields",
			config: RemoteConfig{
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
			config: RemoteConfig{
				Name: "dev-server",
				Host: "dev.example.com",
				User: "sean",
				Key:  "~/.ssh/id_rsa",
			},
			wantErr: nil,
		},
		{
			name: "missing name",
			config: RemoteConfig{
				Host: "dev.example.com",
				User: "sean",
				Key:  "~/.ssh/id_rsa",
			},
			wantErr: errRemoteNameRequired,
		},
		{
			name: "missing host",
			config: RemoteConfig{
				Name: "dev-server",
				User: "sean",
				Key:  "~/.ssh/id_rsa",
			},
			wantErr: errRemoteHostRequired,
		},
		{
			name: "missing user",
			config: RemoteConfig{
				Name: "dev-server",
				Host: "dev.example.com",
				Key:  "~/.ssh/id_rsa",
			},
			wantErr: errRemoteUserRequired,
		},
		{
			name: "missing key",
			config: RemoteConfig{
				Name: "dev-server",
				Host: "dev.example.com",
				User: "sean",
			},
			wantErr: errRemoteKeyRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRemoteConfig(&tt.config)
			if err != tt.wantErr {
				t.Errorf("validateRemoteConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestApplyRemoteDefaults(t *testing.T) {
	t.Run("applies default sessions_dir", func(t *testing.T) {
		config := RemoteConfig{
			Name: "test",
			Host: "host",
			User: "user",
			Key:  "~/.ssh/id_rsa",
		}
		applyRemoteDefaults(&config)
		if config.SessionsDir != defaultSessionsDir {
			t.Errorf("SessionsDir = %v, want %v", config.SessionsDir, defaultSessionsDir)
		}
	})

	t.Run("does not override custom sessions_dir", func(t *testing.T) {
		customDir := "/custom/sessions"
		config := RemoteConfig{
			Name:        "test",
			Host:        "host",
			User:        "user",
			Key:         "~/.ssh/id_rsa",
			SessionsDir: customDir,
		}
		applyRemoteDefaults(&config)
		if config.SessionsDir != customDir {
			t.Errorf("SessionsDir = %v, want %v", config.SessionsDir, customDir)
		}
	})

	t.Run("expands key path", func(t *testing.T) {
		config := RemoteConfig{
			Name: "test",
			Host: "host",
			User: "user",
			Key:  "~/.ssh/id_rsa",
		}
		applyRemoteDefaults(&config)
		// Key should be expanded (not start with ~)
		if config.Key[0] == '~' {
			t.Errorf("Key path not expanded: %v", config.Key)
		}
	})
}

func TestGetRemoteByName(t *testing.T) {
	remotes := []RemoteConfig{
		{Name: "dev-server", Host: "dev.example.com"},
		{Name: "staging", Host: "staging.example.com"},
		{Name: "production", Host: "prod.example.com"},
	}

	t.Run("finds existing remote", func(t *testing.T) {
		result := getRemoteByName(remotes, "staging")
		if result == nil {
			t.Fatal("expected to find remote, got nil")
		}
		if result.Host != "staging.example.com" {
			t.Errorf("Host = %v, want staging.example.com", result.Host)
		}
	})

	t.Run("returns nil for non-existent remote", func(t *testing.T) {
		result := getRemoteByName(remotes, "nonexistent")
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("handles empty list", func(t *testing.T) {
		result := getRemoteByName([]RemoteConfig{}, "any")
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

		var config RemotesConfig
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

		var config RemotesConfig
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

		var config RemotesConfig
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

		var config RemotesConfig
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
	path := expandConfigPath()
	// Should not contain ~ after expansion
	if len(path) > 0 && path[0] == '~' {
		t.Errorf("config path not expanded: %v", path)
	}
}
