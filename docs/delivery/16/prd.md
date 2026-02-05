# PBI-16: Git Integration

[View in Backlog](../backlog.md)

## Overview

Display git information (branch, status, diffs) for each session's working directory, and provide links to GitHub issues/PRs when detectable from branch names.

## Problem Statement

Claude sessions often work on git repositories, but users can't see git context without attaching or opening another terminal. Knowing the branch, dirty status, and related PR helps users understand session context at a glance.

## User Stories

- As a user, I want to see the git branch for each session so I know what feature is being worked on
- As a user, I want to see if there are uncommitted changes so I know the state of the work
- As a user, I want quick access to related GitHub PRs so I can review changes
- As a user, I want to preview diffs without attaching to the session

## Technical Approach

### Git Status Collection

Extend session polling to gather git info:

```bash
# In session directory
git rev-parse --abbrev-ref HEAD  # branch name
git status --porcelain           # dirty status
git log -1 --format="%h %s"      # last commit
git remote get-url origin        # for GitHub detection
```

### Extended Session Data

```json
{
  "tmux_session": "hyperion",
  "status": "working",
  "cwd": "/home/user/projects/hyperion",
  "git": {
    "branch": "feature/auth-flow",
    "dirty": true,
    "ahead": 3,
    "behind": 0,
    "last_commit": "abc1234 Add login validation",
    "remote": "github.com/user/hyperion"
  }
}
```

### Display

Session row with git info:
```
  ⚙️  hyperion                                      2m ago
      ~/projects/hyperion
      feature/auth-flow ● +3 [PR #42]
      "Implementing the OAuth callback..."
```

Legend:
- Branch name shown
- `●` indicates dirty (uncommitted changes)
- `+3` means 3 commits ahead of remote
- `[PR #42]` links to detected PR (from branch name pattern)

### GitHub Integration

1. Parse remote URL to detect GitHub repos
2. Detect PR from branch name patterns:
   - `feature/123-description` → PR might exist
   - `fix/issue-456` → Issue #456
3. Use `gh pr view --json number` if available
4. Cache PR info to avoid repeated API calls

### Diff Preview

1. Press `G` to open git detail view for selected session
2. Show:
   - Full branch info
   - File change summary
   - Diff preview (using `git diff --stat`)
3. Option to view full diff in preview pane

## UX/UI Considerations

- Git info should be compact in list view
- Dirty indicator should be prominent (uncommitted work at risk)
- PR links should be actionable (open in browser)
- Don't slow down polling with git commands (run async)
- Cache git info, update less frequently than status

## Acceptance Criteria

1. Branch name displayed for sessions in git repos
2. Dirty/clean status indicator shown
3. Ahead/behind remote count displayed
4. GitHub PR detected from branch name (when applicable)
5. `G` opens git detail view with full info
6. Diff preview available for uncommitted changes
7. PR number links to GitHub (can open in browser)
8. Git info cached and updated periodically (not every poll)
9. Non-git directories handled gracefully

## Dependencies

- PBI-3: Session polling (base polling mechanism)
- PBI-11: Preview pane (for diff preview display)

## Open Questions

- Should we support GitLab/Bitbucket PR detection?
- Should we show git info for all branches or just the current one?
- Should there be a git log view?

## Related Tasks

See [Tasks](./tasks.md)
