package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/stwalsh4118/navi/internal/task"
)

// Task panel header title
const taskPanelTitle = "Tasks"

// Task panel empty state messages
const (
	taskEmptyMessage = "No task providers configured. Create a .navi.yaml in your project root."
)

// renderTaskPanel renders the task panel that sits below the session list.
// It displays tasks for the focused project with collapsible groups.
func (m Model) renderTaskPanel(width, height int) string {
	if !m.taskPanelVisible {
		return ""
	}

	var b strings.Builder

	// Header line with project name and task count
	b.WriteString(m.renderTaskPanelHeader(width))
	b.WriteString("\n")

	// Content area
	contentWidth := width - 4 // Account for box padding and borders
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Search bar (when actively typing or search persisted after Enter)
	if m.taskSearchMode || m.taskSearchQuery != "" {
		var searchContent string
		if m.taskSearchMode {
			searchContent = "/ " + m.taskSearchInput.View()
		} else {
			searchContent = "/ " + m.taskSearchQuery
		}
		// Show match counter
		if m.taskSearchQuery != "" {
			if len(m.taskSearchMatches) > 0 {
				counter := fmt.Sprintf(" [%d/%d]", m.taskCurrentMatchIdx+1, len(m.taskSearchMatches))
				searchContent += searchMatchCountStyle.Render(counter)
			} else {
				searchContent += " " + searchNoMatchStyle.Render("No matches")
			}
		}
		b.WriteString(searchContent)
		b.WriteString("\n")
	}

	if m.taskFocusedProject == "" {
		// No config for this session's project
		b.WriteString(taskEmptyStyle.Render(taskEmptyMessage))
	} else if err, ok := m.taskErrors[m.taskFocusedProject]; ok {
		// Show error for focused project
		parts := strings.Split(m.taskFocusedProject, "/")
		name := m.taskFocusedProject
		if len(parts) > 0 {
			name = parts[len(parts)-1]
		}
		b.WriteString(taskErrorStyle.Render(fmt.Sprintf("  ✗ %s: %s", name, err.Error())))
	} else if len(m.taskGroups) == 0 {
		b.WriteString(taskEmptyStyle.Render("  No tasks found"))
	} else {
		// Render task list with collapsible groups
		maxLines := height - 3 // header(1) + borders(2)
		if m.taskSearchMode {
			maxLines-- // search bar takes 1 line
		}
		if maxLines < 1 {
			maxLines = 1
		}
		b.WriteString(m.renderTaskPanelList(contentWidth, maxLines))
	}

	// Render the box - use focused style when panel has focus
	boxContent := b.String()
	boxStyle := taskPanelBoxStyle
	if m.taskPanelFocused {
		boxStyle = taskPanelFocusedBoxStyle
	}
	rendered := boxStyle.Width(width - 2).Render(boxContent)

	// Ensure the final output is exactly `height` lines by truncating or padding
	renderedLines := strings.Split(rendered, "\n")
	if len(renderedLines) > height {
		renderedLines = renderedLines[:height]
	}
	for len(renderedLines) < height {
		renderedLines = append(renderedLines, "")
	}

	return strings.Join(renderedLines, "\n")
}

// renderTaskPanelHeader renders the task panel header with project name and counts.
func (m Model) renderTaskPanelHeader(width int) string {
	// Project name
	projectName := ""
	if m.taskFocusedProject != "" {
		parts := strings.Split(m.taskFocusedProject, "/")
		if len(parts) > 0 {
			projectName = parts[len(parts)-1]
		}
	}

	// Count tasks
	totalTasks := totalTaskCount(m.taskGroups)

	var headerParts []string
	headerParts = append(headerParts, previewHeaderStyle.Render("─ "+taskPanelTitle))
	if projectName != "" {
		headerParts = append(headerParts, dimStyle.Render(projectName))
	}
	if totalTasks > 0 {
		headerParts = append(headerParts, dimStyle.Render(fmt.Sprintf("(%d tasks, %d groups)", totalTasks, len(m.taskGroups))))
	}

	return strings.Join(headerParts, " ")
}

// isTaskSearchMatch checks if a task item index is in the taskSearchMatches list.
func (m Model) isTaskSearchMatch(idx int) bool {
	for _, matchIdx := range m.taskSearchMatches {
		if matchIdx == idx {
			return true
		}
	}
	return false
}

// isCurrentTaskSearchMatch checks if a task item index is the current task search match.
func (m Model) isCurrentTaskSearchMatch(idx int) bool {
	if len(m.taskSearchMatches) == 0 {
		return false
	}
	return m.taskSearchMatches[m.taskCurrentMatchIdx] == idx
}

// renderTaskPanelList renders the task list with collapsible groups.
// When focused, shows a cursor. Groups respect taskExpandedGroups state.
func (m Model) renderTaskPanelList(width, maxLines int) string {
	var b strings.Builder
	items := m.getVisibleTaskItems()

	if len(items) == 0 {
		return taskEmptyStyle.Render("  No tasks found")
	}

	lineCount := 0
	for i, item := range items {
		if lineCount >= maxLines {
			break
		}

		isCursorHere := m.taskPanelFocused && i == m.taskCursor
		isMatch := m.taskSearchQuery != "" && m.isTaskSearchMatch(i)
		isCurrentMatch := m.taskSearchQuery != "" && m.isCurrentTaskSearchMatch(i)

		var line string
		if item.isGroup {
			line = m.renderTaskPanelGroupHeader(item, isCursorHere, width)
		} else {
			line = m.renderTaskPanelRow(item, isCursorHere, width)
		}

		// Apply search match highlighting via left-side gutter indicator
		if isCurrentMatch && !isCursorHere {
			line = searchCurrentMatchIndicatorStyle.Render(searchMatchIndicator) + line
		} else if isMatch && !isCursorHere {
			line = searchMatchIndicatorStyle.Render(searchMatchIndicator) + line
		}

		b.WriteString(line)
		b.WriteString("\n")
		lineCount++
	}

	return b.String()
}

// renderTaskPanelGroupHeader renders a group header line in the task panel.
func (m Model) renderTaskPanelGroupHeader(item taskItem, selected bool, width int) string {
	// Selection marker when focused
	marker := unselectedMarker
	if selected {
		marker = selectedMarker
	} else if !m.taskPanelFocused {
		marker = "" // No markers when not focused
	}

	// Chevron shows expand/collapse state
	chevron := taskChevronCollapsed
	if m.taskExpandedGroups[item.groupID] {
		chevron = taskChevronExpanded
	}

	// Count tasks in this group
	taskCount := 0
	for _, g := range m.taskGroups {
		if g.ID == item.groupID {
			taskCount = len(g.Tasks)
			break
		}
	}

	styledChevron := taskGroupChevronStyle.Render(chevron)
	groupNum := dimStyle.Render(fmt.Sprintf("%d.", item.number))
	styledTitle := taskGroupStyle.Render(item.title)
	countLabel := dimStyle.Render(fmt.Sprintf("(%d)", taskCount))

	line := fmt.Sprintf("%s%s %s %s %s", marker, styledChevron, groupNum, styledTitle, countLabel)

	// Status on the right
	if item.status != "" {
		badge := TaskStatusBadge(item.status)
		padding := width - lipgloss.Width(line) - lipgloss.Width(badge) - 2
		if padding < 1 {
			padding = 1
		}
		line += strings.Repeat(" ", padding) + badge
	}

	if selected {
		return selectedStyle.Width(width).Render(line)
	}
	return line
}

// renderTaskPanelRow renders a single task row in the task panel.
func (m Model) renderTaskPanelRow(item taskItem, selected bool, width int) string {
	// Selection marker when focused
	marker := unselectedMarker
	if selected {
		marker = selectedMarker
	} else if !m.taskPanelFocused {
		marker = "  " // Indent to match unfocused groups
	}

	// Status badge (fixed width for alignment)
	badge := TaskStatusBadge(padRight(item.status, 8))

	// Task ID (dimmed)
	id := taskIDStyle.Render(item.taskID)

	// Title
	title := item.title

	line := fmt.Sprintf("%s  %s %s %s", marker, badge, id, title)

	// Truncate if needed
	if lipgloss.Width(line) > width {
		line = truncate(line, width)
	}

	if selected {
		return selectedStyle.Width(width).Render(line)
	}
	return line
}

// padRight pads a string to the given width with spaces.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

// totalTaskCount returns the total number of tasks across all groups.
func totalTaskCount(groups []task.TaskGroup) int {
	total := 0
	for _, g := range groups {
		total += len(g.Tasks)
	}
	return total
}
