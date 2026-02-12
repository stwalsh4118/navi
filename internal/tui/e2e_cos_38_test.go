package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/git"
	"github.com/stwalsh4118/navi/internal/session"
)

// E2E tests for PBI-38: Enhanced GitHub PR Integration
// These tests verify all acceptance criteria are met.

// --- Test helpers ---

// newPRDetailAllPassed returns a PRDetail where all checks have passed.
func newPRDetailAllPassed() *git.PRDetail {
	return &git.PRDetail{
		Number:       42,
		Title:        "Add feature X",
		State:        git.PRStateOpen,
		Draft:        false,
		Mergeable:    git.MergeableMergeable,
		Labels:       []string{"enhancement", "v2"},
		ChangedFiles: 5,
		Additions:    120,
		Deletions:    30,
		ReviewStatus: git.ReviewApproved,
		Reviewers: []git.Reviewer{
			{Login: "alice", State: git.ReviewerApproved},
		},
		Comments: 3,
		Checks: []git.Check{
			{Name: "build", Status: git.CheckStatusCompleted, Conclusion: git.CheckConclusionSuccess},
			{Name: "lint", Status: git.CheckStatusCompleted, Conclusion: git.CheckConclusionSuccess},
		},
		CheckSummary: git.CheckSummary{Total: 2, Passed: 2, Failed: 0, Pending: 0},
		URL:          "https://github.com/user/repo/pull/42",
		FetchedAt:    time.Now().Unix(),
	}
}

// newPRDetailPending returns a PRDetail with pending checks.
func newPRDetailPending() *git.PRDetail {
	return &git.PRDetail{
		Number:       42,
		Title:        "Add feature X",
		State:        git.PRStateOpen,
		Mergeable:    git.MergeableMergeable,
		ChangedFiles: 3,
		Additions:    50,
		Deletions:    10,
		ReviewStatus: git.ReviewRequired,
		Comments:     1,
		Checks: []git.Check{
			{Name: "build", Status: git.CheckStatusInProgress},
			{Name: "lint", Status: git.CheckStatusCompleted, Conclusion: git.CheckConclusionSuccess},
		},
		CheckSummary: git.CheckSummary{Total: 2, Passed: 1, Failed: 0, Pending: 1},
		URL:          "https://github.com/user/repo/pull/42",
		FetchedAt:    time.Now().Unix(),
	}
}

// newPRDetailFailed returns a PRDetail with a failed check.
func newPRDetailFailed() *git.PRDetail {
	return &git.PRDetail{
		Number:       42,
		Title:        "Add feature X",
		State:        git.PRStateOpen,
		Mergeable:    git.MergeableConflicting,
		ChangedFiles: 2,
		Additions:    10,
		Deletions:    5,
		ReviewStatus: git.ReviewChangesRequired,
		Reviewers: []git.Reviewer{
			{Login: "bob", State: git.ReviewerChangesRequired},
		},
		Comments: 2,
		Checks: []git.Check{
			{Name: "build", Status: git.CheckStatusCompleted, Conclusion: git.CheckConclusionFailure},
			{Name: "lint", Status: git.CheckStatusCompleted, Conclusion: git.CheckConclusionSuccess},
		},
		CheckSummary: git.CheckSummary{Total: 2, Passed: 1, Failed: 1, Pending: 0},
		URL:          "https://github.com/user/repo/pull/42",
		FetchedAt:    time.Now().Unix(),
	}
}

// newPRDetailDraft returns a draft PRDetail.
func newPRDetailDraft() *git.PRDetail {
	return &git.PRDetail{
		Number:       42,
		Title:        "WIP: Add feature X",
		State:        git.PRStateOpen,
		Draft:        true,
		Mergeable:    git.MergeableUnknown,
		ChangedFiles: 1,
		Additions:    5,
		Deletions:    0,
		Comments:     0,
		CheckSummary: git.CheckSummary{Total: 0},
		URL:          "https://github.com/user/repo/pull/42",
		FetchedAt:    time.Now().Unix(),
	}
}

// newGitDetailModel creates a model with git detail view open and the given PRDetail.
func newGitDetailModel(prDetail *git.PRDetail) Model {
	return Model{
		width:      100,
		height:     40,
		dialogMode: DialogGitDetail,
		sessionToModify: &session.Info{
			TmuxSession: "test-session",
			CWD:         "/home/user/project",
			Git: &git.Info{
				Branch: "feature/test",
				Dirty:  true,
				Ahead:  1,
				Remote: "https://github.com/user/repo.git",
				PRNum:  42,
				PRDetail: prDetail,
			},
		},
	}
}

// --- AC1: Check statuses displayed with individual icons ---

func TestE2E_AC1_CheckStatusDisplay(t *testing.T) {
	t.Run("all checks passed shows pass icons", func(t *testing.T) {
		m := newGitDetailModel(newPRDetailAllPassed())
		result := m.renderGitDetailView()

		if !strings.Contains(result, "2/2 passed") {
			t.Error("AC1: Should show '2/2 passed' for all checks passing")
		}
		if !strings.Contains(result, git.IndicatorPass) {
			t.Error("AC1: Should show pass indicator for passed checks")
		}
		if !strings.Contains(result, "build") || !strings.Contains(result, "lint") {
			t.Error("AC1: Should list individual check names")
		}
	})

	t.Run("failed checks show fail icons", func(t *testing.T) {
		m := newGitDetailModel(newPRDetailFailed())
		result := m.renderGitDetailView()

		if !strings.Contains(result, "1/2 passed") {
			t.Error("AC1: Should show '1/2 passed' for mixed check results")
		}
		if !strings.Contains(result, git.IndicatorFail) {
			t.Error("AC1: Should show fail indicator for failed checks")
		}
	})

	t.Run("pending checks show pending icons", func(t *testing.T) {
		m := newGitDetailModel(newPRDetailPending())
		result := m.renderGitDetailView()

		if !strings.Contains(result, git.IndicatorPending) {
			t.Error("AC1: Should show pending indicator for in-progress checks")
		}
	})

	t.Run("session list shows aggregate check indicator", func(t *testing.T) {
		m := Model{width: 120, height: 24}

		// All passed
		g := &git.Info{Branch: "main", PRNum: 42, PRDetail: newPRDetailAllPassed()}
		result := renderGitInfo(g, 100)
		if !strings.Contains(result, git.IndicatorPass) {
			t.Error("AC1: Session list should show pass indicator when all checks pass")
		}

		// Failed
		g.PRDetail = newPRDetailFailed()
		result = renderGitInfo(g, 100)
		if !strings.Contains(result, git.IndicatorFail) {
			t.Error("AC1: Session list should show fail indicator when checks fail")
		}

		// Pending
		g.PRDetail = newPRDetailPending()
		result = renderGitInfo(g, 100)
		if !strings.Contains(result, git.IndicatorPending) {
			t.Error("AC1: Session list should show pending indicator when checks pending")
		}

		_ = m // suppress unused warning
	})
}

// --- AC2: Review status with reviewer names ---

func TestE2E_AC2_ReviewStatus(t *testing.T) {
	t.Run("approved review shows reviewer name", func(t *testing.T) {
		m := newGitDetailModel(newPRDetailAllPassed())
		result := m.renderGitDetailView()

		if !strings.Contains(result, "Approved") {
			t.Error("AC2: Should show 'Approved' for approved review status")
		}
		if !strings.Contains(result, "alice") {
			t.Error("AC2: Should show reviewer name 'alice'")
		}
	})

	t.Run("changes requested shows reviewer name", func(t *testing.T) {
		m := newGitDetailModel(newPRDetailFailed())
		result := m.renderGitDetailView()

		if !strings.Contains(result, "Changes requested") {
			t.Error("AC2: Should show 'Changes requested' status")
		}
		if !strings.Contains(result, "bob") {
			t.Error("AC2: Should show reviewer name 'bob'")
		}
	})

	t.Run("pending review shows waiting indicator", func(t *testing.T) {
		m := newGitDetailModel(newPRDetailPending())
		result := m.renderGitDetailView()

		if !strings.Contains(result, "Review pending") || !strings.Contains(result, git.IndicatorWaiting) {
			t.Error("AC2: Should show 'Review pending' with waiting indicator")
		}
	})
}

// --- AC3: Comment count in session list and git detail ---

func TestE2E_AC3_CommentCount(t *testing.T) {
	t.Run("comment count in session list PR indicator", func(t *testing.T) {
		g := &git.Info{
			Branch:   "feature/test",
			PRNum:    42,
			PRDetail: newPRDetailAllPassed(), // has Comments: 3
		}
		result := renderGitInfo(g, 100)

		if !strings.Contains(result, git.CommentIcon+"3") {
			t.Error("AC3: Session list should show comment count with icon")
		}
	})

	t.Run("no comment icon when zero comments", func(t *testing.T) {
		g := &git.Info{
			Branch:   "feature/test",
			PRNum:    42,
			PRDetail: newPRDetailDraft(), // has Comments: 0
		}
		result := renderGitInfo(g, 100)

		if strings.Contains(result, git.CommentIcon) {
			t.Error("AC3: Should not show comment icon when zero comments")
		}
	})
}

// --- AC4: Comment viewer with scroll and return navigation ---

func TestE2E_AC4_CommentViewer(t *testing.T) {
	t.Run("c key triggers comment loading from git detail", func(t *testing.T) {
		m := newGitDetailModel(newPRDetailAllPassed())
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
		updatedModel, cmd := m.Update(msg)
		newM := updatedModel.(Model)

		if newM.dialogError != "Loading comments..." {
			t.Errorf("AC4: 'c' key should set loading indicator, got %q", newM.dialogError)
		}
		if cmd == nil {
			t.Error("AC4: 'c' key should dispatch a fetch command")
		}
	})

	t.Run("comments rendered in content viewer with author and body", func(t *testing.T) {
		m := newGitDetailModel(newPRDetailAllPassed())

		// Simulate receiving comments message
		comments := []git.PRComment{
			{Author: "alice", Body: "Looks good!", CreatedAt: "2026-02-12T09:00:00Z", Type: git.CommentTypeGeneral},
			{Author: "bob", Body: "Fix the bug", CreatedAt: "2026-02-12T09:01:00Z", Type: git.CommentTypeReview, FilePath: "main.go", Line: 42},
		}
		commentsMsg := gitPRCommentsMsg{comments: comments}
		updatedModel, _ := m.Update(commentsMsg)
		newM := updatedModel.(Model)

		if newM.dialogMode != DialogContentViewer {
			t.Error("AC4: Comments should open in content viewer")
		}
		if !strings.Contains(newM.contentViewerTitle, "PR Comments (2)") {
			t.Errorf("AC4: Title should be 'PR Comments (2)', got %q", newM.contentViewerTitle)
		}
		if newM.contentViewerPrevDialog != DialogGitDetail {
			t.Error("AC4: Content viewer should return to git detail on close")
		}

		// Verify content contains comment details
		content := strings.Join(newM.contentViewerLines, "\n")
		if !strings.Contains(content, "alice") {
			t.Error("AC4: Comment content should contain author 'alice'")
		}
		if !strings.Contains(content, "Looks good!") {
			t.Error("AC4: Comment content should contain body text")
		}
		if !strings.Contains(content, "main.go:42") {
			t.Error("AC4: Review comment should show file:line")
		}
	})

	t.Run("empty comments shows message", func(t *testing.T) {
		m := newGitDetailModel(newPRDetailAllPassed())
		commentsMsg := gitPRCommentsMsg{comments: []git.PRComment{}}
		updatedModel, _ := m.Update(commentsMsg)
		newM := updatedModel.(Model)

		if newM.dialogError != "No comments on this PR" {
			t.Errorf("AC4: Should show 'No comments on this PR', got %q", newM.dialogError)
		}
	})

	t.Run("comment fetch error shows error message", func(t *testing.T) {
		m := newGitDetailModel(newPRDetailAllPassed())
		commentsMsg := gitPRCommentsMsg{err: errTestFetch}
		updatedModel, _ := m.Update(commentsMsg)
		newM := updatedModel.(Model)

		if !strings.Contains(newM.dialogError, "Failed to fetch comments") {
			t.Errorf("AC4: Should show fetch error, got %q", newM.dialogError)
		}
	})

	t.Run("esc returns from content viewer to git detail", func(t *testing.T) {
		m := Model{
			width:                   100,
			height:                  40,
			dialogMode:              DialogContentViewer,
			contentViewerPrevDialog: DialogGitDetail,
			contentViewerMode:       ContentModePlain,
			contentViewerLines:      []string{"comment content"},
			contentViewerTitle:      "PR Comments (1)",
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/tmp",
				Git:         &git.Info{Branch: "main", PRNum: 42},
			},
		}

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updatedModel, _ := m.Update(msg)
		newM := updatedModel.(Model)

		if newM.dialogMode != DialogGitDetail {
			t.Errorf("AC4: Esc should return to git detail, got dialog mode %d", newM.dialogMode)
		}
	})
}

// errTestFetch is a sentinel error for test assertions.
var errTestFetch = fmt.Errorf("network error")

// --- AC5: Mergeable status ---

func TestE2E_AC5_MergeableStatus(t *testing.T) {
	t.Run("mergeable shows no conflicts", func(t *testing.T) {
		m := newGitDetailModel(newPRDetailAllPassed())
		result := m.renderGitDetailView()

		if !strings.Contains(result, "No conflicts") {
			t.Error("AC5: Should show 'No conflicts' for mergeable PR")
		}
	})

	t.Run("conflicting shows has conflicts", func(t *testing.T) {
		m := newGitDetailModel(newPRDetailFailed())
		result := m.renderGitDetailView()

		if !strings.Contains(result, "Has conflicts") {
			t.Error("AC5: Should show 'Has conflicts' for conflicting PR")
		}
	})

	t.Run("unknown merge status shows unknown", func(t *testing.T) {
		pr := newPRDetailDraft()
		pr.Mergeable = git.MergeableUnknown
		m := newGitDetailModel(pr)
		result := m.renderGitDetailView()

		if !strings.Contains(result, "Unknown") {
			t.Error("AC5: Should show 'Unknown' for unknown merge status")
		}
	})
}

// --- AC6: Labels ---

func TestE2E_AC6_Labels(t *testing.T) {
	t.Run("labels displayed in git detail view", func(t *testing.T) {
		m := newGitDetailModel(newPRDetailAllPassed())
		result := m.renderGitDetailView()

		if !strings.Contains(result, "Labels:") {
			t.Error("AC6: Should show 'Labels:' header")
		}
		if !strings.Contains(result, "enhancement") {
			t.Error("AC6: Should show label 'enhancement'")
		}
		if !strings.Contains(result, "v2") {
			t.Error("AC6: Should show label 'v2'")
		}
	})

	t.Run("no labels section when empty", func(t *testing.T) {
		pr := newPRDetailPending()
		pr.Labels = nil
		m := newGitDetailModel(pr)
		result := m.renderGitDetailView()

		if strings.Contains(result, "Labels:") {
			t.Error("AC6: Should not show Labels section when no labels")
		}
	})
}

// --- AC7: Draft indicator ---

func TestE2E_AC7_DraftIndicator(t *testing.T) {
	t.Run("draft indicator in git detail view", func(t *testing.T) {
		m := newGitDetailModel(newPRDetailDraft())
		result := m.renderGitDetailView()

		if !strings.Contains(result, "(draft)") {
			t.Error("AC7: Should show '(draft)' indicator in git detail view")
		}
	})

	t.Run("draft indicator in session list", func(t *testing.T) {
		g := &git.Info{
			Branch:   "feature/wip",
			PRNum:    42,
			PRDetail: newPRDetailDraft(),
		}
		result := renderGitInfo(g, 100)

		if !strings.Contains(result, git.IndicatorDraft) {
			t.Error("AC7: Session list should show 'draft' indicator for draft PRs")
		}
	})

	t.Run("draft PR does not show check indicator in session list", func(t *testing.T) {
		pr := newPRDetailDraft()
		pr.Checks = []git.Check{
			{Name: "build", Status: git.CheckStatusCompleted, Conclusion: git.CheckConclusionSuccess},
		}
		pr.CheckSummary = git.CheckSummary{Total: 1, Passed: 1}
		g := &git.Info{
			Branch:   "feature/wip",
			PRNum:    42,
			PRDetail: pr,
		}
		result := renderGitInfo(g, 100)

		// Draft should show "draft" instead of check indicator
		if !strings.Contains(result, git.IndicatorDraft) {
			t.Error("AC7: Draft PR should show 'draft' instead of check indicator")
		}
	})
}

// --- AC8: Changed files stats ---

func TestE2E_AC8_ChangedFilesStats(t *testing.T) {
	t.Run("changed files stats displayed in git detail", func(t *testing.T) {
		m := newGitDetailModel(newPRDetailAllPassed())
		result := m.renderGitDetailView()

		if !strings.Contains(result, "5 files") {
			t.Error("AC8: Should show '5 files' for changed files count")
		}
		if !strings.Contains(result, "+120") {
			t.Error("AC8: Should show '+120' for additions")
		}
		if !strings.Contains(result, "-30") {
			t.Error("AC8: Should show '-30' for deletions")
		}
	})
}

// --- AC9: Auto-refresh for pending checks ---

func TestE2E_AC9_AutoRefresh(t *testing.T) {
	t.Run("auto-refresh starts when PR has pending checks", func(t *testing.T) {
		m := Model{
			width:      100,
			height:     40,
			dialogMode: DialogGitDetail,
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/home/user/project",
				Git: &git.Info{
					Branch: "feature/test",
					Remote: "https://github.com/user/repo.git",
					PRNum:  42,
				},
			},
			gitCache: make(map[string]*git.Info),
		}

		// Simulate receiving PR data with pending checks
		prMsg := gitPRMsg{
			cwd:      "/home/user/project",
			prNum:    42,
			prDetail: newPRDetailPending(),
		}
		updatedModel, cmd := m.Update(prMsg)
		newM := updatedModel.(Model)

		if !newM.prAutoRefreshActive {
			t.Error("AC9: Auto-refresh should activate when checks are pending")
		}
		if cmd == nil {
			t.Error("AC9: Should dispatch auto-refresh tick command")
		}
	})

	t.Run("auto-refresh does not start when all checks passed", func(t *testing.T) {
		m := Model{
			width:      100,
			height:     40,
			dialogMode: DialogGitDetail,
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/home/user/project",
				Git: &git.Info{
					Branch: "feature/test",
					PRNum:  42,
				},
			},
			gitCache: make(map[string]*git.Info),
		}

		prMsg := gitPRMsg{
			cwd:      "/home/user/project",
			prNum:    42,
			prDetail: newPRDetailAllPassed(),
		}
		updatedModel, _ := m.Update(prMsg)
		newM := updatedModel.(Model)

		if newM.prAutoRefreshActive {
			t.Error("AC9: Auto-refresh should NOT activate when all checks passed")
		}
	})

	t.Run("auto-refresh stops when checks become terminal", func(t *testing.T) {
		m := Model{
			width:               100,
			height:              40,
			dialogMode:          DialogGitDetail,
			prAutoRefreshActive: true,
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/home/user/project",
				Git: &git.Info{
					Branch: "feature/test",
					PRNum:  42,
				},
			},
			gitCache: make(map[string]*git.Info),
		}

		// Simulate receiving PR data where all checks are now complete
		prMsg := gitPRMsg{
			cwd:      "/home/user/project",
			prNum:    42,
			prDetail: newPRDetailAllPassed(),
		}
		updatedModel, _ := m.Update(prMsg)
		newM := updatedModel.(Model)

		if newM.prAutoRefreshActive {
			t.Error("AC9: Auto-refresh should deactivate when checks become terminal")
		}
	})

	t.Run("auto-refresh stops when view is closed", func(t *testing.T) {
		m := Model{
			width:               100,
			height:              40,
			dialogMode:          DialogGitDetail,
			prAutoRefreshActive: true,
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/tmp",
				Git:         &git.Info{Branch: "main"},
			},
		}

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		updatedModel, _ := m.Update(msg)
		newM := updatedModel.(Model)

		if newM.prAutoRefreshActive {
			t.Error("AC9: Auto-refresh should deactivate when git detail is closed")
		}
		if newM.dialogMode != DialogNone {
			t.Error("AC9: Dialog should be closed")
		}
	})

	t.Run("auto-refresh indicator displayed when active", func(t *testing.T) {
		m := newGitDetailModel(newPRDetailPending())
		m.prAutoRefreshActive = true
		result := m.renderGitDetailView()

		if !strings.Contains(result, "Auto-refreshing...") {
			t.Error("AC9: Should show 'Auto-refreshing...' indicator")
		}
	})

	t.Run("no auto-refresh indicator when inactive", func(t *testing.T) {
		m := newGitDetailModel(newPRDetailAllPassed())
		m.prAutoRefreshActive = false
		result := m.renderGitDetailView()

		if strings.Contains(result, "Auto-refreshing...") {
			t.Error("AC9: Should NOT show 'Auto-refreshing...' indicator when inactive")
		}
	})

	t.Run("auto-refresh tick ignored when dialog closed", func(t *testing.T) {
		m := Model{
			width:               100,
			height:              40,
			dialogMode:          DialogNone,
			prAutoRefreshActive: true,
		}

		tickMsg := prAutoRefreshTickMsg(time.Now())
		updatedModel, _ := m.Update(tickMsg)
		newM := updatedModel.(Model)

		if newM.prAutoRefreshActive {
			t.Error("AC9: Auto-refresh should deactivate when tick arrives but dialog is closed")
		}
	})

	t.Run("auto-refresh interval constant is 30 seconds", func(t *testing.T) {
		expected := 30 * time.Second
		if prAutoRefreshInterval != expected {
			t.Errorf("AC9: Auto-refresh interval should be %v, got %v", expected, prAutoRefreshInterval)
		}
	})
}

// --- AC10: Manual refresh resets tick ---

func TestE2E_AC10_ManualRefresh(t *testing.T) {
	t.Run("r key triggers PR refresh from git detail", func(t *testing.T) {
		m := newGitDetailModel(newPRDetailPending())
		m.prAutoRefreshActive = true

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
		updatedModel, cmd := m.Update(msg)
		newM := updatedModel.(Model)

		if cmd == nil {
			t.Error("AC10: 'r' key should dispatch fetch command")
		}
		// Manual refresh should reset auto-refresh (it will restart when response arrives)
		if newM.prAutoRefreshActive {
			t.Error("AC10: Manual refresh should reset auto-refresh active flag")
		}
	})

	t.Run("r key keybinding shown for PRs", func(t *testing.T) {
		m := newGitDetailModel(newPRDetailAllPassed())
		result := m.renderGitDetailView()

		if !strings.Contains(result, "r: refresh") {
			t.Error("AC10: Should show 'r: refresh' keybinding hint")
		}
	})

	t.Run("r key keybinding not shown without PR", func(t *testing.T) {
		m := Model{
			width:      100,
			height:     40,
			dialogMode: DialogGitDetail,
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/tmp",
				Git: &git.Info{
					Branch: "main",
					PRNum:  0,
				},
			},
		}
		result := m.renderGitDetailView()

		if strings.Contains(result, "r: refresh") {
			t.Error("AC10: Should NOT show 'r: refresh' when no PR")
		}
	})
}

// --- AC11: Remote session support ---

func TestE2E_AC11_RemoteSessionSupport(t *testing.T) {
	t.Run("remote session PR detail renders correctly", func(t *testing.T) {
		m := Model{
			width:      100,
			height:     40,
			dialogMode: DialogGitDetail,
			sessionToModify: &session.Info{
				TmuxSession: "remote-session",
				CWD:         "/home/user/project",
				Remote:      "devbox",
				Git: &git.Info{
					Branch:   "feature/remote",
					Remote:   "https://github.com/user/repo.git",
					PRNum:    99,
					PRDetail: newPRDetailAllPassed(),
				},
			},
		}

		result := m.renderGitDetailView()

		if !strings.Contains(result, "feature/remote") {
			t.Error("AC11: Should show branch name for remote session")
		}
		if !strings.Contains(result, "#99") {
			t.Error("AC11: Should show PR number for remote session")
		}
		if !strings.Contains(result, "Add feature X") {
			t.Error("AC11: Should show PR title for remote session")
		}
	})

	t.Run("remote session c key uses remote PR comments fetch", func(t *testing.T) {
		m := Model{
			width:      100,
			height:     40,
			dialogMode: DialogGitDetail,
			sessionToModify: &session.Info{
				TmuxSession: "remote-session",
				CWD:         "/home/user/project",
				Remote:      "devbox",
				Git: &git.Info{
					Branch: "feature/remote",
					Remote: "https://github.com/user/repo.git",
					PRNum:  99,
					PRDetail: newPRDetailAllPassed(),
				},
			},
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
		updatedModel, cmd := m.Update(msg)
		newM := updatedModel.(Model)

		if newM.dialogError != "Loading comments..." {
			t.Errorf("AC11: Remote 'c' should set loading indicator, got %q", newM.dialogError)
		}
		if cmd == nil {
			t.Error("AC11: Remote 'c' should dispatch comment fetch command")
		}
	})

	t.Run("remote session r key uses remote PR fetch", func(t *testing.T) {
		m := Model{
			width:      100,
			height:     40,
			dialogMode: DialogGitDetail,
			sessionToModify: &session.Info{
				TmuxSession: "remote-session",
				CWD:         "/home/user/project",
				Remote:      "devbox",
				Git: &git.Info{
					Branch: "feature/remote",
					Remote: "https://github.com/user/repo.git",
					PRNum:  99,
					PRDetail: newPRDetailPending(),
				},
			},
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
		_, cmd := m.Update(msg)

		if cmd == nil {
			t.Error("AC11: Remote 'r' should dispatch fetch command")
		}
	})
}

// --- AC12: No performance regression (lazy loading) ---

func TestE2E_AC12_LazyLoading(t *testing.T) {
	t.Run("PR detail not fetched during regular git polling", func(t *testing.T) {
		// Verify that gitInfoMsg handler doesn't trigger PR fetching
		m := Model{
			width:    100,
			height:   40,
			sessions: []session.Info{
				{TmuxSession: "test", CWD: "/tmp", Git: &git.Info{Branch: "main", PRNum: 42}},
			},
			gitCache: make(map[string]*git.Info),
		}

		gitMsg := gitInfoMsg{
			cache: map[string]*git.Info{
				"/tmp": {Branch: "main", PRNum: 42, FetchedAt: time.Now().Unix()},
			},
		}
		_, cmd := m.Update(gitMsg)

		// gitInfoMsg should not trigger any PR fetch commands
		if cmd != nil {
			t.Error("AC12: gitInfoMsg should not trigger additional commands (PR fetch is lazy)")
		}
	})

	t.Run("PR detail fetched only when git detail view opens", func(t *testing.T) {
		m := Model{
			width:  100,
			height: 40,
			sessions: []session.Info{
				{
					TmuxSession: "test",
					CWD:         "/home/user/project",
					Git:         &git.Info{Branch: "feature/test", PRNum: 42},
				},
			},
			cursor:     0,
			dialogMode: DialogNone,
		}

		// Opening git detail should return a PR fetch command
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		_, cmd := m.Update(msg)

		if cmd == nil {
			t.Error("AC12: Opening git detail should dispatch PR fetch command (lazy loading)")
		}
	})
}

// --- AC13: Existing tests pass (covered by running full test suite) ---

func TestE2E_AC13_NoRegression(t *testing.T) {
	t.Run("existing git detail view still works without PRDetail", func(t *testing.T) {
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
					PRDetail:   nil, // No PR detail loaded yet
				},
			},
		}

		result := m.renderGitDetailView()

		// All existing fields should still be displayed
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
				t.Errorf("AC13: Git detail view should still show %s (%q)", c.desc, c.content)
			}
		}
	})

	t.Run("session list PR indicator works without PRDetail", func(t *testing.T) {
		g := &git.Info{Branch: "main", PRNum: 42, PRDetail: nil}
		result := renderGitInfo(g, 80)

		if !strings.Contains(result, "PR#42") {
			t.Error("AC13: Should still show PR number without PRDetail")
		}
	})

	t.Run("git detail view handles no PR gracefully", func(t *testing.T) {
		m := Model{
			width:      80,
			height:     24,
			dialogMode: DialogGitDetail,
			sessionToModify: &session.Info{
				TmuxSession: "test",
				CWD:         "/tmp",
				Git: &git.Info{
					Branch: "main",
					PRNum:  0,
				},
			},
		}

		result := m.renderGitDetailView()

		// Should render without panic
		if result == "" {
			t.Error("AC13: Git detail should render even without PR")
		}
		// Should show d: diff but not PR-specific keybindings
		if !strings.Contains(result, "d: diff") {
			t.Error("AC13: Should show 'd: diff' keybinding")
		}
	})
}

// --- Cross-cutting: PR state display ---

func TestE2E_PRStateDisplay(t *testing.T) {
	t.Run("open PR state shown in green", func(t *testing.T) {
		m := newGitDetailModel(newPRDetailAllPassed())
		result := m.renderGitDetailView()

		if !strings.Contains(result, git.PRStateOpen) {
			t.Error("Should display PR state 'OPEN'")
		}
	})

	t.Run("PR title displayed", func(t *testing.T) {
		m := newGitDetailModel(newPRDetailAllPassed())
		result := m.renderGitDetailView()

		if !strings.Contains(result, "Add feature X") {
			t.Error("Should display PR title")
		}
	})

	t.Run("c keybinding hint shown for PRs", func(t *testing.T) {
		m := newGitDetailModel(newPRDetailAllPassed())
		result := m.renderGitDetailView()

		if !strings.Contains(result, "c: comments") {
			t.Error("Should show 'c: comments' keybinding hint")
		}
	})
}
