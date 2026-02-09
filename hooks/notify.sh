#!/bin/bash
# notify.sh - Hook script for navi to receive Claude Code status updates.
# Called by Claude Code hooks to write status JSON files to the shared status directory.
# Tracks session metrics including start time and status durations.
# Supports teammate-aware status writing for agent teams.

STATUS="$1"
MESSAGE="${CLAUDE_NOTIFICATION:-}"
DIR="$HOME/.claude-sessions"
mkdir -p "$DIR"

SESSION=$(tmux display-message -p '#{session_name}' 2>/dev/null || echo "unknown")
CWD=$(tmux display-message -p '#{pane_current_path}' 2>/dev/null || echo "")
CURRENT_TIME=$(date +%s)

# Read stdin JSON (hook event data from Claude Code)
STDIN_JSON=""
if [ ! -t 0 ]; then
    STDIN_JSON=$(cat)
fi

# Extract teammate fields from stdin JSON
TEAMMATE_NAME=""
TEAM_NAME=""
if [ -n "$STDIN_JSON" ] && command -v jq &> /dev/null; then
    TEAMMATE_NAME=$(echo "$STDIN_JSON" | jq -r '.teammate_name // ""' 2>/dev/null)
    TEAM_NAME=$(echo "$STDIN_JSON" | jq -r '.team_name // ""' 2>/dev/null)
    # Handle jq null output
    [ "$TEAMMATE_NAME" = "null" ] && TEAMMATE_NAME=""
    [ "$TEAM_NAME" = "null" ] && TEAM_NAME=""
fi

SESSION_FILE="$DIR/$SESSION.json"

# ---- STALE EVENT GUARD ----
# Teammate processes fire hooks (PostToolUse, Stop, SessionEnd) with their own session_id
# but no teammate_name. These would corrupt the main session's status if allowed through.
# Detect them by comparing the stdin session_id to the session_id stored in the session JSON.
# Also handle SessionEnd from teammates by setting their agent to "stopped".
if [ -n "$STDIN_JSON" ] && command -v jq &> /dev/null; then
    HOOK_EVENT=$(echo "$STDIN_JSON" | jq -r '.hook_event_name // ""' 2>/dev/null)
    HOOK_SESSION_ID=$(echo "$STDIN_JSON" | jq -r '.session_id // ""' 2>/dev/null)

    # UserPromptSubmit is ALWAYS from the main agent.
    # Clean up stopped agents from team data so they don't linger forever.
    if [ "$HOOK_EVENT" = "UserPromptSubmit" ] && [ -f "$SESSION_FILE" ]; then
        CLEANED=$(jq '
            if .team and .team.agents then
                .team.agents = [.team.agents[] | select(.status != "stopped")]
            else . end |
            if .team and (.team.agents | length) == 0 then del(.team) else . end
        ' "$SESSION_FILE" 2>/dev/null)
        if [ -n "$CLEANED" ]; then
            TMPFILE=$(mktemp "$DIR/.tmp.XXXXXX")
            echo "$CLEANED" > "$TMPFILE"
            mv "$TMPFILE" "$SESSION_FILE"
        fi
    fi

    # For non-teammate events, check if session_id matches the main session_id in the JSON
    if [ -z "$TEAMMATE_NAME" ] && [ -n "$HOOK_SESSION_ID" ] && [ -f "$SESSION_FILE" ]; then
        STORED_MAIN_SID=$(jq -r '.session_id // ""' "$SESSION_FILE" 2>/dev/null)

        if [ -n "$STORED_MAIN_SID" ] && [ "$HOOK_SESSION_ID" != "$STORED_MAIN_SID" ]; then
            # This is a teammate event — don't let it corrupt the main session
            # For Stop/SessionEnd, try to mark the agent as stopped via session_id match
            if [ "$HOOK_EVENT" = "SessionEnd" ] || [ "$HOOK_EVENT" = "Stop" ]; then
                UPDATED=$(jq \
                    --arg sid "$HOOK_SESSION_ID" \
                    --argjson ts "$CURRENT_TIME" \
                    '
                    if .team and .team.agents then
                        .team.agents = [.team.agents[] |
                            if .session_id == $sid then
                                .status = "stopped" | .timestamp = $ts
                            else . end]
                    else . end
                    ' "$SESSION_FILE" 2>/dev/null)
                if [ -n "$UPDATED" ]; then
                    TMPFILE=$(mktemp "$DIR/.tmp.XXXXXX")
                    echo "$UPDATED" > "$TMPFILE"
                    mv "$TMPFILE" "$SESSION_FILE"
                fi
            fi
            exit 0
        fi
    fi
fi

# ---- SKIP SubagentStart/SubagentStop FOR MAIN SESSION ----
# These events fire on the main agent when a subagent spawns/stops, but carry no useful
# teammate info. Agent status is handled by TeammateIdle (idle), SendMessage inference
# (working), and shutdown_request inference (stopped). Letting these through would
# corrupt the main session status.
if [ -n "$STDIN_JSON" ] && command -v jq &> /dev/null; then
    _HOOK_EVENT=$(echo "$STDIN_JSON" | jq -r '.hook_event_name // ""' 2>/dev/null)
    if [ "$_HOOK_EVENT" = "SubagentStop" ] || [ "$_HOOK_EVENT" = "SubagentStart" ]; then
        exit 0
    fi
fi

# ---- TEAMMATE BRANCH ----
# When teammate_name is present, update team.agents[] instead of main session status
if [ -n "$TEAMMATE_NAME" ] && command -v jq &> /dev/null; then
    # Read existing session file or create minimal structure
    if [ -f "$SESSION_FILE" ]; then
        EXISTING=$(cat "$SESSION_FILE")
    else
        EXISTING='{"tmux_session":"'"$SESSION"'","status":"working","message":"","cwd":"'"$CWD"'","timestamp":'"$CURRENT_TIME"'}'
    fi

    # For SessionEnd events from teammates, set status to "offline"
    AGENT_STATUS="$STATUS"

    # Race condition guard: prevent idle from overwriting stopped
    if [ "$AGENT_STATUS" = "idle" ] && command -v jq &> /dev/null && [ -f "$SESSION_FILE" ]; then
        CURRENT_AGENT_STATUS=$(jq -r --arg name "$TEAMMATE_NAME" \
            '.team.agents[] | select(.name == $name) | .status // ""' \
            "$SESSION_FILE" 2>/dev/null)
        if [ "$CURRENT_AGENT_STATUS" = "stopped" ]; then
            exit 0
        fi
    fi

    # Extract session_id from the teammate's stdin to store on the agent record
    TEAMMATE_SID=""
    if [ -n "$STDIN_JSON" ]; then
        TEAMMATE_SID=$(echo "$STDIN_JSON" | jq -r '.session_id // ""' 2>/dev/null)
        [ "$TEAMMATE_SID" = "null" ] && TEAMMATE_SID=""
    fi

    # Use jq to upsert the agent in team.agents[]
    # If the agent already exists (matched by name), update it in place.
    # If the agent does not exist, append it.
    # Also store session_id so we can detect teammate events in the stale event guard.
    UPDATED=$(echo "$EXISTING" | jq \
        --arg team_name "$TEAM_NAME" \
        --arg agent_name "$TEAMMATE_NAME" \
        --arg agent_status "$AGENT_STATUS" \
        --arg agent_sid "$TEAMMATE_SID" \
        --argjson timestamp "$CURRENT_TIME" \
        '
        # Ensure .team exists
        .team //= {"name": $team_name, "agents": []} |
        # Set team name
        .team.name = $team_name |
        # Ensure .team.agents is an array
        .team.agents //= [] |
        # Check if agent already exists
        if (.team.agents | map(.name) | index($agent_name)) != null then
            # Update existing agent
            .team.agents = [.team.agents[] | if .name == $agent_name then
                .status = $agent_status | .timestamp = $timestamp |
                (if $agent_sid != "" then .session_id = $agent_sid else . end)
            else . end]
        else
            # Append new agent
            .team.agents += [{"name": $agent_name, "status": $agent_status, "timestamp": $timestamp} |
                (if $agent_sid != "" then .session_id = $agent_sid else . end)]
        end
        ' 2>/dev/null)

    # Write atomically via temp file
    if [ -n "$UPDATED" ]; then
        TMPFILE=$(mktemp "$DIR/.tmp.XXXXXX")
        echo "$UPDATED" > "$TMPFILE"
        mv "$TMPFILE" "$SESSION_FILE"
    fi

    exit 0
fi

# ---- MAIN SESSION BRANCH ----
# Original behavior for lead agent (no teammate_name)

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

# Preserve existing team data
EXISTING_TEAM="null"

# Read existing session data if it exists
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

        # Preserve existing team data so main session updates don't blow it away
        EXISTING_TEAM=$(jq '.team // null' "$SESSION_FILE" 2>/dev/null || echo "null")

        # Handle null/empty values from jq
        [ "$STARTED" = "null" ] && STARTED=0
        [ "$WORKING_SECONDS" = "null" ] && WORKING_SECONDS=0
        [ "$WAITING_SECONDS" = "null" ] && WAITING_SECONDS=0
        [ "$PREV_TIMESTAMP" = "null" ] && PREV_TIMESTAMP=0
        [ "$PREV_STATUS" = "null" ] && PREV_STATUS=""
        [ "$TOOL_COUNTS" = "null" ] && TOOL_COUNTS="{}"
        [ "$RECENT_TOOLS" = "null" ] && RECENT_TOOLS="[]"
        [ "$EXISTING_TEAM" = "null" ] && EXISTING_TEAM=""
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

# Build the JSON output, preserving team data if it exists
TEAM_FIELD=""
if [ -n "$EXISTING_TEAM" ] && [ "$EXISTING_TEAM" != "null" ]; then
    TEAM_FIELD=",
  \"team\": $EXISTING_TEAM"
fi

# Extract session_id from stdin for storage in the session JSON
MAIN_SESSION_ID=""
if [ -n "$STDIN_JSON" ] && command -v jq &> /dev/null; then
    MAIN_SESSION_ID=$(echo "$STDIN_JSON" | jq -r '.session_id // ""' 2>/dev/null)
    [ "$MAIN_SESSION_ID" = "null" ] && MAIN_SESSION_ID=""
fi

TMPFILE=$(mktemp "$DIR/.tmp.XXXXXX")
cat > "$TMPFILE" <<EOF
{
  "tmux_session": "$SESSION",
  "session_id": "$MAIN_SESSION_ID",
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
  }$TEAM_FIELD
}
EOF
mv "$TMPFILE" "$SESSION_FILE"

# ---- INFER AGENT STATUS FROM MAIN AGENT TOOL USE ----
# Handle Task spawn (register new agent) and SendMessage (infer working status).
if [ -n "$STDIN_JSON" ] && command -v jq &> /dev/null; then
    TOOL_NAME=$(echo "$STDIN_JSON" | jq -r '.tool_name // ""' 2>/dev/null)

    # When main agent spawns a teammate via Task tool, register the agent with session_id
    if [ "$TOOL_NAME" = "Task" ]; then
        SPAWN_TEAM=$(echo "$STDIN_JSON" | jq -r '.tool_input.team_name // ""' 2>/dev/null)
        SPAWN_NAME=$(echo "$STDIN_JSON" | jq -r '.tool_input.name // ""' 2>/dev/null)
        SPAWN_STATUS=$(echo "$STDIN_JSON" | jq -r '.tool_response.status // ""' 2>/dev/null)
        # Get the teammate's session_id from tool_response if available
        SPAWN_SID=$(echo "$STDIN_JSON" | jq -r '.tool_response.session_id // ""' 2>/dev/null)
        [ "$SPAWN_SID" = "null" ] && SPAWN_SID=""

        if [ "$SPAWN_STATUS" = "teammate_spawned" ] && [ -n "$SPAWN_NAME" ] && [ "$SPAWN_NAME" != "null" ]; then
            [ "$SPAWN_TEAM" = "null" ] && SPAWN_TEAM=""
            UPDATED_SESSION=$(jq \
                --arg team_name "${SPAWN_TEAM}" \
                --arg agent_name "$SPAWN_NAME" \
                --arg agent_sid "$SPAWN_SID" \
                --argjson ts "$CURRENT_TIME" \
                '
                .team //= {"name": $team_name, "agents": []} |
                .team.name = (if $team_name != "" then $team_name else .team.name end) |
                .team.agents //= [] |
                if (.team.agents | map(.name) | index($agent_name)) != null then
                    .team.agents = [.team.agents[] |
                        if .name == $agent_name then
                            .status = "working" | .timestamp = $ts |
                            (if $agent_sid != "" then .session_id = $agent_sid else . end)
                        else . end]
                else
                    .team.agents += [{"name": $agent_name, "status": "working", "timestamp": $ts} |
                        (if $agent_sid != "" then .session_id = $agent_sid else . end)]
                end
                ' "$SESSION_FILE" 2>/dev/null)
            if [ -n "$UPDATED_SESSION" ]; then
                TMPFILE2=$(mktemp "$DIR/.tmp.XXXXXX")
                echo "$UPDATED_SESSION" > "$TMPFILE2"
                mv "$TMPFILE2" "$SESSION_FILE"
            fi
        fi
    fi

    if [ "$TOOL_NAME" = "SendMessage" ]; then
        # Read directly from STDIN_JSON to avoid stringification issues with nested objects
        MSG_TYPE=$(echo "$STDIN_JSON" | jq -r '.tool_input.type // ""' 2>/dev/null)
        RECIPIENT=$(echo "$STDIN_JSON" | jq -r '.tool_input.recipient // ""' 2>/dev/null)
        if [ -n "$MSG_TYPE" ] && [ "$MSG_TYPE" != "null" ]; then

            if [ "$MSG_TYPE" = "shutdown_request" ] && [ -n "$RECIPIENT" ] && [ "$RECIPIENT" != "null" ]; then
                # Mark the agent as stopped immediately on shutdown request
                UPDATED_SESSION=$(jq \
                    --arg name "$RECIPIENT" \
                    --argjson ts "$CURRENT_TIME" \
                    '
                    if .team and .team.agents then
                        .team.agents = [.team.agents[] |
                            if .name == $name then
                                .status = "stopped" | .timestamp = $ts
                            else . end]
                    else . end
                    ' "$SESSION_FILE" 2>/dev/null)
                if [ -n "$UPDATED_SESSION" ]; then
                    TMPFILE2=$(mktemp "$DIR/.tmp.XXXXXX")
                    echo "$UPDATED_SESSION" > "$TMPFILE2"
                    mv "$TMPFILE2" "$SESSION_FILE"
                fi
            elif [ "$MSG_TYPE" = "message" ] && [ -n "$RECIPIENT" ] && [ "$RECIPIENT" != "null" ]; then
                # Set recipient agent to "working" if not stopped
                UPDATED_SESSION=$(jq \
                    --arg name "$RECIPIENT" \
                    --argjson ts "$CURRENT_TIME" \
                    '
                    if .team and .team.agents then
                        .team.agents = [.team.agents[] |
                            if .name == $name and .status != "stopped" then
                                .status = "working" | .timestamp = $ts
                            else . end]
                    else . end
                    ' "$SESSION_FILE" 2>/dev/null)
                if [ -n "$UPDATED_SESSION" ]; then
                    TMPFILE2=$(mktemp "$DIR/.tmp.XXXXXX")
                    echo "$UPDATED_SESSION" > "$TMPFILE2"
                    mv "$TMPFILE2" "$SESSION_FILE"
                fi
            elif [ "$MSG_TYPE" = "broadcast" ]; then
                # Set all non-stopped agents to "working"
                UPDATED_SESSION=$(jq \
                    --argjson ts "$CURRENT_TIME" \
                    '
                    if .team and .team.agents then
                        .team.agents = [.team.agents[] |
                            if .status != "stopped" then
                                .status = "working" | .timestamp = $ts
                            else . end]
                    else . end
                    ' "$SESSION_FILE" 2>/dev/null)
                if [ -n "$UPDATED_SESSION" ]; then
                    TMPFILE2=$(mktemp "$DIR/.tmp.XXXXXX")
                    echo "$UPDATED_SESSION" > "$TMPFILE2"
                    mv "$TMPFILE2" "$SESSION_FILE"
                fi
            fi
        fi
    fi
fi
