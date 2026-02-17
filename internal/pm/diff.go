package pm

import (
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var commitsBetweenFunc = GetCommitsBetween

// DiffSnapshots compares project snapshots and emits state-change events.
func DiffSnapshots(oldSnapshot, newSnapshot ProjectSnapshot) []Event {
	if oldSnapshot.ProjectDir == "" {
		return nil
	}

	now := time.Now().UTC()
	events := make([]Event, 0, 7)

	if oldSnapshot.HeadSHA != "" && newSnapshot.HeadSHA != "" && oldSnapshot.HeadSHA != newSnapshot.HeadSHA {
		commits := commitsBetweenFunc(newSnapshot.ProjectDir, oldSnapshot.HeadSHA, newSnapshot.HeadSHA)
		events = append(events, newEvent(newSnapshot, EventCommit, now, map[string]string{
			"old_head_sha": oldSnapshot.HeadSHA,
			"new_head_sha": newSnapshot.HeadSHA,
			"commits":      strings.Join(commits, "\n"),
		}))
	}

	if newSnapshot.TaskCounts.Done > oldSnapshot.TaskCounts.Done {
		events = append(events, newEvent(newSnapshot, EventTaskCompleted, now, map[string]string{
			"old_done": strconv.Itoa(oldSnapshot.TaskCounts.Done),
			"new_done": strconv.Itoa(newSnapshot.TaskCounts.Done),
		}))
	}

	if newSnapshot.TaskCounts.InProgress > oldSnapshot.TaskCounts.InProgress {
		events = append(events, newEvent(newSnapshot, EventTaskStarted, now, map[string]string{
			"old_in_progress": strconv.Itoa(oldSnapshot.TaskCounts.InProgress),
			"new_in_progress": strconv.Itoa(newSnapshot.TaskCounts.InProgress),
		}))
	}

	if oldSnapshot.SessionStatus != "" && newSnapshot.SessionStatus != "" && oldSnapshot.SessionStatus != newSnapshot.SessionStatus {
		events = append(events, newEvent(newSnapshot, EventSessionStatusChange, now, map[string]string{
			"old_status": oldSnapshot.SessionStatus,
			"new_status": newSnapshot.SessionStatus,
		}))
	}

	if newSnapshot.CurrentPBIID != "" && oldSnapshot.CurrentPBIID == newSnapshot.CurrentPBIID &&
		newSnapshot.TaskCounts.Total > 0 && newSnapshot.TaskCounts.Done == newSnapshot.TaskCounts.Total &&
		oldSnapshot.TaskCounts.Done < oldSnapshot.TaskCounts.Total {
		events = append(events, newEvent(newSnapshot, EventPBICompleted, now, map[string]string{
			"pbi_id":    newSnapshot.CurrentPBIID,
			"pbi_title": newSnapshot.CurrentPBITitle,
		}))
	}

	if oldSnapshot.Branch != "" && oldSnapshot.Branch != newSnapshot.Branch {
		events = append(events, newEvent(newSnapshot, EventBranchCreated, now, map[string]string{
			"old_branch": oldSnapshot.Branch,
			"new_branch": newSnapshot.Branch,
		}))
	}

	if oldSnapshot.PRNumber == 0 && newSnapshot.PRNumber > 0 {
		events = append(events, newEvent(newSnapshot, EventPRCreated, now, map[string]string{
			"pr_number": strconv.Itoa(newSnapshot.PRNumber),
		}))
	}

	return events
}

// GetCommitsBetween returns commit summary lines in old..new range.
func GetCommitsBetween(dir, oldSHA, newSHA string) []string {
	if oldSHA == "" || newSHA == "" {
		return nil
	}

	cmd := exec.Command("git", "log", "--oneline", oldSHA+".."+newSHA)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	commits := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			commits = append(commits, trimmed)
		}
	}

	return commits
}

func newEvent(snapshot ProjectSnapshot, eventType EventType, timestamp time.Time, payload map[string]string) Event {
	return Event{
		Type:        eventType,
		Timestamp:   timestamp,
		ProjectName: snapshot.ProjectName,
		ProjectDir:  snapshot.ProjectDir,
		Payload:     payload,
	}
}
