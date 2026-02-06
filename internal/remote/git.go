package remote

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/stwalsh4118/navi/internal/debug"
	"github.com/stwalsh4118/navi/internal/git"
)

// Git output label prefixes used in the bundled SSH command.
const (
	labelBranch      = "BRANCH:"
	labelDirty       = "DIRTY:"
	labelRemote      = "REMOTE:"
	labelLastCommit  = "LASTCOMMIT:"
	labelAheadBehind = "AHEADBEHIND:"
)

// FetchGitInfo runs a bundled git command over SSH and returns parsed git.Info.
// Returns (nil, nil) for non-git directories. Returns an error only on SSH failures.
func FetchGitInfo(pool *SSHPool, remoteName, cwd string) (*git.Info, error) {
	cmd := buildGitCommand(cwd)

	output, err := pool.Execute(remoteName, cmd)
	if err != nil {
		// Command failure likely means the directory doesn't exist or isn't a git repo.
		debug.Log("remote[%s]: git info command failed for %s: %v", remoteName, cwd, err)
		return nil, nil
	}

	info := parseGitOutput(string(output))
	return info, nil
}

// buildGitCommand constructs a shell command that cd's to cwd and runs
// multiple git commands with labeled output, all in a single SSH call.
func buildGitCommand(cwd string) string {
	// Escape double quotes in the path for safe shell embedding.
	safeCwd := strings.ReplaceAll(cwd, "\"", "\\\"")

	return fmt.Sprintf(
		`cd "%s" && echo "BRANCH:$(git rev-parse --abbrev-ref HEAD 2>/dev/null)" && echo "DIRTY:$(git status --porcelain 2>/dev/null | head -1)" && echo "REMOTE:$(git remote get-url origin 2>/dev/null)" && echo "LASTCOMMIT:$(git log -1 --format='%%h %%s' 2>/dev/null)" && echo "AHEADBEHIND:$(git rev-list --left-right --count @{u}...HEAD 2>/dev/null)"`,
		safeCwd,
	)
}

// parseGitOutput parses the labeled output from the bundled git command
// into a *git.Info struct. Returns nil if the output indicates a non-git directory
// (empty BRANCH line).
func parseGitOutput(output string) *git.Info {
	lines := strings.Split(strings.TrimSpace(output), "\n")

	values := make(map[string]string)
	for _, line := range lines {
		for _, label := range []string{labelBranch, labelDirty, labelRemote, labelLastCommit, labelAheadBehind} {
			if strings.HasPrefix(line, label) {
				values[label] = strings.TrimPrefix(line, label)
				break
			}
		}
	}

	branch := values[labelBranch]
	if branch == "" {
		return nil
	}

	ahead, behind := parseAheadBehind(values[labelAheadBehind])

	return &git.Info{
		Branch:     branch,
		Dirty:      values[labelDirty] != "",
		Remote:     values[labelRemote],
		LastCommit: values[labelLastCommit],
		Ahead:      ahead,
		Behind:     behind,
		FetchedAt:  time.Now().Unix(),
	}
}

// parseAheadBehind parses the tab-separated "behind\tahead" output from
// git rev-list --left-right --count. Returns (0, 0) on any parse failure.
func parseAheadBehind(s string) (ahead, behind int) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0
	}

	parts := strings.Fields(s)
	if len(parts) != 2 {
		return 0, 0
	}

	behind, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0
	}

	ahead, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0
	}

	return ahead, behind
}
