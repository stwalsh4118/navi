package tui

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/git"
	"github.com/stwalsh4118/navi/internal/session"
)

// E2E tests for PBI-16: Git Integration
// These tests verify all acceptance criteria are met

// TestE2E_GitInfoDisplayed verifies that branch name and git status
// are displayed for sessions in git repositories.
func TestE2E_GitInfoDisplayed(t *testing.T) {
	t.Run("session with git info shows branch in row", func(t *testing.T) {
		m := Model{width: 80, height: 24}
		s := session.Info{
			TmuxSession: "test-session",
			Status:      "working",
			CWD:         "/tmp",
			Timestamp:   time.Now().Unix(),
			Git: &git.Info{
				Branch: "feature/test-branch",
				Dirty:  true,
				Ahead:  2,
				Behind: 1,
			},
		}

		result := m.renderSession(s, false, 80)

		// Verify branch name is displayed
		if !strings.Contains(result, "feature/test-branch") {
			t.Error("CoS 1 failed: Branch name should be displayed in s row")
		}

		// Verify dirty indicator is shown
		if !strings.Contains(result, git.DirtyIndicator) {
			t.Error("CoS 2 failed: Dirty indicator should be shown")
		}

		// Verify ahead/behind counts are displayed
		if !strings.Contains(result, "+2") {
			t.Error("CoS 3 failed: Ahead count should be displayed")
		}
		if !strings.Contains(result, "-1") {
			t.Error("CoS 3 failed: Behind count should be displayed")
		}
	})

	t.Run("session without git info has no git line", func(t *testing.T) {
		m := Model{width: 80, height: 24}
		s := session.Info{
			TmuxSession: "test-session",
			Status:      "working",
			CWD:         "/tmp",
			Timestamp:   time.Now().Unix(),
			Git:         nil,
		}

		result := m.renderSession(s, false, 80)
		lines := strings.Split(result, "\n")

		// Non-git sessions should have 2 lines (name+status, cwd)
		if len(lines) != 2 {
			t.Errorf("CoS 9 failed: Non-git s should have 2 lines, got %d", len(lines))
		}
	})
}

// TestE2E_DirtyCleanStatus verifies the dirty/clean status indicator.
func TestE2E_DirtyCleanStatus(t *testing.T) {
	t.Run("dirty indicator shown when uncommitted changes", func(t *testing.T) {
		g := &git.Info{Branch: "main", Dirty: true}
		result := renderGitInfo(g, 80)

		if !strings.Contains(result, git.DirtyIndicator) {
			t.Error("CoS 2 failed: Dirty indicator should be shown for uncommitted changes")
		}
	})

	t.Run("no dirty indicator for clean repo", func(t *testing.T) {
		g := &git.Info{Branch: "main", Dirty: false}
		result := renderGitInfo(g, 80)

		if strings.Contains(result, git.DirtyIndicator) {
			t.Error("CoS 2 failed: Dirty indicator should NOT be shown for clean repo")
		}
	})
}

// TestE2E_PRDetection verifies GitHub PR detection via gh CLI.
func TestE2E_PRDetection(t *testing.T) {
	t.Run("PR number displayed in git info line when present", func(t *testing.T) {
		git := &git.Info{Branch: "feature/test", PRNum: 42}
		result := renderGitInfo(git, 80)

		if !strings.Contains(result, "[PR#42]") {
			t.Error("CoS 4 failed: PR number should be displayed in git info line")
		}
	})

	t.Run("no PR indicator when PRNum is 0", func(t *testing.T) {
		git := &git.Info{Branch: "main", PRNum: 0}
		result := renderGitInfo(git, 80)

		if strings.Contains(result, "PR#") {
			t.Error("CoS 4 failed: No PR indicator should show when PRNum is 0")
		}
	})

	t.Run("git.GetPRNumber returns 0 for non-git dir", func(t *testing.T) {
		prNum := git.GetPRNumber("/")
		if prNum != 0 {
			t.Errorf("CoS 4 failed: git.GetPRNumber(/) = %d, want 0", prNum)
		}
	})
}

// TestE2E_GitDetailView verifies the G keybinding and git detail view.
func TestE2E_GitDetailView(t *testing.T) {
	t.Run("G key opens git detail view", func(t *testing.T) {
		m := Model{
			width:  80,
			height: 24,
			sessions: []session.Info{
				{
					TmuxSession: "test",
					CWD:         "/tmp",
					Git:         &git.Info{Branch: "main"},
				},
			},
			cursor:     0,
			dialogMode: DialogNone,
		}

		// Simulate G key press
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		updatedModel, _ := m.Update(msg)
		newM := updatedModel.(Model)

		if newM.dialogMode != DialogGitDetail {
			t.Error("CoS 5 failed: G key should open git detail view")
		}
	})

	t.Run("git detail view shows all info", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogGitDetail,
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/home/user/project",
				Git: &git.Info{
					Branch:     "feature/auth",
					Dirty:      true,
					Ahead:      3,
					Behind:     1,
					LastCommit: "abc1234 Add login",
					Remote:     "https://github.com/user/repo.git",
					PRNum:      123,
				},
			},
		}

		result := m.renderGitDetailView()

		// Verify all info is displayed
		checks := []struct {
			content string
			desc    string
		}{
			{"feature/auth", "branch name"},
			{"uncommitted changes", "dirty status"},
			{"3 ahead", "ahead count"},
			{"1 behind", "behind count"},
			{"abc1234 Add login", "last commit"},
			{"github.com", "remote URL"},
			{"#123", "PR number"},
		}

		for _, c := range checks {
			if !strings.Contains(result, c.content) {
				t.Errorf("CoS 5 failed: Git detail view should show %s (%q)", c.desc, c.content)
			}
		}
	})

	t.Run("Esc closes git detail view", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogGitDetail,
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/tmp",
				Git:         &git.Info{Branch: "main"},
			},
		}

		// Simulate Esc key press
		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updatedModel, _ := m.Update(msg)
		newM := updatedModel.(Model)

		if newM.dialogMode != DialogNone {
			t.Error("CoS 5 failed: Esc should close git detail view")
		}
	})
}

// TestE2E_DiffPreview verifies the diff preview functionality.
func TestE2E_DiffPreview(t *testing.T) {
	t.Run("d key opens content viewer with diff mode from git detail", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogGitDetail,
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/tmp",
				Git:         &git.Info{Branch: "main", Dirty: true},
			},
		}

		// Simulate d key press
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
		updatedModel, _ := m.Update(msg)
		newM := updatedModel.(Model)

		if newM.dialogMode != DialogContentViewer {
			t.Errorf("d key should open content viewer, got dialog mode %d", newM.dialogMode)
		}
		if newM.contentViewerMode != ContentModeDiff {
			t.Error("content viewer should be in diff mode")
		}
		if !strings.Contains(newM.contentViewerTitle, "main") {
			t.Error("content viewer title should contain branch name")
		}
		if newM.contentViewerPrevDialog != DialogGitDetail {
			t.Error("content viewer should return to git detail on close")
		}
	})

	t.Run("diff content viewer shows branch in title", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogGitDetail,
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/",
				Git:         &git.Info{Branch: "feature/test", Dirty: false},
			},
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
		updatedModel, _ := m.Update(msg)
		newM := updatedModel.(Model)

		if !strings.Contains(newM.contentViewerTitle, "feature/test") {
			t.Error("diff content viewer title should show branch name")
		}
	})

	t.Run("Esc returns to git detail from diff content viewer", func(t *testing.T) {
		m := Model{
			width:                   80,
			height:                  24,
			dialogMode:              DialogContentViewer,
			contentViewerPrevDialog: DialogGitDetail,
			contentViewerMode:       ContentModeDiff,
			contentViewerLines:      []string{"diff content"},
			contentViewerTitle:      "Git Diff: main",
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/tmp",
				Git:         &git.Info{Branch: "main"},
			},
		}

		// Simulate Esc key press
		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updatedModel, _ := m.Update(msg)
		newM := updatedModel.(Model)

		if newM.dialogMode != DialogGitDetail {
			t.Errorf("Esc should return to git detail, got dialog mode %d", newM.dialogMode)
		}
	})
}

// TestE2E_PRLink verifies the PR link functionality.
func TestE2E_PRLink(t *testing.T) {
	t.Run("GitHub remote URL parsing", func(t *testing.T) {
		testCases := []struct {
			remote    string
			wantOwner string
			wantRepo  string
		}{
			{"https://github.com/user/repo.git", "user", "repo"},
			{"git@github.com:user/repo.git", "user", "repo"},
			{"ssh://git@github.com/user/repo.git", "user", "repo"},
		}

		for _, tc := range testCases {
			info := git.ParseGitHubRemote(tc.remote)
			if info == nil {
				t.Errorf("CoS 7 failed: Should parse %q", tc.remote)
				continue
			}
			if info.Owner != tc.wantOwner || info.Repo != tc.wantRepo {
				t.Errorf("CoS 7 failed: git.ParseGitHubRemote(%q) = (%q, %q), want (%q, %q)",
					tc.remote, info.Owner, info.Repo, tc.wantOwner, tc.wantRepo)
			}
		}
	})

	t.Run("PR URL constructed correctly", func(t *testing.T) {
		ghInfo := &git.GitHubInfo{Owner: "user", Repo: "project"}
		url := ghInfo.PRURL(123)
		expected := "https://github.com/user/project/pull/123"

		if url != expected {
			t.Errorf("CoS 7 failed: PRURL(123) = %q, want %q", url, expected)
		}
	})

	t.Run("PR link shown in detail view", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogGitDetail,
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/tmp",
				Git: &git.Info{
					Branch: "feature/test",
					PRNum:  123,
					Remote: "https://github.com/user/repo.git",
				},
			},
		}

		result := m.renderGitDetailView()

		if !strings.Contains(result, "github.com") {
			t.Error("CoS 7 failed: Git detail should show GitHub link")
		}
		if !strings.Contains(result, "o: open PR") {
			t.Error("CoS 7 failed: Git detail should show open PR keybinding")
		}
	})
}

// TestE2E_GitInfoCaching verifies the git info caching mechanism.
func TestE2E_GitInfoCaching(t *testing.T) {
	t.Run("git cache populated on git poll", func(t *testing.T) {
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get cwd: %v", err)
		}

		sessions := []session.Info{
			{TmuxSession: "test", CWD: cwd},
		}

		// Execute poll command
		cmd := pollGitInfoCmd(sessions)
		msg := cmd()

		gitMsg, ok := msg.(gitInfoMsg)
		if !ok {
			t.Fatal("CoS 8 failed: pollGitInfoCmd should return gitInfoMsg")
		}

		if gitMsg.cache[cwd] == nil {
			t.Error("CoS 8 failed: Git info should be cached for git repo")
		}
	})

	t.Run("stale cache entries detected", func(t *testing.T) {
		// Recent fetch should not be stale
		recent := &git.Info{FetchedAt: time.Now().Unix()}
		if recent.IsStale() {
			t.Error("CoS 8 failed: Recent fetch should not be stale")
		}

		// Old fetch should be stale
		old := &git.Info{FetchedAt: time.Now().Add(-20 * time.Second).Unix()}
		if !old.IsStale() {
			t.Error("CoS 8 failed: Old fetch should be stale")
		}
	})

	t.Run("git poll interval is longer than s poll", func(t *testing.T) {
		// Session poll is 500ms, git poll should be longer
		if git.PollInterval <= 500*time.Millisecond {
			t.Error("CoS 8 failed: Git poll interval should be longer than s poll")
		}
	})
}

// TestE2E_NonGitHandling verifies graceful handling of non-git directories.
func TestE2E_NonGitHandling(t *testing.T) {
	t.Run("non-git directory returns nil git info", func(t *testing.T) {
		info := git.GetInfo("/")
		if info != nil {
			t.Error("CoS 9 failed: Non-git directory should return nil git.Info")
		}
	})

	t.Run("non-existent directory returns nil", func(t *testing.T) {
		info := git.GetInfo("/nonexistent/path")
		if info != nil {
			t.Error("CoS 9 failed: Non-existent directory should return nil")
		}
	})

	t.Run("git detail view handles non-git gracefully", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogGitDetail,
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/",
				Git:         nil,
			},
		}

		result := m.renderGitDetailView()
		if !strings.Contains(result, "Not a git repository") {
			t.Error("CoS 9 failed: Git detail should show message for non-git dirs")
		}
	})
}

// TestE2E_GitFunctionality verifies the core git functions work.
func TestE2E_GitFunctionality(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get cwd: %v", err)
	}

	t.Run("git.IsRepo detects git repositories", func(t *testing.T) {
		if !git.IsRepo(cwd) {
			t.Error("Should detect navi project as git repo")
		}
		if git.IsRepo("/") {
			t.Error("Should not detect root as git repo")
		}
	})

	t.Run("git.GetBranch returns branch name", func(t *testing.T) {
		branch := git.GetBranch(cwd)
		if branch == "" {
			t.Error("Should get branch name for git repo")
		}
	})

	t.Run("git.GetInfo returns complete info", func(t *testing.T) {
		info := git.GetInfo(cwd)
		if info == nil {
			t.Fatal("Should get git.Info for git repo")
		}
		if info.Branch == "" {
			t.Error("git.Info should have branch")
		}
		if info.FetchedAt == 0 {
			t.Error("git.Info should have FetchedAt timestamp")
		}
	})
}
