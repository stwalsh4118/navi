#!/bin/bash
# notify.sh - Hook script for navi to receive Claude Code status updates.
# Implementation will be added in PBI-2.

STATUS="$1"
MESSAGE="${CLAUDE_NOTIFICATION:-}"
DIR="$HOME/.claude-sessions"
mkdir -p "$DIR"

SESSION=$(tmux display-message -p '#{session_name}' 2>/dev/null || echo "unknown")
CWD=$(tmux display-message -p '#{pane_current_path}' 2>/dev/null || echo "")

cat > "$DIR/$SESSION.json" <<EOF
{
  "tmux_session": "$SESSION",
  "status": "$STATUS",
  "message": "$MESSAGE",
  "cwd": "$CWD",
  "timestamp": $(date +%s)
}
EOF
