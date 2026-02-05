#!/bin/bash
# notify.sh - Hook script for navi to receive Claude Code status updates.
# Called by Claude Code hooks to write status JSON files to the shared status directory.
# Tracks session metrics including start time and status durations.

STATUS="$1"
MESSAGE="${CLAUDE_NOTIFICATION:-}"
DIR="$HOME/.claude-sessions"
mkdir -p "$DIR"

SESSION=$(tmux display-message -p '#{session_name}' 2>/dev/null || echo "unknown")
CWD=$(tmux display-message -p '#{pane_current_path}' 2>/dev/null || echo "")
CURRENT_TIME=$(date +%s)

# Initialize metrics variables
STARTED=0
TOTAL_SECONDS=0
WORKING_SECONDS=0
WAITING_SECONDS=0
PREV_STATUS=""
PREV_TIMESTAMP=0

# Initialize tool metrics (preserved from previous state)
TOOL_COUNTS="{}"
RECENT_TOOLS="[]"

# Read existing session data if it exists
SESSION_FILE="$DIR/$SESSION.json"
if [ -f "$SESSION_FILE" ]; then
    # Check if jq is available for JSON parsing
    if command -v jq &> /dev/null; then
        PREV_STATUS=$(jq -r '.status // ""' "$SESSION_FILE" 2>/dev/null)
        PREV_TIMESTAMP=$(jq -r '.timestamp // 0' "$SESSION_FILE" 2>/dev/null)
        STARTED=$(jq -r '.metrics.time.started // 0' "$SESSION_FILE" 2>/dev/null)
        WORKING_SECONDS=$(jq -r '.metrics.time.working_seconds // 0' "$SESSION_FILE" 2>/dev/null)
        WAITING_SECONDS=$(jq -r '.metrics.time.waiting_seconds // 0' "$SESSION_FILE" 2>/dev/null)

        # Read existing tool metrics to preserve them
        TOOL_COUNTS=$(jq -r '.metrics.tools.counts // {}' "$SESSION_FILE" 2>/dev/null || echo "{}")
        RECENT_TOOLS=$(jq -r '.metrics.tools.recent // []' "$SESSION_FILE" 2>/dev/null || echo "[]")

        # Handle null/empty values from jq
        [ "$STARTED" = "null" ] && STARTED=0
        [ "$WORKING_SECONDS" = "null" ] && WORKING_SECONDS=0
        [ "$WAITING_SECONDS" = "null" ] && WAITING_SECONDS=0
        [ "$PREV_TIMESTAMP" = "null" ] && PREV_TIMESTAMP=0
        [ "$PREV_STATUS" = "null" ] && PREV_STATUS=""
        [ "$TOOL_COUNTS" = "null" ] && TOOL_COUNTS="{}"
        [ "$RECENT_TOOLS" = "null" ] && RECENT_TOOLS="[]"
    fi
fi

# If this is a new session or coming back from offline, set start time
if [ "$STARTED" = "0" ] || [ -z "$STARTED" ] || [ "$PREV_STATUS" = "offline" ]; then
    STARTED=$CURRENT_TIME
    # Reset time counters for new session
    WORKING_SECONDS=0
    WAITING_SECONDS=0
fi

# Accumulate time based on PREVIOUS status
# This tracks how long we spent in the previous status before this update
if [ "$PREV_TIMESTAMP" != "0" ] && [ -n "$PREV_TIMESTAMP" ] && [ "$PREV_STATUS" != "offline" ]; then
    ELAPSED=$((CURRENT_TIME - PREV_TIMESTAMP))

    # Only accumulate positive elapsed time (handle clock drift)
    if [ "$ELAPSED" -gt 0 ]; then
        case "$PREV_STATUS" in
            working|done)
                WORKING_SECONDS=$((WORKING_SECONDS + ELAPSED))
                ;;
            waiting|permission)
                WAITING_SECONDS=$((WAITING_SECONDS + ELAPSED))
                ;;
        esac
    fi
fi

# Calculate total session time
TOTAL_SECONDS=$((CURRENT_TIME - STARTED))

# Ensure total_seconds is not negative
if [ "$TOTAL_SECONDS" -lt 0 ]; then
    TOTAL_SECONDS=0
fi

cat > "$SESSION_FILE" <<EOF
{
  "tmux_session": "$SESSION",
  "status": "$STATUS",
  "message": "$MESSAGE",
  "cwd": "$CWD",
  "timestamp": $CURRENT_TIME,
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
  }
}
EOF
