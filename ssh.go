package main

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSH connection constants
const (
	// sshDefaultPort is the default SSH port
	sshDefaultPort = 22

	// sshConnectTimeout is the timeout for establishing SSH connections
	sshConnectTimeout = 10 * time.Second

	// sshCommandTimeout is the timeout for executing commands over SSH
	sshCommandTimeout = 30 * time.Second
)

// ConnectionStatus represents the state of a remote connection.
type ConnectionStatus int

const (
	// StatusDisconnected means no connection has been established
	StatusDisconnected ConnectionStatus = iota

	// StatusConnected means the connection is active
	StatusConnected

	// StatusError means the connection failed
	StatusError
)

// String returns a human-readable status string.
func (s ConnectionStatus) String() string {
	switch s {
	case StatusConnected:
		return "connected"
	case StatusDisconnected:
		return "disconnected"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

// RemoteStatus holds the connection status and last error for a remote.
type RemoteStatus struct {
	Status    ConnectionStatus
	LastError error
	LastPoll  time.Time
}

// SSHPool manages SSH connections to multiple remote machines.
// It provides connection pooling, thread-safe access, and status tracking.
type SSHPool struct {
	mu      sync.RWMutex
	remotes map[string]*RemoteConfig // Map of remote name to config
	clients map[string]*ssh.Client   // Map of remote name to active client
	status  map[string]*RemoteStatus // Map of remote name to status
}

// NewSSHPool creates a new SSH connection pool for the given remotes.
func NewSSHPool(remotes []RemoteConfig) *SSHPool {
	pool := &SSHPool{
		remotes: make(map[string]*RemoteConfig),
		clients: make(map[string]*ssh.Client),
		status:  make(map[string]*RemoteStatus),
	}

	for i := range remotes {
		pool.remotes[remotes[i].Name] = &remotes[i]
		pool.status[remotes[i].Name] = &RemoteStatus{
			Status: StatusDisconnected,
		}
	}

	return pool
}

// GetStatus returns the connection status for a remote.
// Returns StatusDisconnected if the remote is unknown.
func (p *SSHPool) GetStatus(remoteName string) *RemoteStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	status, ok := p.status[remoteName]
	if !ok {
		return &RemoteStatus{Status: StatusDisconnected}
	}
	return status
}

// GetAllStatus returns the status of all configured remotes.
func (p *SSHPool) GetAllStatus() map[string]*RemoteStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]*RemoteStatus, len(p.status))
	for name, status := range p.status {
		// Return a copy to avoid race conditions
		result[name] = &RemoteStatus{
			Status:    status.Status,
			LastError: status.LastError,
			LastPoll:  status.LastPoll,
		}
	}
	return result
}

// Connect establishes or returns an existing connection to the named remote.
func (p *SSHPool) Connect(remoteName string) (*ssh.Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if we have an existing connection
	if client, ok := p.clients[remoteName]; ok {
		// Verify the connection is still alive by sending a keepalive
		_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
		if err == nil {
			debugLog("ssh[%s]: reusing existing connection", remoteName)
			return client, nil
		}
		// Connection is dead, clean up
		debugLog("ssh[%s]: existing connection dead, reconnecting: %v", remoteName, err)
		client.Close()
		delete(p.clients, remoteName)
	}

	// Get the remote config
	remote, ok := p.remotes[remoteName]
	if !ok {
		return nil, fmt.Errorf("unknown remote: %s", remoteName)
	}

	debugLog("ssh[%s]: connecting to %s@%s (key: %s)", remoteName, remote.User, remote.Host, remote.Key)

	// Establish new connection
	client, err := p.dialRemote(remote)
	if err != nil {
		debugLog("ssh[%s]: connection failed: %v", remoteName, err)
		p.status[remoteName] = &RemoteStatus{
			Status:    StatusError,
			LastError: err,
			LastPoll:  time.Now(),
		}
		return nil, err
	}

	debugLog("ssh[%s]: connected successfully", remoteName)

	// Store the connection
	p.clients[remoteName] = client
	p.status[remoteName] = &RemoteStatus{
		Status:   StatusConnected,
		LastPoll: time.Now(),
	}

	return client, nil
}

// Execute runs a command on the named remote and returns the output.
// It handles connection establishment and session management.
func (p *SSHPool) Execute(remoteName, command string) ([]byte, error) {
	client, err := p.Connect(remoteName)
	if err != nil {
		return nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		debugLog("ssh[%s]: session creation failed, reconnecting: %v", remoteName, err)
		// Connection may be stale, try to reconnect once
		p.mu.Lock()
		if oldClient, ok := p.clients[remoteName]; ok {
			oldClient.Close()
			delete(p.clients, remoteName)
		}
		p.mu.Unlock()

		// Retry connection
		client, err = p.Connect(remoteName)
		if err != nil {
			return nil, err
		}

		session, err = client.NewSession()
		if err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
	}
	defer session.Close()

	// Execute the command
	output, err := session.CombinedOutput(command)
	if err != nil {
		debugLog("ssh[%s]: command failed: %v (output: %q)", remoteName, err, string(output))
		return output, err
	}

	return output, nil
}

// Close closes all connections in the pool.
func (p *SSHPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for name, client := range p.clients {
		client.Close()
		delete(p.clients, name)
		p.status[name] = &RemoteStatus{
			Status:   StatusDisconnected,
			LastPoll: time.Now(),
		}
	}
}

// dialRemote establishes an SSH connection to a remote, optionally through a jump host.
func (p *SSHPool) dialRemote(remote *RemoteConfig) (*ssh.Client, error) {
	// Load SSH private key
	signer, err := loadSSHKey(remote.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to load SSH key: %w", err)
	}

	// Create SSH client config
	config := &ssh.ClientConfig{
		User: remote.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Support known hosts
		Timeout:         sshConnectTimeout,
	}

	// Build target address
	targetAddr := fmt.Sprintf("%s:%d", remote.Host, sshDefaultPort)

	// Connect through jump host if configured
	if remote.JumpHost != "" {
		return p.dialThroughJumpHost(remote, config, targetAddr)
	}

	// Direct connection
	client, err := ssh.Dial("tcp", targetAddr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", remote.Host, err)
	}

	return client, nil
}

// dialThroughJumpHost establishes an SSH connection through a bastion/jump host.
func (p *SSHPool) dialThroughJumpHost(remote *RemoteConfig, targetConfig *ssh.ClientConfig, targetAddr string) (*ssh.Client, error) {
	// Load SSH key for jump host (use same key for simplicity)
	signer, err := loadSSHKey(remote.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to load SSH key for jump host: %w", err)
	}

	// Create jump host config
	jumpConfig := &ssh.ClientConfig{
		User: remote.User, // Use same user for jump host
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         sshConnectTimeout,
	}

	// Connect to jump host
	jumpAddr := fmt.Sprintf("%s:%d", remote.JumpHost, sshDefaultPort)
	jumpClient, err := ssh.Dial("tcp", jumpAddr, jumpConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to jump host %s: %w", remote.JumpHost, err)
	}

	// Dial target through jump host
	conn, err := jumpClient.Dial("tcp", targetAddr)
	if err != nil {
		jumpClient.Close()
		return nil, fmt.Errorf("failed to dial target through jump host: %w", err)
	}

	// Create SSH client over the tunneled connection
	ncc, chans, reqs, err := ssh.NewClientConn(conn, targetAddr, targetConfig)
	if err != nil {
		conn.Close()
		jumpClient.Close()
		return nil, fmt.Errorf("failed to create SSH connection through jump host: %w", err)
	}

	client := ssh.NewClient(ncc, chans, reqs)
	return client, nil
}

// loadSSHKey loads and parses an SSH private key from a file.
func loadSSHKey(keyPath string) (ssh.Signer, error) {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file %s: %w", keyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return signer, nil
}

// GetRemoteConfig returns the configuration for a named remote.
// Returns nil if the remote doesn't exist.
func (p *SSHPool) GetRemoteConfig(remoteName string) *RemoteConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.remotes[remoteName]
}

// RemoteNames returns a list of all configured remote names.
func (p *SSHPool) RemoteNames() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	names := make([]string, 0, len(p.remotes))
	for name := range p.remotes {
		names = append(names, name)
	}
	return names
}

// buildSSHAttachCommand builds the SSH command for attaching to a remote tmux session.
// This is used for spawning an SSH process to attach to remote sessions.
func buildSSHAttachCommand(remote *RemoteConfig, sessionName string) []string {
	args := []string{"ssh"}

	// Add key option
	args = append(args, "-i", remote.Key)

	// Add jump host if configured
	if remote.JumpHost != "" {
		args = append(args, "-J", fmt.Sprintf("%s@%s", remote.User, remote.JumpHost))
	}

	// Add terminal allocation (required for tmux)
	args = append(args, "-t")

	// Add target host
	args = append(args, fmt.Sprintf("%s@%s", remote.User, remote.Host))

	// Add tmux attach command
	args = append(args, "tmux", "attach-session", "-t", sessionName)

	return args
}

// IsConnected returns true if the pool has at least one connected remote.
func (p *SSHPool) IsConnected() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, status := range p.status {
		if status.Status == StatusConnected {
			return true
		}
	}
	return false
}

// Disconnect closes the connection to a specific remote.
func (p *SSHPool) Disconnect(remoteName string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if client, ok := p.clients[remoteName]; ok {
		client.Close()
		delete(p.clients, remoteName)
	}

	if status, ok := p.status[remoteName]; ok {
		status.Status = StatusDisconnected
		status.LastPoll = time.Now()
	}
}

// Ensure we use the net package to avoid import errors
var _ = net.Dial
