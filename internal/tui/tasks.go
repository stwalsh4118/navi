package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/stwalsh4118/navi/internal/task"
)

// Task view timing constants
const (
	// taskDefaultRefreshInterval is how often task data is refreshed.
	taskDefaultRefreshInterval = 60 * time.Second
)

// tasksMsg carries refreshed task data from provider execution.
type tasksMsg struct {
	groupsByProject  map[string][]task.TaskGroup // keyed by project dir
	errors           map[string]error            // keyed by project dir
	resultsByProject map[string]*task.ProviderResult
}

// taskConfigsMsg carries discovered project configs from session CWDs.
type taskConfigsMsg struct {
	configs []task.ProjectConfig
}

// taskTickMsg triggers periodic task data refresh.
type taskTickMsg time.Time

// taskRefreshCmd runs all discovered providers and returns results keyed by project.
func taskRefreshCmd(configs []task.ProjectConfig, cache *task.ResultCache, globalConfig *task.GlobalConfig, timeout time.Duration) tea.Cmd {
	return func() tea.Msg {
		groupsByProject := make(map[string][]task.TaskGroup)
		errors := make(map[string]error)
		resultsByProject := make(map[string]*task.ProviderResult)

		for _, cfg := range configs {
			// Check cache first
			if cached, ok := cache.Get(cfg.ProjectDir, taskDefaultRefreshInterval); ok {
				if cached.Error != nil {
					errors[cfg.ProjectDir] = cached.Error
				} else if cached.Result != nil {
					groups := normalizeGroups(cached.Result, cfg, globalConfig)
					groupsByProject[cfg.ProjectDir] = groups
					resultsByProject[cfg.ProjectDir] = cached.Result
				}
				continue
			}

			// Execute provider
			result, err := task.ExecuteProvider(cfg, timeout)
			cache.Set(cfg.ProjectDir, result, err)

			if err != nil {
				errors[cfg.ProjectDir] = err
				continue
			}

			groups := normalizeGroups(result, cfg, globalConfig)
			groupsByProject[cfg.ProjectDir] = groups
			resultsByProject[cfg.ProjectDir] = result
		}

		return tasksMsg{groupsByProject: groupsByProject, errors: errors, resultsByProject: resultsByProject}
	}
}

// normalizeGroups applies status normalization to a provider result and returns groups.
// If the result has no groups (flat format), wraps tasks in a single group named after the project.
func normalizeGroups(result *task.ProviderResult, cfg task.ProjectConfig, globalConfig *task.GlobalConfig) []task.TaskGroup {
	statusMap := make(map[string]string)
	if globalConfig != nil {
		statusMap = globalConfig.Tasks.StatusMap
	}

	if len(result.Groups) > 0 {
		groups := make([]task.TaskGroup, len(result.Groups))
		for i, g := range result.Groups {
			groups[i] = g
			groups[i].Status = task.NormalizeStatus(g.Status, statusMap)
			normalizedTasks := make([]task.Task, len(g.Tasks))
			for j, t := range g.Tasks {
				normalizedTasks[j] = t
				normalizedTasks[j].Status = task.NormalizeStatus(t.Status, statusMap)
			}
			groups[i].Tasks = normalizedTasks
		}
		return groups
	}

	// Flat format: wrap in a single group
	if len(result.Tasks) > 0 {
		normalizedTasks := make([]task.Task, len(result.Tasks))
		for i, t := range result.Tasks {
			normalizedTasks[i] = t
			normalizedTasks[i].Status = task.NormalizeStatus(t.Status, statusMap)
		}
		// Use the last directory component as group title
		parts := strings.Split(cfg.ProjectDir, "/")
		groupTitle := cfg.ProjectDir
		if len(parts) > 0 {
			groupTitle = parts[len(parts)-1]
		}
		return []task.TaskGroup{{
			ID:    cfg.ProjectDir,
			Title: groupTitle,
			Tasks: normalizedTasks,
		}}
	}

	return nil
}

// taskTickCmd returns a command that fires after the task refresh interval.
func taskTickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return taskTickMsg(t)
	})
}

// discoverTaskConfigsCmd discovers project configs from session CWDs.
func discoverTaskConfigsCmd(sessions []sessionCWD, globalConfig *task.GlobalConfig) tea.Cmd {
	return func() tea.Msg {
		var cwds []string
		for _, s := range sessions {
			if s.cwd != "" {
				cwds = append(cwds, s.cwd)
			}
		}
		configs := task.DiscoverProjects(cwds, globalConfig)
		return taskConfigsMsg{configs: configs}
	}
}

// sessionCWD is a lightweight struct for passing session CWDs to config discovery.
type sessionCWD struct {
	cwd string
}

// extractSessionCWDs extracts unique CWDs from the current session list.
func extractSessionCWDs(sessions []sessionCWD) []string {
	seen := make(map[string]bool)
	var cwds []string
	for _, s := range sessions {
		if s.cwd != "" && !seen[s.cwd] {
			seen[s.cwd] = true
			cwds = append(cwds, s.cwd)
		}
	}
	return cwds
}

// findProjectForCWD returns the project dir that contains the given CWD,
// or empty string if no matching project config is found.
func findProjectForCWD(cwd string, configs []task.ProjectConfig) string {
	if cwd == "" {
		return ""
	}
	for _, cfg := range configs {
		if cwd == cfg.ProjectDir || strings.HasPrefix(cwd, cfg.ProjectDir+"/") {
			return cfg.ProjectDir
		}
	}
	return ""
}
