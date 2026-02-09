#!/bin/bash
# tool-tracker.sh - Hook script for tracking tool usage in navi.
# Called by Claude Code PostToolUse hooks to track tool counts and recent tools.
# Receives JSON input via stdin with tool_name, tool_input, tool_response, tool_use_id.
# Skips processing for teammate events (agent-level tool metrics are not tracked).

DIR="$HOME/.claude-sessions"
mkdir -p "$DIR"

SESSION=$(tmux display-message -p '#{session_name}' 2>/dev/null || echo "unknown")
SESSION_FILE="$DIR/$SESSION.json"

# Read stdin JSON
# The hook receives JSON like: {"tool_name": "Read", "tool_input": {...}, ...}
if command -v jq &> /dev/null; then
    INPUT=$(cat)

    # Skip processing for teammate events - agent-level tool metrics are not tracked
    TEAMMATE_NAME=$(echo "$INPUT" | jq -r '.teammate_name // ""' 2>/dev/null)
    [ "$TEAMMATE_NAME" = "null" ] && TEAMMATE_NAME=""
    if [ -n "$TEAMMATE_NAME" ]; then
        exit 0
    fi

    TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // ""')
else
    # Fallback: try to extract tool_name with grep/sed
    INPUT=$(cat)
    TOOL_NAME=$(echo "$INPUT" | grep -o '"tool_name"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*: *"\([^"]*\)".*/\1/')
fi

# Exit if we couldn't get a tool name
if [ -z "$TOOL_NAME" ] || [ "$TOOL_NAME" = "null" ]; then
    exit 0
fi

# Constants
MAX_RECENT_TOOLS=10

# Read existing session data
TOOL_COUNTS="{}"
RECENT_TOOLS="[]"
EXISTING_TEAM=""

if [ -f "$SESSION_FILE" ] && command -v jq &> /dev/null; then
    TOOL_COUNTS=$(jq -r '.metrics.tools.counts // {}' "$SESSION_FILE" 2>/dev/null || echo "{}")
    RECENT_TOOLS=$(jq -r '.metrics.tools.recent // []' "$SESSION_FILE" 2>/dev/null || echo "[]")
    # Preserve existing team data
    EXISTING_TEAM=$(jq '.team // null' "$SESSION_FILE" 2>/dev/null || echo "null")
    [ "$EXISTING_TEAM" = "null" ] && EXISTING_TEAM=""
fi

# Update tool counts - increment count for this tool
if command -v jq &> /dev/null; then
    CURRENT_COUNT=$(echo "$TOOL_COUNTS" | jq -r --arg tool "$TOOL_NAME" '.[$tool] // 0')
    NEW_COUNT=$((CURRENT_COUNT + 1))
    TOOL_COUNTS=$(echo "$TOOL_COUNTS" | jq --arg tool "$TOOL_NAME" --argjson count "$NEW_COUNT" '.[$tool] = $count')

    # Update recent tools - add to front, keep only last N
    RECENT_TOOLS=$(echo "$RECENT_TOOLS" | jq --arg tool "$TOOL_NAME" --argjson max "$MAX_RECENT_TOOLS" '[$tool] + . | .[:$max]')
fi

# Read other existing session data to preserve it
if [ -f "$SESSION_FILE" ] && command -v jq &> /dev/null; then
    TMUX_SESSION=$(jq -r '.tmux_session // ""' "$SESSION_FILE")
    STATUS=$(jq -r '.status // "working"' "$SESSION_FILE")
    MESSAGE=$(jq -r '.message // ""' "$SESSION_FILE")
    CWD=$(jq -r '.cwd // ""' "$SESSION_FILE")
    TIMESTAMP=$(jq -r '.timestamp // 0' "$SESSION_FILE")
    STARTED=$(jq -r '.metrics.time.started // 0' "$SESSION_FILE")
    TOTAL_SECONDS=$(jq -r '.metrics.time.total_seconds // 0' "$SESSION_FILE")
    WORKING_SECONDS=$(jq -r '.metrics.time.working_seconds // 0' "$SESSION_FILE")
    WAITING_SECONDS=$(jq -r '.metrics.time.waiting_seconds // 0' "$SESSION_FILE")

    # Handle null values
    [ "$TMUX_SESSION" = "null" ] && TMUX_SESSION="$SESSION"
    [ "$STATUS" = "null" ] && STATUS="working"
    [ "$MESSAGE" = "null" ] && MESSAGE=""
    [ "$TIMESTAMP" = "null" ] && TIMESTAMP=$(date +%s)
    [ "$STARTED" = "null" ] && STARTED=0
    [ "$TOTAL_SECONDS" = "null" ] && TOTAL_SECONDS=0
    [ "$WORKING_SECONDS" = "null" ] && WORKING_SECONDS=0
    [ "$WAITING_SECONDS" = "null" ] && WAITING_SECONDS=0
else
    # Session file doesn't exist, use defaults
    TMUX_SESSION="$SESSION"
    STATUS="working"
    MESSAGE=""
    CWD=$(tmux display-message -p '#{pane_current_path}' 2>/dev/null || echo "")
    TIMESTAMP=$(date +%s)
    STARTED=$(date +%s)
    TOTAL_SECONDS=0
    WORKING_SECONDS=0
    WAITING_SECONDS=0
fi

# Build team field if it exists
TEAM_FIELD=""
if [ -n "$EXISTING_TEAM" ] && [ "$EXISTING_TEAM" != "null" ]; then
    TEAM_FIELD=",
  \"team\": $EXISTING_TEAM"
fi

# Write updated session file with tool metrics (atomic via temp file)
TMPFILE=$(mktemp "$DIR/.tmp.XXXXXX")
cat > "$TMPFILE" <<EOF
{
  "tmux_session": "$TMUX_SESSION",
  "status": "$STATUS",
  "message": "$MESSAGE",
  "cwd": "$CWD",
  "timestamp": $TIMESTAMP,
  "metrics": {
    "time": {
      "started": $STARTED,
      "total_seconds": $TOTAL_SECONDS,
      "working_seconds": $WORKING_SECONDS,
      "waiting_seconds": $WAITING_SECONDS
    },
    "tools": {
      "recent": $RECENT_TOOLS,
      "counts": $TOOL_COUNTS
    }
  }$TEAM_FIELD
}
EOF
mv "$TMPFILE" "$SESSION_FILE"
