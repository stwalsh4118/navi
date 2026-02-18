#!/usr/bin/env bash
set -euo pipefail

# markdown-tasks.sh — Navi built-in provider
# Parses a local markdown-based task management structure (docs/delivery/)
# and outputs tasks in the standard task JSON format.
#
# Environment variables:
#   NAVI_TASK_ARG_PATH          — path to the delivery docs directory (default: docs/delivery)
#   NAVI_TASK_ARG_STATUS_FILTER — comma-separated PBI statuses to include (optional; e.g. "Agreed,InProgress")

# ── dependency checks ────────────────────────────────────────────────
if ! command -v jq &>/dev/null; then
  echo "error: 'jq' is not installed. Install from https://jqlang.github.io/jq/" >&2
  exit 1
fi

# ── resolve delivery docs path ───────────────────────────────────────
path="${NAVI_TASK_ARG_PATH:-docs/delivery}"

if [[ ! -d "$path" ]]; then
  echo "error: directory not found: $path" >&2
  exit 1
fi

backlog_file="$path/backlog.md"
if [[ ! -f "$backlog_file" ]]; then
  echo "error: backlog file not found: $backlog_file" >&2
  exit 1
fi

# ── parse optional status filter ─────────────────────────────────────
status_filter="${NAVI_TASK_ARG_STATUS_FILTER:-}"
has_filter=false

# Convert comma-separated filter into an associative array for fast lookup.
declare -A filter_map
if [[ -n "$status_filter" ]]; then
  has_filter=true
  IFS=',' read -ra filter_parts <<< "$status_filter"
  for f in "${filter_parts[@]}"; do
    # Trim whitespace
    trimmed="$(echo "$f" | xargs)"
    # Store lowercase for case-insensitive matching
    filter_map["${trimmed,,}"]=1
  done
fi

# ── helper: trim whitespace ──────────────────────────────────────────
trim() {
  local s="$1"
  # Remove leading/trailing whitespace
  s="${s#"${s%%[![:space:]]*}"}"
  s="${s%"${s##*[![:space:]]}"}"
  echo "$s"
}

normalize_status() {
  local status="$1"
  status="${status,,}"
  status="${status//[[:space:]]/}"
  status="${status//_/}"
  status="${status//-/}"
  echo "$status"
}

# ── helper: extract link text from markdown link ─────────────────────
# Given "[Some Text](./path.md)", returns "Some Text"
# If no link, returns the input as-is.
extract_link_text() {
  local s="$1"
  if [[ "$s" =~ ^\[([^]]*)\]\(.*\)$ ]]; then
    echo "${BASH_REMATCH[1]}"
  else
    echo "$s"
  fi
}

# ── parse backlog.md for PBIs ────────────────────────────────────────
# Format: | ID | Actor | User Story | Status | CoS |
# We skip non-table lines, the header row, and separator rows.

declare -a pbi_ids=()
declare -A pbi_titles=()
declare -A pbi_statuses=()

backlog_found=false
in_table=false

while IFS= read -r line; do
  # If we were in the backlog table and hit a non-table line, we're done
  if [[ ! "$line" =~ ^[[:space:]]*\| ]]; then
    if [[ "$in_table" == true ]]; then
      break
    fi
    continue
  fi

  # Skip separator rows (contain ---)
  if [[ "$line" =~ ---+ ]]; then
    continue
  fi

  # Detect the backlog table header (contains "ID" and "Status" and "User Story")
  if [[ "$in_table" == false ]]; then
    if [[ "$line" == *"ID"* ]] && [[ "$line" == *"Status"* ]] && [[ "$line" == *"User Story"* ]]; then
      backlog_found=true
      in_table=true
    fi
    continue
  fi

  # Parse table row: | ID | Actor | User Story | Status | CoS |
  # Split by | and extract fields
  IFS='|' read -ra fields <<< "$line"

  # fields[0] is empty (before first |), fields[1]=ID, fields[2]=Actor, fields[3]=User Story, fields[4]=Status
  if [[ ${#fields[@]} -lt 5 ]]; then
    continue
  fi

  pbi_id="$(trim "${fields[1]}")"
  pbi_title="$(trim "${fields[3]}")"
  pbi_status="$(trim "${fields[4]}")"

  # Skip empty IDs
  if [[ -z "$pbi_id" ]]; then
    continue
  fi

  # Apply status filter if set
  if [[ "$has_filter" == true ]]; then
    status_lower="${pbi_status,,}"
    if [[ -z "${filter_map[$status_lower]:-}" ]]; then
      continue
    fi
  fi

  pbi_ids+=("$pbi_id")
  pbi_titles["$pbi_id"]="$pbi_title"
  pbi_statuses["$pbi_id"]="$pbi_status"

done < "$backlog_file"

if [[ "$backlog_found" == false ]]; then
  echo "error: no backlog table found in $backlog_file (expected header with ID, User Story, Status columns)" >&2
  exit 1
fi

# Identify current PBI hint: first InProgress, then first Agreed.
current_pbi_id=""
current_pbi_title=""

for pbi_id in "${pbi_ids[@]}"; do
  normalized_status="$(normalize_status "${pbi_statuses[$pbi_id]}")"
  if [[ "$normalized_status" == "inprogress" ]]; then
    current_pbi_id="$pbi_id"
    break
  fi
done

if [[ -z "$current_pbi_id" ]]; then
  for pbi_id in "${pbi_ids[@]}"; do
    normalized_status="$(normalize_status "${pbi_statuses[$pbi_id]}")"
    if [[ "$normalized_status" == "agreed" ]]; then
      current_pbi_id="$pbi_id"
      break
    fi
  done
fi

# ── parse tasks for each PBI ─────────────────────────────────────────

# Build a JSON array of groups
groups_json="[]"

for pbi_id in "${pbi_ids[@]}"; do
  tasks_file="$path/$pbi_id/tasks.md"
  tasks_json="[]"

  if [[ -f "$tasks_file" ]]; then
    in_table=false

    while IFS= read -r line; do
      # If we were in the task table and hit a non-table line, we're done
      if [[ ! "$line" =~ ^[[:space:]]*\| ]]; then
        if [[ "$in_table" == true ]]; then
          break
        fi
        continue
      fi

      # Skip separator rows
      if [[ "$line" =~ ---+ ]]; then
        continue
      fi

      # Detect the task table header
      if [[ "$in_table" == false ]]; then
        if [[ "$line" == *"Task ID"* ]] && [[ "$line" == *"Status"* ]]; then
          in_table=true
        fi
        continue
      fi

      # Parse: | Task ID | Name | Status | Description |
      IFS='|' read -ra fields <<< "$line"

      if [[ ${#fields[@]} -lt 5 ]]; then
        continue
      fi

      task_id="$(trim "${fields[1]}")"
      task_name_raw="$(trim "${fields[2]}")"
      task_status="$(trim "${fields[3]}")"

      # Skip empty task IDs
      if [[ -z "$task_id" ]]; then
        continue
      fi

      # Extract link text from markdown links like [Task Name](./28-1.md)
      task_title="$(extract_link_text "$task_name_raw")"

      # Build task JSON object and append to tasks array
      task_obj="$(jq -n \
        --arg id "$task_id" \
        --arg title "$task_title" \
        --arg status "$task_status" \
        '{id: $id, title: $title, status: $status}')"

      tasks_json="$(echo "$tasks_json" | jq --argjson obj "$task_obj" '. + [$obj]')"

    done < "$tasks_file"
  fi

  # Read title from prd.md first line (e.g. "# PBI-28: Task View with Pluggable Providers")
  prd_file="$path/$pbi_id/prd.md"
  if [[ -f "$prd_file" ]]; then
    prd_title="$(head -1 "$prd_file")"
    # Strip leading "# " markdown heading prefix
    prd_title="${prd_title#\# }"
    # Strip "PBI-<id>: " prefix to keep just the name
    prd_title="${prd_title#PBI-${pbi_id}: }"
  else
    prd_title="${pbi_titles[$pbi_id]}"
  fi

  is_current=false
  if [[ -n "$current_pbi_id" && "$pbi_id" == "$current_pbi_id" ]]; then
    is_current=true
    current_pbi_title="$prd_title"
  fi

  # Build group JSON and append to groups array
  group_obj="$(jq -n \
    --arg id "PBI-$pbi_id" \
    --arg title "$prd_title" \
    --arg status "${pbi_statuses[$pbi_id]}" \
    --argjson tasks "$tasks_json" \
    --argjson is_current "$is_current" \
    '{id: $id, title: $title, status: $status, tasks: $tasks} + (if $is_current then {is_current: true} else {} end)')"

  groups_json="$(echo "$groups_json" | jq --argjson obj "$group_obj" '. + [$obj]')"
done

# ── output final JSON ────────────────────────────────────────────────
jq -n \
  --argjson groups "$groups_json" \
  --arg current_pbi_id "$current_pbi_id" \
  --arg current_pbi_title "$current_pbi_title" \
  '{groups: $groups} + (if $current_pbi_id != "" then {current_pbi_id: ("PBI-" + $current_pbi_id), current_pbi_title: $current_pbi_title} else {} end)'
