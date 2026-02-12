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
		headerLines := m.taskPanelHeaderLines()
		maxLines := height - headerLines - 2 // header(N) + borders(2)
		if m.taskSearchMode || m.taskSearchQuery != "" {
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

// statusSummaryOrder defines the display order for status summary categories.
var statusSummaryOrder = []string{"done", "active", "review", "blocked", "todo"}

// renderTaskPanelHeader renders the task panel header with project name, counts,
// and optionally a second line with status breakdown summary.
// Returns (headerString, numberOfHeaderLines).
func (m Model) renderTaskPanelHeader(width int) string {
	// Project name
	projectName := ""
	if m.taskFocusedProject != "" {
		parts := strings.Split(m.taskFocusedProject, "/")
		if len(parts) > 0 {
			projectName = parts[len(parts)-1]
		}
	}

	// Use unfiltered groups for counts (always show full context)
	allGroups := m.taskGroups
	totalTasks := totalTaskCount(allGroups)

	var headerParts []string
	headerParts = append(headerParts, previewHeaderStyle.Render("─ "+taskPanelTitle))
	if projectName != "" {
		headerParts = append(headerParts, dimStyle.Render(projectName))
	}
	if totalTasks > 0 {
		headerParts = append(headerParts, dimStyle.Render(fmt.Sprintf("(%d tasks, %d groups)", totalTasks, len(allGroups))))
	}

	// Add sort mode indicator (hidden when source + ascending, to reduce noise)
	if m.taskSortMode != taskSortSource || m.taskSortReversed {
		arrow := "↑"
		if m.taskSortReversed {
			arrow = "↓"
		}
		headerParts = append(headerParts, dimStyle.Render(fmt.Sprintf("sort:%s%s", m.taskSortMode, arrow)))
	}

	// Add filter mode indicator (hidden when all)
	if m.taskFilterMode != taskFilterAll {
		filterInfo := fmt.Sprintf("filter:%s", m.taskFilterMode)
		hiddenCount := len(allGroups) - len(m.getFilteredTaskGroups())
		if hiddenCount > 0 {
			filterInfo += fmt.Sprintf(" (%d hidden)", hiddenCount)
		}
		headerParts = append(headerParts, dimStyle.Render(filterInfo))
	}

	// Add accordion indicator
	if m.taskAccordionMode {
		headerParts = append(headerParts, dimStyle.Render("accordion"))
	}

	// Add refreshing indicator
	if m.taskRefreshing {
		headerParts = append(headerParts, dimStyle.Render(italicStyle.Render("Refreshing...")))
	}

	firstLine := strings.Join(headerParts, " ")

	// Second line: status summary (only when panel height allows)
	height := m.getTaskPanelHeight()
	contentLines := height - 3 // header(1) + borders(2)
	if contentLines >= 4 && len(allGroups) > 0 {
		summary := groupStatusSummary(allGroups)
		summaryLine := renderStatusSummary(summary)
		if summaryLine != "" {
			return firstLine + "\n" + "  " + summaryLine
		}
	}

	return firstLine
}

// groupStatusCategory maps a group status string to a canonical category.
func groupStatusCategory(status string) string {
	lower := strings.ToLower(status)
	switch {
	case lower == "done" || lower == "closed" || lower == "completed":
		return "done"
	case lower == "active" || lower == "in_progress" || lower == "inprogress" || lower == "working":
		return "active"
	case lower == "review" || lower == "inreview":
		return "review"
	case lower == "blocked":
		return "blocked"
	default:
		return "todo"
	}
}

// groupStatusSummary counts groups by canonical status category.
func groupStatusSummary(groups []task.TaskGroup) map[string]int {
	counts := make(map[string]int)
	for _, g := range groups {
		cat := groupStatusCategory(g.Status)
		counts[cat]++
	}
	return counts
}

// renderStatusSummary formats a status summary line like "12 done · 3 active · 1 review · 18 todo".
// Only includes categories with non-zero counts.
func renderStatusSummary(counts map[string]int) string {
	var parts []string
	for _, cat := range statusSummaryOrder {
		if n, ok := counts[cat]; ok && n > 0 {
			var styled string
			switch cat {
			case "done":
				styled = greenStyle.Render(fmt.Sprintf("%d", n)) + " " + dimStyle.Render(cat)
			case "active":
				styled = cyanStyle.Render(fmt.Sprintf("%d", n)) + " " + dimStyle.Render(cat)
			case "review":
				styled = magentaStyle.Render(fmt.Sprintf("%d", n)) + " " + dimStyle.Render(cat)
			case "blocked":
				styled = redStyle.Render(fmt.Sprintf("%d", n)) + " " + dimStyle.Render(cat)
			case "todo":
				styled = yellowStyle.Render(fmt.Sprintf("%d", n)) + " " + dimStyle.Render(cat)
			}
			parts = append(parts, styled)
		}
	}
	return strings.Join(parts, dimStyle.Render(" · "))
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
// Uses taskScrollOffset for viewport-aware rendering, showing only visible items.
// Shows scroll indicators when content extends beyond the viewport.
func (m Model) renderTaskPanelList(width, maxLines int) string {
	var b strings.Builder
	items := m.getVisibleTaskItems()

	if len(items) == 0 {
		return taskEmptyStyle.Render("  No tasks found")
	}

	// Determine visible range based on scroll offset
	startIdx := m.taskScrollOffset
	if startIdx > len(items) {
		startIdx = len(items)
	}

	// Check if scroll indicators are needed and reduce content lines accordingly.
	// Important: hasBelow must be checked AFTER reducing for hasAbove, because
	// the top indicator reduces the number of visible items, making it more likely
	// that items extend below the viewport.
	hasAbove := startIdx > 0
	contentLines := maxLines
	if hasAbove {
		contentLines--
	}
	hasBelow := startIdx+contentLines < len(items)
	if hasBelow {
		contentLines--
	}
	if contentLines < 1 {
		contentLines = 1
	}

	endIdx := startIdx + contentLines
	if endIdx > len(items) {
		endIdx = len(items)
	}

	// Render top scroll indicator
	if hasAbove {
		b.WriteString(dimStyle.Render(fmt.Sprintf(scrollIndicatorAbove, startIdx)))
		b.WriteString("\n")
	}

	for i := startIdx; i < endIdx; i++ {
		item := items[i]
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
	}

	// Render bottom scroll indicator
	if hasBelow {
		belowCount := len(items) - endIdx
		b.WriteString(dimStyle.Render(fmt.Sprintf(scrollIndicatorBelow, belowCount)))
		b.WriteString("\n")
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

	// Find group to compute progress
	var group *task.TaskGroup
	for i := range m.taskGroups {
		if m.taskGroups[i].ID == item.groupID {
			group = &m.taskGroups[i]
			break
		}
	}

	styledChevron := taskGroupChevronStyle.Render(chevron)
	groupNum := dimStyle.Render(fmt.Sprintf("%d.", item.number))
	styledTitle := taskGroupStyle.Render(item.title)

	// Progress display: [done/total] ██░░ for groups with tasks, (0) for empty groups
	var progressLabel string
	if group != nil && len(group.Tasks) > 0 {
		done, total := groupProgress(*group)
		bar := renderProgressBar(done, total)
		progressLabel = dimStyle.Render(fmt.Sprintf("[%d/%d]", done, total)) + " " + dimStyle.Render(bar)
	} else {
		progressLabel = dimStyle.Render("(0)")
	}

	line := fmt.Sprintf("%s%s %s %s %s", marker, styledChevron, groupNum, styledTitle, progressLabel)

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

// groupProgress counts the number of done-equivalent tasks in a group.
// Returns (done, total) where done includes tasks with status "done", "closed", or "completed".
func groupProgress(group task.TaskGroup) (done int, total int) {
	total = len(group.Tasks)
	for _, t := range group.Tasks {
		if isDoneStatus(t.Status) {
			done++
		}
	}
	return done, total
}

// isDoneStatus returns true if the status represents a completed task.
func isDoneStatus(status string) bool {
	lower := strings.ToLower(status)
	return lower == "done" || lower == "closed" || lower == "completed"
}

// progressBarWidth is the number of characters in the mini progress bar.
const progressBarWidth = 4

// renderProgressBar produces a 4-character progress bar using █ (filled) and ░ (empty).
// Returns an empty string if total is 0.
func renderProgressBar(done, total int) string {
	if total == 0 {
		return ""
	}
	filled := done * progressBarWidth / total
	if filled > progressBarWidth {
		filled = progressBarWidth
	}
	// Ensure at least 1 filled block when there's partial progress
	if done > 0 && filled == 0 {
		filled = 1
	}
	empty := progressBarWidth - filled
	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}
