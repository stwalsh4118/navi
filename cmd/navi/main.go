package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/cli"
	"github.com/stwalsh4118/navi/internal/tui"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "status":
			os.Exit(cli.RunStatus(os.Args[2:]))
		case "sound":
			os.Exit(cli.RunSound(os.Args[2:]))
		}
	}

	p := tea.NewProgram(tui.InitialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
