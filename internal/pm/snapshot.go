package pm

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/stwalsh4118/navi/internal/git"
	"github.com/stwalsh4118/navi/internal/pathutil"
	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/task"
)

const (
	normalizedDoneStatus = "done"
)

var inProgressTaskStatuses = map[string]bool{
	"inprogress":  true,
	"in_progress": true,
	"in-progress": true,
}

var gitInfoFunc = git.GetInfo
var getHeadSHAFunc = GetHeadSHA

// DiscoverProjects groups sessions by expanded project directory.
func DiscoverProjects(sessions []session.Info) map[string][]session.Info {
	projects := make(map[string][]session.Info)

	for _, s := range sessions {
		if strings.TrimSpace(s.CWD) == "" {
			continue
		}

		expanded := pathutil.ExpandPath(s.CWD)
		abs, err := filepath.Abs(expanded)
		if err != nil {
			abs = expanded
		}

		projects[abs] = append(projects[abs], s)
	}

	return projects
}

// CaptureSnapshot captures project state for a project directory and grouped sessions.
func CaptureSnapshot(projectDir string, sessions []session.Info, taskResult *task.ProviderResult) ProjectSnapshot {
	snapshot := ProjectSnapshot{
		ProjectName:  filepath.Base(projectDir),
		ProjectDir:   projectDir,
		SessionCount: len(sessions),
	}

	if info := gitInfoFunc(projectDir); info != nil {
		snapshot.Branch = info.Branch
		snapshot.CommitsAhead = info.Ahead
		snapshot.Dirty = info.Dirty
		snapshot.PRNumber = info.PRNum
		snapshot.HeadSHA = getHeadSHAFunc(projectDir)
	}

	if taskResult != nil {
		snapshot.TaskCounts = getTaskCounts(taskResult)
	}

	resolvedCurrentPBI := ResolveCurrentPBI(ResolverInput{
		TaskResult: taskResult,
		Sessions:   sessions,
		ProjectDir: projectDir,
		Branch:     snapshot.Branch,
	})
	snapshot.CurrentPBIID = resolvedCurrentPBI.PBIID
	snapshot.CurrentPBITitle = resolvedCurrentPBI.Title
	snapshot.CurrentPBISource = resolvedCurrentPBI.Source

	status, activity := aggregateSessionState(sessions)
	snapshot.SessionStatus = status
	if activity > 0 {
		snapshot.LastActivity = time.Unix(activity, 0).UTC()
	}

	return snapshot
}

// GetHeadSHA returns the full HEAD SHA for a git repository directory.
func GetHeadSHA(dir string) string {
	if _, err := os.Stat(dir); err != nil {
		return ""
	}

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

func getTaskCounts(taskResult *task.ProviderResult) TaskCounts {
	var counts TaskCounts
	for _, t := range taskResult.AllTasks() {
		counts.Total++

		normalized := strings.ToLower(strings.TrimSpace(task.NormalizeStatus(t.Status, nil)))
		switch {
		case normalized == normalizedDoneStatus:
			counts.Done++
		case inProgressTaskStatuses[normalized]:
			counts.InProgress++
		}
	}

	return counts
}

func aggregateSessionState(sessions []session.Info) (string, int64) {
	bestStatus := ""
	bestRank := len(session.StatusPriority) + 1
	latest := int64(0)

	for _, s := range sessions {
		status, _ := session.CompositeStatus(s)
		rank := statusRank(status)
		if rank < bestRank {
			bestRank = rank
			bestStatus = status
		}
		if s.Timestamp > latest {
			latest = s.Timestamp
		}
	}

	return bestStatus, latest
}

func statusRank(status string) int {
	for i, candidate := range session.StatusPriority {
		if status == candidate {
			return i
		}
	}
	return len(session.StatusPriority)
}
