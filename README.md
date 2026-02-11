# navi

A terminal dashboard for monitoring Claude Code sessions. If you run multiple Claude Code instances in tmux, navi gives you a single pane of glass to see what they're all doing.

## What it does

navi polls status files written by Claude Code hooks and renders a live TUI showing every session's state — working, waiting for input, needs permission, done, errored, or offline. You can attach to any session, kill or rename them, preview their output, see git branch info, and track token usage, all without leaving the dashboard.

## Features

- **Live session status** — see all your Claude Code tmux sessions at a glance
- **Attach/detach** — jump into any session directly from the dashboard
- **Session management** — create, kill, and rename sessions
- **Preview pane** — read recent output without attaching (side or bottom layout)
- **Git integration** — branch name, dirty/clean status, ahead/behind counts
- **Agent team awareness** — when a session spawns a team, see each agent's status inline
- **Token metrics** — track token usage and tool activity per session
- **Task panel** — view project tasks from pluggable providers (GitHub Issues, markdown files)
- **Content viewer** — browse files, diffs, and task details in-app
- **Search** — vim-style `/` search with `n`/`N` to cycle matches
- **Remote sessions** — aggregate sessions from remote machines over SSH
- **Scrollable everything** — all panels scroll when content overflows

## Requirements

- Go 1.25+
- tmux
- jq (for automatic config merging during install)
- Linux (macOS may work but isn't tested)

## Install

```bash
git clone https://github.com/stwalsh4118/navi.git
cd navi
./install.sh
```

The install script will:
1. Create `~/.claude-sessions/` and install the hook scripts
2. Merge hook config into your Claude Code settings (`~/.claude/settings.json`)
3. Build the binary and optionally copy it to `~/.local/bin/`

If you already have Claude Code hooks configured, the installer will ask how you want to handle conflicts (merge, override, skip, or save for manual merging).

## Usage

```bash
# Start some Claude Code sessions in tmux
tmux new-session -d -s myproject -c ~/projects/myproject
tmux send-keys -t myproject 'claude' Enter

# Launch the dashboard
navi
```

### Keybindings

#### Session list

| Key | Action |
|-----|--------|
| `j`/`k` or arrows | Navigate sessions |
| `Enter` | Attach to session |
| `d` | Detach from current session |
| `n`/`N` | Next/previous search match |
| `x` | Kill session |
| `R` | Rename session |
| `p` | Toggle preview pane |
| `L` | Toggle preview layout (side/bottom) |
| `W` | Toggle preview word wrap |
| `T` | Toggle task panel |
| `G` | Git detail view |
| `i` | Metrics detail view |
| `/` | Search |
| `s` | Cycle sort mode |
| `o` | Toggle offline sessions |
| `f` | Cycle filter (all/local/remote) |
| `r` | Refresh |
| `q` | Quit |

#### Preview pane

| Key | Action |
|-----|--------|
| `j`/`k` | Scroll |
| `g`/`G` | Top/bottom |
| `Tab`/`Esc` | Return focus to session list |

#### Task panel

| Key | Action |
|-----|--------|
| `j`/`k` | Navigate tasks |
| `Enter` | View task detail / toggle group |
| `/` | Search tasks |
| `n`/`N` | Next/previous task match |
| `Tab` | Return focus to session list |

## How it works

navi uses [Claude Code hooks](https://docs.anthropic.com/en/docs/claude-code/hooks) to track session state. The hooks fire on events like prompt submission, permission requests, tool use, and session end, writing JSON status files to `~/.claude-sessions/`. The TUI polls these files and renders the dashboard.

```
Claude Code (tmux) ──hook──> ~/.claude-sessions/<session>.json ──poll──> navi TUI
```

### Status directory

Each session gets a JSON file in `~/.claude-sessions/` containing:
- Session name and status
- Working directory
- Git branch and status
- Agent team info (if running a team)
- Timestamp

### Configuration

Create a `.navi.yaml` in your project root to configure task providers:

```yaml
tasks:
  provider: "markdown-tasks"
  args:
    path: "docs/delivery"
```

Available task providers live in the `providers/` directory. Currently ships with:
- `markdown-tasks` — parse tasks from markdown files
- `github-issues` — fetch issues from GitHub

## Documentation

Full docs are at [navi-docs.pages.dev](https://navi-docs.pages.dev/).

## Built with

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — terminal styling
- [Bubbles](https://github.com/charmbracelet/bubbles) — TUI components

## License

MIT
