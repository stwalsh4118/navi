# SSH Package Guide for Task 19-2

**Date**: 2026-02-05
**Package**: `golang.org/x/crypto/ssh`
**Documentation**: https://pkg.go.dev/golang.org/x/crypto/ssh

## Installation

```bash
go get golang.org/x/crypto/ssh
```

## Import

```go
import "golang.org/x/crypto/ssh"
```

## Key Types

### ClientConfig

```go
type ClientConfig struct {
    User              string           // Username to authenticate as
    Auth              []AuthMethod     // Authentication methods
    HostKeyCallback   HostKeyCallback  // Server key verification
    Timeout           time.Duration    // TCP connection timeout
}
```

### Key-Based Authentication

```go
// Load and parse private key
key, err := os.ReadFile("/home/user/.ssh/id_rsa")
if err != nil {
    log.Fatalf("unable to read private key: %v", err)
}

signer, err := ssh.ParsePrivateKey(key)
if err != nil {
    log.Fatalf("unable to parse private key: %v", err)
}

config := &ssh.ClientConfig{
    User: "user",
    Auth: []ssh.AuthMethod{
        ssh.PublicKeys(signer),
    },
    HostKeyCallback: ssh.InsecureIgnoreHostKey(), // For testing only
    Timeout: 10 * time.Second,
}
```

## Connection

### Direct Connection

```go
client, err := ssh.Dial("tcp", "host.com:22", config)
if err != nil {
    log.Fatalf("unable to connect: %v", err)
}
defer client.Close()
```

### Jump Host / Bastion Connection

```go
// First connect to bastion
bastionClient, err := ssh.Dial("tcp", "bastion:22", bastionConfig)
if err != nil {
    return nil, err
}

// Then tunnel through to target
conn, err := bastionClient.Dial("tcp", "target:22")
if err != nil {
    return nil, err
}

// Create SSH client on tunneled connection
ncc, chans, reqs, err := ssh.NewClientConn(conn, "target:22", targetConfig)
if err != nil {
    return nil, err
}

client := ssh.NewClient(ncc, chans, reqs)
```

## Executing Commands

```go
session, err := client.NewSession()
if err != nil {
    log.Fatal("Failed to create session: ", err)
}
defer session.Close()

// Option 1: Get output directly
output, err := session.Output("cat ~/.claude-sessions/*.json")

// Option 2: Use combined output (stdout + stderr)
output, err := session.CombinedOutput("ls -la")
```

## Host Key Verification Options

```go
// Accept any host key (insecure, for testing)
ssh.InsecureIgnoreHostKey()

// Accept only specific key
ssh.FixedHostKey(knownHostKey)
```

## Our Implementation Pattern

```go
type SSHPool struct {
    mu      sync.RWMutex
    remotes map[string]*RemoteConfig
    clients map[string]*ssh.Client
    status  map[string]ConnectionStatus
}

func (p *SSHPool) Execute(remoteName, command string) ([]byte, error) {
    client, err := p.getOrConnect(remoteName)
    if err != nil {
        return nil, err
    }

    session, err := client.NewSession()
    if err != nil {
        return nil, err
    }
    defer session.Close()

    return session.CombinedOutput(command)
}
```
