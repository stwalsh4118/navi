# CLI API

Package: `internal/cli`

## Status Command

```go
func RunStatus(args []string) int
```

Flags:
- `--verbose`: include all non-zero status counts
- `--format=plain|tmux`: plain text output mode (tmux currently same formatting)

Behavior:
- Reads sessions from `session.ReadStatusFiles(pathutil.ExpandPath(session.StatusDir))`
- Default output includes only priority statuses: `waiting`, `permission`
- Verbose output includes non-zero counts in this order: `working`, `waiting`, `permission`, `idle`, `stopped`
- Prints empty output when nothing matches the selected mode
- Returns exit code `0` on success and `1` on flag/IO errors
