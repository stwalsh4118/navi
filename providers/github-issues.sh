#!/usr/bin/env bash
set -euo pipefail

# github-issues.sh — Navi built-in provider
# Fetches GitHub Issues via the gh CLI and outputs standard task JSON.
# Issues are grouped by milestone; issues without a milestone go into "Ungrouped".
#
# Environment variables:
#   NAVI_TASK_ARG_REPO   — owner/repo (optional; auto-detected if unset)
#   NAVI_TASK_ARG_LIMIT  — max issues to fetch (default 100)

# ── dependency checks ────────────────────────────────────────────────
if ! command -v gh &>/dev/null; then
  echo "error: 'gh' CLI is not installed. Install from https://cli.github.com/" >&2
  exit 1
fi

if ! command -v jq &>/dev/null; then
  echo "error: 'jq' is not installed. Install from https://jqlang.github.io/jq/" >&2
  exit 1
fi

# ── determine target repository ──────────────────────────────────────
repo="${NAVI_TASK_ARG_REPO:-}"
if [[ -z "$repo" ]]; then
  repo="$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null)" || {
    echo "error: could not auto-detect repository. Set NAVI_TASK_ARG_REPO or run from inside a git repo." >&2
    exit 1
  }
fi

# ── configuration ────────────────────────────────────────────────────
limit="${NAVI_TASK_ARG_LIMIT:-100}"

# ── fetch issues ─────────────────────────────────────────────────────
issues="$(gh issue list \
  --repo "$repo" \
  --json number,title,state,labels,assignees,milestone,url,createdAt,updatedAt \
  --limit "$limit" 2>/dev/null)" || {
  echo "error: failed to fetch issues from $repo. Check authentication (gh auth status) and network." >&2
  exit 1
}

# ── transform into standard task JSON ────────────────────────────────
echo "$issues" | jq --arg repo "$repo" '
  # Map a single issue to the standard task format.
  def to_task:
    {
      id:       ("#" + (.number | tostring)),
      title:    .title,
      status:   (if .state == "OPEN" then "open" elif .state == "CLOSED" then "closed" else (.state | ascii_downcase) end),
      assignee: (if (.assignees | length) > 0 then .assignees[0].login else null end),
      labels:   [.labels[].name],
      priority: 0,
      url:      .url,
      created:  .createdAt,
      updated:  .updatedAt
    }
    # Remove null optional fields.
    | with_entries(select(.value != null));

  # Separate issues with and without milestones.
  (map(select(.milestone != null and .milestone.title != null))) as $with_ms |
  (map(select(.milestone == null or .milestone.title == null))) as $without_ms |

  # Build groups from milestoned issues.
  (
    [$with_ms | group_by(.milestone.title)[] |
      {
        id:     (.[0].milestone.title | gsub("[^a-zA-Z0-9]"; "-") | ascii_downcase),
        title:  .[0].milestone.title,
        status: (if .[0].milestone.state == "open" then "open" elif .[0].milestone.state == "closed" then "closed" else "open" end),
        url:    .[0].milestone.url,
        tasks:  [.[] | to_task]
      }
    ]
  ) as $ms_groups |

  # Build the ungrouped group (if any issues lack a milestone).
  (
    if ($without_ms | length) > 0 then
      [{
        id:     "ungrouped",
        title:  "Ungrouped",
        status: "open",
        tasks:  [$without_ms[] | to_task]
      }]
    else
      []
    end
  ) as $ungrouped |

  # Combine: milestone groups first, then ungrouped.
  { groups: ($ms_groups + $ungrouped) }
'
