package pm

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stwalsh4118/navi/internal/git"
	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/task"
)

func TestDiscoverProjects(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sessions := []session.Info{
		{TmuxSession: "s1", CWD: "~/proj", Timestamp: 1},
		{TmuxSession: "s2", CWD: filepath.Join(home, "proj"), Timestamp: 2},
		{TmuxSession: "s3", CWD: filepath.Join(home, "other"), Timestamp: 3},
		{TmuxSession: "s4", CWD: "", Timestamp: 4},
	}

	projects := DiscoverProjects(sessions)
	if len(projects) != 2 {
		t.Fatalf("project count = %d, want 2", len(projects))
	}

	mainProject := filepath.Join(home, "proj")
	if len(projects[mainProject]) != 2 {
		t.Fatalf("sessions in %q = %d, want 2", mainProject, len(projects[mainProject]))
	}
}

func TestCaptureSnapshot(t *testing.T) {
	projectDir := t.TempDir()

	originalGitInfo := gitInfoFunc
	originalHeadSHA := getHeadSHAFunc
	t.Cleanup(func() {
		gitInfoFunc = originalGitInfo
		getHeadSHAFunc = originalHeadSHA
	})

	gitInfoFunc = func(dir string) *git.Info {
		if dir != projectDir {
			t.Fatalf("unexpected git dir %q", dir)
		}
		return &git.Info{Branch: "main", Ahead: 3, Dirty: true, PRNum: 14}
	}
	getHeadSHAFunc = func(dir string) string {
		if dir != projectDir {
			t.Fatalf("unexpected sha dir %q", dir)
		}
		return "0123456789abcdef0123456789abcdef01234567"
	}

	result := &task.ProviderResult{
		Groups: []task.TaskGroup{{
			ID:    "PBI-46",
			Title: "PM Engine",
			Tasks: []task.Task{
				{ID: "46-1", Status: "done"},
				{ID: "46-2", Status: "in_progress"},
				{ID: "46-3", Status: "todo"},
			},
		}},
	}

	sessions := []session.Info{
		{Status: session.StatusIdle, Timestamp: time.Now().Add(-time.Minute).Unix()},
		{Status: session.StatusWorking, Timestamp: time.Now().Unix()},
	}

	snapshot := CaptureSnapshot(projectDir, sessions, result)

	if snapshot.ProjectName != filepath.Base(projectDir) {
		t.Fatalf("project name = %q, want %q", snapshot.ProjectName, filepath.Base(projectDir))
	}
	if snapshot.Branch != "main" {
		t.Fatalf("branch = %q, want main", snapshot.Branch)
	}
	if snapshot.HeadSHA == "" {
		t.Fatal("expected non-empty head sha")
	}
	if snapshot.TaskCounts.Total != 3 || snapshot.TaskCounts.Done != 1 || snapshot.TaskCounts.InProgress != 1 {
		t.Fatalf("unexpected task counts: %+v", snapshot.TaskCounts)
	}
	if snapshot.CurrentPBIID != "PBI-46" || snapshot.CurrentPBITitle != "PM Engine" {
		t.Fatalf("unexpected pbi fields: id=%q title=%q", snapshot.CurrentPBIID, snapshot.CurrentPBITitle)
	}
	if snapshot.CurrentPBISource != "first_group_fallback" {
		t.Fatalf("current pbi source = %q, want %q", snapshot.CurrentPBISource, "first_group_fallback")
	}
	if snapshot.SessionStatus != session.StatusWorking {
		t.Fatalf("session status = %q, want %q", snapshot.SessionStatus, session.StatusWorking)
	}
	if snapshot.SessionCount != 2 {
		t.Fatalf("session count = %d, want 2", snapshot.SessionCount)
	}
}

func TestCaptureSnapshotGitFailureReturnsPartialSnapshot(t *testing.T) {
	originalGitInfo := gitInfoFunc
	originalHeadSHA := getHeadSHAFunc
	t.Cleanup(func() {
		gitInfoFunc = originalGitInfo
		getHeadSHAFunc = originalHeadSHA
	})

	gitInfoFunc = func(_ string) *git.Info { return nil }
	getHeadSHAFunc = func(_ string) string { return "" }

	snapshot := CaptureSnapshot("/does/not/exist", nil, nil)
	if snapshot.Branch != "" || snapshot.HeadSHA != "" {
		t.Fatalf("expected empty git fields, got branch=%q sha=%q", snapshot.Branch, snapshot.HeadSHA)
	}
}

func TestCaptureSnapshotBranchPatternResolution(t *testing.T) {
	projectDir := t.TempDir()

	originalGitInfo := gitInfoFunc
	originalHeadSHA := getHeadSHAFunc
	t.Cleanup(func() {
		gitInfoFunc = originalGitInfo
		getHeadSHAFunc = originalHeadSHA
	})

	gitInfoFunc = func(_ string) *git.Info {
		return &git.Info{Branch: "feature/pbi-46-pm-engine", Ahead: 0, Dirty: false, PRNum: 0}
	}
	getHeadSHAFunc = func(_ string) string { return "sha" }

	taskResult := &task.ProviderResult{
		Groups: []task.TaskGroup{
			{ID: "PBI-45", Title: "Old Work", Status: "Done"},
			{ID: "PBI-46", Title: "PM Engine", Status: "Done"},
		},
	}

	snapshot := CaptureSnapshot(projectDir, nil, taskResult)
	if snapshot.CurrentPBIID != "PBI-46" {
		t.Fatalf("current pbi id = %q, want %q", snapshot.CurrentPBIID, "PBI-46")
	}
	if snapshot.CurrentPBISource != "branch_pattern" {
		t.Fatalf("current pbi source = %q, want %q", snapshot.CurrentPBISource, "branch_pattern")
	}
}
