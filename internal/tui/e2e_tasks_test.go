package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/task"
)

// testProjectDir is the project directory used in task test models.
const testProjectDir = "/home/user/api"

// newTaskTestModel creates a model pre-populated with task data for testing.
func newTaskTestModel() Model {
	groups := []task.TaskGroup{
		{
			ID:     "g1",
			Title:  "Search & Filter",
			Status: "in_progress",
			Tasks: []task.Task{
				{ID: "13-1", Title: "Implement fuzzy search", Status: "done", URL: "https://github.com/owner/repo/issues/42"},
				{ID: "13-2", Title: "Add status filter", Status: "done"},
				{ID: "13-3", Title: "Sort mode cycling", Status: "active", Labels: []string{"feat", "tui"}},
			},
		},
		{
			ID:     "g2",
			Title:  "API Server",
			Status: "open",
			Tasks: []task.Task{
				{ID: "#142", Title: "Add rate limiting endpoint", Status: "active"},
				{ID: "#138", Title: "Fix auth token refresh", Status: "todo"},
			},
		},
	}

	m := Model{
		width:  120,
		height: 40,
		sessions: []session.Info{
			{TmuxSession: "api-server", Status: session.StatusWorking, CWD: testProjectDir, Timestamp: time.Now().Unix()},
			{TmuxSession: "frontend", Status: session.StatusWaiting, CWD: "/home/user/web", Timestamp: time.Now().Unix()},
		},
		searchInput:        initSearchInput(),
		taskSearchInput:    initTaskSearchInput(),
		taskCache:          task.NewResultCache(),
		taskGlobalConfig:   &task.GlobalConfig{},
		taskGroups:         groups,
		taskExpandedGroups: make(map[string]bool),
		taskFocusedProject: testProjectDir,
		taskGroupsByProject: map[string][]task.TaskGroup{
			testProjectDir: groups,
		},
		taskProjectConfigs: []task.ProjectConfig{
			{Tasks: task.ProjectTaskConfig{Provider: "test"}, ProjectDir: testProjectDir},
		},
	}
	return m
}

// TestE2E_TaskPanelToggle tests T key toggles the task panel.
func TestE2E_TaskPanelToggle(t *testing.T) {
	t.Run("T key shows task panel scoped to selected session project", func(t *testing.T) {
		m := newTaskTestModel()

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !updated.taskPanelVisible {
			t.Error("task panel should be visible after pressing T")
		}
		if updated.taskFocusedProject != testProjectDir {
			t.Errorf("focused project should be %q, got %q", testProjectDir, updated.taskFocusedProject)
		}
	})

	t.Run("T key hides task panel when visible", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.taskPanelVisible {
			t.Error("task panel should be hidden after pressing T again")
		}
	})
}

// TestE2E_TaskPanelMutualExclusivity tests that task panel and preview are mutually exclusive.
func TestE2E_TaskPanelMutualExclusivity(t *testing.T) {
	t.Run("T key closes preview pane", func(t *testing.T) {
		m := newTaskTestModel()
		m.previewVisible = true
		m.previewUserEnabled = true

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !updated.taskPanelVisible {
			t.Error("task panel should be visible")
		}
		if updated.previewVisible {
			t.Error("preview should be hidden when task panel opens")
		}
	})

	t.Run("p key closes task panel", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelUserEnabled = true

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !updated.previewVisible {
			t.Error("preview should be visible")
		}
		if updated.taskPanelVisible {
			t.Error("task panel should be hidden when preview opens")
		}
	})
}

// TestE2E_TaskPanelDisplaysGroupsAndTasks tests that the task panel shows groups and tasks.
func TestE2E_TaskPanelDisplaysGroupsAndTasks(t *testing.T) {
	t.Run("task panel renders group titles (collapsed by default)", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true

		result := m.View()

		// Should contain group titles
		if !strings.Contains(result, "Search & Filter") {
			t.Error("view should contain 'Search & Filter' group title")
		}
		if !strings.Contains(result, "API Server") {
			t.Error("view should contain 'API Server' group title")
		}

		// Tasks should NOT be visible when groups are collapsed
		if strings.Contains(result, "Implement fuzzy search") {
			t.Error("tasks should be hidden when groups are collapsed")
		}
	})

	t.Run("expanded groups show tasks", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskExpandedGroups["g1"] = true
		m.taskExpandedGroups["g2"] = true

		result := m.View()

		if !strings.Contains(result, "Implement fuzzy search") {
			t.Error("view should contain task title when group is expanded")
		}
		if !strings.Contains(result, "Add rate limiting endpoint") {
			t.Error("view should contain task title when group is expanded")
		}
	})

	t.Run("task panel header shows counts", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true

		result := m.View()

		if !strings.Contains(result, "5 tasks") {
			t.Error("view should show task count")
		}
		if !strings.Contains(result, "2 groups") {
			t.Error("view should show group count")
		}
	})
}

// TestE2E_TaskPanelCursorTracking tests that cursor movement updates the task panel.
func TestE2E_TaskPanelCursorTracking(t *testing.T) {
	t.Run("moving cursor updates focused project", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.cursor = 0 // on api-server (has project config)

		// Move to frontend (no project config)
		msg := tea.KeyMsg{Type: tea.KeyDown}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.cursor != 1 {
			t.Errorf("cursor should be 1, got %d", updated.cursor)
		}
		// frontend has CWD /home/user/web which has no project config
		if updated.taskFocusedProject != "" {
			t.Errorf("focused project should be empty for unconfigured session, got %q", updated.taskFocusedProject)
		}
	})

	t.Run("cursor back to configured session restores tasks", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.cursor = 1 // on frontend
		m.taskFocusedProject = ""
		m.taskGroups = nil

		// Move back to api-server
		msg := tea.KeyMsg{Type: tea.KeyUp}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.cursor != 0 {
			t.Errorf("cursor should be 0, got %d", updated.cursor)
		}
		if updated.taskFocusedProject != testProjectDir {
			t.Errorf("focused project should be %q, got %q", testProjectDir, updated.taskFocusedProject)
		}
		if len(updated.taskGroups) == 0 {
			t.Error("task groups should be populated for configured project")
		}
	})
}

// TestE2E_TaskPanelResize tests that [/] resize the task panel.
func TestE2E_TaskPanelResize(t *testing.T) {
	t.Run("bracket keys resize task panel", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelHeight = 15

		// Shrink
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.taskPanelHeight >= 15 {
			t.Errorf("panel height should shrink, got %d", updated.taskPanelHeight)
		}

		// Expand
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
		newModel, _ = updated.Update(msg)
		updated = newModel.(Model)

		if updated.taskPanelHeight <= 10 {
			t.Errorf("panel height should grow after ], got %d", updated.taskPanelHeight)
		}
	})

	t.Run("bracket keys do not resize when panel hidden", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = false
		m.taskPanelHeight = 15

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.taskPanelHeight != 15 {
			t.Errorf("panel height should not change when hidden, got %d", updated.taskPanelHeight)
		}
	})
}

// TestE2E_TaskPanelErrorDisplay tests that provider errors are displayed gracefully.
func TestE2E_TaskPanelErrorDisplay(t *testing.T) {
	t.Run("provider errors shown for focused project", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskFocusedProject = testProjectDir
		m.taskGroups = nil
		m.taskErrors = map[string]error{
			testProjectDir: &task.ProviderError{
				Type:    task.ErrExec,
				Message: "provider exited with error",
				Stderr:  "gh: not found",
			},
		}

		result := m.View()

		if !strings.Contains(result, "api") {
			t.Error("view should show project name from error")
		}
	})
}

// TestE2E_TaskPanelFocusMode tests entering and using focus mode in the task panel.
func TestE2E_TaskPanelFocusMode(t *testing.T) {
	t.Run("tab enters focus mode when task panel visible", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true

		msg := tea.KeyMsg{Type: tea.KeyTab}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !updated.taskPanelFocused {
			t.Error("task panel should be focused after pressing tab")
		}
		if updated.taskCursor != 0 {
			t.Errorf("task cursor should be 0, got %d", updated.taskCursor)
		}
	})

	t.Run("tab exits focus mode when focused", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true

		msg := tea.KeyMsg{Type: tea.KeyTab}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.taskPanelFocused {
			t.Error("task panel should not be focused after pressing tab again")
		}
	})

	t.Run("esc exits focus mode", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true

		msg := tea.KeyMsg{Type: tea.KeyEscape}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.taskPanelFocused {
			t.Error("task panel should not be focused after pressing esc")
		}
	})

	t.Run("up/down navigates task cursor when focused", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true
		m.taskCursor = 0

		msg := tea.KeyMsg{Type: tea.KeyDown}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.taskCursor != 1 {
			t.Errorf("task cursor should be 1, got %d", updated.taskCursor)
		}
	})

	t.Run("space expands collapsed group when focused", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true
		m.taskCursor = 0 // On first group header

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !updated.taskExpandedGroups["g1"] {
			t.Error("group g1 should be expanded after pressing space")
		}
	})

	t.Run("space collapses expanded group when focused", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true
		m.taskExpandedGroups["g1"] = true
		m.taskCursor = 0 // On first group header

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.taskExpandedGroups["g1"] {
			t.Error("group g1 should be collapsed after pressing space")
		}
	})

	t.Run("session keys blocked when task panel focused", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true

		// 'n' should not open dialog
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogNone {
			t.Error("'n' should not open dialog when task panel is focused")
		}
	})

	t.Run("focused panel shows cursor highlight", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true
		m.taskCursor = 0

		result := m.View()

		// Should contain the selection marker
		if !strings.Contains(result, selectedMarker) {
			t.Error("focused task panel should show selection marker")
		}
	})
}

// TestE2E_TaskPanelSearch tests searching within the focused task panel.
func TestE2E_TaskPanelSearch(t *testing.T) {
	t.Run("slash enters task search mode when focused", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !updated.taskSearchMode {
			t.Error("task search mode should be active after pressing /")
		}
	})

	t.Run("search filters tasks by title", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true
		m.taskSearchMode = true
		m.taskSearchQuery = "fuzzy"

		items := m.getVisibleTaskItems()

		// Should show group "Search & Filter" (has matching task) + the matching task
		foundTask := false
		for _, item := range items {
			if !item.isGroup && item.title == "Implement fuzzy search" {
				foundTask = true
			}
		}
		if !foundTask {
			t.Error("search for 'fuzzy' should show 'Implement fuzzy search' task")
		}

		// Should NOT show tasks from unmatched groups
		for _, item := range items {
			if !item.isGroup && item.title == "Add rate limiting endpoint" {
				t.Error("search for 'fuzzy' should not show unmatched task 'Add rate limiting endpoint'")
			}
		}
	})

	t.Run("search filters by task ID", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true
		m.taskSearchMode = true
		m.taskSearchQuery = "#142"

		items := m.getVisibleTaskItems()

		foundTask := false
		for _, item := range items {
			if !item.isGroup && item.taskID == "#142" {
				foundTask = true
			}
		}
		if !foundTask {
			t.Error("search for '#142' should show task with ID #142")
		}
	})

	t.Run("search filters by group title", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true
		m.taskSearchMode = true
		m.taskSearchQuery = "API Server"

		items := m.getVisibleTaskItems()

		// Should show the API Server group and all its tasks
		foundGroup := false
		taskCount := 0
		for _, item := range items {
			if item.isGroup && item.title == "API Server" {
				foundGroup = true
			}
			if !item.isGroup && item.groupID == "g2" {
				taskCount++
			}
		}
		if !foundGroup {
			t.Error("search for 'API Server' should show the API Server group")
		}
		if taskCount != 2 {
			t.Errorf("search matching group title should show all group tasks, got %d", taskCount)
		}
	})

	t.Run("esc clears search and stays in focus mode", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true
		m.taskSearchMode = true
		m.taskSearchQuery = "test"
		m.taskSearchInput.SetValue("test")

		msg := tea.KeyMsg{Type: tea.KeyEscape}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.taskSearchMode {
			t.Error("search mode should be cleared after esc")
		}
		if updated.taskSearchQuery != "" {
			t.Errorf("search query should be empty, got %q", updated.taskSearchQuery)
		}
		if !updated.taskPanelFocused {
			t.Error("panel should remain focused after clearing search")
		}
	})

	t.Run("enter accepts search and exits search mode", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true
		m.taskSearchMode = true
		m.taskSearchQuery = "fuzzy"
		m.taskSearchInput.SetValue("fuzzy")

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.taskSearchMode {
			t.Error("search mode should be exited after enter")
		}
		if updated.taskSearchQuery != "fuzzy" {
			t.Error("search query should be preserved after enter")
		}
		if !updated.taskPanelFocused {
			t.Error("panel should remain focused after enter")
		}
	})

	t.Run("no match shows empty state", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true
		m.taskSearchMode = true
		m.taskSearchQuery = "xyznonexistent"

		result := m.View()

		if !strings.Contains(result, "No tasks matching") {
			t.Error("should show 'no tasks matching' for non-matching search")
		}
	})

	t.Run("footer shows / search in focus mode", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true

		result := m.View()

		if !strings.Contains(result, "/ search") {
			t.Error("footer should show '/ search' in focus mode")
		}
	})
}

// TestE2E_TaskPanelEmptyState tests empty state when no configs are found.
func TestE2E_TaskPanelEmptyState(t *testing.T) {
	t.Run("empty state shows guidance when no focused project", func(t *testing.T) {
		m := Model{
			width:               120,
			height:              40,
			taskPanelVisible:    true,
			taskFocusedProject:  "",
			taskExpandedGroups:  make(map[string]bool),
			taskGroupsByProject: make(map[string][]task.TaskGroup),
			taskCache:           task.NewResultCache(),
			taskGlobalConfig:    &task.GlobalConfig{},
			searchInput:         initSearchInput(),
			taskSearchInput:     initTaskSearchInput(),
		}

		result := m.View()

		if !strings.Contains(result, ".navi.yaml") {
			t.Error("empty state should mention .navi.yaml setup")
		}
	})

	t.Run("T on session without project config shows empty state", func(t *testing.T) {
		m := Model{
			width:  120,
			height: 40,
			sessions: []session.Info{
				{TmuxSession: "no-config", Status: session.StatusWorking, CWD: "/home/user/unconfigured", Timestamp: time.Now().Unix()},
			},
			searchInput:         initSearchInput(),
			taskSearchInput:     initTaskSearchInput(),
			taskExpandedGroups:  make(map[string]bool),
			taskGroupsByProject: make(map[string][]task.TaskGroup),
			taskCache:           task.NewResultCache(),
			taskGlobalConfig:    &task.GlobalConfig{},
			taskProjectConfigs:  []task.ProjectConfig{},
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if !updated.taskPanelVisible {
			t.Error("task panel should be visible")
		}
		if updated.taskFocusedProject != "" {
			t.Errorf("focused project should be empty, got %q", updated.taskFocusedProject)
		}

		result := updated.View()
		if !strings.Contains(result, ".navi.yaml") {
			t.Error("should show setup guidance for unconfigured project")
		}
	})
}

// TestE2E_TasksMsg_UpdatesModel tests that tasksMsg properly updates the model.
func TestE2E_TasksMsg_UpdatesModel(t *testing.T) {
	t.Run("tasksMsg updates task groups for focused project", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskGroups = nil

		msg := tasksMsg{
			groupsByProject: map[string][]task.TaskGroup{
				testProjectDir: {
					{ID: "new", Title: "New Group", Tasks: []task.Task{{ID: "1", Title: "New Task", Status: "open"}}},
				},
			},
			errors: nil,
		}

		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if len(updated.taskGroups) != 1 {
			t.Errorf("expected 1 group, got %d", len(updated.taskGroups))
		}
		if updated.taskGroups[0].Title != "New Group" {
			t.Errorf("expected group title 'New Group', got %q", updated.taskGroups[0].Title)
		}
	})

	t.Run("tasksMsg does not show groups from other projects", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskGroups = nil

		msg := tasksMsg{
			groupsByProject: map[string][]task.TaskGroup{
				"/other/project": {
					{ID: "other", Title: "Other Project", Tasks: []task.Task{{ID: "1", Title: "Other Task", Status: "open"}}},
				},
			},
			errors: nil,
		}

		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if len(updated.taskGroups) != 0 {
			t.Errorf("expected 0 groups (other project), got %d", len(updated.taskGroups))
		}
	})
}

// TestE2E_TaskConfigsMsg_TriggersRefresh tests that config discovery triggers a refresh.
func TestE2E_TaskConfigsMsg_TriggersRefresh(t *testing.T) {
	t.Run("taskConfigsMsg with configs triggers refresh", func(t *testing.T) {
		m := newTaskTestModel()

		msg := taskConfigsMsg{
			configs: []task.ProjectConfig{
				{Tasks: task.ProjectTaskConfig{Provider: "test"}, ProjectDir: "/tmp/test"},
			},
		}

		newModel, cmd := m.Update(msg)
		updated := newModel.(Model)

		if len(updated.taskProjectConfigs) != 1 {
			t.Errorf("expected 1 project config, got %d", len(updated.taskProjectConfigs))
		}
		if cmd == nil {
			t.Error("taskConfigsMsg with configs should trigger a refresh command")
		}
	})
}

// TestE2E_TaskPanelReadOnly tests that session keybindings still work when task panel is open.
func TestE2E_TaskPanelReadOnly(t *testing.T) {
	t.Run("session keybindings work with task panel open", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true

		// 'n' (new session) should open dialog even with task panel visible
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		newModel, _ := m.Update(msg)
		updated := newModel.(Model)

		if updated.dialogMode != DialogNewSession {
			t.Error("session 'n' key should still open dialog with task panel visible")
		}
	})
}

// TestE2E_TaskPanelFooter tests the footer shows task panel keybindings.
func TestE2E_TaskPanelFooter(t *testing.T) {
	t.Run("footer shows T tasks keybinding", func(t *testing.T) {
		m := newTaskTestModel()

		result := m.View()

		if !strings.Contains(result, "T tasks") {
			t.Error("footer should show T tasks keybinding")
		}
	})

	t.Run("footer shows Tab focus when task panel visible", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true

		result := m.View()

		if !strings.Contains(result, "Tab focus") {
			t.Error("footer should show Tab focus when task panel is visible")
		}
	})

	t.Run("footer shows focus-mode keybindings when focused", func(t *testing.T) {
		m := newTaskTestModel()
		m.taskPanelVisible = true
		m.taskPanelFocused = true

		result := m.View()

		if !strings.Contains(result, "Space expand") {
			t.Error("footer should show Space expand when task panel is focused")
		}
		if !strings.Contains(result, "Tab/Esc back") {
			t.Error("footer should show Tab/Esc back when task panel is focused")
		}
	})
}
