package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

// Selection marker constants
const (
	selectedMarker   = "▸ "
	unselectedMarker = "  "
	rowIndent        = "      " // 6 spaces for indented lines
)

// renderSession renders a single session row with icon, name, age, cwd, and message.
func (m Model) renderSession(s SessionInfo, selected bool, width int) string {
	var b strings.Builder

	// Selection marker
	marker := unselectedMarker
	if selected {
		marker = selectedMarker
	}

	// First line: marker + icon + name + [remote] + age
	icon := statusIcon(s.Status)
	name := boldStyle.Render(s.TmuxSession)

	// Add remote label if this is a remote session
	remoteLabel := ""
	if s.Remote != "" {
		remoteLabel = " " + dimStyle.Render(fmt.Sprintf("[%s]", s.Remote))
	}

	age := formatAge(s.Timestamp)

	// Calculate padding for right-aligned age
	firstLine := fmt.Sprintf("%s%s  %s%s", marker, icon, name, remoteLabel)
	padding := width - lipgloss.Width(firstLine) - len(age) - 2
	if padding < 1 {
		padding = 1
	}

	b.WriteString(firstLine)
	b.WriteString(strings.Repeat(" ", padding))
	b.WriteString(age)
	b.WriteString("\n")

	// Second line: working directory (indented, dimmed)
	cwd := shortenPath(s.CWD)
	b.WriteString(rowIndent)
	b.WriteString(dimStyle.Render(cwd))

	// Third line: git info if present (indented)
	if s.Git != nil {
		b.WriteString("\n")
		b.WriteString(rowIndent)
		b.WriteString(renderGitInfo(s.Git, width-len(rowIndent)))
	}

	// Metrics badges line if metrics present (indented, dimmed)
	if s.Metrics != nil {
		metricsLine := renderMetricsBadges(s.Metrics)
		if metricsLine != "" {
			b.WriteString("\n")
			b.WriteString(rowIndent)
			b.WriteString(dimStyle.Render(metricsLine))
		}
	}

	// Message line if present (indented, dimmed/italic)
	if s.Message != "" {
		b.WriteString("\n")
		msg := truncate(s.Message, width-len(rowIndent))
		b.WriteString(rowIndent)
		b.WriteString(dimStyle.Render(italicStyle.Render(fmt.Sprintf("\"%s\"", msg))))
	}

	return b.String()
}

// renderMetricsBadges returns a compact metrics display string.
// Format: "⏱ 1h 23m  🔧 45  📊 57k"
func renderMetricsBadges(metrics *Metrics) string {
	if metrics == nil {
		return ""
	}

	var parts []string

	// Time badge
	if metrics.Time != nil && metrics.Time.TotalSeconds > 0 {
		duration := formatDuration(metrics.Time.TotalSeconds)
		parts = append(parts, fmt.Sprintf("⏱ %s", duration))
	}

	// Tool count badge
	toolCount := formatToolCount(metrics.Tools)
	if toolCount > 0 {
		parts = append(parts, fmt.Sprintf("🔧 %d", toolCount))
	}

	// Token count badge
	if metrics.Tokens != nil && metrics.Tokens.Total > 0 {
		parts = append(parts, fmt.Sprintf("📊 %s", formatTokenCount(metrics.Tokens.Total)))
	}

	return strings.Join(parts, "  ")
}

// renderGitInfo renders git status info with appropriate coloring.
// Format: "branch-name ● +3 -1 [PR#42]"
func renderGitInfo(git *GitInfo, maxWidth int) string {
	if git == nil {
		return ""
	}

	var parts []string

	// Branch name in cyan (truncate if too long)
	branch := git.Branch
	if len(branch) > gitMaxBranchLength {
		branch = branch[:gitMaxBranchLength-3] + "..."
	}
	parts = append(parts, cyanStyle.Render(branch))

	// Dirty indicator in yellow
	if git.Dirty {
		parts = append(parts, yellowStyle.Render(gitDirtyIndicator))
	}

	// Ahead count in green
	if git.Ahead > 0 {
		parts = append(parts, greenStyle.Render(fmt.Sprintf("%s%d", gitAheadPrefix, git.Ahead)))
	}

	// Behind count in red
	if git.Behind > 0 {
		parts = append(parts, redStyle.Render(fmt.Sprintf("%s%d", gitBehindPrefix, git.Behind)))
	}

	// PR number if detected via gh CLI
	if git.PRNum > 0 {
		parts = append(parts, dimStyle.Render(fmt.Sprintf("[%s%d]", gitPRPrefix, git.PRNum)))
	}

	result := strings.Join(parts, " ")

	// Note: We don't truncate here since colors add ANSI codes that affect length calculation
	// The branch truncation above handles the main size concern

	return result
}

// formatAge formats a Unix timestamp as a human-readable age string.
func formatAge(timestamp int64) string {
	elapsed := time.Since(time.Unix(timestamp, 0))
	if elapsed < time.Minute {
		return fmt.Sprintf("%ds ago", int(elapsed.Seconds()))
	}
	if elapsed < time.Hour {
		return fmt.Sprintf("%dm ago", int(elapsed.Minutes()))
	}
	return fmt.Sprintf("%dh ago", int(elapsed.Hours()))
}

// shortenPath replaces the home directory prefix with ~.
func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// truncate truncates a string to maxLen characters, adding ellipsis if needed.
func truncate(s string, maxLen int) string {
	if maxLen < 4 {
		maxLen = 4
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Header constant
const headerTitle = "Claude Sessions"

// renderHeader renders the header box with title, session count, and remote status.
func (m Model) renderHeader() string {
	// Count local and remote sessions
	localCount, remoteCount := 0, 0
	for _, s := range m.sessions {
		if s.Remote == "" {
			localCount++
		} else {
			remoteCount++
		}
	}

	// Build count string
	var countStr string
	if remoteCount > 0 {
		countStr = fmt.Sprintf("%d local, %d remote", localCount, remoteCount)
	} else {
		countStr = fmt.Sprintf("%d active", len(m.sessions))
	}

	// Build remote status indicators if remotes are configured
	remoteStatus := ""
	if m.sshPool != nil && len(m.remotes) > 0 {
		var indicators []string
		for _, remote := range m.remotes {
			status := m.sshPool.GetStatus(remote.Name)
			var indicator string
			switch status.Status {
			case StatusConnected:
				indicator = greenStyle.Render(fmt.Sprintf("[%s:✓]", remote.Name))
			case StatusError:
				indicator = redStyle.Render(fmt.Sprintf("[%s:✗]", remote.Name))
			default:
				indicator = dimStyle.Render(fmt.Sprintf("[%s:-]", remote.Name))
			}
			indicators = append(indicators, indicator)
		}
		remoteStatus = " " + strings.Join(indicators, " ")
	}

	// Build aggregate metrics string
	aggregateStr := ""
	aggregate := AggregateMetrics(m.sessions)
	if aggregate != nil {
		var parts []string
		if aggregate.Tokens != nil && aggregate.Tokens.Total > 0 {
			parts = append(parts, fmt.Sprintf("📊 %s", formatTokenCount(aggregate.Tokens.Total)))
		}
		if aggregate.Time != nil && aggregate.Time.TotalSeconds > 0 {
			parts = append(parts, fmt.Sprintf("⏱ %s", formatDuration(aggregate.Time.TotalSeconds)))
		}
		toolCount := formatToolCount(aggregate.Tools)
		if toolCount > 0 {
			parts = append(parts, fmt.Sprintf("🔧 %d", toolCount))
		}
		if len(parts) > 0 {
			aggregateStr = "  " + strings.Join(parts, " ")
		}
	}

	// Calculate padding for count on right
	// Account for box border padding (1 on each side)
	contentWidth := m.width - 4
	leftPart := headerTitle + remoteStatus + aggregateStr
	padding := contentWidth - lipgloss.Width(leftPart) - lipgloss.Width(countStr)
	if padding < 1 {
		padding = 1
	}

	content := leftPart + strings.Repeat(" ", padding) + countStr
	return boxStyle.Width(m.width - 2).Render(content)
}

// renderFooter renders the footer box with keybinding help.
func (m Model) renderFooter() string {
	var parts []string

	// Check if currently selected session is remote
	isRemoteSelected := false
	filteredSessions := m.getFilteredSessions()
	if len(filteredSessions) > 0 && m.cursor < len(filteredSessions) {
		isRemoteSelected = filteredSessions[m.cursor].Remote != ""
	}

	// Always show these
	parts = append(parts, "↑/↓ nav", "⏎ attach", "p preview")

	// Show preview-specific keys only when preview is visible
	if m.previewVisible {
		parts = append(parts, "L layout", "W wrap", "[/] resize")
	}

	// Show filter option with current state if remotes are configured
	if len(m.remotes) > 0 {
		filterLabel := fmt.Sprintf("f filter:%s", m.filterModeString())
		parts = append(parts, filterLabel)
	}

	// Show local-only options only when a local session is selected
	if !isRemoteSelected {
		parts = append(parts, "d dismiss", "n new", "x kill", "R rename", "G git")
	} else {
		parts = append(parts, "n new") // new session is always available
	}

	parts = append(parts, "r refresh", "q quit")

	footerHelp := strings.Join(parts, "  ")
	return boxStyle.Width(m.width - 2).Render(footerHelp)
}

// Dialog width constant
const dialogWidth = 56

// renderPreview renders the preview pane showing captured tmux output.
// Returns empty string if preview is not visible.
func (m Model) renderPreview(width, height int) string {
	if !m.previewVisible {
		return ""
	}

	var b strings.Builder

	// Get session name for header
	filteredSessions := m.getFilteredSessions()
	sessionName := ""
	if len(filteredSessions) > 0 && m.cursor < len(filteredSessions) {
		sessionName = filteredSessions[m.cursor].TmuxSession
	}

	// Build header with session name
	if sessionName != "" {
		b.WriteString(previewHeaderStyle.Render("─ " + sessionName + " "))
	}
	b.WriteString("\n")

	// Content area
	contentWidth := width - 4 // Account for box padding and borders
	if contentWidth < 10 {
		contentWidth = 10
	}

	if m.previewContent == "" {
		b.WriteString(previewEmptyStyle.Render("No preview available"))
	} else {
		var content string
		if m.previewWrap {
			// Wrap content to fit width
			content = wordwrap.String(m.previewContent, contentWidth)
		} else {
			// Truncate long lines
			lines := strings.Split(m.previewContent, "\n")
			for i, line := range lines {
				if lipgloss.Width(line) > contentWidth {
					lines[i] = truncate(line, contentWidth)
				}
			}
			content = strings.Join(lines, "\n")
		}

		// Calculate max content lines: total height - borders(2) - header(1)
		maxLines := height - 3
		if maxLines < 1 {
			maxLines = 1
		}

		// Split into lines and limit to available height
		lines := strings.Split(content, "\n")
		if len(lines) > maxLines {
			lines = lines[len(lines)-maxLines:]
		}

		b.WriteString(strings.Join(lines, "\n"))
	}

	// Render the box
	boxContent := b.String()
	rendered := previewBoxStyle.Width(width - 2).Render(boxContent)

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

// renderSessionList renders the session list portion of the view.
func (m Model) renderSessionList(width int) string {
	var b strings.Builder

	filteredSessions := m.getFilteredSessions()
	if len(filteredSessions) == 0 {
		// Show appropriate message based on filter state
		var message string
		if len(m.sessions) == 0 {
			message = noSessionsMessage
		} else {
			message = fmt.Sprintf("  No %s sessions", m.filterModeString())
		}
		b.WriteString(dimStyle.Render(message))
		b.WriteString("\n")
	} else {
		for i, session := range filteredSessions {
			selected := i == m.cursor
			b.WriteString(m.renderSession(session, selected, width))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// gitDetailWidth is the width of the git detail dialog
const gitDetailWidth = 70

// renderGitDetailView renders the git detail view dialog.
func (m Model) renderGitDetailView() string {
	var b strings.Builder

	// Title
	b.WriteString(dialogTitleStyle.Render("Git Information"))
	b.WriteString("\n\n")

	if m.sessionToModify == nil {
		b.WriteString(dimStyle.Render("No session selected"))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("Esc: close"))
		content := b.String()
		dialog := dialogBoxStyle.Width(gitDetailWidth).Render(content)
		return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, dialog)
	}

	session := m.sessionToModify

	// Session name
	b.WriteString(fmt.Sprintf("Session: %s\n", boldStyle.Render(session.TmuxSession)))
	b.WriteString(fmt.Sprintf("CWD: %s\n\n", dimStyle.Render(session.CWD)))

	// Check if git info is available
	if session.Git == nil {
		b.WriteString(dimStyle.Render("Not a git repository"))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("Esc: close"))
		content := b.String()
		dialog := dialogBoxStyle.Width(gitDetailWidth).Render(content)
		return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, dialog)
	}

	git := session.Git

	// Branch info
	b.WriteString(fmt.Sprintf("Branch: %s", cyanStyle.Render(git.Branch)))
	if git.Dirty {
		b.WriteString(fmt.Sprintf(" %s", yellowStyle.Render(gitDirtyIndicator+" (uncommitted changes)")))
	} else {
		b.WriteString(fmt.Sprintf(" %s", greenStyle.Render("(clean)")))
	}
	b.WriteString("\n")

	// Ahead/behind info
	if git.Ahead > 0 || git.Behind > 0 {
		b.WriteString("Status: ")
		if git.Ahead > 0 {
			b.WriteString(greenStyle.Render(fmt.Sprintf("%d ahead", git.Ahead)))
		}
		if git.Ahead > 0 && git.Behind > 0 {
			b.WriteString(", ")
		}
		if git.Behind > 0 {
			b.WriteString(redStyle.Render(fmt.Sprintf("%d behind", git.Behind)))
		}
		b.WriteString(" remote\n")
	}

	// Last commit
	if git.LastCommit != "" {
		b.WriteString(fmt.Sprintf("Last commit: %s\n", dimStyle.Render(git.LastCommit)))
	}

	// Remote URL
	if git.Remote != "" {
		b.WriteString(fmt.Sprintf("Remote: %s\n", dimStyle.Render(git.Remote)))

		// GitHub info if available
		ghInfo := parseGitHubRemote(git.Remote)
		if ghInfo != nil {
			b.WriteString(fmt.Sprintf("GitHub: %s/%s\n", ghInfo.Owner, ghInfo.Repo))
		}
	}

	// PR number if detected via gh CLI
	if git.PRNum > 0 {
		b.WriteString(fmt.Sprintf("Pull Request: %s\n", highlightStyle.Render(fmt.Sprintf("#%d", git.PRNum))))

		// Show URL if we have GitHub info
		ghInfo := parseGitHubRemote(git.Remote)
		if ghInfo != nil {
			url := ghInfo.PRURL(git.PRNum)
			b.WriteString(fmt.Sprintf("Link: %s\n", dimStyle.Render(url)))
		}
	}

	b.WriteString("\n")

	// Keybindings
	var keys []string
	keys = append(keys, "d: diff")
	if git.PRNum > 0 && git.Remote != "" {
		keys = append(keys, "o: open PR")
	}
	keys = append(keys, "Esc: close")
	b.WriteString(dimStyle.Render(strings.Join(keys, "  ")))

	content := b.String()
	dialog := dialogBoxStyle.Width(gitDetailWidth).Render(content)
	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, dialog)
}

// renderGitDiffView renders the git diff view dialog.
func (m Model) renderGitDiffView() string {
	var b strings.Builder

	// Title
	b.WriteString(dialogTitleStyle.Render("Git Changes"))
	b.WriteString("\n\n")

	if m.sessionToModify == nil || m.sessionToModify.Git == nil {
		b.WriteString(dimStyle.Render("No git information available"))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("Esc: back"))
		content := b.String()
		dialog := dialogBoxStyle.Width(gitDetailWidth).Render(content)
		return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, dialog)
	}

	session := m.sessionToModify
	git := session.Git

	// Show branch and dirty status
	b.WriteString(fmt.Sprintf("Branch: %s", cyanStyle.Render(git.Branch)))
	if git.Dirty {
		b.WriteString(fmt.Sprintf(" %s", yellowStyle.Render(gitDirtyIndicator)))
	}
	b.WriteString("\n\n")

	// Get diff stat
	dir := expandPath(session.CWD)
	diffStat := getGitDiffStat(dir)

	if diffStat == "" {
		if git.Dirty {
			// Dirty but no diff - might be staged changes only
			b.WriteString(dimStyle.Render("No unstaged changes (changes may be staged)"))
		} else {
			b.WriteString(greenStyle.Render("Working tree clean - no changes"))
		}
	} else {
		// Show the diff stat output
		b.WriteString(boldStyle.Render("Changed files:"))
		b.WriteString("\n")

		// Split and render each line of diff stat
		lines := strings.Split(diffStat, "\n")
		maxDisplayLines := 20 // Limit lines to keep dialog manageable
		displayLines := lines
		if len(lines) > maxDisplayLines {
			displayLines = lines[:maxDisplayLines-1]
			displayLines = append(displayLines, dimStyle.Render(fmt.Sprintf("... and %d more files", len(lines)-maxDisplayLines+1)))
		}

		for _, line := range displayLines {
			// Color insertions green, deletions red
			if strings.Contains(line, "|") {
				// File stat line: "filename | 10 ++--"
				parts := strings.SplitN(line, "|", 2)
				if len(parts) == 2 {
					filename := parts[0]
					stats := parts[1]
					// Color the + and - characters
					stats = strings.ReplaceAll(stats, "+", greenStyle.Render("+"))
					stats = strings.ReplaceAll(stats, "-", redStyle.Render("-"))
					b.WriteString(fmt.Sprintf("%s|%s\n", filename, stats))
				} else {
					b.WriteString(line + "\n")
				}
			} else if strings.Contains(line, "insertion") || strings.Contains(line, "deletion") {
				// Summary line
				b.WriteString(dimStyle.Render(line) + "\n")
			} else {
				b.WriteString(line + "\n")
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Esc: back to git info"))

	content := b.String()
	dialog := dialogBoxStyle.Width(gitDetailWidth).Render(content)
	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, dialog)
}

// metricsDetailWidth is the width of the metrics detail dialog
const metricsDetailWidth = 60

// renderMetricsDetailView renders the metrics detail view dialog.
func (m Model) renderMetricsDetailView() string {
	var b strings.Builder

	// Title
	b.WriteString(dialogTitleStyle.Render("Session Metrics"))
	b.WriteString("\n\n")

	if m.sessionToModify == nil {
		b.WriteString(dimStyle.Render("No session selected"))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("Esc: close"))
		content := b.String()
		dialog := dialogBoxStyle.Width(metricsDetailWidth).Render(content)
		return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, dialog)
	}

	session := m.sessionToModify

	// Session name
	b.WriteString(fmt.Sprintf("Session: %s\n", boldStyle.Render(session.TmuxSession)))
	b.WriteString(fmt.Sprintf("Status: %s %s\n", statusIcon(session.Status), session.Status))
	b.WriteString("\n")

	// Check if metrics is available
	if session.Metrics == nil {
		b.WriteString(dimStyle.Render("No metrics data available"))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("Esc: close"))
		content := b.String()
		dialog := dialogBoxStyle.Width(metricsDetailWidth).Render(content)
		return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, dialog)
	}

	metrics := session.Metrics

	// Token metrics section
	b.WriteString(boldStyle.Render("Token Usage"))
	b.WriteString("\n")

	if metrics.Tokens != nil && metrics.Tokens.Total > 0 {
		b.WriteString(fmt.Sprintf("  Total: %s\n", formatTokenCount(metrics.Tokens.Total)))
		b.WriteString(fmt.Sprintf("  Input: %s\n", formatTokenCount(metrics.Tokens.Input)))
		b.WriteString(fmt.Sprintf("  Output: %s\n", formatTokenCount(metrics.Tokens.Output)))
	} else {
		b.WriteString(dimStyle.Render("  No token data"))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Time metrics section
	b.WriteString(boldStyle.Render("Time Tracking"))
	b.WriteString("\n")

	if metrics.Time != nil {
		// Session start time
		if metrics.Time.Started > 0 {
			startTime := time.Unix(metrics.Time.Started, 0)
			b.WriteString(fmt.Sprintf("  Started: %s\n", startTime.Format("15:04:05")))
		}

		// Total duration
		b.WriteString(fmt.Sprintf("  Duration: %s\n", formatDuration(metrics.Time.TotalSeconds)))

		// Working vs waiting breakdown
		if metrics.Time.WorkingSeconds > 0 || metrics.Time.WaitingSeconds > 0 {
			b.WriteString(fmt.Sprintf("  Working: %s", greenStyle.Render(formatDuration(metrics.Time.WorkingSeconds))))
			if metrics.Time.TotalSeconds > 0 {
				pct := float64(metrics.Time.WorkingSeconds) / float64(metrics.Time.TotalSeconds) * 100
				b.WriteString(fmt.Sprintf(" (%.0f%%)", pct))
			}
			b.WriteString("\n")

			b.WriteString(fmt.Sprintf("  Waiting: %s", yellowStyle.Render(formatDuration(metrics.Time.WaitingSeconds))))
			if metrics.Time.TotalSeconds > 0 {
				pct := float64(metrics.Time.WaitingSeconds) / float64(metrics.Time.TotalSeconds) * 100
				b.WriteString(fmt.Sprintf(" (%.0f%%)", pct))
			}
			b.WriteString("\n")
		}
	} else {
		b.WriteString(dimStyle.Render("  No time data"))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Tool activity section
	b.WriteString(boldStyle.Render("Tool Activity"))
	b.WriteString("\n")

	if metrics.Tools != nil && len(metrics.Tools.Counts) > 0 {
		// Total tool calls
		totalTools := formatToolCount(metrics.Tools)
		b.WriteString(fmt.Sprintf("  Total calls: %d\n", totalTools))

		// Tool counts (sorted by frequency)
		type toolCount struct {
			name  string
			count int
		}
		var counts []toolCount
		for name, count := range metrics.Tools.Counts {
			counts = append(counts, toolCount{name, count})
		}
		// Sort by count descending
		for i := 0; i < len(counts)-1; i++ {
			for j := i + 1; j < len(counts); j++ {
				if counts[j].count > counts[i].count {
					counts[i], counts[j] = counts[j], counts[i]
				}
			}
		}

		// Show top tools (limit to 8)
		maxTools := 8
		if len(counts) < maxTools {
			maxTools = len(counts)
		}
		for i := 0; i < maxTools; i++ {
			b.WriteString(fmt.Sprintf("  %s: %d\n", counts[i].name, counts[i].count))
		}
		if len(counts) > maxTools {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  ... and %d more tools\n", len(counts)-maxTools)))
		}

		// Recent tools
		if len(metrics.Tools.Recent) > 0 {
			b.WriteString("\n")
			b.WriteString(fmt.Sprintf("  Recent: %s\n", strings.Join(metrics.Tools.Recent, ", ")))
		}
	} else {
		b.WriteString(dimStyle.Render("  No tool data"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Esc: close"))

	content := b.String()
	dialog := dialogBoxStyle.Width(metricsDetailWidth).Render(content)
	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, dialog)
}

// renderDialog renders the dialog overlay when dialogMode is set.
// Returns empty string if no dialog is open.
func (m Model) renderDialog() string {
	if m.dialogMode == DialogNone {
		return ""
	}

	var b strings.Builder

	// Title
	title := dialogTitle(m.dialogMode)
	b.WriteString(dialogTitleStyle.Render(title))
	b.WriteString("\n\n")

	// Dialog-specific content
	switch m.dialogMode {
	case DialogNewSession:
		b.WriteString("Name:      ")
		b.WriteString(m.nameInput.View())
		b.WriteString("\n")
		b.WriteString("Directory: ")
		b.WriteString(m.dirInput.View())
		b.WriteString("\n")
		// Checkbox for skip permissions
		checkbox := "[ ]"
		if m.skipPermissions {
			checkbox = "[x]"
		}
		checkboxLine := checkbox + " Skip permissions"
		if m.focusedInput == focusSkipPerms {
			checkboxLine = highlightStyle.Render(checkboxLine)
		}
		b.WriteString(checkboxLine)
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("Tab: switch  Space: toggle  Enter: create  Esc: cancel"))
	case DialogKillConfirm:
		if m.sessionToModify != nil {
			b.WriteString(fmt.Sprintf("Kill session '%s'?\n\n", m.sessionToModify.TmuxSession))
		}
		b.WriteString(dimStyle.Render("y: yes  n: no  Esc: cancel"))
	case DialogRename:
		if m.sessionToModify != nil {
			b.WriteString(fmt.Sprintf("Current: %s\n\n", dimStyle.Render(m.sessionToModify.TmuxSession)))
		}
		b.WriteString("New name: ")
		b.WriteString(m.nameInput.View())
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("Enter: rename  Esc: cancel"))
	case DialogGitDetail:
		b.Reset() // Clear the builder for custom git view
		return m.renderGitDetailView()
	case DialogGitDiff:
		b.Reset() // Clear the builder for custom diff view
		return m.renderGitDiffView()
	case DialogMetricsDetail:
		b.Reset() // Clear the builder for custom metrics view
		return m.renderMetricsDetailView()
	}

	// Error message if present
	if m.dialogError != "" {
		b.WriteString("\n\n")
		b.WriteString(dialogErrorStyle.Render(m.dialogError))
	}

	// Render the dialog box
	content := b.String()
	dialog := dialogBoxStyle.Width(dialogWidth).Render(content)

	// Center the dialog horizontally using lipgloss.Place for proper multi-line handling
	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, dialog)
}
