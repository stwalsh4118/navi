package pm

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

type Breadcrumb struct {
	Timestamp string `json:"timestamp"`
	Summary   string `json:"summary"`
}
