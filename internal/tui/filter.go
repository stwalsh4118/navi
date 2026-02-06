package tui

import (
	"sort"
	"strings"

	"github.com/stwalsh4118/navi/internal/session"
)

// SortMode represents the session list sort mode.
type SortMode int

// Sort mode constants
const (
	SortPriority  SortMode = iota // Attention-needed first, then by time (default)
	SortName                      // Alphabetical by session name
	SortAge                       // Most recent activity first
	SortStatus                    // Grouped by status type
	SortDirectory                 // Grouped by working directory
)

// sortModeCount is the total number of sort modes (used for cycling).
const sortModeCount = 5

// SortModeLabel returns a display label for the given sort mode.
func SortModeLabel(mode SortMode) string {
	switch mode {
	case SortPriority:
		return "priority"
	case SortName:
		return "name"
	case SortAge:
		return "age"
	case SortStatus:
		return "status"
	case SortDirectory:
		return "directory"
	default:
		return "priority"
	}
}

// statusFilterKeys maps number key characters to session status values.
var statusFilterKeys = map[string]string{
	"1": session.StatusWaiting,
	"2": session.StatusPermission,
	"3": session.StatusWorking,
	"4": "done",
	"5": "error",
}

// statusOrder defines a canonical ordering of statuses for SortStatus mode.
var statusOrder = map[string]int{
	session.StatusPermission: 0,
	session.StatusWaiting:    1,
	session.StatusWorking:    2,
	"done":                   3,
	"error":                  4,
}

// sortSessions returns a sorted copy of the sessions slice according to the given mode.
// The original slice is not modified.
func sortSessions(sessions []session.Info, mode SortMode) []session.Info {
	if len(sessions) <= 1 {
		return sessions
	}

	// Make a copy to avoid mutating the original
	sorted := make([]session.Info, len(sessions))
	copy(sorted, sessions)

	switch mode {
	case SortPriority:
		session.SortSessions(sorted)

	case SortName:
		sort.SliceStable(sorted, func(i, j int) bool {
			return strings.ToLower(sorted[i].TmuxSession) < strings.ToLower(sorted[j].TmuxSession)
		})

	case SortAge:
		sort.SliceStable(sorted, func(i, j int) bool {
			return sorted[i].Timestamp > sorted[j].Timestamp
		})

	case SortStatus:
		sort.SliceStable(sorted, func(i, j int) bool {
			iOrder, iOk := statusOrder[sorted[i].Status]
			jOrder, jOk := statusOrder[sorted[j].Status]
			if !iOk {
				iOrder = len(statusOrder) // Unknown statuses go last
			}
			if !jOk {
				jOrder = len(statusOrder)
			}
			if iOrder != jOrder {
				return iOrder < jOrder
			}
			return sorted[i].Timestamp > sorted[j].Timestamp
		})

	case SortDirectory:
		sort.SliceStable(sorted, func(i, j int) bool {
			if sorted[i].CWD != sorted[j].CWD {
				return strings.ToLower(sorted[i].CWD) < strings.ToLower(sorted[j].CWD)
			}
			return strings.ToLower(sorted[i].TmuxSession) < strings.ToLower(sorted[j].TmuxSession)
		})
	}

	return sorted
}

// filterByStatus returns sessions matching the given status.
func filterByStatus(sessions []session.Info, status string) []session.Info {
	var filtered []session.Info
	for _, s := range sessions {
		if s.Status == status {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// filterOffline removes "done" sessions from the list.
func filterOffline(sessions []session.Info) []session.Info {
	var filtered []session.Info
	for _, s := range sessions {
		if s.Status != "done" {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// fuzzyMatch checks if query is a subsequence of target (case-insensitive).
// Returns whether it matches and a relevance score.
// Higher scores indicate better matches.
func fuzzyMatch(query, target string) (bool, int) {
	if query == "" {
		return true, 0
	}

	lowerQuery := strings.ToLower(query)
	lowerTarget := strings.ToLower(target)

	queryIdx := 0
	score := 0
	lastMatchIdx := -1

	for i := 0; i < len(lowerTarget) && queryIdx < len(lowerQuery); i++ {
		if lowerTarget[i] == lowerQuery[queryIdx] {
			// Consecutive match bonus
			if lastMatchIdx == i-1 {
				score += 2
			}
			// Start-of-word bonus
			if i == 0 || lowerTarget[i-1] == '/' || lowerTarget[i-1] == '-' || lowerTarget[i-1] == '_' || lowerTarget[i-1] == ' ' {
				score += 3
			}
			score++ // Base match point
			lastMatchIdx = i
			queryIdx++
		}
	}

	if queryIdx == len(lowerQuery) {
		return true, score
	}
	return false, 0
}

// fuzzyFilter returns sessions whose name, CWD, or message match the query.
// Results are sorted by best match score (descending).
func fuzzyFilter(sessions []session.Info, query string) []session.Info {
	if query == "" {
		return sessions
	}

	type scored struct {
		session session.Info
		score   int
	}

	var matches []scored
	for _, s := range sessions {
		bestScore := 0

		if ok, sc := fuzzyMatch(query, s.TmuxSession); ok && sc > bestScore {
			bestScore = sc
		}
		if ok, sc := fuzzyMatch(query, s.CWD); ok && sc > bestScore {
			bestScore = sc
		}
		if ok, sc := fuzzyMatch(query, s.Message); ok && sc > bestScore {
			bestScore = sc
		}

		if bestScore > 0 {
			matches = append(matches, scored{session: s, score: bestScore})
		}
	}

	// Sort by score descending
	sort.SliceStable(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	result := make([]session.Info, len(matches))
	for i, m := range matches {
		result[i] = m.session
	}
	return result
}
