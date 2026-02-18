package pm

import (
	"encoding/json"
	"time"
)

type TriggerType string

const (
	TriggerTaskCompleted TriggerType = "task_completed"
	TriggerCommit        TriggerType = "commit"
	TriggerOnDemand      TriggerType = "on_demand"
)

type InboxPayload struct {
	Timestamp   time.Time         `json:"timestamp"`
	TriggerType TriggerType       `json:"trigger_type"`
	Events      []Event           `json:"events"`
	Snapshots   []ProjectSnapshot `json:"snapshots"`
}

func BuildInbox(trigger TriggerType, snapshots []ProjectSnapshot, events []Event) (*InboxPayload, error) {
	inbox := &InboxPayload{
		Timestamp:   time.Now().UTC(),
		TriggerType: trigger,
		Events:      append([]Event{}, events...),
		Snapshots:   append([]ProjectSnapshot{}, snapshots...),
	}

	return inbox, nil
}

func InboxToJSON(inbox *InboxPayload) ([]byte, error) {
	return json.Marshal(inbox)
}
