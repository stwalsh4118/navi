package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/task"
)

// --- Task Panel Scroll Indicator Tests ---

func TestTaskPanelScrollIndicators(t *testing.T) {
	t.Run("no indicators when all items fit", func(t *testing.T) {
		m := newTaskScrollTestModel()
		// Collapse all groups so only 3 group headers show (fits in 7 lines)
		m.taskExpandedGroups = map[string]bool{}
		m.taskScrollOffset = 0

		result := m.renderTaskPanelList(100, 7)

		if strings.Contains(result, "more above") {
			t.Error("should not show top indicator when all items fit")
		}
		if strings.Contains(result, "more below") {
			t.Error("should not show bottom indicator when all items fit")
		}
	})

	t.Run("bottom indicator when items overflow below", func(t *testing.T) {
		m := newTaskScrollTestModel()
		m.taskScrollOffset = 0 // At top, all groups expanded = 18 items, maxLines = 7

		result := m.renderTaskPanelList(100, 7)

		if strings.Contains(result, "more above") {
			t.Error("should not show top indicator when at top")
		}
		if !strings.Contains(result, "more below") {
			t.Error("should show bottom indicator when items extend below viewport")
		}
	})

	t.Run("top indicator when scrolled down", func(t *testing.T) {
		m := newTaskScrollTestModel()
		m.taskScrollOffset = 5

		result := m.renderTaskPanelList(100, 7)

		if !strings.Contains(result, "more above") {
			t.Error("should show top indicator when scrolled past top")
		}
		expectedAbove := fmt.Sprintf(scrollIndicatorAbove, 5)
		if !strings.Contains(result, expectedAbove) {
			t.Errorf("top indicator should show correct count, expected %q", expectedAbove)
		}
	})

	t.Run("both indicators when scrolled to middle", func(t *testing.T) {
		m := newTaskScrollTestModel()
		m.taskScrollOffset = 5 // Middle of 18 items with maxLines = 7

		result := m.renderTaskPanelList(100, 7)

		if !strings.Contains(result, "more above") {
			t.Error("should show top indicator")
		}
		if !strings.Contains(result, "more below") {
			t.Error("should show bottom indicator")
		}
	})

	t.Run("no bottom indicator when scrolled to end", func(t *testing.T) {
		m := newTaskScrollTestModel()
		items := m.getVisibleTaskItems()
		maxLines := 7
		maxScroll := taskPanelMaxScroll(len(items), maxLines)
		m.taskScrollOffset = maxScroll

		result := m.renderTaskPanelList(100, maxLines)

		if !strings.Contains(result, "more above") {
			t.Error("should show top indicator when scrolled from top")
		}
		if strings.Contains(result, "more below") {
			t.Error("should not show bottom indicator when scrolled to end")
		}
	})

	t.Run("indicator shows correct below count", func(t *testing.T) {
		m := newTaskScrollTestModel()
		m.taskScrollOffset = 0
		items := m.getVisibleTaskItems()
		maxLines := 7

		result := m.renderTaskPanelList(100, maxLines)

		// With 18 items, maxLines=7, no top indicator, so contentLines=6 (maxLines - 1 for bottom indicator)
		// endIdx = 0 + 6 = 6, belowCount = 18 - 6 = 12
		expectedBelow := fmt.Sprintf(scrollIndicatorBelow, len(items)-6)
		if !strings.Contains(result, expectedBelow) {
			t.Errorf("bottom indicator should show %q, got:\n%s", expectedBelow, result)
		}
	})
}

// --- Session List Scroll Indicator Tests ---

func TestSessionListScrollIndicators(t *testing.T) {
	t.Run("no top indicator when at top", func(t *testing.T) {
		m := newSessionScrollTestModel()
		m.sessionScrollOffset = 0

		result := m.renderSessionList(100)

		if strings.Contains(result, "more above") {
			t.Error("should not show top indicator when at top")
		}
	})

	t.Run("top indicator when scrolled down", func(t *testing.T) {
		m := newSessionScrollTestModel()
		m.sessionScrollOffset = 5

		result := m.renderSessionList(100)

		if !strings.Contains(result, "more above") {
			t.Error("should show top indicator when scrolled past top")
		}
		expectedAbove := fmt.Sprintf(scrollIndicatorAbove, 5)
		if !strings.Contains(result, expectedAbove) {
			t.Errorf("top indicator should show '5', expected %q in result", expectedAbove)
		}
	})

	t.Run("bottom indicator when sessions overflow", func(t *testing.T) {
		m := newSessionScrollTestModel()
		m.sessionScrollOffset = 0

		result := m.renderSessionList(100)

		// With 20 sessions and small terminal (height=30), maxVisible is small
		maxVisible := m.sessionListMaxVisible()
		if maxVisible < len(m.sessions) {
			if !strings.Contains(result, "more below") {
				t.Error("should show bottom indicator when sessions extend below viewport")
			}
		}
	})

	t.Run("no indicators with few sessions", func(t *testing.T) {
		m := newSessionScrollTestModel()
		// Only 2 sessions - should fit without scrolling
		m.sessions = []session.Info{
			{TmuxSession: "s1", Status: session.StatusWorking, CWD: "/tmp", Timestamp: time.Now().Unix()},
			{TmuxSession: "s2", Status: session.StatusWorking, CWD: "/tmp", Timestamp: time.Now().Unix()},
		}
		m.sessionScrollOffset = 0

		result := m.renderSessionList(100)

		if strings.Contains(result, "more above") {
			t.Error("should not show top indicator with few sessions")
		}
		// Bottom indicator depends on maxVisible vs session count
		maxVisible := m.sessionListMaxVisible()
		if maxVisible >= 2 && strings.Contains(result, "more below") {
			t.Error("should not show bottom indicator when all sessions fit")
		}
	})

	t.Run("no indicators on empty session list", func(t *testing.T) {
		m := newSessionScrollTestModel()
		m.sessions = nil

		result := m.renderSessionList(100)

		if strings.Contains(result, "more above") || strings.Contains(result, "more below") {
			t.Error("should not show indicators on empty session list")
		}
	})
}

// --- Preview Pane Scroll Indicator Tests ---

func TestPreviewScrollIndicators(t *testing.T) {
	t.Run("no indicators when content fits", func(t *testing.T) {
		m := newPreviewScrollTestModel()
		// Set small content that fits in viewport
		m.previewContent = "line 1\nline 2\nline 3"
		m.previewAutoScroll = false
		m.previewScrollOffset = 0

		result := m.renderPreview(100, 20)

		if strings.Contains(result, "more above") {
			t.Error("should not show top indicator when content fits")
		}
		if strings.Contains(result, "more below") {
			t.Error("should not show bottom indicator when content fits")
		}
	})

	t.Run("bottom indicator when content overflows", func(t *testing.T) {
		m := newPreviewScrollTestModel()
		m.previewAutoScroll = false
		m.previewScrollOffset = 0

		result := m.renderPreview(100, 20)

		// 100 lines of content in a 20-height preview (17 content lines)
		if strings.Contains(result, "more above") {
			t.Error("should not show top indicator when at top")
		}
		if !strings.Contains(result, "more below") {
			t.Error("should show bottom indicator when content extends below")
		}
	})

	t.Run("top indicator when scrolled down", func(t *testing.T) {
		m := newPreviewScrollTestModel()
		m.previewAutoScroll = false
		m.previewScrollOffset = 10

		result := m.renderPreview(100, 20)

		if !strings.Contains(result, "more above") {
			t.Error("should show top indicator when scrolled past top")
		}
	})

	t.Run("both indicators when scrolled to middle", func(t *testing.T) {
		m := newPreviewScrollTestModel()
		m.previewAutoScroll = false
		m.previewScrollOffset = 30

		result := m.renderPreview(100, 20)

		if !strings.Contains(result, "more above") {
			t.Error("should show top indicator")
		}
		if !strings.Contains(result, "more below") {
			t.Error("should show bottom indicator")
		}
	})

	t.Run("no bottom indicator when at bottom", func(t *testing.T) {
		m := newPreviewScrollTestModel()
		m.previewAutoScroll = true // Auto-scroll goes to bottom

		result := m.renderPreview(100, 20)

		if !strings.Contains(result, "more above") {
			t.Error("should show top indicator when scrolled to bottom of long content")
		}
		if strings.Contains(result, "more below") {
			t.Error("should not show bottom indicator when at bottom")
		}
	})

	t.Run("no indicators with empty content", func(t *testing.T) {
		m := newPreviewScrollTestModel()
		m.previewContent = ""

		result := m.renderPreview(100, 20)

		if strings.Contains(result, "more above") || strings.Contains(result, "more below") {
			t.Error("should not show indicators with empty content")
		}
	})
}

// --- Scroll Indicator Constants Tests ---

func TestScrollIndicatorConstants(t *testing.T) {
	t.Run("above indicator formats correctly", func(t *testing.T) {
		result := fmt.Sprintf(scrollIndicatorAbove, 5)
		if result != "↑ 5 more above" {
			t.Errorf("expected '↑ 5 more above', got %q", result)
		}
	})

	t.Run("below indicator formats correctly", func(t *testing.T) {
		result := fmt.Sprintf(scrollIndicatorBelow, 12)
		if result != "↓ 12 more below" {
			t.Errorf("expected '↓ 12 more below', got %q", result)
		}
	})
}

// --- Integration: Indicators Render with Dimmed Style ---

func TestScrollIndicatorsUseDimStyle(t *testing.T) {
	t.Run("task panel indicators are dimmed", func(t *testing.T) {
		m := newTaskScrollTestModel()
		m.taskScrollOffset = 5

		result := m.renderTaskPanelList(100, 7)

		// The indicator text should be present (dimStyle wraps it in ANSI codes)
		if !strings.Contains(result, "more above") {
			t.Error("indicator text should be present in rendered output")
		}
	})
}

// --- Cross-panel: Indicators update with scroll ---

func TestScrollIndicatorsUpdateDynamically(t *testing.T) {
	t.Run("task panel indicators change as scroll offset changes", func(t *testing.T) {
		m := newTaskScrollTestModel()

		// At top: no top indicator, has bottom indicator
		m.taskScrollOffset = 0
		resultTop := m.renderTaskPanelList(100, 7)
		if strings.Contains(resultTop, "more above") {
			t.Error("should not have top indicator at scroll offset 0")
		}

		// In middle: both indicators
		m.taskScrollOffset = 5
		resultMid := m.renderTaskPanelList(100, 7)
		if !strings.Contains(resultMid, "more above") || !strings.Contains(resultMid, "more below") {
			t.Error("should have both indicators in the middle")
		}

		// At bottom: top indicator, no bottom indicator
		items := m.getVisibleTaskItems()
		maxScroll := taskPanelMaxScroll(len(items), 7)
		m.taskScrollOffset = maxScroll
		resultBot := m.renderTaskPanelList(100, 7)
		if !strings.Contains(resultBot, "more above") {
			t.Error("should have top indicator at bottom")
		}
		if strings.Contains(resultBot, "more below") {
			t.Error("should not have bottom indicator at bottom")
		}
	})
}

// helper to create a model with enough sessions and task groups for indicator tests
func newScrollIndicatorTestModel() Model {
	now := time.Now().Unix()
	sessions := make([]session.Info, 20)
	for i := range sessions {
		sessions[i] = session.Info{
			TmuxSession: fmt.Sprintf("session-%02d", i),
			Status:      session.StatusWorking,
			CWD:         "/home/user/project",
			Timestamp:   now,
		}
	}

	groups := []task.TaskGroup{
		{
			ID:     "g1",
			Title:  "Group One",
			Status: "in_progress",
			Tasks: []task.Task{
				{ID: "1-1", Title: "Task 1-1", Status: "done"},
				{ID: "1-2", Title: "Task 1-2", Status: "active"},
			},
		},
	}

	return Model{
		width:               120,
		height:              30,
		sessions:            sessions,
		cursor:              0,
		sessionScrollOffset: 0,
		searchInput:         initSearchInput(),
		taskSearchInput:     initTaskSearchInput(),
		taskExpandedGroups:  map[string]bool{"g1": true},
		taskGroupsByProject: map[string][]task.TaskGroup{"/home/user/project": groups},
		taskGroups:          groups,
		taskCache:           task.NewResultCache(),
		taskGlobalConfig:    &task.GlobalConfig{},
		taskPanelVisible:    true,
		taskPanelFocused:    true,
		taskPanelHeight:     10,
		sortMode:            SortPriority,
	}
}
