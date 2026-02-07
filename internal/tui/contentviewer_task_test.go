package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/task"
)

// newTaskViewerTestModel creates a model with tasks for content viewer integration testing.
// It creates a temporary project directory structure with a real task file.
func newTaskViewerTestModel(t *testing.T) (Model, string) {
	t.Helper()

	// Create a temp project directory with a markdown task file
	tmpDir := t.TempDir()
	taskDir := filepath.Join(tmpDir, "docs", "delivery", "29")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("failed to create task dir: %v", err)
	}
	taskFile := filepath.Join(taskDir, "29-1.md")
	if err := os.WriteFile(taskFile, []byte("# Task 29-1\n\nThis is the task content."), 0o644); err != nil {
		t.Fatalf("failed to write task file: %v", err)
	}

	groups := []task.TaskGroup{
		{
			ID:     "PBI-29",
			Title:  "Content Viewer",
			Status: "InProgress",
			Tasks: []task.Task{
				{ID: "29-1", Title: "Build ContentViewer", Status: "done"},
				{ID: "29-2", Title: "Add diff coloring", Status: "active", URL: "https://github.com/owner/repo/issues/42"},
			},
		},
	}

	m := Model{
		width:  120,
		height: 40,
		sessions: []session.Info{
			{TmuxSession: "dev", Status: session.StatusWorking, CWD: tmpDir, Timestamp: time.Now().Unix()},
		},
		searchInput:        initSearchInput(),
		taskSearchInput:    initTaskSearchInput(),
		taskCache:          task.NewResultCache(),
		taskGlobalConfig:   &task.GlobalConfig{},
		taskGroups:         groups,
		taskExpandedGroups: map[string]bool{"PBI-29": true},
		taskFocusedProject: tmpDir,
		taskGroupsByProject: map[string][]task.TaskGroup{
			tmpDir: groups,
		},
		taskProjectConfigs: []task.ProjectConfig{
			{Tasks: task.ProjectTaskConfig{Provider: "markdown-tasks"}, ProjectDir: tmpDir},
		},
		taskPanelVisible: true,
		taskPanelFocused: true,
	}

	return m, tmpDir
}

func TestTaskPanelEnterOpensContentViewer(t *testing.T) {
	t.Run("enter on local task opens content viewer with file content", func(t *testing.T) {
		m, _ := newTaskViewerTestModel(t)
		// Cursor on task "29-1" (group header is index 0, task 29-1 is index 1)
		m.taskCursor = 1

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogContentViewer {
			t.Errorf("expected DialogContentViewer, got %d", updated.dialogMode)
		}
		if updated.contentViewerTitle != "Build ContentViewer" {
			t.Errorf("expected title 'Build ContentViewer', got %q", updated.contentViewerTitle)
		}
		// Content should include the file content
		fullContent := strings.Join(updated.contentViewerLines, "\n")
		if !strings.Contains(fullContent, "Task 29-1") {
			t.Error("content viewer should contain the task file content")
		}
		if !strings.Contains(fullContent, "task content") {
			t.Error("content viewer should contain full file content")
		}
	})

	t.Run("enter on task with URL does not open content viewer", func(t *testing.T) {
		m, _ := newTaskViewerTestModel(t)
		// Cursor on task "29-2" which has a URL (index 2)
		m.taskCursor = 2

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		// Should NOT open content viewer (it would try to open URL externally)
		if updated.dialogMode == DialogContentViewer {
			t.Error("should not open content viewer for tasks with URL")
		}
	})

	t.Run("enter on group toggles expansion", func(t *testing.T) {
		m, _ := newTaskViewerTestModel(t)
		m.taskCursor = 0 // On group header

		// Group is currently expanded, enter should collapse it
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.taskExpandedGroups["PBI-29"] {
			t.Error("group should be collapsed after enter on expanded group")
		}
		if updated.dialogMode == DialogContentViewer {
			t.Error("should not open content viewer when pressing enter on group")
		}
	})

	t.Run("enter on group when collapsed expands it", func(t *testing.T) {
		m, _ := newTaskViewerTestModel(t)
		m.taskExpandedGroups = map[string]bool{} // Collapse all
		m.taskCursor = 0                         // On group header

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !updated.taskExpandedGroups["PBI-29"] {
			t.Error("group should be expanded after enter on collapsed group")
		}
	})

	t.Run("file read error shows error in content viewer", func(t *testing.T) {
		m, _ := newTaskViewerTestModel(t)
		// Add a task with an ID that doesn't have a file
		m.taskGroups[0].Tasks = append(m.taskGroups[0].Tasks, task.Task{
			ID: "29-99", Title: "Nonexistent task", Status: "proposed",
		})
		// Navigate to the nonexistent task
		m.taskCursor = 3

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogContentViewer {
			t.Errorf("expected DialogContentViewer for error display, got %d", updated.dialogMode)
		}
		fullContent := strings.Join(updated.contentViewerLines, "\n")
		if !strings.Contains(fullContent, "Error") {
			t.Error("content viewer should show error message for missing file")
		}
	})
}

func TestTaskPanelEnterDoesNotAffectNonFocusedPanel(t *testing.T) {
	t.Run("enter on session list does not trigger task viewer", func(t *testing.T) {
		m, _ := newTaskViewerTestModel(t)
		m.taskPanelFocused = false // Focus is on session list

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		// Should not open content viewer (session list enter attaches to session)
		if updated.dialogMode == DialogContentViewer {
			t.Error("session list enter should not open content viewer")
		}
	})
}
