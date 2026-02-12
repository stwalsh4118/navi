package git

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"time"
)

// PR state constants (from GitHub API)
const (
	PRStateOpen   = "OPEN"
	PRStateClosed = "CLOSED"
	PRStateMerged = "MERGED"
)

// Mergeable state constants (from GitHub API)
const (
	MergeableMergeable   = "MERGEABLE"
	MergeableConflicting = "CONFLICTING"
	MergeableUnknown     = "UNKNOWN"
)

// Review decision constants (from GitHub API)
const (
	ReviewApproved        = "APPROVED"
	ReviewChangesRequired = "CHANGES_REQUESTED"
	ReviewRequired        = "REVIEW_REQUIRED"
)

// Reviewer state constants (from GitHub API)
const (
	ReviewerApproved        = "APPROVED"
	ReviewerChangesRequired = "CHANGES_REQUESTED"
	ReviewerCommented       = "COMMENTED"
	ReviewerPending         = "PENDING"
)

// Check status constants (from GitHub API statusCheckRollup)
const (
	CheckStatusCompleted  = "COMPLETED"
	CheckStatusInProgress = "IN_PROGRESS"
	CheckStatusQueued     = "QUEUED"
	CheckStatusPending    = "PENDING"
)

// Check conclusion constants (from GitHub API)
const (
	CheckConclusionSuccess        = "SUCCESS"
	CheckConclusionFailure        = "FAILURE"
	CheckConclusionNeutral        = "NEUTRAL"
	CheckConclusionCancelled      = "CANCELLED"
	CheckConclusionTimedOut       = "TIMED_OUT"
	CheckConclusionActionRequired = "ACTION_REQUIRED"
)

// Display indicator constants for check/review/merge status
const (
	IndicatorPass    = "✓"
	IndicatorFail    = "✗"
	IndicatorPending = "●"
	IndicatorWaiting = "⏳"
	IndicatorUnknown = "?"
	IndicatorDraft   = "draft"
	CommentIcon      = "💬"
)

// PR cache TTL constant (separate from git info cache)
const PRCacheMaxAge = 60 * time.Second

// PRDetail represents extended PR metadata fetched from the GitHub CLI.
type PRDetail struct {
	Number       int          `json:"number"`
	Title        string       `json:"title"`
	State        string       `json:"state"`                   // OPEN, CLOSED, MERGED
	Draft        bool         `json:"is_draft"`                // Whether PR is a draft
	Mergeable    string       `json:"mergeable"`               // MERGEABLE, CONFLICTING, UNKNOWN
	Labels       []string     `json:"labels,omitempty"`        // Label names
	ChangedFiles int          `json:"changed_files"`           // Number of changed files
	Additions    int          `json:"additions"`               // Lines added
	Deletions    int          `json:"deletions"`               // Lines deleted
	ReviewStatus string       `json:"review_status,omitempty"` // APPROVED, CHANGES_REQUESTED, REVIEW_REQUIRED
	Reviewers    []Reviewer   `json:"reviewers,omitempty"`     // Individual reviewer decisions
	Comments     int          `json:"comments"`                // Total comment count
	Checks       []Check      `json:"checks,omitempty"`        // Individual CI/CD check runs
	CheckSummary CheckSummary `json:"check_summary"`           // Aggregated check counts
	URL          string       `json:"url"`                     // PR URL on GitHub
	FetchedAt    int64        `json:"fetched_at,omitempty"`    // Unix timestamp of last fetch
}

// Reviewer represents a PR reviewer and their decision.
type Reviewer struct {
	Login string `json:"login"` // GitHub username
	State string `json:"state"` // APPROVED, CHANGES_REQUESTED, COMMENTED, PENDING
}

// Check represents a single CI/CD check run.
type Check struct {
	Name       string `json:"name"`       // Check name (e.g., "build", "lint")
	Status     string `json:"status"`     // COMPLETED, IN_PROGRESS, QUEUED, PENDING
	Conclusion string `json:"conclusion"` // SUCCESS, FAILURE, NEUTRAL, etc. (empty if not completed)
}

// CheckSummary contains aggregated counts of check statuses.
type CheckSummary struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Pending int `json:"pending"`
}

// IsAllPassed returns true if all checks have passed.
func (cs CheckSummary) IsAllPassed() bool {
	return cs.Total > 0 && cs.Passed == cs.Total
}

// HasFailures returns true if any check has failed.
func (cs CheckSummary) HasFailures() bool {
	return cs.Failed > 0
}

// IsPending returns true if any check is still pending or in progress.
func (cs CheckSummary) IsPending() bool {
	return cs.Pending > 0
}

// CheckIndicator returns the display indicator for the aggregate check status.
func (cs CheckSummary) CheckIndicator() string {
	if cs.Total == 0 {
		return ""
	}
	if cs.HasFailures() {
		return IndicatorFail
	}
	if cs.IsPending() {
		return IndicatorPending
	}
	if cs.IsAllPassed() {
		return IndicatorPass
	}
	return ""
}

// IsStale returns true if the PR detail data is older than PRCacheMaxAge.
func (pr *PRDetail) IsStale() bool {
	if pr == nil || pr.FetchedAt == 0 {
		return true
	}
	return time.Since(time.Unix(pr.FetchedAt, 0)) > PRCacheMaxAge
}

// MergeIndicator returns a display string for the mergeable status.
func (pr *PRDetail) MergeIndicator() string {
	switch pr.Mergeable {
	case MergeableMergeable:
		return IndicatorPass + " No conflicts"
	case MergeableConflicting:
		return IndicatorFail + " Has conflicts"
	default:
		return IndicatorUnknown + " Unknown"
	}
}

// ReviewIndicator returns a display string for the review status.
func (pr *PRDetail) ReviewIndicator() string {
	switch pr.ReviewStatus {
	case ReviewApproved:
		names := pr.reviewerNames(ReviewerApproved)
		if len(names) > 0 {
			return IndicatorPass + " Approved by " + joinNames(names)
		}
		return IndicatorPass + " Approved"
	case ReviewChangesRequired:
		names := pr.reviewerNames(ReviewerChangesRequired)
		if len(names) > 0 {
			return IndicatorFail + " Changes requested by " + joinNames(names)
		}
		return IndicatorFail + " Changes requested"
	case ReviewRequired:
		return IndicatorWaiting + " Review pending"
	default:
		if len(pr.Reviewers) == 0 {
			return "No reviewers assigned"
		}
		return IndicatorWaiting + " Review pending"
	}
}

// reviewerNames returns the login names of reviewers with the given state.
func (pr *PRDetail) reviewerNames(state string) []string {
	var names []string
	for _, r := range pr.Reviewers {
		if r.State == state {
			names = append(names, r.Login)
		}
	}
	return names
}

// joinNames joins a slice of names with commas and "and" for the last element.
func joinNames(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	default:
		result := ""
		for i, name := range names {
			if i == len(names)-1 {
				result += "and " + name
			} else {
				result += name + ", "
			}
		}
		return result
	}
}

// PRComment represents a single comment on a PR (either review or general).
type PRComment struct {
	Author    string `json:"author"`               // GitHub username
	Body      string `json:"body"`                  // Comment text
	CreatedAt string `json:"created_at"`            // ISO 8601 timestamp
	UpdatedAt string `json:"updated_at,omitempty"`  // ISO 8601 timestamp
	Type      string `json:"type"`                  // "review" or "comment"
	FilePath  string `json:"file_path,omitempty"`   // File path for review comments
	Line      int    `json:"line,omitempty"`         // Line number for review comments
}

// Comment type constants
const (
	CommentTypeReview  = "review"
	CommentTypeGeneral = "comment"
)

// ghAPIReviewComment represents the JSON structure of a review comment from the GitHub API.
type ghAPIReviewComment struct {
	User         ghAPIUser `json:"user"`
	Body         string    `json:"body"`
	CreatedAt    string    `json:"created_at"`
	UpdatedAt    string    `json:"updated_at"`
	Path         string    `json:"path"`
	Line         int       `json:"line"`
	OriginalLine int       `json:"original_line"`
}

// ghAPIGeneralComment represents the JSON structure of an issue comment from the GitHub API.
type ghAPIGeneralComment struct {
	User      ghAPIUser `json:"user"`
	Body      string    `json:"body"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

// ghAPIUser represents the user object in GitHub API responses.
type ghAPIUser struct {
	Login string `json:"login"`
}

// parseReviewComments parses raw JSON from the GitHub pulls/comments API into PRComment slices.
func parseReviewComments(data []byte) ([]PRComment, error) {
	var raw []ghAPIReviewComment
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse review comments: %w", err)
	}

	comments := make([]PRComment, 0, len(raw))
	for _, r := range raw {
		line := r.Line
		if line == 0 {
			line = r.OriginalLine
		}
		comments = append(comments, PRComment{
			Author:    r.User.Login,
			Body:      r.Body,
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
			Type:      CommentTypeReview,
			FilePath:  r.Path,
			Line:      line,
		})
	}
	return comments, nil
}

// parseGeneralComments parses raw JSON from the GitHub issues/comments API into PRComment slices.
func parseGeneralComments(data []byte) ([]PRComment, error) {
	var raw []ghAPIGeneralComment
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse general comments: %w", err)
	}

	comments := make([]PRComment, 0, len(raw))
	for _, r := range raw {
		comments = append(comments, PRComment{
			Author:    r.User.Login,
			Body:      r.Body,
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
			Type:      CommentTypeGeneral,
		})
	}
	return comments, nil
}

// sortCommentsByCreatedAt sorts comments chronologically by CreatedAt (ISO 8601 format).
func sortCommentsByCreatedAt(comments []PRComment) {
	sort.Slice(comments, func(i, j int) bool {
		return comments[i].CreatedAt < comments[j].CreatedAt
	})
}

// GetPRComments fetches both review comments and general comments for a PR.
// It uses the local git directory to determine the owner/repo from the remote URL.
func GetPRComments(dir string, prNum int) ([]PRComment, error) {
	remoteURL := GetRemote(dir)
	ghInfo := ParseGitHubRemote(remoteURL)
	if ghInfo == nil {
		return nil, fmt.Errorf("could not determine GitHub owner/repo from remote URL")
	}
	return GetPRCommentsByRepo(ghInfo.Owner, ghInfo.Repo, prNum)
}

// GetPRCommentsByRepo fetches PR comments using owner/repo directly (for remote sessions).
// It fetches both review comments and general issue comments, then merges and sorts them chronologically.
func GetPRCommentsByRepo(owner, repo string, prNum int) ([]PRComment, error) {
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("owner and repo must not be empty")
	}
	if prNum <= 0 {
		return nil, fmt.Errorf("PR number must be positive")
	}

	// Fetch review comments (pull request inline comments)
	reviewPath := fmt.Sprintf("repos/%s/%s/pulls/%d/comments", owner, repo, prNum)
	reviewCmd := exec.Command("gh", "api", "--paginate", reviewPath)
	reviewOutput, err := reviewCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch review comments: %w", err)
	}

	// Fetch general comments (issue-level comments)
	generalPath := fmt.Sprintf("repos/%s/%s/issues/%d/comments", owner, repo, prNum)
	generalCmd := exec.Command("gh", "api", "--paginate", generalPath)
	generalOutput, err := generalCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch general comments: %w", err)
	}

	// Parse review comments
	var reviewComments []PRComment
	if len(reviewOutput) > 0 {
		reviewComments, err = parseReviewComments(reviewOutput)
		if err != nil {
			return nil, err
		}
	}

	// Parse general comments
	var generalComments []PRComment
	if len(generalOutput) > 0 {
		generalComments, err = parseGeneralComments(generalOutput)
		if err != nil {
			return nil, err
		}
	}

	// Merge and sort chronologically
	allComments := make([]PRComment, 0, len(reviewComments)+len(generalComments))
	allComments = append(allComments, reviewComments...)
	allComments = append(allComments, generalComments...)
	sortCommentsByCreatedAt(allComments)

	return allComments, nil
}

// ghPRFields is the comma-separated list of fields to request from gh pr view --json.
const ghPRFields = "number,title,state,isDraft,mergeable,labels,changedFiles,additions,deletions,reviewDecision,reviews,comments,statusCheckRollup,url"

// ghPRViewJSON is the raw JSON structure returned by gh pr view --json.
// It differs from PRDetail in field naming and nested structure.
type ghPRViewJSON struct {
	Number            int               `json:"number"`
	Title             string            `json:"title"`
	State             string            `json:"state"`
	IsDraft           bool              `json:"isDraft"`
	Mergeable         string            `json:"mergeable"`
	Labels            []ghLabel         `json:"labels"`
	ChangedFiles      int               `json:"changedFiles"`
	Additions         int               `json:"additions"`
	Deletions         int               `json:"deletions"`
	ReviewDecision    string            `json:"reviewDecision"`
	Reviews           []ghReview        `json:"reviews"`
	Comments          []json.RawMessage `json:"comments"`
	StatusCheckRollup []ghStatusCheck   `json:"statusCheckRollup"`
	URL               string            `json:"url"`
}

// ghLabel represents a label in the gh CLI JSON output.
type ghLabel struct {
	Name string `json:"name"`
}

// ghReview represents a review in the gh CLI JSON output.
type ghReview struct {
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
	State string `json:"state"`
	Body  string `json:"body"`
}

// ghStatusCheck represents a status check in the gh CLI JSON output.
// The __typename field determines whether it is a CheckRun or StatusContext.
type ghStatusCheck struct {
	TypeName   string `json:"__typename"`
	Name       string `json:"name"`       // CheckRun
	Status     string `json:"status"`     // CheckRun: COMPLETED, IN_PROGRESS, etc.
	Conclusion string `json:"conclusion"` // CheckRun: SUCCESS, FAILURE, etc.
	Context    string `json:"context"`    // StatusContext: used as name
	State      string `json:"state"`      // StatusContext: SUCCESS, FAILURE, PENDING, ERROR
}

// statusContextTypeName is the __typename value for StatusContext checks.
const statusContextTypeName = "StatusContext"

// parsePRDetailJSON parses the raw gh pr view --json output into a PRDetail.
// Returns nil if the data cannot be parsed.
func parsePRDetailJSON(data []byte) *PRDetail {
	var raw ghPRViewJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	pr := &PRDetail{
		Number:       raw.Number,
		Title:        raw.Title,
		State:        raw.State,
		Draft:        raw.IsDraft,
		Mergeable:    raw.Mergeable,
		ChangedFiles: raw.ChangedFiles,
		Additions:    raw.Additions,
		Deletions:    raw.Deletions,
		ReviewStatus: raw.ReviewDecision,
		URL:          raw.URL,
		FetchedAt:    time.Now().Unix(),
	}

	// Extract label names
	for _, l := range raw.Labels {
		pr.Labels = append(pr.Labels, l.Name)
	}

	// Map reviews to Reviewer structs
	reviewsWithBody := 0
	for _, r := range raw.Reviews {
		pr.Reviewers = append(pr.Reviewers, Reviewer{
			Login: r.Author.Login,
			State: r.State,
		})
		if r.Body != "" {
			reviewsWithBody++
		}
	}

	// Comment count: general comments + reviews with a body
	pr.Comments = len(raw.Comments) + reviewsWithBody

	// Parse status checks
	for _, sc := range raw.StatusCheckRollup {
		check := parseStatusCheck(sc)
		pr.Checks = append(pr.Checks, check)
	}

	// Compute check summary from parsed checks
	pr.CheckSummary = computeCheckSummary(pr.Checks)

	return pr
}

// parseStatusCheck converts a ghStatusCheck into a Check, handling both
// CheckRun and StatusContext types.
func parseStatusCheck(sc ghStatusCheck) Check {
	if sc.TypeName == statusContextTypeName {
		return Check{
			Name:       sc.Context,
			Status:     CheckStatusCompleted,
			Conclusion: mapStatusContextState(sc.State),
		}
	}
	// CheckRun (default)
	return Check{
		Name:       sc.Name,
		Status:     sc.Status,
		Conclusion: sc.Conclusion,
	}
}

// mapStatusContextState maps StatusContext state values to Check conclusion values.
func mapStatusContextState(state string) string {
	switch state {
	case "SUCCESS":
		return CheckConclusionSuccess
	case "FAILURE":
		return CheckConclusionFailure
	case "PENDING":
		return CheckStatusPending
	case "ERROR":
		return CheckConclusionFailure
	default:
		return state
	}
}

// computeCheckSummary aggregates check results into a CheckSummary.
func computeCheckSummary(checks []Check) CheckSummary {
	summary := CheckSummary{Total: len(checks)}
	for _, c := range checks {
		switch {
		case c.Status == CheckStatusCompleted && c.Conclusion == CheckConclusionSuccess:
			summary.Passed++
		case c.Status == CheckStatusCompleted && (c.Conclusion == CheckConclusionFailure ||
			c.Conclusion == CheckConclusionCancelled ||
			c.Conclusion == CheckConclusionTimedOut):
			summary.Failed++
		default:
			// IN_PROGRESS, QUEUED, PENDING, or completed with neutral/action_required
			summary.Pending++
		}
	}
	return summary
}

// GetPRDetail fetches extended PR metadata for the current branch using gh CLI.
// Returns nil on any error (gh not installed, no PR, parse failure, etc.).
func GetPRDetail(dir string) *PRDetail {
	cmd := exec.Command("gh", "pr", "view", "--json", ghPRFields)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return nil
	}
	return parsePRDetailJSON(output)
}

// GetPRDetailByRepo fetches extended PR metadata using -R flag for remote sessions.
// Uses the branch name and remote URL to identify the PR without a local checkout.
// Returns nil on any error.
func GetPRDetailByRepo(branch, remoteURL string) *PRDetail {
	ghInfo := ParseGitHubRemote(remoteURL)
	if ghInfo == nil || branch == "" {
		return nil
	}

	repo := fmt.Sprintf("%s/%s", ghInfo.Owner, ghInfo.Repo)
	cmd := exec.Command("gh", "pr", "view", branch, "-R", repo, "--json", ghPRFields)
	output, err := cmd.Output()
	if err != nil {
		return nil
	}
	return parsePRDetailJSON(output)
}
