# PBI-23: CLI Mode

[View in Backlog](../backlog.md)

## Overview

Add non-interactive CLI commands for scripting navi into automation workflows, allowing users to list sessions, attach programmatically, and query session status from shell scripts.

## Problem Statement

The TUI is great for interactive use, but users can't script navi into their workflows. Common needs include: checking session status in scripts, creating sessions from automation, integrating with other tools.

## User Stories

- As a user, I want to list sessions from a script to check what's running
- As a user, I want to create sessions programmatically for automation
- As a user, I want to check specific session status in conditionals
- As a user, I want machine-readable output for parsing

## Technical Approach

### CLI Commands

```bash
# List sessions (default: human-readable)
navi list
navi ls

# List with machine-readable output
navi list --json
navi list --format "{{.Name}}\t{{.Status}}"

# Session details
navi status hyperion
navi status hyperion --json

# Create session
navi new myproject --dir ~/work/myproject
navi new myproject --dir ~/work/myproject --attach

# Attach to session
navi attach hyperion

# Kill session
navi kill hyperion
navi kill hyperion --force

# Dismiss notification
navi dismiss hyperion

# Bookmark operations
navi bookmark add hyperion --note "Auth work"
navi bookmark list
navi bookmark remove hyperion

# Export session
navi export hyperion --format md --output transcript.md

# Remote operations
navi remote list
navi remote status dev-server

# Configuration
navi config show
navi config edit

# Server (for team dashboard)
navi server --port 8080
```

### Output Formats

**Human-readable (default)**
```
NAME        STATUS      AGE      DIRECTORY
hyperion    working     2m       ~/projects/hyperion
api         done        15m      ~/projects/api
scratch     permission  30s      ~/tmp/scratch
```

**JSON**
```json
[
  {
    "name": "hyperion",
    "status": "working",
    "message": "Implementing feature...",
    "cwd": "/home/user/projects/hyperion",
    "age_seconds": 120
  }
]
```

**Template**
```bash
$ navi list --format "{{.Name}}:{{.Status}}"
hyperion:working
api:done
scratch:permission
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Session not found |
| 3 | Session already exists |
| 4 | Permission denied |

### Implementation

1. **Command Structure**
   - Use cobra for CLI framework
   - Subcommands for each operation
   - Global flags for output format

2. **Shared Core**
   - CLI and TUI share session management code
   - Single source of truth for operations

3. **Mode Detection**
   ```go
   func main() {
       if len(os.Args) > 1 {
           runCLI()
       } else {
           runTUI()
       }
   }
   ```

### Script Examples

```bash
# Wait for session to need input
while [ "$(navi status myproject --format "{{.Status}}")" = "working" ]; do
    sleep 5
done
echo "Session needs attention!"

# Auto-start sessions from project
for project in ~/work/*/; do
    name=$(basename "$project")
    if ! navi status "$name" &>/dev/null; then
        navi new "$name" --dir "$project"
    fi
done

# Export all done sessions
navi list --json | jq -r '.[] | select(.status == "done") | .name' | while read name; do
    navi export "$name" --format md --output "exports/${name}.md"
done
```

## UX/UI Considerations

- Commands should feel natural to shell users
- Error messages should be clear and actionable
- JSON output should be consistent across commands
- Support for shell completion (bash, zsh, fish)

## Acceptance Criteria

1. `navi list` shows sessions in human-readable table
2. `navi list --json` outputs valid JSON
3. `navi status <session>` shows single session details
4. `navi new <name>` creates new session
5. `navi attach <session>` attaches to session
6. `navi kill <session>` kills session (with confirmation)
7. `navi export <session>` exports session transcript
8. Template format support with --format flag
9. Appropriate exit codes for scripting
10. Shell completion scripts for major shells
11. `navi --help` shows comprehensive help

## Dependencies

- PBI-7: Session management (for create/kill)
- PBI-17: History (for export functionality)
- PBI-18: Session logs (for export content)

## Open Questions

- Should there be a `navi watch` command for continuous output?
- Should CLI support remote sessions?
- Should there be a daemon mode for background monitoring?

## Related Tasks

See [Tasks](./tasks.md)
