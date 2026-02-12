package tui

import (
	"sort"
	"strings"

	"github.com/stwalsh4118/navi/internal/task"
)

// taskSortMode defines the available sort modes for task groups.
type taskSortMode string

const (
	taskSortSource   taskSortMode = "source"
	taskSortStatus   taskSortMode = "status"
	taskSortName     taskSortMode = "name"
	taskSortProgress taskSortMode = "progress"
)

// taskSortModes lists all sort modes in cycle order.
var taskSortModes = []taskSortMode{taskSortSource, taskSortStatus, taskSortName, taskSortProgress}

// nextTaskSortMode returns the next sort mode in the cycle.
func nextTaskSortMode(current taskSortMode) taskSortMode {
	for i, m := range taskSortModes {
		if m == current {
			return taskSortModes[(i+1)%len(taskSortModes)]
		}
	}
	return taskSortSource
}

// taskFilterMode defines the available filter modes for task groups.
type taskFilterMode string

const (
	taskFilterAll        taskFilterMode = "all"
	taskFilterActive     taskFilterMode = "active"
	taskFilterIncomplete taskFilterMode = "incomplete"
)

// taskFilterModes lists all filter modes in cycle order.
var taskFilterModes = []taskFilterMode{taskFilterAll, taskFilterActive, taskFilterIncomplete}

// nextTaskFilterMode returns the next filter mode in the cycle.
func nextTaskFilterMode(current taskFilterMode) taskFilterMode {
	for i, m := range taskFilterModes {
		if m == current {
			return taskFilterModes[(i+1)%len(taskFilterModes)]
		}
	}
	return taskFilterAll
}

// statusPriority returns a numeric priority for status sorting.
// Lower numbers sort first: active > review > blocked > todo > done.
func statusPriority(status string) int {
	cat := groupStatusCategory(status)
	switch cat {
	case "active":
		return 0
	case "review":
		return 1
	case "blocked":
		return 2
	case "todo":
		return 3
	case "done":
		return 4
	default:
		return 5
	}
}

// getFilteredTaskGroups returns task groups after applying the current filter mode.
// This does NOT apply sorting — use getSortedAndFilteredTaskGroups for the full pipeline.
func (m Model) getFilteredTaskGroups() []task.TaskGroup {
	return filterTaskGroups(m.taskGroups, m.taskFilterMode)
}

// filterTaskGroups filters groups based on the given filter mode.
func filterTaskGroups(groups []task.TaskGroup, mode taskFilterMode) []task.TaskGroup {
	if mode == taskFilterAll || mode == "" {
		return groups
	}

	var filtered []task.TaskGroup
	for _, g := range groups {
		cat := groupStatusCategory(g.Status)
		switch mode {
		case taskFilterActive:
			// Show only active, review, or blocked
			if cat == "active" || cat == "review" || cat == "blocked" {
				filtered = append(filtered, g)
			}
		case taskFilterIncomplete:
			// Show everything except done
			if cat != "done" {
				filtered = append(filtered, g)
			}
		}
	}
	return filtered
}

// sortTaskGroups returns a sorted copy of groups based on the given sort mode.
// Source order is preserved as a stable secondary sort.
func sortTaskGroups(groups []task.TaskGroup, mode taskSortMode) []task.TaskGroup {
	if mode == taskSortSource {
		return groups
	}

	// Make a copy with original indices for stable sort
	type indexedGroup struct {
		group task.TaskGroup
		index int
	}
	indexed := make([]indexedGroup, len(groups))
	for i, g := range groups {
		indexed[i] = indexedGroup{group: g, index: i}
	}

	sort.SliceStable(indexed, func(i, j int) bool {
		gi, gj := indexed[i], indexed[j]
		switch mode {
		case taskSortStatus:
			pi, pj := statusPriority(gi.group.Status), statusPriority(gj.group.Status)
			if pi != pj {
				return pi < pj
			}
			return gi.index < gj.index
		case taskSortName:
			ni, nj := strings.ToLower(gi.group.Title), strings.ToLower(gj.group.Title)
			if ni != nj {
				return ni < nj
			}
			return gi.index < gj.index
		case taskSortProgress:
			di, ti := groupProgress(gi.group)
			dj, tj := groupProgress(gj.group)
			// Lowest completion first
			var pi, pj float64
			if ti > 0 {
				pi = float64(di) / float64(ti)
			}
			if tj > 0 {
				pj = float64(dj) / float64(tj)
			}
			if pi != pj {
				return pi < pj
			}
			return gi.index < gj.index
		default:
			return gi.index < gj.index
		}
	})

	result := make([]task.TaskGroup, len(indexed))
	for i, ig := range indexed {
		result[i] = ig.group
	}
	return result
}

// sortTasksByStatus returns a sorted copy of tasks based on status priority.
func sortTasksByStatus(tasks []task.Task) []task.Task {
	type indexedTask struct {
		task  task.Task
		index int
	}
	indexed := make([]indexedTask, len(tasks))
	for i, t := range tasks {
		indexed[i] = indexedTask{task: t, index: i}
	}

	sort.SliceStable(indexed, func(i, j int) bool {
		pi, pj := statusPriority(indexed[i].task.Status), statusPriority(indexed[j].task.Status)
		if pi != pj {
			return pi < pj
		}
		return indexed[i].index < indexed[j].index
	})

	result := make([]task.Task, len(indexed))
	for i, it := range indexed {
		result[i] = it.task
	}
	return result
}

// getSortedAndFilteredTaskGroups returns groups after applying filter and sort.
// When sort mode is status, tasks within groups are also sorted by status.
func (m Model) getSortedAndFilteredTaskGroups() []task.TaskGroup {
	// Filter first
	groups := filterTaskGroups(m.taskGroups, m.taskFilterMode)

	// Sort groups
	groups = sortTaskGroups(groups, m.taskSortMode)

	// When sorting by status, also sort tasks within groups
	if m.taskSortMode == taskSortStatus {
		for i := range groups {
			groups[i].Tasks = sortTasksByStatus(groups[i].Tasks)
		}
	}

	// Reverse if requested (must copy first to avoid mutating the source slice)
	if m.taskSortReversed {
		reversed := make([]task.TaskGroup, len(groups))
		copy(reversed, groups)
		reverseTaskGroups(reversed)
		groups = reversed
	}

	return groups
}

// reverseTaskGroups reverses a slice of task groups in place.
func reverseTaskGroups(groups []task.TaskGroup) {
	for i, j := 0, len(groups)-1; i < j; i, j = i+1, j-1 {
		groups[i], groups[j] = groups[j], groups[i]
	}
}
