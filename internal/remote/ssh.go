package remote

import (
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/stwalsh4118/navi/internal/debug"
)

// SSH connection constants
const (
	SSHDefaultPort    = 22
	SSHConnectTimeout = 10 * time.Second
	SSHCommandTimeout = 30 * time.Second
)

// ConnectionStatus represents the state of a remote connection.
type ConnectionStatus int

const (
	StatusDisconnected ConnectionStatus = iota
	StatusConnected
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
type SSHPool struct {
	mu      sync.RWMutex
	remotes map[string]*Config
	clients map[string]*ssh.Client
	status  map[string]*RemoteStatus
}

// NewSSHPool creates a new SSH connection pool for the given remotes.
func NewSSHPool(remotes []Config) *SSHPool {
	pool := &SSHPool{
		remotes: make(map[string]*Config),
		clients: make(map[string]*ssh.Client),
		status:  make(map[string]*RemoteStatus),
	}

	for _, r := range remotes {
		rCopy := r
		pool.remotes[rCopy.Name] = &rCopy
		pool.status[rCopy.Name] = &RemoteStatus{
			Status: StatusDisconnected,
		}
	}

	return pool
}

// GetStatus returns the connection status for a remote.
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

	if client, ok := p.clients[remoteName]; ok {
		_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
		if err == nil {
			debug.Log("ssh[%s]: reusing existing connection", remoteName)
			return client, nil
		}
		debug.Log("ssh[%s]: existing connection dead, reconnecting: %v", remoteName, err)
		client.Close()
		delete(p.clients, remoteName)
	}

	remote, ok := p.remotes[remoteName]
	if !ok {
		return nil, fmt.Errorf("unknown remote: %s", remoteName)
	}

	debug.Log("ssh[%s]: connecting to %s@%s (key: %s)", remoteName, remote.User, remote.Host, remote.Key)

	client, err := p.dialRemote(remote)
	if err != nil {
		debug.Log("ssh[%s]: connection failed: %v", remoteName, err)
		p.status[remoteName] = &RemoteStatus{
			Status:    StatusError,
			LastError: err,
			LastPoll:  time.Now(),
		}
		return nil, err
	}

	debug.Log("ssh[%s]: connected successfully", remoteName)

	p.clients[remoteName] = client
	p.status[remoteName] = &RemoteStatus{
		Status:   StatusConnected,
		LastPoll: time.Now(),
	}

	return client, nil
}

// Execute runs a command on the named remote and returns the output.
func (p *SSHPool) Execute(remoteName, command string) ([]byte, error) {
	client, err := p.Connect(remoteName)
	if err != nil {
		return nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		debug.Log("ssh[%s]: session creation failed, reconnecting: %v", remoteName, err)
		p.mu.Lock()
		if oldClient, ok := p.clients[remoteName]; ok {
			oldClient.Close()
			delete(p.clients, remoteName)
		}
		p.mu.Unlock()

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

	output, err := session.CombinedOutput(command)
	if err != nil {
		debug.Log("ssh[%s]: command failed: %v (output: %q)", remoteName, err, string(output))
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

// GetRemoteConfig returns the configuration for a named remote.
func (p *SSHPool) GetRemoteConfig(remoteName string) *Config {
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

// BuildSSHAttachCommand builds the SSH command for attaching to a remote tmux session.
func BuildSSHAttachCommand(remote *Config, sessionName string) []string {
	args := []string{"ssh"}

	args = append(args, "-i", remote.Key)

	if remote.JumpHost != "" {
		args = append(args, "-J", fmt.Sprintf("%s@%s", remote.User, remote.JumpHost))
	}

	args = append(args, "-t")
	args = append(args, fmt.Sprintf("%s@%s", remote.User, remote.Host))
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

func (p *SSHPool) dialRemote(remote *Config) (*ssh.Client, error) {
	signer, err := LoadSSHKey(remote.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to load SSH key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: remote.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         SSHConnectTimeout,
	}

	targetAddr := fmt.Sprintf("%s:%d", remote.Host, SSHDefaultPort)

	if remote.JumpHost != "" {
		return p.dialThroughJumpHost(remote, config, targetAddr)
	}

	client, err := ssh.Dial("tcp", targetAddr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", remote.Host, err)
	}

	return client, nil
}

func (p *SSHPool) dialThroughJumpHost(remote *Config, targetConfig *ssh.ClientConfig, targetAddr string) (*ssh.Client, error) {
	signer, err := LoadSSHKey(remote.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to load SSH key for jump host: %w", err)
	}

	jumpConfig := &ssh.ClientConfig{
		User: remote.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         SSHConnectTimeout,
	}

	jumpAddr := fmt.Sprintf("%s:%d", remote.JumpHost, SSHDefaultPort)
	jumpClient, err := ssh.Dial("tcp", jumpAddr, jumpConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to jump host %s: %w", remote.JumpHost, err)
	}

	conn, err := jumpClient.Dial("tcp", targetAddr)
	if err != nil {
		jumpClient.Close()
		return nil, fmt.Errorf("failed to dial target through jump host: %w", err)
	}

	ncc, chans, reqs, err := ssh.NewClientConn(conn, targetAddr, targetConfig)
	if err != nil {
		conn.Close()
		jumpClient.Close()
		return nil, fmt.Errorf("failed to create SSH connection through jump host: %w", err)
	}

	client := ssh.NewClient(ncc, chans, reqs)
	return client, nil
}

// LoadSSHKey loads and parses an SSH private key from a file.
func LoadSSHKey(keyPath string) (ssh.Signer, error) {
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

