package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stwalsh4118/navi/internal/git"
	"github.com/stwalsh4118/navi/internal/session"
)

func TestGitInfoJSON(t *testing.T) {
	// Test marshaling and unmarshaling
	original := &git.Info{
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
		t.Fatalf("Failed to marshal git.Info: %v", err)
	}

	var decoded git.Info
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal git.Info: %v", err)
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

func TestFormatGitStatusLine(t *testing.T) {
	tests := []struct {
		name     string
		git      *git.Info
		maxWidth int
		want     string
	}{
		{
			name:     "nil git info",
			git:      nil,
			maxWidth: 50,
			want:     "",
		},
		{
			name: "clean branch",
			git: &git.Info{
				Branch: "main",
				Dirty:  false,
			},
			maxWidth: 50,
			want:     "main",
		},
		{
			name: "dirty branch",
			git: &git.Info{
				Branch: "feature/auth",
				Dirty:  true,
			},
			maxWidth: 50,
			want:     "feature/auth ●",
		},
		{
			name: "ahead of remote",
			git: &git.Info{
				Branch: "main",
				Dirty:  false,
				Ahead:  3,
			},
			maxWidth: 50,
			want:     "main +3",
		},
		{
			name: "behind remote",
			git: &git.Info{
				Branch: "main",
				Dirty:  false,
				Behind: 2,
			},
			maxWidth: 50,
			want:     "main -2",
		},
		{
			name: "full status",
			git: &git.Info{
				Branch: "feature/auth",
				Dirty:  true,
				Ahead:  3,
				Behind: 1,
				PRNum:  42,
			},
			maxWidth: 50,
			want:     "feature/auth ● +3 -1 [PR#42]",
		},
		{
			name: "long branch truncated",
			git: &git.Info{
				Branch: "feature/very-long-branch-name-that-exceeds-limit",
				Dirty:  false,
			},
			maxWidth: 50,
			want:     "feature/very-long-branch-na...", // git.MaxBranchLength = 30
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := git.FormatStatusLine(tt.git, tt.maxWidth)
			if got != tt.want {
				t.Errorf("git.FormatStatusLine() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGitInfoIsStale(t *testing.T) {
	tests := []struct {
		name      string
		git       *git.Info
		wantStale bool
	}{
		{
			name:      "nil git info is stale",
			git:       nil,
			wantStale: true,
		},
		{
			name:      "zero fetched at is stale",
			git:       &git.Info{FetchedAt: 0},
			wantStale: true,
		},
		{
			name:      "recent fetch is not stale",
			git:       &git.Info{FetchedAt: time.Now().Unix()},
			wantStale: false,
		},
		{
			name:      "old fetch is stale",
			git:       &git.Info{FetchedAt: time.Now().Add(-20 * time.Second).Unix()},
			wantStale: true,
		},
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

func TestGitHubInfo(t *testing.T) {
	gh := &git.GitHubInfo{Owner: "user", Repo: "repo"}

	tests := []struct {
		name string
		fn   func() string
		want string
	}{
		{
			name: "base URL",
			fn:   func() string { return gh.GitHubURL("") },
			want: "https://github.com/user/repo",
		},
		{
			name: "issue URL",
			fn:   func() string { return gh.IssueURL(123) },
			want: "https://github.com/user/repo/issues/123",
		},
		{
			name: "PR URL",
			fn:   func() string { return gh.PRURL(42) },
			want: "https://github.com/user/repo/pull/42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}

	// Test nil git.GitHubInfo
	var nilGH *git.GitHubInfo
	if nilGH.GitHubURL("/test") != "" {
		t.Error("nil git.GitHubInfo should return empty URL")
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
		// HTTPS URLs
		{
			name:      "HTTPS with .git",
			remoteURL: "https://github.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "HTTPS without .git",
			remoteURL: "https://github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "HTTPS with trailing slash",
			remoteURL: "https://github.com/owner/repo/",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "HTTP URL",
			remoteURL: "http://github.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		// SSH URLs
		{
			name:      "SSH with .git",
			remoteURL: "git@github.com:owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "SSH without .git",
			remoteURL: "git@github.com:owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		// SSH protocol URLs
		{
			name:      "SSH protocol with .git",
			remoteURL: "ssh://git@github.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "SSH protocol without .git",
			remoteURL: "ssh://git@github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		// Real-world examples
		{
			name:      "real HTTPS example",
			remoteURL: "https://github.com/stwalsh4118/navi.git",
			wantOwner: "stwalsh4118",
			wantRepo:  "navi",
		},
		{
			name:      "real SSH example",
			remoteURL: "git@github.com:anthropics/claude-code.git",
			wantOwner: "anthropics",
			wantRepo:  "claude-code",
		},
		// Non-GitHub remotes
		{
			name:      "empty string",
			remoteURL: "",
			wantNil:   true,
		},
		{
			name:      "GitLab HTTPS",
			remoteURL: "https://gitlab.com/owner/repo.git",
			wantNil:   true,
		},
		{
			name:      "GitLab SSH",
			remoteURL: "git@gitlab.com:owner/repo.git",
			wantNil:   true,
		},
		{
			name:      "Bitbucket HTTPS",
			remoteURL: "https://bitbucket.org/owner/repo.git",
			wantNil:   true,
		},
		{
			name:      "self-hosted GitHub Enterprise",
			remoteURL: "https://github.example.com/owner/repo.git",
			wantNil:   true,
		},
		{
			name:      "invalid URL",
			remoteURL: "not-a-valid-url",
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := git.ParseGitHubRemote(tt.remoteURL)
			if tt.wantNil {
				if got != nil {
					t.Errorf("git.ParseGitHubRemote(%q) = %+v, want nil", tt.remoteURL, got)
				}
				return
			}

			if got == nil {
				t.Fatalf("git.ParseGitHubRemote(%q) = nil, want non-nil", tt.remoteURL)
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

func TestIsGitRepo(t *testing.T) {
	// Current directory should be a git repo (navi project)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	if !git.IsRepo(cwd) {
		t.Error("Expected current directory to be a git repo")
	}

	// Root directory is typically not a git repo
	if git.IsRepo("/") {
		t.Error("Expected root directory to not be a git repo")
	}

	// Non-existent directory
	if git.IsRepo("/nonexistent/path/that/does/not/exist") {
		t.Error("Expected non-existent directory to not be a git repo")
	}
}

func TestGetGitBranch(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Current directory should have a branch
	branch := git.GetBranch(cwd)
	if branch == "" {
		t.Error("Expected to get a branch name for current directory")
	}

	// Non-git directory should return empty
	branch = git.GetBranch("/")
	if branch != "" {
		t.Errorf("Expected empty branch for non-git directory, got %q", branch)
	}
}

func TestIsGitDirty(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Just verify it doesn't error - actual dirty state depends on repo
	_ = git.IsDirty(cwd)

	// Non-git directory should return false
	if git.IsDirty("/") {
		t.Error("Expected non-git directory to not be dirty")
	}
}

func TestGetGitLastCommit(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	commit := git.GetLastCommit(cwd)
	if commit == "" {
		t.Error("Expected to get last commit for current directory")
	}

	// Non-git directory should return empty
	commit = git.GetLastCommit("/")
	if commit != "" {
		t.Errorf("Expected empty last commit for non-git directory, got %q", commit)
	}
}

func TestGetGitRemote(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Current directory should have a remote (navi project)
	remote := git.GetRemote(cwd)
	if remote == "" {
		t.Error("Expected to get remote URL for current directory")
	}

	// Non-git directory should return empty
	remote = git.GetRemote("/")
	if remote != "" {
		t.Errorf("Expected empty remote for non-git directory, got %q", remote)
	}
}

func TestGetGitInfo(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Current directory should return valid git.Info
	info := git.GetInfo(cwd)
	if info == nil {
		t.Fatal("Expected git.Info for current directory, got nil")
	}

	if info.Branch == "" {
		t.Error("Expected Branch to be set")
	}
	if info.FetchedAt == 0 {
		t.Error("Expected FetchedAt to be set")
	}

	// Non-git directory should return nil
	info = git.GetInfo("/")
	if info != nil {
		t.Error("Expected nil git.Info for non-git directory")
	}

	// Non-existent directory should return nil
	info = git.GetInfo("/nonexistent/path")
	if info != nil {
		t.Error("Expected nil git.Info for non-existent directory")
	}
}

func TestGetGitInfoWithTilde(t *testing.T) {
	// Test that tilde expansion works
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Could not get home directory")
	}

	// Find a git repo in home dir to test with (skip if not found)
	testPath := filepath.Join(home, ".config")
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Skip("~/.config does not exist")
	}

	// Just verify it doesn't panic with tilde path
	_ = git.GetInfo("~/.config")
}

func TestGetGitPRNumber(t *testing.T) {
	t.Run("returns 0 for non-git directory", func(t *testing.T) {
		prNum := git.GetPRNumber("/")
		if prNum != 0 {
			t.Errorf("git.GetPRNumber(/) = %d, want 0", prNum)
		}
	})

	t.Run("returns 0 when gh CLI not available or no PR", func(t *testing.T) {
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get cwd: %v", err)
		}

		// This will return 0 if no PR exists for current branch or gh not installed
		// We just verify it doesn't panic
		prNum := git.GetPRNumber(cwd)
		// prNum could be 0 or a real PR number - both are valid
		_ = prNum
	})
}

func TestGitCacheIntegration(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Test pollGitInfoCmd with current directory
	sessions := []session.Info{
		{TmuxSession: "test1", CWD: cwd},
		{TmuxSession: "test2", CWD: "/"},            // non-git
		{TmuxSession: "test3", CWD: "/nonexistent"}, // non-existent
	}

	msg := pollGitInfoCmd(sessions)()

	gitMsg, ok := msg.(gitInfoMsg)
	if !ok {
		t.Fatalf("Expected gitInfoMsg, got %T", msg)
	}

	// Current directory should be in cache
	if gitMsg.cache[cwd] == nil {
		t.Error("Expected current directory to be in cache")
	}

	// Non-git directories should not be in cache
	if gitMsg.cache["/"] != nil {
		t.Error("Expected root directory to not be in cache")
	}

	// Non-existent directories should not be in cache
	if gitMsg.cache["/nonexistent"] != nil {
		t.Error("Expected non-existent directory to not be in cache")
	}
}

func TestGetGitDiffStat(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Just verify it doesn't error on a git repo
	// Actual output depends on repo state
	_ = git.GetDiffStat(cwd)

	// Non-git directory should return empty
	diffStat := git.GetDiffStat("/")
	if diffStat != "" {
		t.Errorf("Expected empty diff stat for non-git directory, got %q", diffStat)
	}
}

func TestGetGitDiff(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Just verify it doesn't error on a git repo
	// Actual output depends on repo state
	_ = git.GetDiff(cwd)

	// Non-git directory should return empty
	diff := git.GetDiff("/")
	if diff != "" {
		t.Errorf("Expected empty diff for non-git directory, got %q", diff)
	}
}

func TestGitDiffConstants(t *testing.T) {
	// Verify constants are set to reasonable values
	if git.DiffMaxLines <= 0 {
		t.Errorf("git.DiffMaxLines should be positive, got %d", git.DiffMaxLines)
	}

	if git.DiffTruncatedMsg == "" {
		t.Error("git.DiffTruncatedMsg should not be empty")
	}
}

func TestOpenURL(t *testing.T) {
	t.Run("empty URL returns error", func(t *testing.T) {
		err := git.OpenURL("")
		if err == nil {
			t.Error("expected error for empty URL")
		}
		if !strings.Contains(err.Error(), "empty URL") {
			t.Errorf("expected error message to mention 'empty URL', got: %v", err)
		}
	})

}
