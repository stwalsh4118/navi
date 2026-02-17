package pm

import "time"

type EventType string

const (
	EventTaskCompleted       EventType = "task_completed"
	EventTaskStarted         EventType = "task_started"
	EventCommit              EventType = "commit"
	EventSessionStatusChange EventType = "session_status_change"
	EventPBICompleted        EventType = "pbi_completed"
	EventBranchCreated       EventType = "branch_created"
	EventPRCreated           EventType = "pr_created"
)

type TaskCounts struct {
	Total      int `json:"total"`
	Done       int `json:"done"`
	InProgress int `json:"in_progress"`
}

type ProjectSnapshot struct {
	ProjectName     string     `json:"project_name"`
	ProjectDir      string     `json:"project_dir"`
	HeadSHA         string     `json:"head_sha"`
	Branch          string     `json:"branch"`
	CommitsAhead    int        `json:"commits_ahead"`
	Dirty           bool       `json:"dirty"`
	CurrentPBIID    string     `json:"current_pbi_id"`
	CurrentPBITitle string     `json:"current_pbi_title"`
	TaskCounts      TaskCounts `json:"task_counts"`
	SessionStatus   string     `json:"session_status"`
	LastActivity    time.Time  `json:"last_activity"`
	SessionCount    int        `json:"session_count"`
	PRNumber        int        `json:"pr_number,omitempty"`
}

type Event struct {
	Type        EventType         `json:"type"`
	Timestamp   time.Time         `json:"timestamp"`
	ProjectName string            `json:"project_name"`
	ProjectDir  string            `json:"project_dir"`
	Payload     map[string]string `json:"payload,omitempty"`
}

type PMOutput struct {
	Snapshots []ProjectSnapshot `json:"snapshots"`
	Events    []Event           `json:"events"`
}
