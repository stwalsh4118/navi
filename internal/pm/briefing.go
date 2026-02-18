package pm

import (
	"encoding/json"
	"fmt"
)

type PMBriefing struct {
	Summary        string            `json:"summary"`
	Projects       []ProjectBriefing `json:"projects"`
	AttentionItems []AttentionItem   `json:"attention_items"`
	Breadcrumbs    []Breadcrumb      `json:"breadcrumbs"`
}

type ProjectBriefing struct {
	Name           string `json:"name"`
	Status         string `json:"status"`
	CurrentWork    string `json:"current_work"`
	RecentActivity string `json:"recent_activity"`
}

type AttentionItem struct {
	Priority    string `json:"priority"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ProjectName string `json:"project_name"`
}

// UnmarshalJSON handles Claude sometimes sending priority as a number instead
// of a string (e.g., 1 instead of "high").
func (a *AttentionItem) UnmarshalJSON(data []byte) error {
	type raw struct {
		Priority    json.RawMessage `json:"priority"`
		Title       string          `json:"title"`
		Description string          `json:"description"`
		ProjectName string          `json:"project_name"`
	}
	var r raw
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	a.Title = r.Title
	a.Description = r.Description
	a.ProjectName = r.ProjectName

	// Try string first, fall back to number.
	var s string
	if err := json.Unmarshal(r.Priority, &s); err == nil {
		a.Priority = s
	} else {
		var n float64
		if err := json.Unmarshal(r.Priority, &n); err == nil {
			a.Priority = fmt.Sprintf("%g", n)
		}
	}
	return nil
}

type Breadcrumb struct {
	Timestamp string `json:"timestamp"`
	Summary   string `json:"summary"`
}
