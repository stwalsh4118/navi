package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	ansi "github.com/charmbracelet/x/ansi"

	"github.com/stwalsh4118/navi/internal/pm"
)

const (
	pmZoneProjects = iota
	pmZoneEvents
)

const (
	pmMinTerminalWidth        = 80
	pmShortTerminalHeight     = 30
	pmVeryShortTerminalHeight = 15
	pmNarrowWidthThreshold    = 100
	pmEventPageScrollAmt      = 8
)

func (m *Model) togglePMView() {
	if m.pmViewVisible {
		m.pmViewVisible = false
		return
	}

	m.pmViewVisible = true
	m.pmZoneFocus = pmZoneProjects
	m.pmProjectCursor = 0
	m.pmProjectScrollOffset = 0
	m.pmEventScrollOffset = 0

	// PM view is mutually exclusive with task panel and preview.
	m.taskPanelVisible = false
	m.taskPanelUserEnabled = false
	m.taskPanelFocused = false
	m.clearTaskSearchState()

	m.previewVisible = false
	m.previewUserEnabled = false
	m.previewFocused = false
	m.previewContent = ""

	m.clearSearchState()
}

func (m Model) updatePMView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "P", "esc":
		m.pmViewVisible = false
		return m, nil
	case "tab":
		if m.pmZoneFocus == pmZoneProjects {
			m.pmZoneFocus = pmZoneEvents
		} else {
			m.pmZoneFocus = pmZoneProjects
		}
		return m, nil
	}

	if m.pmZoneFocus == pmZoneProjects {
		switch msg.String() {
		case "up", "k":
			m.movePMProjectCursor(-1)
		case "down", "j":
			m.movePMProjectCursor(1)
		case "enter":
			m.pmSelectCurrentProject()
		case " ":
			m.togglePMProjectExpansion()
		}
		return m, nil
	}

	maxScroll := m.pmMaxEventScroll()
	switch msg.String() {
	case "up", "k", "j":
		if msg.String() == "j" {
			m.pmEventScrollOffset++
		} else {
			m.pmEventScrollOffset--
		}
	case "down":
		m.pmEventScrollOffset++
	case "pgup":
		m.pmEventScrollOffset -= pmEventPageScrollAmt
	case "pgdown":
		m.pmEventScrollOffset += pmEventPageScrollAmt
	case "g":
		m.pmEventScrollOffset = 0
	case "G":
		m.pmEventScrollOffset = maxScroll
	}

	if m.pmEventScrollOffset < 0 {
		m.pmEventScrollOffset = 0
	}
	if m.pmEventScrollOffset > maxScroll {
		m.pmEventScrollOffset = maxScroll
	}

	return m, nil
}

func (m *Model) movePMProjectCursor(delta int) {
	snapshots := m.sortedPMSnapshots()
	if len(snapshots) == 0 {
		m.pmProjectCursor = 0
		m.pmProjectScrollOffset = 0
		return
	}

	m.pmProjectCursor += delta
	if m.pmProjectCursor < 0 {
		m.pmProjectCursor = 0
	}
	if m.pmProjectCursor >= len(snapshots) {
		m.pmProjectCursor = len(snapshots) - 1
	}

	maxVisible := m.pmProjectsVisibleRows()
	if maxVisible < 1 {
		maxVisible = 1
	}
	if m.pmProjectCursor < m.pmProjectScrollOffset {
		m.pmProjectScrollOffset = m.pmProjectCursor
	}
	if m.pmProjectCursor >= m.pmProjectScrollOffset+maxVisible {
		m.pmProjectScrollOffset = m.pmProjectCursor - maxVisible + 1
	}

	maxScroll := len(snapshots) - maxVisible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.pmProjectScrollOffset > maxScroll {
		m.pmProjectScrollOffset = maxScroll
	}
}

func (m *Model) togglePMProjectExpansion() {
	snapshots := m.sortedPMSnapshots()
	if len(snapshots) == 0 || m.pmProjectCursor >= len(snapshots) {
		return
	}
	if m.pmExpandedProjects == nil {
		m.pmExpandedProjects = make(map[string]bool)
	}
	projectDir := snapshots[m.pmProjectCursor].ProjectDir
	if projectDir == "" {
		return
	}
	m.pmExpandedProjects[projectDir] = !m.pmExpandedProjects[projectDir]
}

func (m *Model) pmSelectCurrentProject() {
	snapshots := m.sortedPMSnapshots()
	if len(snapshots) == 0 || m.pmProjectCursor >= len(snapshots) {
		return
	}
	projectDir := snapshots[m.pmProjectCursor].ProjectDir
	if projectDir == "" {
		return
	}

	m.pmViewVisible = false
	m.pmProjectFilterDir = projectDir
	m.cursor = 0
	m.sessionScrollOffset = 0
}

func (m Model) renderPMView(width, height int) string {
	if width < pmMinTerminalWidth {
		message := dimStyle.Render("Terminal too narrow for PM view (minimum 80 columns)")
		return pmBoxStyle.Width(width - 2).Render(message)
	}

	briefingH, projectsH, eventsH := pmZoneHeights(height)
	sections := make([]string, 0, 3)
	if briefingH > 0 {
		sections = append(sections, m.renderPMBriefing(width, briefingH))
	}
	sections = append(sections, m.renderPMProjects(width, projectsH))
	sections = append(sections, m.renderPMEvents(width, eventsH))

	return strings.Join(sections, "\n")
}

func pmZoneHeights(contentHeight int) (int, int, int) {
	if contentHeight <= 0 {
		return 0, 0, 0
	}

	if contentHeight < pmVeryShortTerminalHeight {
		projectsH := (contentHeight * 40) / 100
		if projectsH < 1 {
			projectsH = 1
		}
		eventsH := contentHeight - projectsH
		if eventsH < 1 {
			eventsH = 1
			projectsH = contentHeight - eventsH
		}
		return 0, projectsH, eventsH
	}

	if contentHeight < pmShortTerminalHeight {
		briefingH := 1
		remaining := contentHeight - briefingH
		projectsH := (remaining * 40) / 100
		if projectsH < 1 {
			projectsH = 1
		}
		eventsH := remaining - projectsH
		if eventsH < 1 {
			eventsH = 1
			projectsH = remaining - eventsH
		}
		return briefingH, projectsH, eventsH
	}

	briefingH := (contentHeight * 30) / 100
	projectsH := (contentHeight * 30) / 100
	eventsH := contentHeight - briefingH - projectsH
	if briefingH < 1 {
		briefingH = 1
	}
	if projectsH < 1 {
		projectsH = 1
	}
	if eventsH < 1 {
		eventsH = 1
	}
	return briefingH, projectsH, eventsH
}

func (m Model) renderPMBriefing(width, height int) string {
	placeholder := dimStyle.Italic(true).Render("No PM briefing yet")
	if m.pmLastError != "" {
		placeholder = redStyle.Render("PM refresh error: " + m.pmLastError)
	}
	if height <= 1 {
		return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(placeholder)
	}
	contentHeight := height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}
	content := lipgloss.NewStyle().Width(width-4).Height(contentHeight).Align(lipgloss.Center, lipgloss.Center).Render(placeholder)
	return pmBoxStyle.Width(width - 2).Render(content)
}

func (m Model) renderPMProjects(width, height int) string {
	if height <= 0 {
		return ""
	}

	boxStyleToUse := pmBoxStyle
	if m.pmZoneFocus == pmZoneProjects {
		boxStyleToUse = pmFocusedBoxStyle
	}

	projectRows := m.sortedPMSnapshots()
	if len(projectRows) == 0 {
		empty := taskEmptyStyle.Render("No projects detected")
		return boxStyleToUse.Width(width - 2).Render(empty)
	}

	cursor := m.pmProjectCursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(projectRows) {
		cursor = len(projectRows) - 1
	}

	maxVisible := height - 2
	if maxVisible < 1 {
		maxVisible = 1
	}
	offset := m.pmProjectScrollOffset
	if offset < 0 {
		offset = 0
	}
	if offset > len(projectRows)-1 {
		offset = len(projectRows) - 1
	}
	if cursor < offset {
		offset = cursor
	}
	if cursor >= offset+maxVisible {
		offset = cursor - maxVisible + 1
	}

	hasAbove := offset > 0
	lines := make([]string, 0, maxVisible)
	if hasAbove {
		lines = append(lines, dimStyle.Render(fmt.Sprintf(scrollIndicatorAbove, offset)))
	}

	slotsRemaining := maxVisible - len(lines)
	if slotsRemaining < 0 {
		slotsRemaining = 0
	}

	nextRowIndex := offset
	for nextRowIndex < len(projectRows) && slotsRemaining > 0 {
		i := nextRowIndex
		isSelected := i == cursor
		row := m.renderPMProjectRow(projectRows[i], width-4)
		if isSelected {
			row = selectedStyle.Render(row)
		}
		lines = append(lines, row)
		slotsRemaining--

		if isSelected && slotsRemaining > 0 && m.pmExpandedProjects[projectRows[i].ProjectDir] {
			details := m.renderPMProjectExpansion(projectRows[i], width-4)
			for _, detail := range details {
				if slotsRemaining == 0 {
					break
				}
				lines = append(lines, detail)
				slotsRemaining--
			}
		}

		nextRowIndex++
	}

	if nextRowIndex < len(projectRows) {
		below := len(projectRows) - nextRowIndex
		indicator := dimStyle.Render(fmt.Sprintf(scrollIndicatorBelow, below))
		if len(lines) < maxVisible {
			lines = append(lines, indicator)
		} else if len(lines) > 0 {
			lines[len(lines)-1] = indicator
		}
	}

	return boxStyleToUse.Width(width - 2).Render(strings.Join(lines, "\n"))
}

func (m Model) renderPMProjectRow(snapshot pm.ProjectSnapshot, width int) string {
	projectName := snapshot.ProjectName
	if width < pmNarrowWidthThreshold {
		projectName = truncate(projectName, 20)
	}
	projectName = boldStyle.Render(projectName)

	pbiText := "No active PBI"
	if snapshot.CurrentPBIID != "" {
		pbiText = snapshot.CurrentPBIID
		if snapshot.CurrentPBITitle != "" {
			pbiText += ": " + snapshot.CurrentPBITitle
		}
	}
	pbiMax := 40
	if width < 130 {
		pbiMax = 30
	}
	if width < pmNarrowWidthThreshold {
		pbiMax = 22
	}
	pbiText = truncate(pbiText, pbiMax)

	progress := fmt.Sprintf("%d/%d tasks", snapshot.TaskCounts.Done, snapshot.TaskCounts.Total)
	status := snapshot.SessionStatus
	if status == "" {
		status = "unknown"
	}

	parts := []string{StatusIcon(status), projectName, pbiText, progress, status}
	if width >= pmMinTerminalWidth {
		parts = append(parts, fmt.Sprintf("%d sess", snapshot.SessionCount))
	}
	if width >= pmNarrowWidthThreshold {
		parts = append(parts, formatPMRelativeTime(snapshot.LastActivity))
	}

	joined := strings.Join(parts, "  ")
	if lipgloss.Width(joined) > width {
		joined = pmTruncateANSI(joined, width)
	}
	return joined
}

func (m Model) renderPMProjectExpansion(snapshot pm.ProjectSnapshot, width int) []string {
	lines := []string{
		dimStyle.Render("  branch: " + snapshot.Branch + pmDirtySuffix(snapshot.Dirty)),
		dimStyle.Render("  commits: " + pmCommitSummary(snapshot.HeadSHA, snapshot.CommitsAhead)),
	}
	if snapshot.PRNumber > 0 {
		lines = append(lines, dimStyle.Render("  pr: #"+strconv.Itoa(snapshot.PRNumber)))
	}

	if result, ok := m.pmTaskResults[snapshot.ProjectDir]; ok && result != nil {
		tasks := result.AllTasks()
		if len(tasks) > 0 {
			lines = append(lines, dimStyle.Render("  tasks:"))
			maxTasks := 3
			if len(tasks) < maxTasks {
				maxTasks = len(tasks)
			}
			for i := 0; i < maxTasks; i++ {
				line := "    - " + TaskStatusBadge(tasks[i].Status) + " " + truncate(tasks[i].Title, width-14)
				lines = append(lines, line)
			}
		}
	}

	for i := range lines {
		if lipgloss.Width(lines[i]) > width {
			lines[i] = pmTruncateANSI(lines[i], width)
		}
	}
	return lines
}

func (m Model) renderPMEvents(width, height int) string {
	if height <= 0 {
		return ""
	}

	boxStyleToUse := pmBoxStyle
	if m.pmZoneFocus == pmZoneEvents {
		boxStyleToUse = pmFocusedBoxStyle
	}

	events := m.sortedPMEvents()
	if len(events) == 0 {
		empty := taskEmptyStyle.Render("No events yet")
		return boxStyleToUse.Width(width - 2).Render(empty)
	}

	maxVisible := height - 2
	if maxVisible < 1 {
		maxVisible = 1
	}
	offset := m.pmEventScrollOffset
	maxScroll := len(events) - maxVisible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if offset < 0 {
		offset = 0
	}
	if offset > maxScroll {
		offset = maxScroll
	}

	hasAbove := offset > 0
	available := maxVisible
	if hasAbove {
		available--
	}
	end := offset + available
	if end > len(events) {
		end = len(events)
	}
	hasBelow := end < len(events)
	if hasBelow {
		available--
		end = offset + available
		if end > len(events) {
			end = len(events)
		}
	}

	lines := make([]string, 0, maxVisible)
	if hasAbove {
		lines = append(lines, dimStyle.Render(fmt.Sprintf(scrollIndicatorAbove, offset)))
	}
	for _, event := range events[offset:end] {
		line := m.renderPMEventRow(event, width-4)
		lines = append(lines, line)
	}
	if hasBelow {
		lines = append(lines, dimStyle.Render(fmt.Sprintf(scrollIndicatorBelow, len(events)-end)))
	}

	return boxStyleToUse.Width(width - 2).Render(strings.Join(lines, "\n"))
}

func (m Model) renderPMEventRow(event pm.Event, width int) string {
	timePart := dimStyle.Render(formatPMRelativeTime(event.Timestamp))
	typePart := pmEventTypeLabel(event.Type)
	projectPart := dimStyle.Render(event.ProjectName)
	detail := dimStyle.Render(pmEventDetail(event))
	line := strings.Join([]string{timePart, typePart, projectPart, detail}, "  ")
	if lipgloss.Width(line) > width {
		line = pmTruncateANSI(line, width)
	}
	return line
}

func pmTruncateANSI(s string, width int) string {
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(s, width, "")
}

func pmEventTypeLabel(eventType pm.EventType) string {
	label := string(eventType)
	switch eventType {
	case pm.EventTaskCompleted, pm.EventTaskStarted, pm.EventPBICompleted:
		return greenStyle.Render(label)
	case pm.EventSessionStatusChange:
		return yellowStyle.Render(label)
	case pm.EventCommit, pm.EventBranchCreated, pm.EventPRCreated:
		return cyanStyle.Render(label)
	default:
		return dimStyle.Render(label)
	}
}

func pmEventDetail(event pm.Event) string {
	payload := event.Payload
	if payload == nil {
		return ""
	}

	switch event.Type {
	case pm.EventTaskCompleted:
		return fmt.Sprintf("done %s->%s", payload["old_done"], payload["new_done"])
	case pm.EventTaskStarted:
		return fmt.Sprintf("in-progress %s->%s", payload["old_in_progress"], payload["new_in_progress"])
	case pm.EventSessionStatusChange:
		return payload["old_status"] + " -> " + payload["new_status"]
	case pm.EventCommit:
		commitLines := strings.Split(payload["commits"], "\n")
		if len(commitLines) > 0 && strings.TrimSpace(commitLines[0]) != "" {
			return commitLines[0]
		}
		return shortSHA(payload["new_head_sha"])
	case pm.EventBranchCreated:
		return payload["old_branch"] + " -> " + payload["new_branch"]
	case pm.EventPRCreated:
		if payload["pr_number"] != "" {
			return "PR #" + payload["pr_number"]
		}
	case pm.EventPBICompleted:
		if payload["pbi_id"] != "" {
			return payload["pbi_id"] + " completed"
		}
	}

	return ""
}

func formatPMRelativeTime(ts time.Time) string {
	if ts.IsZero() {
		return "unknown"
	}
	return formatAge(ts.Unix())
}

func pmDirtySuffix(dirty bool) string {
	if dirty {
		return " (dirty)"
	}
	return ""
}

func pmCommitSummary(headSHA string, ahead int) string {
	head := shortSHA(headSHA)
	if head == "" {
		head = "unknown"
	}
	return fmt.Sprintf("%s (+%d)", head, ahead)
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func (m Model) sortedPMSnapshots() []pm.ProjectSnapshot {
	if m.pmOutput == nil || len(m.pmOutput.Snapshots) == 0 {
		return nil
	}
	snapshots := append([]pm.ProjectSnapshot(nil), m.pmOutput.Snapshots...)
	sort.SliceStable(snapshots, func(i, j int) bool {
		iRank := pmAttentionRank(snapshots[i].SessionStatus)
		jRank := pmAttentionRank(snapshots[j].SessionStatus)
		if iRank != jRank {
			return iRank < jRank
		}
		return snapshots[i].LastActivity.After(snapshots[j].LastActivity)
	})
	return snapshots
}

func (m Model) sortedPMEvents() []pm.Event {
	if m.pmOutput == nil || len(m.pmOutput.Events) == 0 {
		return nil
	}
	events := append([]pm.Event(nil), m.pmOutput.Events...)
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Timestamp.After(events[j].Timestamp)
	})
	return events
}

func pmAttentionRank(status string) int {
	switch status {
	case "permission":
		return 0
	case "error":
		return 1
	case "waiting":
		return 2
	case "working":
		return 3
	case "idle":
		return 4
	case "offline":
		return 5
	default:
		return 6
	}
}

func (m Model) pmProjectsVisibleRows() int {
	contentHeight := m.height - 8
	_, projectsH, _ := pmZoneHeights(contentHeight)
	maxVisible := projectsH - 2
	if maxVisible < 1 {
		maxVisible = 1
	}
	return maxVisible
}

func (m Model) pmMaxEventScroll() int {
	contentHeight := m.height - 8
	_, _, eventsH := pmZoneHeights(contentHeight)
	maxVisible := eventsH - 2
	if maxVisible < 1 {
		maxVisible = 1
	}
	events := m.sortedPMEvents()
	maxScroll := len(events) - maxVisible
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}
