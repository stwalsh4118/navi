package remote

import (
	"testing"
)

func TestParseGitOutput_NormalOutput(t *testing.T) {
	output := `BRANCH:feature/my-branch
DIRTY: M file.go
REMOTE:git@github.com:user/repo.git
LASTCOMMIT:abc1234 Add new feature
AHEADBEHIND:2	3`

	info := parseGitOutput(output)
	if info == nil {
		t.Fatal("expected non-nil info for normal output")
	}

	if info.Branch != "feature/my-branch" {
		t.Errorf("Branch = %q, want %q", info.Branch, "feature/my-branch")
	}
	if !info.Dirty {
		t.Error("Dirty = false, want true")
	}
	if info.Remote != "git@github.com:user/repo.git" {
		t.Errorf("Remote = %q, want %q", info.Remote, "git@github.com:user/repo.git")
	}
	if info.LastCommit != "abc1234 Add new feature" {
		t.Errorf("LastCommit = %q, want %q", info.LastCommit, "abc1234 Add new feature")
	}
	if info.Behind != 2 {
		t.Errorf("Behind = %d, want 2", info.Behind)
	}
	if info.Ahead != 3 {
		t.Errorf("Ahead = %d, want 3", info.Ahead)
	}
	if info.FetchedAt == 0 {
		t.Error("FetchedAt = 0, want non-zero timestamp")
	}
}

func TestParseGitOutput_NonGitDirectory(t *testing.T) {
	// Empty output means non-git directory
	info := parseGitOutput("")
	if info != nil {
		t.Errorf("expected nil for empty output, got %+v", info)
	}
}

func TestParseGitOutput_EmptyBranch(t *testing.T) {
	output := `BRANCH:
DIRTY:
REMOTE:
LASTCOMMIT:
AHEADBEHIND:`

	info := parseGitOutput(output)
	if info != nil {
		t.Errorf("expected nil for empty BRANCH, got %+v", info)
	}
}

func TestParseGitOutput_MissingUpstream(t *testing.T) {
	output := `BRANCH:main
DIRTY:
REMOTE:git@github.com:user/repo.git
LASTCOMMIT:def5678 Initial commit
AHEADBEHIND:`

	info := parseGitOutput(output)
	if info == nil {
		t.Fatal("expected non-nil info")
	}

	if info.Ahead != 0 {
		t.Errorf("Ahead = %d, want 0", info.Ahead)
	}
	if info.Behind != 0 {
		t.Errorf("Behind = %d, want 0", info.Behind)
	}
}

func TestParseGitOutput_DirtyRepo(t *testing.T) {
	output := `BRANCH:develop
DIRTY:?? untracked.txt
REMOTE:https://github.com/user/repo.git
LASTCOMMIT:aaa1111 Some commit
AHEADBEHIND:0	0`

	info := parseGitOutput(output)
	if info == nil {
		t.Fatal("expected non-nil info")
	}

	if !info.Dirty {
		t.Error("Dirty = false, want true (porcelain output present)")
	}
}

func TestParseGitOutput_CleanRepo(t *testing.T) {
	output := `BRANCH:main
DIRTY:
REMOTE:git@github.com:user/repo.git
LASTCOMMIT:bbb2222 Clean state
AHEADBEHIND:0	0`

	info := parseGitOutput(output)
	if info == nil {
		t.Fatal("expected non-nil info")
	}

	if info.Dirty {
		t.Error("Dirty = true, want false (empty porcelain)")
	}
}

func TestParseGitOutput_PartialOutput(t *testing.T) {
	// Only branch and dirty, no other labels
	output := `BRANCH:partial-branch
DIRTY:`

	info := parseGitOutput(output)
	if info == nil {
		t.Fatal("expected non-nil info for partial output with branch")
	}

	if info.Branch != "partial-branch" {
		t.Errorf("Branch = %q, want %q", info.Branch, "partial-branch")
	}
	if info.Remote != "" {
		t.Errorf("Remote = %q, want empty", info.Remote)
	}
	if info.LastCommit != "" {
		t.Errorf("LastCommit = %q, want empty", info.LastCommit)
	}
	if info.Ahead != 0 || info.Behind != 0 {
		t.Errorf("Ahead/Behind = %d/%d, want 0/0", info.Ahead, info.Behind)
	}
}

func TestParseAheadBehind(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantAhead   int
		wantBehind  int
	}{
		{"normal", "2\t3", 3, 2},
		{"zeros", "0\t0", 0, 0},
		{"empty", "", 0, 0},
		{"spaces only", "   ", 0, 0},
		{"single value", "5", 0, 0},
		{"non-numeric", "abc\tdef", 0, 0},
		{"large values", "100\t200", 200, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ahead, behind := parseAheadBehind(tt.input)
			if ahead != tt.wantAhead {
				t.Errorf("ahead = %d, want %d", ahead, tt.wantAhead)
			}
			if behind != tt.wantBehind {
				t.Errorf("behind = %d, want %d", behind, tt.wantBehind)
			}
		})
	}
}

func TestBuildGitCommand(t *testing.T) {
	cmd := buildGitCommand("/home/user/project")

	if !contains(cmd, `cd "/home/user/project"`) {
		t.Errorf("command should contain cd with quoted path, got: %s", cmd)
	}
	if !contains(cmd, "BRANCH:") {
		t.Errorf("command should contain BRANCH label, got: %s", cmd)
	}
	if !contains(cmd, "AHEADBEHIND:") {
		t.Errorf("command should contain AHEADBEHIND label, got: %s", cmd)
	}
}

func TestBuildGitCommand_PathWithSpaces(t *testing.T) {
	cmd := buildGitCommand("/home/user/my project")

	if !contains(cmd, `cd "/home/user/my project"`) {
		t.Errorf("command should handle path with spaces, got: %s", cmd)
	}
}

func TestBuildGitCommand_PathWithQuotes(t *testing.T) {
	cmd := buildGitCommand(`/home/user/project"name`)

	if !contains(cmd, `cd "/home/user/project\"name"`) {
		t.Errorf("command should escape quotes in path, got: %s", cmd)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
