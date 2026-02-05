package main

import (
	"testing"
	"time"
)

func TestConnectionStatusString(t *testing.T) {
	tests := []struct {
		status ConnectionStatus
		want   string
	}{
		{StatusConnected, "connected"},
		{StatusDisconnected, "disconnected"},
		{StatusError, "error"},
		{ConnectionStatus(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.status.String()
		if got != tt.want {
			t.Errorf("ConnectionStatus(%d).String() = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestNewSSHPool(t *testing.T) {
	remotes := []RemoteConfig{
		{Name: "dev", Host: "dev.example.com", User: "user", Key: "~/.ssh/id_rsa"},
		{Name: "staging", Host: "staging.example.com", User: "deploy", Key: "~/.ssh/deploy_key"},
	}

	pool := NewSSHPool(remotes)

	t.Run("initializes remotes map", func(t *testing.T) {
		if len(pool.remotes) != 2 {
			t.Errorf("remotes count = %d, want 2", len(pool.remotes))
		}
	})

	t.Run("initializes status map", func(t *testing.T) {
		if len(pool.status) != 2 {
			t.Errorf("status count = %d, want 2", len(pool.status))
		}
	})

	t.Run("all remotes start disconnected", func(t *testing.T) {
		for name := range pool.remotes {
			status := pool.GetStatus(name)
			if status.Status != StatusDisconnected {
				t.Errorf("remote %s status = %v, want disconnected", name, status.Status)
			}
		}
	})
}

func TestSSHPoolGetStatus(t *testing.T) {
	remotes := []RemoteConfig{
		{Name: "test", Host: "test.example.com", User: "user", Key: "~/.ssh/id_rsa"},
	}

	pool := NewSSHPool(remotes)

	t.Run("returns status for known remote", func(t *testing.T) {
		status := pool.GetStatus("test")
		if status.Status != StatusDisconnected {
			t.Errorf("status = %v, want disconnected", status.Status)
		}
	})

	t.Run("returns disconnected for unknown remote", func(t *testing.T) {
		status := pool.GetStatus("unknown")
		if status.Status != StatusDisconnected {
			t.Errorf("status = %v, want disconnected", status.Status)
		}
	})
}

func TestSSHPoolGetAllStatus(t *testing.T) {
	remotes := []RemoteConfig{
		{Name: "dev", Host: "dev.example.com", User: "user", Key: "~/.ssh/id_rsa"},
		{Name: "staging", Host: "staging.example.com", User: "deploy", Key: "~/.ssh/deploy_key"},
	}

	pool := NewSSHPool(remotes)

	statuses := pool.GetAllStatus()

	if len(statuses) != 2 {
		t.Errorf("statuses count = %d, want 2", len(statuses))
	}

	for name, status := range statuses {
		if status.Status != StatusDisconnected {
			t.Errorf("remote %s status = %v, want disconnected", name, status.Status)
		}
	}
}

func TestSSHPoolRemoteNames(t *testing.T) {
	remotes := []RemoteConfig{
		{Name: "alpha", Host: "alpha.example.com", User: "user", Key: "~/.ssh/id_rsa"},
		{Name: "beta", Host: "beta.example.com", User: "user", Key: "~/.ssh/id_rsa"},
	}

	pool := NewSSHPool(remotes)

	names := pool.RemoteNames()

	if len(names) != 2 {
		t.Errorf("names count = %d, want 2", len(names))
	}

	// Check both names are present (order may vary)
	found := make(map[string]bool)
	for _, name := range names {
		found[name] = true
	}

	if !found["alpha"] || !found["beta"] {
		t.Errorf("missing expected remote names in %v", names)
	}
}

func TestSSHPoolGetRemoteConfig(t *testing.T) {
	remotes := []RemoteConfig{
		{Name: "test", Host: "test.example.com", User: "testuser", Key: "~/.ssh/id_rsa"},
	}

	pool := NewSSHPool(remotes)

	t.Run("returns config for known remote", func(t *testing.T) {
		config := pool.GetRemoteConfig("test")
		if config == nil {
			t.Fatal("expected config, got nil")
		}
		if config.Host != "test.example.com" {
			t.Errorf("host = %v, want test.example.com", config.Host)
		}
		if config.User != "testuser" {
			t.Errorf("user = %v, want testuser", config.User)
		}
	})

	t.Run("returns nil for unknown remote", func(t *testing.T) {
		config := pool.GetRemoteConfig("unknown")
		if config != nil {
			t.Errorf("expected nil, got %v", config)
		}
	})
}

func TestSSHPoolClose(t *testing.T) {
	remotes := []RemoteConfig{
		{Name: "test", Host: "test.example.com", User: "user", Key: "~/.ssh/id_rsa"},
	}

	pool := NewSSHPool(remotes)

	// Close should not panic even with no connections
	pool.Close()

	// Status should be disconnected after close
	status := pool.GetStatus("test")
	if status.Status != StatusDisconnected {
		t.Errorf("status after close = %v, want disconnected", status.Status)
	}
}

func TestSSHPoolIsConnected(t *testing.T) {
	remotes := []RemoteConfig{
		{Name: "test", Host: "test.example.com", User: "user", Key: "~/.ssh/id_rsa"},
	}

	pool := NewSSHPool(remotes)

	t.Run("returns false when no connections", func(t *testing.T) {
		if pool.IsConnected() {
			t.Error("expected false when no connections")
		}
	})
}

func TestSSHPoolDisconnect(t *testing.T) {
	remotes := []RemoteConfig{
		{Name: "test", Host: "test.example.com", User: "user", Key: "~/.ssh/id_rsa"},
	}

	pool := NewSSHPool(remotes)

	// Disconnect should not panic even with no connection
	pool.Disconnect("test")

	// Status should be disconnected
	status := pool.GetStatus("test")
	if status.Status != StatusDisconnected {
		t.Errorf("status = %v, want disconnected", status.Status)
	}

	// Disconnect unknown remote should not panic
	pool.Disconnect("unknown")
}

func TestBuildSSHAttachCommand(t *testing.T) {
	t.Run("direct connection", func(t *testing.T) {
		remote := &RemoteConfig{
			Name: "test",
			Host: "test.example.com",
			User: "testuser",
			Key:  "/home/user/.ssh/id_rsa",
		}

		args := buildSSHAttachCommand(remote, "my-session")

		// Verify key components are present
		if args[0] != "ssh" {
			t.Errorf("first arg = %v, want ssh", args[0])
		}

		// Check key option
		foundKey := false
		for i, arg := range args {
			if arg == "-i" && i+1 < len(args) && args[i+1] == "/home/user/.ssh/id_rsa" {
				foundKey = true
				break
			}
		}
		if !foundKey {
			t.Error("key option not found in args")
		}

		// Check terminal allocation
		foundTerminal := false
		for _, arg := range args {
			if arg == "-t" {
				foundTerminal = true
				break
			}
		}
		if !foundTerminal {
			t.Error("-t option not found in args")
		}

		// Check host
		foundHost := false
		for _, arg := range args {
			if arg == "testuser@test.example.com" {
				foundHost = true
				break
			}
		}
		if !foundHost {
			t.Error("host not found in args")
		}

		// Check tmux command
		foundTmux := false
		for i, arg := range args {
			if arg == "tmux" && i+3 < len(args) &&
				args[i+1] == "attach-session" &&
				args[i+2] == "-t" &&
				args[i+3] == "my-session" {
				foundTmux = true
				break
			}
		}
		if !foundTmux {
			t.Error("tmux attach command not found in args")
		}
	})

	t.Run("with jump host", func(t *testing.T) {
		remote := &RemoteConfig{
			Name:     "test",
			Host:     "target.example.com",
			User:     "testuser",
			Key:      "/home/user/.ssh/id_rsa",
			JumpHost: "bastion.example.com",
		}

		args := buildSSHAttachCommand(remote, "my-session")

		// Check jump host option
		foundJump := false
		for i, arg := range args {
			if arg == "-J" && i+1 < len(args) && args[i+1] == "testuser@bastion.example.com" {
				foundJump = true
				break
			}
		}
		if !foundJump {
			t.Error("jump host option not found in args")
		}
	})
}

func TestRemoteStatus(t *testing.T) {
	status := &RemoteStatus{
		Status:    StatusConnected,
		LastError: nil,
		LastPoll:  time.Now(),
	}

	if status.Status != StatusConnected {
		t.Errorf("status = %v, want connected", status.Status)
	}

	if status.LastError != nil {
		t.Errorf("last error = %v, want nil", status.LastError)
	}

	if status.LastPoll.IsZero() {
		t.Error("last poll should not be zero")
	}
}

func TestSSHPoolConnectUnknownRemote(t *testing.T) {
	pool := NewSSHPool([]RemoteConfig{})

	_, err := pool.Connect("unknown")
	if err == nil {
		t.Error("expected error for unknown remote")
	}
}
