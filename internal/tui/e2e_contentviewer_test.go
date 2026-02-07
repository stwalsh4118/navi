package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/git"
	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/task"
)

// --- Acceptance Criteria 1: Reusable content viewer with scrolling ---

func TestE2E_AC1_ContentViewerWithScrolling(t *testing.T) {
	t.Run("content viewer displays arbitrary text with scrolling", func(t *testing.T) {
		// Create content exceeding viewport height
		content := multiLineContent(100)
		m := newContentViewerTestModel(content, ContentModePlain)

		if m.dialogMode != DialogContentViewer {
			t.Fatal("content viewer should be open")
		}

		// Verify rendering includes content
		output := m.renderContentViewer()
		if !strings.Contains(output, "Test Title") {
			t.Error("AC1: content viewer should display title")
		}

		// Verify scrolling works
		viewportH := m.contentViewerViewportHeight()
		if viewportH < 1 {
			t.Fatalf("AC1: viewport height should be positive, got %d", viewportH)
		}

		// Scroll down
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		updated := result.(Model)
		if updated.contentViewerScroll != 1 {
			t.Errorf("AC1: scroll should be 1 after j, got %d", updated.contentViewerScroll)
		}

		// Page down
		result, _ = updated.Update(tea.KeyMsg{Type: tea.KeyPgDown})
		updated = result.(Model)
		if updated.contentViewerScroll <= 1 {
			t.Error("AC1: page down should scroll more than 1 line")
		}
	})
}

// --- Acceptance Criteria 2: Task enter opens detail file ---

func TestE2E_AC2_TaskEnterOpensDetailFile(t *testing.T) {
	t.Run("enter on task opens detail file in content viewer", func(t *testing.T) {
		// Set up a project with a real task file
		tmpDir := t.TempDir()
		taskDir := filepath.Join(tmpDir, "docs", "delivery", "29")
		if err := os.MkdirAll(taskDir, 0o755); err != nil {
			t.Fatalf("failed to create task dir: %v", err)
		}
		taskContent := "# Task 29-1: Build ContentViewer\n\nDetailed task content here."
		if err := os.WriteFile(filepath.Join(taskDir, "29-1.md"), []byte(taskContent), 0o644); err != nil {
			t.Fatalf("failed to write task file: %v", err)
		}

		groups := []task.TaskGroup{{
			ID:     "PBI-29",
			Title:  "Content Viewer",
			Status: "InProgress",
			Tasks:  []task.Task{{ID: "29-1", Title: "Build ContentViewer", Status: "done"}},
		}}

		m := Model{
			width: 120, height: 40,
			sessions: []session.Info{
				{TmuxSession: "dev", Status: session.StatusWorking, CWD: tmpDir, Timestamp: time.Now().Unix()},
			},
			searchInput: initSearchInput(), taskSearchInput: initTaskSearchInput(),
			taskCache: task.NewResultCache(), taskGlobalConfig: &task.GlobalConfig{},
			taskGroups: groups, taskExpandedGroups: map[string]bool{"PBI-29": true},
			taskFocusedProject:  tmpDir,
			taskGroupsByProject: map[string][]task.TaskGroup{tmpDir: groups},
			taskProjectConfigs:  []task.ProjectConfig{{Tasks: task.ProjectTaskConfig{Provider: "markdown-tasks"}, ProjectDir: tmpDir}},
			taskPanelVisible:    true,
			taskPanelFocused:    true,
			taskCursor:          1, // On the task row
		}

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		result, _ := m.Update(msg)
		updated := result.(Model)

		if updated.dialogMode != DialogContentViewer {
			t.Fatalf("AC2: expected DialogContentViewer, got %d", updated.dialogMode)
		}
		fullContent := strings.Join(updated.contentViewerLines, "\n")
		if !strings.Contains(fullContent, "Build ContentViewer") {
			t.Error("AC2: content viewer should contain the task file content")
		}
		if updated.contentViewerTitle != "Build ContentViewer" {
			t.Errorf("AC2: title should be task name, got %q", updated.contentViewerTitle)
		}
	})
}

// --- Acceptance Criteria 3: Keybindings ---

func TestE2E_AC3_Keybindings(t *testing.T) {
	content := multiLineContent(100)

	t.Run("j/k scroll one line", func(t *testing.T) {
		m := newContentViewerTestModel(content, ContentModePlain)

		// j scrolls down
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		updated := result.(Model)
		if updated.contentViewerScroll != 1 {
			t.Errorf("AC3: j should scroll to 1, got %d", updated.contentViewerScroll)
		}

		// k scrolls up
		result, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		updated = result.(Model)
		if updated.contentViewerScroll != 0 {
			t.Errorf("AC3: k should scroll to 0, got %d", updated.contentViewerScroll)
		}
	})

	t.Run("up/down arrows scroll one line", func(t *testing.T) {
		m := newContentViewerTestModel(content, ContentModePlain)

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		updated := result.(Model)
		if updated.contentViewerScroll != 1 {
			t.Errorf("AC3: down should scroll to 1, got %d", updated.contentViewerScroll)
		}

		result, _ = updated.Update(tea.KeyMsg{Type: tea.KeyUp})
		updated = result.(Model)
		if updated.contentViewerScroll != 0 {
			t.Errorf("AC3: up should scroll to 0, got %d", updated.contentViewerScroll)
		}
	})

	t.Run("page up/down scroll one page", func(t *testing.T) {
		m := newContentViewerTestModel(content, ContentModePlain)

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
		updated := result.(Model)
		if updated.contentViewerScroll != contentViewerPageScrollAmt {
			t.Errorf("AC3: pgdn should scroll by page amount, got %d", updated.contentViewerScroll)
		}

		result, _ = updated.Update(tea.KeyMsg{Type: tea.KeyPgUp})
		updated = result.(Model)
		if updated.contentViewerScroll != 0 {
			t.Errorf("AC3: pgup should scroll back, got %d", updated.contentViewerScroll)
		}
	})

	t.Run("g goes to top, G goes to bottom", func(t *testing.T) {
		m := newContentViewerTestModel(content, ContentModePlain)
		maxScroll := m.contentViewerMaxScroll()

		// G to bottom
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
		updated := result.(Model)
		if updated.contentViewerScroll != maxScroll {
			t.Errorf("AC3: G should go to bottom (%d), got %d", maxScroll, updated.contentViewerScroll)
		}

		// g to top
		result, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
		updated = result.(Model)
		if updated.contentViewerScroll != 0 {
			t.Errorf("AC3: g should go to top, got %d", updated.contentViewerScroll)
		}
	})

	t.Run("Esc closes viewer", func(t *testing.T) {
		m := newContentViewerTestModel(content, ContentModePlain)

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
		updated := result.(Model)
		if updated.dialogMode != DialogNone {
			t.Errorf("AC3: Esc should close viewer, got dialog mode %d", updated.dialogMode)
		}
	})
}

// --- Acceptance Criteria 4: Diff coloring ---

func TestE2E_AC4_DiffColoring(t *testing.T) {
	t.Run("git diffs display with addition/deletion coloring", func(t *testing.T) {
		diffContent := "diff --git a/main.go b/main.go\n" +
			"--- a/main.go\n" +
			"+++ b/main.go\n" +
			"@@ -1,5 +1,7 @@\n" +
			" package main\n" +
			"+import \"fmt\"\n" +
			"-import \"log\"\n" +
			" func main() {\n"

		m := newContentViewerTestModel(diffContent, ContentModeDiff)
		output := m.renderContentViewer()

		// All diff lines should be present in output
		if !strings.Contains(output, "package main") {
			t.Error("AC4: context lines should be visible")
		}
		if !strings.Contains(output, "import \"fmt\"") {
			t.Error("AC4: addition lines should be visible")
		}
		if !strings.Contains(output, "import \"log\"") {
			t.Error("AC4: deletion lines should be visible")
		}

		// Verify diff mode is set
		if m.contentViewerMode != ContentModeDiff {
			t.Error("AC4: content mode should be diff")
		}
	})

	t.Run("d key from git detail triggers diff mode", func(t *testing.T) {
		m := Model{
			width: 80, height: 24,
			dialogMode: DialogGitDetail,
			sessionToModify: &session.Info{
				TmuxSession: "test", CWD: "/tmp",
				Git: &git.Info{Branch: "main", Dirty: true},
			},
		}

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
		updated := result.(Model)

		if updated.contentViewerMode != ContentModeDiff {
			t.Error("AC4: d from git detail should set diff mode")
		}
	})
}

// --- Acceptance Criteria 5: Terminal resize ---

func TestE2E_AC5_TerminalResize(t *testing.T) {
	t.Run("content viewer adapts to terminal resize", func(t *testing.T) {
		content := multiLineContent(100)
		m := newContentViewerTestModel(content, ContentModePlain)

		h1 := m.contentViewerViewportHeight()

		// Simulate resize
		m.width = 160
		m.height = 60
		h2 := m.contentViewerViewportHeight()

		if h2 <= h1 {
			t.Errorf("AC5: viewport should grow with terminal, was %d now %d", h1, h2)
		}

		// Rendering should work with new dimensions
		output := m.renderContentViewer()
		if output == "" {
			t.Error("AC5: rendering should produce output after resize")
		}
	})

	t.Run("viewport height adapts to smaller terminal", func(t *testing.T) {
		content := multiLineContent(100)
		m := newContentViewerTestModel(content, ContentModePlain)

		h1 := m.contentViewerViewportHeight()

		m.height = 15
		h2 := m.contentViewerViewportHeight()

		if h2 >= h1 {
			t.Errorf("AC5: viewport should shrink with smaller terminal, was %d now %d", h1, h2)
		}
	})
}

// --- Acceptance Criteria 6: Esc closes and returns ---

func TestE2E_AC6_EscClosesViewer(t *testing.T) {
	t.Run("Esc closes standalone content viewer to DialogNone", func(t *testing.T) {
		m := newContentViewerTestModel("content", ContentModePlain)

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
		updated := result.(Model)

		if updated.dialogMode != DialogNone {
			t.Errorf("AC6: Esc should return to DialogNone, got %d", updated.dialogMode)
		}
		if updated.contentViewerLines != nil {
			t.Error("AC6: content viewer lines should be cleared")
		}
	})

	t.Run("Esc from git diff viewer returns to git detail", func(t *testing.T) {
		m := Model{
			width: 80, height: 24,
			dialogMode:              DialogContentViewer,
			contentViewerPrevDialog: DialogGitDetail,
			contentViewerLines:      []string{"diff content"},
			contentViewerTitle:      "Git Diff: main",
			contentViewerMode:       ContentModeDiff,
		}

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
		updated := result.(Model)

		if updated.dialogMode != DialogGitDetail {
			t.Errorf("AC6: Esc should return to DialogGitDetail, got %d", updated.dialogMode)
		}
	})

	t.Run("q also closes content viewer", func(t *testing.T) {
		m := newContentViewerTestModel("content", ContentModePlain)

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		updated := result.(Model)

		if updated.dialogMode != DialogNone {
			t.Errorf("AC6: q should close viewer, got %d", updated.dialogMode)
		}
	})
}
