package pm

import (
	"sort"

	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/task"
)

// Engine orchestrates snapshot capture, diffing, and event logging.
type Engine struct {
	prevSnapshots map[string]ProjectSnapshot
}

// NewEngine creates a PM engine with empty snapshot cache.
func NewEngine() *Engine {
	return &Engine{prevSnapshots: make(map[string]ProjectSnapshot)}
}

// Run executes one PM pipeline cycle.
func (e *Engine) Run(sessions []session.Info, taskResults map[string]*task.ProviderResult) (*PMOutput, error) {
	projects := DiscoverProjects(sessions)
	projectDirs := make([]string, 0, len(projects))
	for projectDir := range projects {
		projectDirs = append(projectDirs, projectDir)
	}
	sort.Strings(projectDirs)

	snapshots := make([]ProjectSnapshot, 0, len(projectDirs))
	allEvents := make([]Event, 0)
	nextSnapshots := make(map[string]ProjectSnapshot, len(projectDirs))

	for _, projectDir := range projectDirs {
		snapshot := CaptureSnapshot(projectDir, projects[projectDir], taskResults[projectDir])
		snapshots = append(snapshots, snapshot)

		if oldSnapshot, ok := e.prevSnapshots[projectDir]; ok {
			allEvents = append(allEvents, DiffSnapshots(oldSnapshot, snapshot)...)
		}

		nextSnapshots[projectDir] = snapshot
	}

	if err := AppendEvents(allEvents); err != nil {
		return nil, err
	}

	e.prevSnapshots = nextSnapshots

	return &PMOutput{
		Snapshots: snapshots,
		Events:    allEvents,
	}, nil
}
