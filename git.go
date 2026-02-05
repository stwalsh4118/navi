package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// GitInfo represents git repository information for a session's working directory.
type GitInfo struct {
	Branch     string `json:"branch"`               // Current branch name
	Dirty      bool   `json:"dirty"`                // Whether there are uncommitted changes
	Ahead      int    `json:"ahead"`                // Number of commits ahead of remote
	Behind     int    `json:"behind"`               // Number of commits behind remote
	LastCommit string `json:"last_commit"`          // Short hash + subject of last commit
	Remote     string `json:"remote"`               // Remote URL (for GitHub detection)
	PRNum      int    `json:"pr_num,omitempty"`     // GitHub PR number for current branch (from gh CLI)
	FetchedAt  int64  `json:"fetched_at,omitempty"` // Unix timestamp when git info was fetched
}

// GitHubInfo contains parsed GitHub repository information from a remote URL.
type GitHubInfo struct {
	Owner string // Repository owner (user or organization)
	Repo  string // Repository name
}

// GitHub remote URL patterns
var (
	// Pattern for HTTPS URLs: https://github.com/owner/repo.git or https://github.com/owner/repo
	githubHTTPSRegex = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+?)(?:\.git)?/?$`)

	// Pattern for SSH URLs: git@github.com:owner/repo.git or git@github.com:owner/repo
	githubSSHRegex = regexp.MustCompile(`^git@github\.com:([^/]+)/([^/]+?)(?:\.git)?$`)

	// Pattern for SSH URLs with protocol: ssh://git@github.com/owner/repo.git
	githubSSHProtocolRegex = regexp.MustCompile(`^ssh://git@github\.com/([^/]+)/([^/]+?)(?:\.git)?/?$`)
)

// getGitPRNumber checks if there's a GitHub PR for the current branch.
// Uses `gh pr view` to detect an open PR and returns its number.
// Returns 0 if no PR exists or gh CLI is not available.
func getGitPRNumber(dir string) int {
	// Use gh CLI to check for PR on current branch
	cmd := exec.Command("gh", "pr", "view", "--json", "number", "--jq", ".number")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		// No PR exists or gh not installed
		return 0
	}

	numStr := strings.TrimSpace(string(output))
	if num, err := strconv.Atoi(numStr); err == nil && num > 0 {
		return num
	}
	return 0
}

// parseGitHubRemote parses a git remote URL and extracts GitHub owner and repo.
// Returns nil if the URL is not a valid GitHub remote.
// Supports various URL formats:
//   - https://github.com/owner/repo.git
//   - https://github.com/owner/repo
//   - git@github.com:owner/repo.git
//   - git@github.com:owner/repo
//   - ssh://git@github.com/owner/repo.git
func parseGitHubRemote(remoteURL string) *GitHubInfo {
	if remoteURL == "" {
		return nil
	}

	// Try HTTPS pattern
	if matches := githubHTTPSRegex.FindStringSubmatch(remoteURL); len(matches) == 3 {
		return &GitHubInfo{Owner: matches[1], Repo: matches[2]}
	}

	// Try SSH pattern (git@github.com:owner/repo)
	if matches := githubSSHRegex.FindStringSubmatch(remoteURL); len(matches) == 3 {
		return &GitHubInfo{Owner: matches[1], Repo: matches[2]}
	}

	// Try SSH protocol pattern (ssh://git@github.com/owner/repo)
	if matches := githubSSHProtocolRegex.FindStringSubmatch(remoteURL); len(matches) == 3 {
		return &GitHubInfo{Owner: matches[1], Repo: matches[2]}
	}

	return nil
}

// Git display constants
const (
	// gitDirtyIndicator is shown when there are uncommitted changes
	gitDirtyIndicator = "●"

	// gitAheadPrefix is shown before the ahead count
	gitAheadPrefix = "+"

	// gitBehindPrefix is shown before the behind count
	gitBehindPrefix = "-"

	// gitPRPrefix is shown before a detected PR number
	gitPRPrefix = "PR#"

	// gitMaxBranchLength is the maximum length for branch name display
	gitMaxBranchLength = 30
)

// Git polling constants
const (
	// gitPollInterval is how often to refresh git info (less frequent than session polling)
	gitPollInterval = 5 * time.Second

	// gitCacheMaxAge is the maximum age for cached git info before it's considered stale
	gitCacheMaxAge = 10 * time.Second
)

// formatGitStatusLine formats git info for display in the session row.
// Returns a string like "feature/auth ● +3 -1" or "main" for clean branches.
func formatGitStatusLine(git *GitInfo, maxWidth int) string {
	if git == nil {
		return ""
	}

	var parts []string

	// Branch name (possibly truncated)
	branch := git.Branch
	if len(branch) > gitMaxBranchLength {
		branch = branch[:gitMaxBranchLength-3] + "..."
	}
	parts = append(parts, branch)

	// Dirty indicator
	if git.Dirty {
		parts = append(parts, gitDirtyIndicator)
	}

	// Ahead/behind counts
	if git.Ahead > 0 {
		parts = append(parts, fmt.Sprintf("%s%d", gitAheadPrefix, git.Ahead))
	}
	if git.Behind > 0 {
		parts = append(parts, fmt.Sprintf("%s%d", gitBehindPrefix, git.Behind))
	}

	// PR number (if detected via gh CLI)
	if git.PRNum > 0 {
		parts = append(parts, fmt.Sprintf("[%s%d]", gitPRPrefix, git.PRNum))
	}

	result := strings.Join(parts, " ")

	// Truncate if too long
	if maxWidth > 0 && len(result) > maxWidth {
		if maxWidth > 3 {
			result = result[:maxWidth-3] + "..."
		} else {
			result = result[:maxWidth]
		}
	}

	return result
}

// IsStale returns true if the git info is older than gitCacheMaxAge.
func (g *GitInfo) IsStale() bool {
	if g == nil || g.FetchedAt == 0 {
		return true
	}
	return time.Since(time.Unix(g.FetchedAt, 0)) > gitCacheMaxAge
}

// GitHubURL constructs a GitHub URL for the repository.
// path is appended to the base URL (e.g., "/issues/123" or "/pull/42").
func (gh *GitHubInfo) GitHubURL(path string) string {
	if gh == nil || gh.Owner == "" || gh.Repo == "" {
		return ""
	}
	return fmt.Sprintf("https://github.com/%s/%s%s", gh.Owner, gh.Repo, path)
}

// IssueURL returns the GitHub issue URL for the given issue number.
func (gh *GitHubInfo) IssueURL(issueNum int) string {
	return gh.GitHubURL(fmt.Sprintf("/issues/%d", issueNum))
}

// PRURL returns the GitHub pull request URL for the given PR number.
func (gh *GitHubInfo) PRURL(prNum int) string {
	return gh.GitHubURL(fmt.Sprintf("/pull/%d", prNum))
}

// openURL opens a URL in the system's default browser.
// Supports Linux (xdg-open), macOS (open), and Windows (start).
func openURL(url string) error {
	if url == "" {
		return fmt.Errorf("empty URL")
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

// isGitRepo checks if the given directory is inside a git repository.
// Returns true if .git exists in the directory or any parent directory.
func isGitRepo(dir string) bool {
	// Use git rev-parse to check if inside a git repo
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = dir
	err := cmd.Run()
	return err == nil
}

// getGitBranch returns the current branch name for the git repository.
// Returns empty string if not in a git repo or on error.
func getGitBranch(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// isGitDirty checks if the repository has uncommitted changes.
// Returns true if there are staged or unstaged changes.
func isGitDirty(dir string) bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) > 0
}

// getGitAheadBehind returns the number of commits ahead and behind the upstream.
// Returns (0, 0) if there's no upstream or on error.
func getGitAheadBehind(dir string) (ahead, behind int) {
	// git rev-list --left-right --count @{u}...HEAD
	// Output format: "<behind>\t<ahead>"
	cmd := exec.Command("git", "rev-list", "--left-right", "--count", "@{u}...HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		// No upstream configured or other error
		return 0, 0
	}

	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) != 2 {
		return 0, 0
	}

	// First number is behind, second is ahead
	behind, _ = strconv.Atoi(parts[0])
	ahead, _ = strconv.Atoi(parts[1])
	return ahead, behind
}

// getGitLastCommit returns the short hash and subject of the last commit.
// Returns empty string if not in a git repo or on error.
func getGitLastCommit(dir string) string {
	cmd := exec.Command("git", "log", "-1", "--format=%h %s")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// getGitRemote returns the URL of the 'origin' remote.
// Returns empty string if no remote is configured or on error.
func getGitRemote(dir string) string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// Diff display constants
const (
	// gitDiffMaxLines is the maximum number of lines to display in the diff view
	gitDiffMaxLines = 100

	// gitDiffTruncatedMsg is shown when diff output is truncated
	gitDiffTruncatedMsg = "\n... (diff truncated)"
)

// getGitDiffStat returns the git diff --stat output for the working directory.
// Shows a summary of file changes with insertions/deletions.
// Returns empty string if no changes or on error.
func getGitDiffStat(dir string) string {
	cmd := exec.Command("git", "diff", "--stat")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// getGitDiff returns the full git diff output for the working directory.
// This includes both staged and unstaged changes.
// The output may be truncated if it exceeds gitDiffMaxLines.
func getGitDiff(dir string) string {
	cmd := exec.Command("git", "diff")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	result := strings.TrimSpace(string(output))

	// Truncate if too long
	lines := strings.Split(result, "\n")
	if len(lines) > gitDiffMaxLines {
		lines = lines[:gitDiffMaxLines]
		result = strings.Join(lines, "\n") + gitDiffTruncatedMsg
	}

	return result
}

// getGitInfo collects all git information for the given directory.
// Returns nil if the directory is not a git repository.
func getGitInfo(dir string) *GitInfo {
	// Expand path if needed
	if strings.HasPrefix(dir, "~/") {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, dir[2:])
	}

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}

	// Check if it's a git repository
	if !isGitRepo(dir) {
		return nil
	}

	// Gather git information
	branch := getGitBranch(dir)
	info := &GitInfo{
		Branch:     branch,
		Dirty:      isGitDirty(dir),
		LastCommit: getGitLastCommit(dir),
		Remote:     getGitRemote(dir),
		FetchedAt:  time.Now().Unix(),
	}

	// Get ahead/behind (may fail if no upstream)
	info.Ahead, info.Behind = getGitAheadBehind(dir)

	// Note: PRNum is fetched lazily via getGitPRNumber() only when needed
	// (e.g., when opening git detail view) to avoid slow gh CLI calls on every poll

	return info
}
