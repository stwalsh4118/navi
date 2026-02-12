# PBI-38: Enhanced GitHub PR Integration

## Overview

The current PR integration shows only the PR number and a link to open it in the browser. This PBI enriches the PR view with CI/CD check statuses, review state, comment counts, comment viewing, labels, draft status, merge readiness, and changed file stats — giving users full PR context without leaving the TUI.

[View in Backlog](../backlog.md#user-content-38)

## Problem Statement

When monitoring Claude Code sessions, developers frequently need to check on the state of pull requests associated with each session's branch. Currently, the TUI shows only the PR number and a clickable link — forcing users to leave the terminal and visit GitHub for critical context:

- **No CI/CD visibility** — Users can't see if checks are passing, failing, or still running. This is the most common reason to visit the PR page.
- **No review status** — No indication of whether the PR has been approved, has changes requested, or is awaiting review.
- **No comment awareness** — Users don't know if there are review comments to address without visiting GitHub.
- **No merge readiness** — No way to tell if the PR has conflicts or is ready to merge.
- **No metadata** — Labels, draft status, and change stats are invisible.

All of this data is available via the `gh` CLI and can be fetched with a single `gh pr view --json` call.

## User Stories

- As a user, I want to see CI/CD check statuses (passing/failing/pending) for my PR so that I know if the build is green without leaving the TUI.
- As a user, I want to see the review status (approved/changes requested/pending) so that I know if my PR needs attention.
- As a user, I want to see the PR comment count so that I know if there are comments to address.
- As a user, I want to browse PR comments in the TUI so that I can read review feedback without switching to a browser.
- As a user, I want to see whether my PR is mergeable so that I know if there are conflicts to resolve.
- As a user, I want to see PR labels so that I have full context on categorization and workflow state.
- As a user, I want to see if a PR is in draft status so that I know it's not yet ready for review.
- As a user, I want to see changed files count and line stats so that I can gauge PR size at a glance.
- As a user, I want PR data to auto-refresh so that check statuses update as CI runs complete.

## Technical Approach

### 1. Extended PR Data Model

Extend the existing `git.Info` struct (or introduce a new `PRDetail` struct) to hold the additional PR metadata. All data will be fetched via `gh pr view --json` with a single call:

```go
type PRDetail struct {
    Number       int
    Title        string
    State        string        // OPEN, CLOSED, MERGED
    Draft        bool
    Mergeable    string        // MERGEABLE, CONFLICTING, UNKNOWN
    Labels       []string
    ChangedFiles int
    Additions    int
    Deletions    int
    ReviewStatus string        // APPROVED, CHANGES_REQUESTED, REVIEW_REQUIRED, ""
    Reviewers    []Reviewer    // Who reviewed and their decision
    Comments     int           // Total comment count (review + issue comments)
    Checks       []Check       // CI/CD check runs
    CheckSummary CheckSummary  // Aggregated: pass/fail/pending counts
    URL          string
    FetchedAt    int64
}

type Reviewer struct {
    Login  string
    State  string  // APPROVED, CHANGES_REQUESTED, COMMENTED, PENDING
}

type Check struct {
    Name       string
    Status     string  // COMPLETED, IN_PROGRESS, QUEUED, PENDING
    Conclusion string  // SUCCESS, FAILURE, NEUTRAL, CANCELLED, TIMED_OUT, ACTION_REQUIRED
}

type CheckSummary struct {
    Total   int
    Passed  int
    Failed  int
    Pending int
}
```

### 2. Data Fetching

#### Primary fetch

Use `gh pr view --json` with all required fields in a single call:

```bash
gh pr view --json number,title,state,isDraft,mergeable,labels,changedFiles,additions,deletions,reviewDecision,reviews,comments,statusCheckRollup,url
```

For remote sessions (no local directory):

```bash
gh pr view <branch> -R <owner/repo> --json <fields>
```

#### Comment fetching

Comments will be fetched on-demand when the user opens the comment viewer, using:

```bash
gh api repos/{owner}/{repo}/pulls/{number}/comments
gh api repos/{owner}/{repo}/issues/{number}/comments
```

This separates review comments (inline on code) from general PR comments.

#### Refresh strategy

- **Lazy initial fetch**: PR detail is fetched when the user first opens the git detail view (existing pattern).
- **Auto-refresh**: When the git detail view is open and checks are pending/in-progress, auto-refresh every 30 seconds.
- **Manual refresh**: A keybinding to force-refresh PR data.
- **Cache**: PR detail is cached per session. Cache is invalidated on manual refresh or when auto-refresh fires.

### 3. Enhanced Git Detail View

The existing git detail dialog will be expanded to show the new PR information. Layout:

```
╭─ Git Info ─────────────────────────────────────────╮
│  Branch:  feature/auth-flow                        │
│  Status:  ● +3 -1                                  │
│  Commit:  a1b2c3d Add OAuth2 flow                  │
│  Remote:  origin (github.com/user/repo)            │
│                                                    │
│  ── Pull Request #42 ──────────────────────────────│
│  Title:   Add OAuth2 authentication flow           │
│  State:   OPEN (draft)                             │
│  Review:  ⏳ Review pending                        │
│  Merge:   ✓ No conflicts                           │
│  Labels:  enhancement, auth                        │
│  Changed: 12 files (+340 / -85)                    │
│                                                    │
│  ── Checks (3/4 passed) ──────────────────────────│
│  ✓ build        passed                             │
│  ✓ lint         passed                             │
│  ✓ unit-tests   passed                             │
│  ● e2e-tests    running...                         │
│                                                    │
│  ── Comments (5) ─────────────────────────────────│
│  Press 'c' to view comments                        │
│                                                    │
│  o: open PR  r: refresh  c: comments  q: close     │
╰────────────────────────────────────────────────────╯
```

### 4. Session List PR Indicator

Enhance the inline PR indicator in the session list to show at-a-glance status:

```
feature/auth ● +3 -1 [PR#42 ✓]     # All checks passing
feature/auth ● +3 -1 [PR#42 ✗]     # Checks failing
feature/auth ● +3 -1 [PR#42 ●]     # Checks pending/running
feature/auth ● +3 -1 [PR#42 ✓ 💬5] # Passing with 5 comments
feature/auth ● +3 -1 [PR#42 draft] # Draft PR
```

The indicator character reflects overall check status:
- `✓` — All checks passed
- `✗` — Any check failed
- `●` — Checks pending or in progress
- No indicator if no checks configured

### 5. Comment Viewer

A scrollable panel (reusing the existing content viewer infrastructure from PBI-29) to display PR comments:

- Opens via `c` from the git detail view
- Shows both review comments (inline on code) and general PR comments
- Each comment displays: author, timestamp, body (rendered markdown where possible)
- Comments are sorted chronologically
- Scrollable with `j/k`, `g/G` for top/bottom
- `q` or `Escape` to close and return to git detail view

### 6. Review Status Display

Show review decision with appropriate indicators:

| Status | Display |
|--------|---------|
| Approved | `✓ Approved by alice, bob` |
| Changes requested | `✗ Changes requested by alice` |
| Review pending | `⏳ Review pending` |
| No reviewers | `No reviewers assigned` |

### 7. Auto-Refresh for Pending Checks

When the git detail view is open and checks are in a non-terminal state (pending, in_progress, queued):
- Start a ticker that refreshes PR data every 30 seconds
- Stop the ticker when all checks reach a terminal state or the view is closed
- Show a subtle "Auto-refreshing..." indicator when active

## UX/UI Considerations

### Visual indicators

- Use consistent status icons: `✓` (success/green), `✗` (failure/red), `●` (pending/yellow), `⏳` (waiting/dim)
- Color-code review status: green for approved, red for changes requested, dim for pending
- Labels displayed as space-separated values with dimmed styling
- Draft PRs shown with a `(draft)` suffix in dimmed style

### Keybinding additions

| Key | Action | Context |
|-----|--------|---------|
| `c` | Open comment viewer | Git detail view with PR |
| `r` | Refresh PR data | Git detail view |

Existing keybinding `o` (open PR in browser) remains unchanged.

### Progressive disclosure

- Session list shows minimal PR status (number + check indicator)
- Git detail view shows full PR metadata
- Comment viewer is a separate layer opened on demand
- This avoids overwhelming the user with data they may not need

### Performance considerations

- Single `gh pr view --json` call fetches all metadata (no multiple round-trips)
- Comments fetched only on demand (not preloaded)
- Auto-refresh only active when git detail view is open and checks are pending
- Cache prevents redundant fetches when switching between sessions

## Acceptance Criteria

1. **Check statuses**: Git detail view shows individual CI/CD check names, statuses, and conclusions. Session list shows aggregate check indicator (pass/fail/pending).
2. **Review status**: Git detail view shows review decision (approved/changes requested/pending) with reviewer names.
3. **Comment count**: Comment count displayed in both the session list PR indicator and git detail view.
4. **Comment viewer**: Users can open a scrollable comment viewer from the git detail view showing all PR comments with author, timestamp, and body.
5. **Mergeable status**: Git detail view shows whether the PR has conflicts or is ready to merge.
6. **Labels**: Git detail view shows PR labels.
7. **Draft indicator**: Draft PRs are clearly marked in both session list and git detail view.
8. **Changed files stats**: Git detail view shows number of changed files, additions, and deletions.
9. **Auto-refresh**: PR data auto-refreshes every 30 seconds when checks are pending and the git detail view is open.
10. **Manual refresh**: `r` keybinding force-refreshes PR data from the git detail view.
11. **Remote session support**: All PR enhancements work for remote sessions using the `-R` flag pattern.
12. **No performance regression**: PR data fetching does not slow down session list rendering or polling.
13. **All existing tests pass**; new tests cover PR data parsing, check aggregation, review status, and comment fetching.

## Dependencies

- Requires `gh` CLI installed and authenticated (existing requirement)
- Builds on existing git integration (PBI-16) and content viewer (PBI-29)

## Open Questions

- None.

## Related Tasks

[View Tasks](./tasks.md)
