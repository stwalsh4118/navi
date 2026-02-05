# claude-sessions: TUI Design Document

## Overview

A terminal UI written in Go that monitors all running Claude Code tmux sessions, displays their real-time status (working, needs input, done, needs permission), and lets you attach to any session directly. When you detach from tmux, the TUI reappears automatically.

---

## Architecture

There are two components: **hooks** that Claude Code fires to write status updates, and a **TUI** that reads those updates and renders the dashboard.

### 1. Status Directory (`~/.claude-sessions/`)

A shared directory where hooks write JSON status files. Each file represents one active Claude Code session, keyed by tmux session name.

```
~/.claude-sessions/
├── hyperion.json
├── dotfiles.json
└── api.json
```

Each status file:

```json
{
  "tmux_session": "hyperion",
  "status": "waiting",
  "message": "Should I proceed with the refactor?",
  "cwd": "/home/sean/projects/hyperion",
  "timestamp": 1738627200
}
```

**Status values:** `working`, `waiting`, `done`, `permission`, `error`

### 2. Claude Code Hooks

Configured in `~/.config/claude/settings.json`. Each hook runs a small shell script that writes/updates the JSON status file for its session.

```json
{
  "hooks": {
    "Notification": [
      {
        "matcher": "",
        "hooks": [{ "type": "command", "command": "~/.claude-sessions/hooks/notify.sh waiting" }]
      }
    ],
    "Stop": [
      {
        "hooks": [{ "type": "command", "command": "~/.claude-sessions/hooks/notify.sh done" }]
      }
    ],
    "PreToolUse": [
      {
        "hooks": [{ "type": "command", "command": "~/.claude-sessions/hooks/notify.sh permission" }]
      }
    ],
    "SubagentStop": [
      {
        "hooks": [{ "type": "command", "command": "~/.claude-sessions/hooks/notify.sh working" }]
      }
    ]
  }
}
```

The hook script (`hooks/notify.sh`):

```bash
#!/bin/bash
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
```

### 3. TUI Application (Go)

**Framework:** [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI, [Lip Gloss](https://github.com/charmbracelet/lipgloss) for styling.

---

## TUI Layout

```
╭─────────────────────────────────────────────────────────╮
│  Claude Sessions                               3 active │
╰─────────────────────────────────────────────────────────╯

  ⏳  hyperion                                      12s ago
      ~/projects/hyperion
      "Should I proceed with the refactor?"

  ✅  api                                            3m ago
      ~/projects/parallel-api
      Done

▸ ⚙️  dotfiles                                      working
      ~/dotfiles

  ❓  scratch                                       45s ago
      ~/tmp/scratch
      "Run: rm -rf ./dist?"

╭─────────────────────────────────────────────────────────╮
│  ↑/↓ navigate  ⏎ attach  d dismiss  q quit  r refresh  │
╰─────────────────────────────────────────────────────────╯
```

### Status Icons

| Status       | Icon | Color   | Meaning                          |
|------------- |------|---------|----------------------------------|
| `waiting`    | ⏳   | Yellow  | Claude asked a question          |
| `done`       | ✅   | Green   | Task completed                   |
| `permission` | ❓   | Magenta | Needs tool use approval          |
| `working`    | ⚙️   | Cyan    | Actively processing              |
| `error`      | ❌   | Red     | Something went wrong             |
| `unknown`    | ○    | Dim     | No status yet / stale            |

### Row Content

Each row shows:
- **Status icon** (colored)
- **Session name** in bold
- **Age** of last status update, right-aligned
- **Second line**: Working directory (shortened with `~`), dimmed
- **Third line** (if present): The message from Claude, truncated to terminal width, dimmed/italic

The currently selected row has a `▸` marker and is highlighted.

---

## Core Data Flow

```
┌──────────────┐    writes JSON     ┌──────────────────────┐
│  Claude Code ├───────────────────▸│ ~/.claude-sessions/   │
│  (hooks)     │                    │   hyperion.json, ...  │
└──────────────┘                    └──────────┬───────────┘
                                               │ reads every 500ms
                                    ┌──────────▼───────────┐
                                    │   TUI (Go)           │
                                    │   Bubble Tea          │
                                    └──────────┬───────────┘
                                               │ user presses Enter
                                    ┌──────────▼───────────┐
                                    │ tmux attach-session   │
                                    │ -t <session_name>     │
                                    └──────────┬───────────┘
                                               │ user detaches (prefix+d)
                                    ┌──────────▼───────────┐
                                    │ TUI re-renders        │
                                    │ (process resumes)     │
                                    └──────────────────────┘
```

---

## Key Behaviors

### Attach/Detach Loop

This is the core UX. When the user presses Enter on a session:

1. TUI calls `bubbletea.ExecProcess` to run `tmux attach-session -t <session_name>`
2. The TUI process suspends while tmux takes over the terminal
3. When the user detaches from tmux (`prefix + d`), the tmux attach process exits
4. Bubble Tea regains control, re-renders the TUI with fresh state

With `bubbletea.ExecProcess`, this is built-in — it hands the TTY to the child process and resumes the TUI when it exits.

### File Watching / Polling

Poll `~/.claude-sessions/` every 500ms using `os.ReadDir` + `json.Unmarshal` on each file. This is simpler and more portable than fsnotify, and the directory will only ever have a handful of small files.

On each tick:
1. Read all `*.json` files from the status directory
2. Parse into `[]SessionInfo` structs
3. Cross-reference with live tmux sessions via `tmux list-sessions -F '#{session_name}'` — remove any status files whose session no longer exists (stale cleanup)
4. Sort: `waiting` and `permission` first (needs attention), then by timestamp descending
5. Send as a message to the Bubble Tea model to trigger re-render

### Dismiss

Pressing `d` on a session resets its status to `working` (clears the notification). This is for when you've seen the notification but don't need to attach right now. It overwrites the JSON file with `"status": "working"`.

### Stale Cleanup

On each poll cycle, run `tmux list-sessions -F '#{session_name}'` and delete any status files whose session name isn't in the list. This handles cases where a tmux session was killed without a clean hook firing.

---

## Go Struct Design

```go
type SessionInfo struct {
    TmuxSession string `json:"tmux_session"`
    Status      string `json:"status"`
    Message     string `json:"message"`
    CWD         string `json:"cwd"`
    Timestamp   int64  `json:"timestamp"`
}

type Model struct {
    sessions []SessionInfo
    cursor   int
    width    int
    height   int
    err      error
}
```

### Bubble Tea Messages

```go
type tickMsg time.Time           // periodic refresh trigger
type sessionsMsg []SessionInfo   // updated session list from polling
type attachDoneMsg struct{}      // returned after tmux detach
```

### Commands

```go
func pollSessions() tea.Msg              // reads dir, parses JSON, cleans stale
func attachSession(name string) tea.Cmd  // tea.ExecProcess for tmux attach-session -t <name>
func tickCmd() tea.Cmd                   // time.After(500ms) -> tickMsg
```

---

## File Structure

```
claude-sessions/
├── main.go              # entry point, Bubble Tea program setup
├── model.go             # Model, Init, Update, View
├── sessions.go          # polling logic, JSON parsing, stale cleanup
├── styles.go            # Lip Gloss style definitions
├── hooks/
│   └── notify.sh        # hook script for Claude Code to write status
├── install.sh           # copies hooks, patches claude settings.json
├── go.mod
├── go.sum
└── README.md
```

---

## Install Script (`install.sh`)

A helper that:

1. Copies `notify.sh` to `~/.claude-sessions/hooks/` and makes it executable
2. Creates `~/.claude-sessions/` if it doesn't exist
3. Reads `~/.config/claude/settings.json`, merges the hook config (preserving existing hooks), writes it back
4. Builds the Go binary and optionally copies it to `~/.local/bin/`

---

## Workflow Example

```bash
# Start named sessions for each project
tmux new-session -d -s hyperion -c ~/projects/hyperion
tmux new-session -d -s api -c ~/projects/parallel-api
tmux new-session -d -s dotfiles -c ~/dotfiles

# Launch claude in each (from another terminal or via tmux send-keys)
tmux send-keys -t hyperion 'claude' Enter
tmux send-keys -t api 'claude' Enter
tmux send-keys -t dotfiles 'claude' Enter

# Run the TUI — this is your home base
claude-sessions
```

From here you see all three sessions. When `hyperion` shows ⏳, hit Enter to jump in, answer Claude's question, `prefix+d` to detach, and you're back in the dashboard.

---

## Optional Enhancements (Post-MVP)

- **Sound on status change**: Play a short beep/sound when a session transitions to `waiting` or `permission` (configurable, off by default)
- **Desktop notification forwarding**: In addition to the TUI, optionally fire `notify-send` / `wsl-notify-send.exe` from the hook script for when the TUI isn't visible
- **Filter/search**: `/` to filter sessions by name when you have many running
- **Transcript preview**: Read the last few lines from the tmux pane via `tmux capture-pane -t <session> -p` and display a preview below the selected session
- **New session from TUI**: `n` to create a new named tmux session and launch Claude Code in it directly from the dashboard
- **Auto-attach on urgent**: Config option to automatically switch to a session when it enters `waiting` status (aggressive but useful for single-monitor setups)
- **Session kill**: `x` to kill a tmux session and clean up its status file from the dashboard
