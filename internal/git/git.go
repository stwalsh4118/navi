package git

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

// Info represents git repository information for a session's working directory.
type Info struct {
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
	githubHTTPSRegex       = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+?)(?:\.git)?/?$`)
	githubSSHRegex         = regexp.MustCompile(`^git@github\.com:([^/]+)/([^/]+?)(?:\.git)?$`)
	githubSSHProtocolRegex = regexp.MustCompile(`^ssh://git@github\.com/([^/]+)/([^/]+?)(?:\.git)?/?$`)
)

// Display constants
const (
	DirtyIndicator  = "●"
	AheadPrefix     = "+"
	BehindPrefix    = "-"
	PRPrefix        = "PR#"
	MaxBranchLength = 30
)

// Polling constants
const (
	PollInterval = 5 * time.Second
	CacheMaxAge  = 10 * time.Second
)

// Diff display constants
const (
	DiffMaxLines    = 100
	DiffTruncatedMsg = "\n... (diff truncated)"
)

// GetPRNumber checks if there's a GitHub PR for the current branch.
func GetPRNumber(dir string) int {
	cmd := exec.Command("gh", "pr", "view", "--json", "number", "--jq", ".number")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	numStr := strings.TrimSpace(string(output))
	if num, err := strconv.Atoi(numStr); err == nil && num > 0 {
		return num
	}
	return 0
}

// ParseGitHubRemote parses a git remote URL and extracts GitHub owner and repo.
func ParseGitHubRemote(remoteURL string) *GitHubInfo {
	if remoteURL == "" {
		return nil
	}

	if matches := githubHTTPSRegex.FindStringSubmatch(remoteURL); len(matches) == 3 {
		return &GitHubInfo{Owner: matches[1], Repo: matches[2]}
	}
	if matches := githubSSHRegex.FindStringSubmatch(remoteURL); len(matches) == 3 {
		return &GitHubInfo{Owner: matches[1], Repo: matches[2]}
	}
	if matches := githubSSHProtocolRegex.FindStringSubmatch(remoteURL); len(matches) == 3 {
		return &GitHubInfo{Owner: matches[1], Repo: matches[2]}
	}

	return nil
}

// FormatStatusLine formats git info for display in the session row.
func FormatStatusLine(g *Info, maxWidth int) string {
	if g == nil {
		return ""
	}

	var parts []string

	branch := g.Branch
	if len(branch) > MaxBranchLength {
		branch = branch[:MaxBranchLength-3] + "..."
	}
	parts = append(parts, branch)

	if g.Dirty {
		parts = append(parts, DirtyIndicator)
	}
	if g.Ahead > 0 {
		parts = append(parts, fmt.Sprintf("%s%d", AheadPrefix, g.Ahead))
	}
	if g.Behind > 0 {
		parts = append(parts, fmt.Sprintf("%s%d", BehindPrefix, g.Behind))
	}
	if g.PRNum > 0 {
		parts = append(parts, fmt.Sprintf("[%s%d]", PRPrefix, g.PRNum))
	}

	result := strings.Join(parts, " ")

	if maxWidth > 0 && len(result) > maxWidth {
		if maxWidth > 3 {
			result = result[:maxWidth-3] + "..."
		} else {
			result = result[:maxWidth]
		}
	}

	return result
}

// IsStale returns true if the git info is older than CacheMaxAge.
func (g *Info) IsStale() bool {
	if g == nil || g.FetchedAt == 0 {
		return true
	}
	return time.Since(time.Unix(g.FetchedAt, 0)) > CacheMaxAge
}

// GitHubURL constructs a GitHub URL for the repository.
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

// OpenURL opens a URL in the system's default browser.
func OpenURL(url string) error {
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

// IsRepo checks if the given directory is inside a git repository.
func IsRepo(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = dir
	err := cmd.Run()
	return err == nil
}

// GetBranch returns the current branch name for the git repository.
func GetBranch(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// IsDirty checks if the repository has uncommitted changes.
func IsDirty(dir string) bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) > 0
}

// GetAheadBehind returns the number of commits ahead and behind the upstream.
func GetAheadBehind(dir string) (ahead, behind int) {
	cmd := exec.Command("git", "rev-list", "--left-right", "--count", "@{u}...HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return 0, 0
	}

	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) != 2 {
		return 0, 0
	}

	behind, _ = strconv.Atoi(parts[0])
	ahead, _ = strconv.Atoi(parts[1])
	return ahead, behind
}

// GetLastCommit returns the short hash and subject of the last commit.
func GetLastCommit(dir string) string {
	cmd := exec.Command("git", "log", "-1", "--format=%h %s")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// GetRemote returns the URL of the 'origin' remote.
func GetRemote(dir string) string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// GetDiffStat returns the git diff --stat output for the working directory.
func GetDiffStat(dir string) string {
	cmd := exec.Command("git", "diff", "--stat")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// GetDiff returns the full git diff output for the working directory.
func GetDiff(dir string) string {
	cmd := exec.Command("git", "diff")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	result := strings.TrimSpace(string(output))

	lines := strings.Split(result, "\n")
	if len(lines) > DiffMaxLines {
		lines = lines[:DiffMaxLines]
		result = strings.Join(lines, "\n") + DiffTruncatedMsg
	}

	return result
}

// GetInfo collects all git information for the given directory.
// Returns nil if the directory is not a git repository.
func GetInfo(dir string) *Info {
	if strings.HasPrefix(dir, "~/") {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, dir[2:])
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}

	if !IsRepo(dir) {
		return nil
	}

	branch := GetBranch(dir)
	info := &Info{
		Branch:     branch,
		Dirty:      IsDirty(dir),
		LastCommit: GetLastCommit(dir),
		Remote:     GetRemote(dir),
		FetchedAt:  time.Now().Unix(),
	}

	info.Ahead, info.Behind = GetAheadBehind(dir)

	return info
}
