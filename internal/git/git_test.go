package git

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInfoJSON(t *testing.T) {
	original := &Info{
		Branch:     "feature/auth-flow",
		Dirty:      true,
		Ahead:      3,
		Behind:     1,
		LastCommit: "abc1234 Add login validation",
		Remote:     "git@github.com:user/repo.git",
		PRNum:      42,
		FetchedAt:  time.Now().Unix(),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal Info: %v", err)
	}

	var decoded Info
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Info: %v", err)
	}

	if decoded.Branch != original.Branch {
		t.Errorf("Branch mismatch: got %q, want %q", decoded.Branch, original.Branch)
	}
	if decoded.Dirty != original.Dirty {
		t.Errorf("Dirty mismatch: got %v, want %v", decoded.Dirty, original.Dirty)
	}
	if decoded.Ahead != original.Ahead {
		t.Errorf("Ahead mismatch: got %d, want %d", decoded.Ahead, original.Ahead)
	}
	if decoded.Behind != original.Behind {
		t.Errorf("Behind mismatch: got %d, want %d", decoded.Behind, original.Behind)
	}
	if decoded.PRNum != original.PRNum {
		t.Errorf("PRNum mismatch: got %d, want %d", decoded.PRNum, original.PRNum)
	}
}

func TestFormatStatusLine(t *testing.T) {
	tests := []struct {
		name     string
		git      *Info
		maxWidth int
		want     string
	}{
		{"nil git info", nil, 50, ""},
		{"clean branch", &Info{Branch: "main", Dirty: false}, 50, "main"},
		{"dirty branch", &Info{Branch: "feature/auth", Dirty: true}, 50, "feature/auth ●"},
		{"ahead of remote", &Info{Branch: "main", Ahead: 3}, 50, "main +3"},
		{"behind remote", &Info{Branch: "main", Behind: 2}, 50, "main -2"},
		{"full status", &Info{Branch: "feature/auth", Dirty: true, Ahead: 3, Behind: 1, PRNum: 42}, 50, "feature/auth ● +3 -1 [PR#42]"},
		{"long branch truncated", &Info{Branch: "feature/very-long-branch-name-that-exceeds-limit"}, 50, "feature/very-long-branch-na..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatStatusLine(tt.git, tt.maxWidth)
			if got != tt.want {
				t.Errorf("FormatStatusLine() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInfoIsStale(t *testing.T) {
	tests := []struct {
		name      string
		git       *Info
		wantStale bool
	}{
		{"nil git info is stale", nil, true},
		{"zero fetched at is stale", &Info{FetchedAt: 0}, true},
		{"recent fetch is not stale", &Info{FetchedAt: time.Now().Unix()}, false},
		{"old fetch is stale", &Info{FetchedAt: time.Now().Add(-20 * time.Second).Unix()}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.git.IsStale()
			if got != tt.wantStale {
				t.Errorf("IsStale() = %v, want %v", got, tt.wantStale)
			}
		})
	}
}

func TestGitHubInfoURLs(t *testing.T) {
	gh := &GitHubInfo{Owner: "user", Repo: "repo"}

	tests := []struct {
		name string
		fn   func() string
		want string
	}{
		{"base URL", func() string { return gh.GitHubURL("") }, "https://github.com/user/repo"},
		{"issue URL", func() string { return gh.IssueURL(123) }, "https://github.com/user/repo/issues/123"},
		{"PR URL", func() string { return gh.PRURL(42) }, "https://github.com/user/repo/pull/42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}

	var nilGH *GitHubInfo
	if nilGH.GitHubURL("/test") != "" {
		t.Error("nil GitHubInfo should return empty URL")
	}
}

func TestParseGitHubRemote(t *testing.T) {
	tests := []struct {
		name      string
		remoteURL string
		wantOwner string
		wantRepo  string
		wantNil   bool
	}{
		{"HTTPS with .git", "https://github.com/owner/repo.git", "owner", "repo", false},
		{"HTTPS without .git", "https://github.com/owner/repo", "owner", "repo", false},
		{"HTTPS with trailing slash", "https://github.com/owner/repo/", "owner", "repo", false},
		{"HTTP URL", "http://github.com/owner/repo.git", "owner", "repo", false},
		{"SSH with .git", "git@github.com:owner/repo.git", "owner", "repo", false},
		{"SSH without .git", "git@github.com:owner/repo", "owner", "repo", false},
		{"SSH protocol with .git", "ssh://git@github.com/owner/repo.git", "owner", "repo", false},
		{"SSH protocol without .git", "ssh://git@github.com/owner/repo", "owner", "repo", false},
		{"real HTTPS example", "https://github.com/stwalsh4118/navi.git", "stwalsh4118", "navi", false},
		{"real SSH example", "git@github.com:anthropics/claude-code.git", "anthropics", "claude-code", false},
		{"empty string", "", "", "", true},
		{"GitLab HTTPS", "https://gitlab.com/owner/repo.git", "", "", true},
		{"GitLab SSH", "git@gitlab.com:owner/repo.git", "", "", true},
		{"Bitbucket HTTPS", "https://bitbucket.org/owner/repo.git", "", "", true},
		{"self-hosted GitHub Enterprise", "https://github.example.com/owner/repo.git", "", "", true},
		{"invalid URL", "not-a-valid-url", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseGitHubRemote(tt.remoteURL)
			if tt.wantNil {
				if got != nil {
					t.Errorf("ParseGitHubRemote(%q) = %+v, want nil", tt.remoteURL, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("ParseGitHubRemote(%q) = nil, want non-nil", tt.remoteURL)
			}
			if got.Owner != tt.wantOwner {
				t.Errorf("Owner = %q, want %q", got.Owner, tt.wantOwner)
			}
			if got.Repo != tt.wantRepo {
				t.Errorf("Repo = %q, want %q", got.Repo, tt.wantRepo)
			}
		})
	}
}

func TestIsRepo(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	// We're inside internal/git/ which is inside the navi git repo
	if !IsRepo(cwd) {
		t.Error("Expected current directory to be a git repo")
	}
	if IsRepo("/") {
		t.Error("Expected root directory to not be a git repo")
	}
}

func TestGetBranch(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	branch := GetBranch(cwd)
	if branch == "" {
		t.Error("Expected to get a branch name for current directory")
	}

	branch = GetBranch("/")
	if branch != "" {
		t.Errorf("Expected empty branch for non-git directory, got %q", branch)
	}
}

func TestIsDirty(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	_ = IsDirty(cwd)

	if IsDirty("/") {
		t.Error("Expected non-git directory to not be dirty")
	}
}

func TestGetLastCommit(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	commit := GetLastCommit(cwd)
	if commit == "" {
		t.Error("Expected to get last commit for current directory")
	}

	commit = GetLastCommit("/")
	if commit != "" {
		t.Errorf("Expected empty last commit for non-git directory, got %q", commit)
	}
}

func TestGetRemoteFunc(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	remote := GetRemote(cwd)
	if remote == "" {
		t.Error("Expected to get remote URL for current directory")
	}

	remote = GetRemote("/")
	if remote != "" {
		t.Errorf("Expected empty remote for non-git directory, got %q", remote)
	}
}

func TestGetInfo(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	info := GetInfo(cwd)
	if info == nil {
		t.Fatal("Expected Info for current directory, got nil")
	}
	if info.Branch == "" {
		t.Error("Expected Branch to be set")
	}
	if info.FetchedAt == 0 {
		t.Error("Expected FetchedAt to be set")
	}

	info = GetInfo("/")
	if info != nil {
		t.Error("Expected nil Info for non-git directory")
	}

	info = GetInfo("/nonexistent/path")
	if info != nil {
		t.Error("Expected nil Info for non-existent directory")
	}
}

func TestGetInfoWithTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Could not get home directory")
	}

	testPath := filepath.Join(home, ".config")
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Skip("~/.config does not exist")
	}

	_ = GetInfo("~/.config")
}

func TestGetPRNumber(t *testing.T) {
	t.Run("returns 0 for non-git directory", func(t *testing.T) {
		prNum := GetPRNumber("/")
		if prNum != 0 {
			t.Errorf("GetPRNumber(/) = %d, want 0", prNum)
		}
	})

	t.Run("returns 0 when gh CLI not available or no PR", func(t *testing.T) {
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get cwd: %v", err)
		}
		_ = GetPRNumber(cwd)
	})
}

func TestGetDiffStat(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	_ = GetDiffStat(cwd)

	diffStat := GetDiffStat("/")
	if diffStat != "" {
		t.Errorf("Expected empty diff stat for non-git directory, got %q", diffStat)
	}
}

func TestGetDiff(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	_ = GetDiff(cwd)

	diff := GetDiff("/")
	if diff != "" {
		t.Errorf("Expected empty diff for non-git directory, got %q", diff)
	}
}

func TestDiffConstants(t *testing.T) {
	if DiffMaxLines <= 0 {
		t.Errorf("DiffMaxLines should be positive, got %d", DiffMaxLines)
	}
	if DiffTruncatedMsg == "" {
		t.Error("DiffTruncatedMsg should not be empty")
	}
}

func TestOpenURL(t *testing.T) {
	t.Run("empty URL returns error", func(t *testing.T) {
		err := OpenURL("")
		if err == nil {
			t.Error("expected error for empty URL")
		}
		if !strings.Contains(err.Error(), "empty URL") {
			t.Errorf("expected error message to mention 'empty URL', got: %v", err)
		}
	})

}
