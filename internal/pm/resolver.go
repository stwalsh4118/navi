package pm

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/stwalsh4118/navi/internal/pathutil"
	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/task"
)

var groupStatusPriority = map[string]int{
	"inprogress": 0,
	"agreed":     1,
	"inreview":   2,
	"review":     2,
	"proposed":   3,
	"done":       4,
}

type ResolverInput struct {
	TaskResult     *task.ProviderResult
	Sessions       []session.Info
	ProjectDir     string
	Branch         string
	BranchPatterns []string
}

type ResolverResult struct {
	PBIID  string
	Title  string
	Source string
}

func ResolveCurrentPBI(input ResolverInput) ResolverResult {
	if result, ok := resolveFromProviderHint(input.TaskResult); ok {
		return result
	}

	if result, ok := resolveFromSessionMetadata(input.Sessions, input.ProjectDir, input.TaskResult); ok {
		return result
	}

	if result, ok := resolveFromBranchPattern(input.Branch, input.BranchPatterns, input.TaskResult); ok {
		return result
	}

	if input.TaskResult != nil {
		if id, title, ok := selectGroupByStatus(input.TaskResult.Groups); ok {
			return ResolverResult{PBIID: id, Title: title, Source: "status_heuristic"}
		}
		if id, title, ok := firstGroupFallback(input.TaskResult.Groups); ok {
			return ResolverResult{PBIID: id, Title: title, Source: "first_group_fallback"}
		}
	}

	return ResolverResult{}
}

func resolveFromProviderHint(taskResult *task.ProviderResult) (ResolverResult, bool) {
	if taskResult == nil {
		return ResolverResult{}, false
	}

	currentID := strings.TrimSpace(taskResult.CurrentPBIID)
	if currentID != "" {
		title := strings.TrimSpace(taskResult.CurrentPBITitle)
		if title == "" {
			title = findTitleByPBIID(taskResult, currentID)
		}
		return ResolverResult{PBIID: currentID, Title: title, Source: "provider_hint"}, true
	}

	for _, group := range taskResult.Groups {
		if !group.IsCurrent {
			continue
		}
		if strings.TrimSpace(group.ID) == "" && strings.TrimSpace(group.Title) == "" {
			continue
		}
		return ResolverResult{PBIID: group.ID, Title: group.Title, Source: "provider_hint"}, true
	}

	return ResolverResult{}, false
}

func resolveFromSessionMetadata(sessions []session.Info, projectDir string, taskResult *task.ProviderResult) (ResolverResult, bool) {
	if len(sessions) == 0 || strings.TrimSpace(projectDir) == "" {
		return ResolverResult{}, false
	}

	targetDir := normalizeProjectDir(projectDir)
	if targetDir == "" {
		return ResolverResult{}, false
	}

	bestTimestamp := int64(0)
	bestID := ""
	bestTitle := ""
	found := false

	for _, s := range sessions {
		if normalizeProjectDir(s.CWD) != targetDir {
			continue
		}
		currentPBI := strings.TrimSpace(s.CurrentPBI)
		if currentPBI == "" {
			continue
		}
		if !found || s.Timestamp > bestTimestamp {
			bestTimestamp = s.Timestamp
			bestID = currentPBI
			bestTitle = strings.TrimSpace(s.CurrentPBITitle)
			found = true
		}
	}

	if !found {
		return ResolverResult{}, false
	}

	if bestTitle == "" {
		bestTitle = findTitleByPBIID(taskResult, bestID)
	}

	return ResolverResult{PBIID: bestID, Title: bestTitle, Source: "session_metadata"}, true
}

func resolveFromBranchPattern(branch string, patterns []string, taskResult *task.ProviderResult) (ResolverResult, bool) {
	branchPBIID, ok := InferPBIFromBranch(branch, patterns)
	if !ok {
		return ResolverResult{}, false
	}

	resolvedID := resolveBranchInferredID(branchPBIID, taskResult)
	title := findTitleByPBIID(taskResult, resolvedID)
	return ResolverResult{PBIID: resolvedID, Title: title, Source: "branch_pattern"}, true
}

func resolveBranchInferredID(inferredID string, taskResult *task.ProviderResult) string {
	trimmedInferred := strings.TrimSpace(inferredID)
	if trimmedInferred == "" {
		return ""
	}

	isNumeric := isDigitsOnly(trimmedInferred)
	bestNumericSuffixMatch := ""

	if taskResult != nil {
		for _, group := range taskResult.Groups {
			groupID := strings.TrimSpace(group.ID)
			if groupID == "" {
				continue
			}

			if strings.EqualFold(groupID, trimmedInferred) {
				return group.ID
			}

			if !isNumeric {
				continue
			}

			if bestNumericSuffixMatch == "" && hasNumericSuffix(groupID, trimmedInferred) {
				bestNumericSuffixMatch = group.ID
			}
		}
	}

	if bestNumericSuffixMatch != "" {
		return bestNumericSuffixMatch
	}

	if isNumeric {
		if numericPrefix := inferNumericIDPrefix(taskResult); numericPrefix != "" {
			return fmt.Sprintf("%s%s", numericPrefix, trimmedInferred)
		}
		return trimmedInferred
	}

	return trimmedInferred
}

func isDigitsOnly(value string) bool {
	if value == "" {
		return false
	}

	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

func hasNumericSuffix(value string, suffix string) bool {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" || suffix == "" {
		return false
	}

	if !strings.HasSuffix(trimmedValue, suffix) {
		return false
	}

	prefixLength := len(trimmedValue) - len(suffix)
	if prefixLength <= 0 {
		return false
	}

	separator := trimmedValue[prefixLength-1]
	return separator == '-' || separator == '_' || separator == '/' || separator == '#'
}

func inferNumericIDPrefix(taskResult *task.ProviderResult) string {
	if taskResult == nil {
		return ""
	}

	prefixCounts := make(map[string]int)
	orderedPrefixes := make([]string, 0)

	for _, group := range taskResult.Groups {
		prefix, ok := numericIDSuffixPrefix(group.ID)
		if !ok {
			continue
		}
		if _, exists := prefixCounts[prefix]; !exists {
			orderedPrefixes = append(orderedPrefixes, prefix)
		}
		prefixCounts[prefix]++
	}

	bestPrefix := ""
	bestCount := 0
	for _, prefix := range orderedPrefixes {
		count := prefixCounts[prefix]
		if count > bestCount {
			bestCount = count
			bestPrefix = prefix
		}
	}

	return bestPrefix
}

func numericIDSuffixPrefix(value string) (string, bool) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return "", false
	}

	i := len(trimmedValue) - 1
	for i >= 0 {
		if trimmedValue[i] < '0' || trimmedValue[i] > '9' {
			break
		}
		i--
	}

	digitStart := i + 1
	if digitStart <= 0 || digitStart >= len(trimmedValue) {
		return "", false
	}

	separator := trimmedValue[digitStart-1]
	if separator != '-' && separator != '_' && separator != '/' && separator != '#' {
		return "", false
	}

	return trimmedValue[:digitStart], true
}

func firstGroupFallback(groups []task.TaskGroup) (string, string, bool) {
	for _, group := range groups {
		if strings.TrimSpace(group.ID) != "" || strings.TrimSpace(group.Title) != "" {
			return group.ID, group.Title, true
		}
	}
	return "", "", false
}

func findTitleByPBIID(taskResult *task.ProviderResult, pbiID string) string {
	if taskResult == nil {
		return ""
	}
	trimmedID := strings.TrimSpace(pbiID)
	for _, group := range taskResult.Groups {
		if strings.EqualFold(strings.TrimSpace(group.ID), trimmedID) {
			return group.Title
		}
	}
	return ""
}

func normalizeProjectDir(dir string) string {
	expanded := pathutil.ExpandPath(strings.TrimSpace(dir))
	if expanded == "" {
		return ""
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return expanded
	}
	return abs
}

func selectGroupByStatus(groups []task.TaskGroup) (id string, title string, ok bool) {
	bestPriority := len(groupStatusPriority) + 1
	bestID := ""
	bestTitle := ""
	found := false

	for _, group := range groups {
		if strings.TrimSpace(group.ID) == "" && strings.TrimSpace(group.Title) == "" {
			continue
		}

		priority, recognized := groupStatusPriority[normalizeGroupStatus(group.Status)]
		if !recognized {
			continue
		}

		if !found || priority < bestPriority {
			bestPriority = priority
			bestID = group.ID
			bestTitle = group.Title
			found = true
		}
	}

	if !found {
		return "", "", false
	}

	return bestID, bestTitle, true
}

func normalizeGroupStatus(status string) string {
	normalized := strings.ToLower(strings.TrimSpace(status))
	normalized = strings.ReplaceAll(normalized, "_", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ReplaceAll(normalized, " ", "")
	return normalized
}
