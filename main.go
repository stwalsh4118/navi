package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func initialModel() Model {
	// Load remote configuration (errors are logged but not fatal)
	remotes, err := loadRemotesConfig()
	if err != nil {
		// Log error but continue - remotes are optional
		fmt.Fprintf(os.Stderr, "Warning: failed to load remotes config: %v\n", err)
		remotes = []RemoteConfig{}
	}

	// Initialize SSH pool if remotes are configured
	var sshPool *SSHPool
	if len(remotes) > 0 {
		sshPool = NewSSHPool(remotes)
	}

	return Model{
		sessions: []SessionInfo{},
		cursor:   0,
		width:    80,
		height:   24,
		remotes:  remotes,
		sshPool:  sshPool,
	}
}
