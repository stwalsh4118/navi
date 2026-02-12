# Git PR API

Package: `internal/git` (file: `pr.go`)

## Data Types

### PRDetail

Extended PR metadata fetched from the GitHub CLI.

```go
type PRDetail struct {
    Number       int          // PR number
    Title        string       // PR title
    State        string       // OPEN, CLOSED, MERGED
    Draft        bool         // Whether PR is a draft
    Mergeable    string       // MERGEABLE, CONFLICTING, UNKNOWN
    Labels       []string     // Label names
    ChangedFiles int          // Number of changed files
    Additions    int          // Lines added
    Deletions    int          // Lines deleted
    ReviewStatus string       // APPROVED, CHANGES_REQUESTED, REVIEW_REQUIRED
    Reviewers    []Reviewer   // Individual reviewer decisions
    Comments     int          // Total comment count
    Checks       []Check      // Individual CI/CD check runs
    CheckSummary CheckSummary // Aggregated check counts
    URL          string       // PR URL on GitHub
    FetchedAt    int64        // Unix timestamp of last fetch
}
```

Methods: `IsStale() bool`, `MergeIndicator() string`, `ReviewIndicator() string`

### Reviewer

```go
type Reviewer struct {
    Login string // GitHub username
    State string // APPROVED, CHANGES_REQUESTED, COMMENTED, PENDING
}
```

### Check

```go
type Check struct {
    Name       string // Check name (e.g., "build", "lint")
    Status     string // COMPLETED, IN_PROGRESS, QUEUED, PENDING
    Conclusion string // SUCCESS, FAILURE, NEUTRAL, etc.
}
```

### CheckSummary

```go
type CheckSummary struct {
    Total, Passed, Failed, Pending int
}
```

Methods: `IsAllPassed() bool`, `HasFailures() bool`, `IsPending() bool`, `CheckIndicator() string`

### PRComment

```go
type PRComment struct {
    Author    string // GitHub username
    Body      string // Comment text
    CreatedAt string // ISO 8601 timestamp
    UpdatedAt string // ISO 8601 timestamp
    Type      string // "review" or "comment"
    FilePath  string // File path (review comments only)
    Line      int    // Line number (review comments only)
}
```

## Public Functions

### PR Detail Fetching

```go
// GetPRDetail fetches PR metadata for the current branch using gh CLI.
// Uses the working directory to determine the repo. Returns nil on error.
func GetPRDetail(dir string) *PRDetail

// GetPRDetailByRepo fetches PR metadata using -R flag for remote sessions.
// Uses branch name and remote URL. Returns nil on error.
func GetPRDetailByRepo(branch, remoteURL string) *PRDetail
```

### PR Comment Fetching

```go
// GetPRComments fetches review + general comments using local git dir.
func GetPRComments(dir string, prNum int) ([]PRComment, error)

// GetPRCommentsByRepo fetches comments using owner/repo directly (remote sessions).
// Fetches both pulls/{n}/comments and issues/{n}/comments, merges and sorts chronologically.
func GetPRCommentsByRepo(owner, repo string, prNum int) ([]PRComment, error)
```

## Constants

| Group | Constants |
|-------|-----------|
| PR State | `PRStateOpen`, `PRStateClosed`, `PRStateMerged` |
| Mergeable | `MergeableMergeable`, `MergeableConflicting`, `MergeableUnknown` |
| Review Decision | `ReviewApproved`, `ReviewChangesRequired`, `ReviewRequired` |
| Reviewer State | `ReviewerApproved`, `ReviewerChangesRequired`, `ReviewerCommented`, `ReviewerPending` |
| Check Status | `CheckStatusCompleted`, `CheckStatusInProgress`, `CheckStatusQueued`, `CheckStatusPending` |
| Check Conclusion | `CheckConclusionSuccess`, `CheckConclusionFailure`, `CheckConclusionNeutral`, `CheckConclusionCancelled`, `CheckConclusionTimedOut`, `CheckConclusionActionRequired` |
| Display | `IndicatorPass` (✓), `IndicatorFail` (✗), `IndicatorPending` (●), `IndicatorWaiting` (⏳), `IndicatorUnknown` (?), `IndicatorDraft`, `CommentIcon` (💬) |
| Comment Type | `CommentTypeReview`, `CommentTypeGeneral` |
| Cache | `PRCacheMaxAge` (60s) |

## TUI Integration

TUI message types (in `internal/tui/model.go`):
- `gitPRMsg` — carries fetched `*PRDetail`
- `gitPRCommentsMsg` — carries fetched `[]PRComment` and error

TUI commands (in `internal/tui/sessions.go`):
- `fetchPRCmd(dir)` — async fetch of PR detail for local session
- `fetchRemotePRCmd(branch, remoteURL)` — async fetch for remote session
- `fetchPRCommentsCmd(dir, prNum)` — async fetch of PR comments for local session
- `fetchRemotePRCommentsCmd(owner, repo, prNum)` — async fetch for remote session
- `prAutoRefreshTickCmd()` — ticker for auto-refreshing pending checks (30s interval)
