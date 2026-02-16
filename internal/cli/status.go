package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/stwalsh4118/navi/internal/pathutil"
	"github.com/stwalsh4118/navi/internal/session"
)

var defaultStatusOrder = []string{session.StatusWaiting, session.StatusPermission}

var verboseStatusOrder = []string{
	session.StatusWorking,
	session.StatusWaiting,
	session.StatusPermission,
	session.StatusIdle,
	session.StatusStopped,
}

// RunStatus executes the one-shot status command.
func RunStatus(args []string) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	verbose := fs.Bool("verbose", false, "show all non-zero status counts")
	format := fs.String("format", "plain", "output format (plain|tmux)")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *format != "plain" && *format != "tmux" {
		fmt.Fprintf(os.Stderr, "invalid format: %s\n", *format)
		return 1
	}

	sessions, err := session.ReadStatusFiles(pathutil.ExpandPath(session.StatusDir))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed reading session statuses: %v\n", err)
		return 1
	}

	counts := countStatuses(sessions)
	output := formatSummary(counts, *verbose)
	if output != "" {
		fmt.Fprintln(os.Stdout, output)
	}

	return 0
}

func countStatuses(sessions []session.Info) map[string]int {
	counts := make(map[string]int)
	for _, s := range sessions {
		counts[s.Status]++
	}
	return counts
}

func formatSummary(counts map[string]int, verbose bool) string {
	order := defaultStatusOrder
	if verbose {
		order = verboseStatusOrder
	}

	parts := make([]string, 0, len(order))
	for _, status := range order {
		if counts[status] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", counts[status], status))
		}
	}

	return strings.Join(parts, ", ")
}
