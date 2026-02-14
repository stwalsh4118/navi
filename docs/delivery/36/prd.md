# PBI-36: Windows Build Support and Cross-Platform Compatibility

## Overview

Navi currently only builds and runs on Unix-like systems (Linux, macOS) due to dependencies on tmux, bash scripts, and hardcoded Unix paths. This PBI enables Windows builds with a "remote viewer" model: on Windows, navi runs natively for remote session management, task views, git integration, and all non-tmux features. Local session management (create, attach, kill) is disabled on Windows since tmux is not available. Git for Windows (which bundles Git Bash) is the sole prerequisite for Windows, providing both git and bash for provider script execution.

[View in Backlog](../backlog.md#user-content-36)

## Problem Statement

### The binary doesn't build/run correctly on Windows

Several areas of the codebase have Unix-specific assumptions:

1. **tmux dependency** — All local session management (create, attach, kill, rename, preview) uses `exec.Command("tmux", ...)`. tmux is not available on Windows. However, remote session management over SSH targets Linux hosts where tmux exists, so that path works fine.

2. **Bash provider scripts** — Built-in providers (`markdown-tasks.sh`, `github-issues.sh`) are bash scripts with shebangs. They cannot execute directly on Windows. However, Git for Windows bundles Git Bash (based on MSYS2), which provides a full bash environment. Since navi already requires git for the git integration panel, Git Bash is effectively already a prerequisite.

3. **Hardcoded Unix paths** — Config and session paths use `~/` prefix:
   - `~/.claude-sessions` (`internal/session/session.go:23`)
   - `~/.navi/config.yaml` (`internal/task/types.go:118`)
   - `~/.config/navi/remotes.yaml` (`internal/remote/config.go:14`)
   - Windows equivalent would use `os.UserConfigDir()` → `%APPDATA%`.

4. **Shell commands over SSH** — Remote actions (`internal/remote/actions.go`) use Unix utilities (`sed -i`, `rm -f`, `mv`, `date +%s`). These run on the **remote** host (Linux) via SSH, so they work fine from a Windows client.

5. **Hook scripts** — `hooks/notify.sh` and `hooks/tool-tracker.sh` are bash scripts. On Windows, these would execute via Git Bash.

6. **Install script** — `install.sh` is bash. Needs a Windows-compatible installation path.

7. **File permissions** — Tests use Unix mode bits (`0o755`, `0o644`). These are no-ops on Windows.

### What already works cross-platform

- **Git integration** — `internal/git/git.go` already has correct `runtime.GOOS` switching for URL opening (`xdg-open`/`open`/`cmd /c start`). This is the pattern to follow elsewhere.
- **Go's `filepath`** — Used consistently for path joining (handles `\` vs `/`).
- **`pathutil.ExpandPath`** — Uses `os.UserHomeDir()` which works on Windows, though `~\` pattern isn't handled.
- **Remote SSH commands** — These execute on the remote Linux host, not locally, so Unix utilities work fine from a Windows client.

## User Stories

- As a user on Windows, I want to run navi to manage remote Claude Code sessions, view tasks, and use git integration, even though local tmux sessions are not available.
- As a user on Windows, I want clear feedback that local session management is not supported and that navi operates in remote-only mode.
- As a user, I want navi to use platform-appropriate config directories so that my config integrates with my OS conventions.
- As a developer, I want Windows CI builds so that I can catch platform issues early.

## Technical Approach

### Prerequisites for Windows users

**Git for Windows** is the sole prerequisite. It provides:
- `git` — Required for the git integration panel (already a dependency)
- `bash` (Git Bash) — Required for executing provider scripts (`.sh` files)
- Unix utilities (`sed`, `grep`, `awk`, `jq` if installed) — Available through MSYS2 bundled with Git Bash

Git Bash's `bash.exe` is typically available at `C:\Program Files\Git\bin\bash.exe` or on `PATH` if Git is installed with the "Add to PATH" option.

### Windows feature matrix

| Feature | Windows support | Notes |
|---------|----------------|-------|
| TUI (dashboard, navigation, keybindings) | Full | Pure Go, cross-platform |
| Task view (provider scripts) | Full | Scripts execute via Git Bash |
| Git integration (branch, status, diff) | Full | Uses git, available on Windows |
| Search (vim-style, filtering) | Full | Pure Go |
| Content viewer | Full | Pure Go |
| Remote session management (SSH) | Full | SSH commands run on remote Linux host |
| Remote session preview | Full | SSH to remote, tmux on remote |
| Remote git info | Full | SSH to remote |
| Local session list | Partial | Can read `.claude-sessions` status files, but no tmux to manage them |
| Local session create/attach/kill/rename | Not supported | Requires tmux |
| Local session preview | Not supported | Requires tmux `capture-pane` |
| Hooks (notify, tool-tracker) | Via Git Bash | Execute through bash |

### Implementation

1. **Cross-platform path resolution** — Extend `internal/pathutil` with platform-aware directory functions:
   - Use `os.UserConfigDir()` for config base:
     - Linux: `~/.config/navi/`
     - macOS: `~/.config/navi/` (prefer Unix convention for CLI tools)
     - Windows: `%APPDATA%\navi\`
   - Session status directory: keep `~/.claude-sessions/` on Unix (Claude Code controls this); on Windows use `%APPDATA%\navi\claude-sessions\` or wherever Claude Code stores sessions on Windows.
   - Update `ExpandPath()` to handle `~\` in addition to `~/`.

2. **tmux feature gating** — At startup, detect `runtime.GOOS == "windows"` and disable local session management features:
   - Disable create/attach/kill/rename session keybindings and menu options.
   - Show a status bar indicator or panel message: "Local sessions not available on Windows — use remote sessions."
   - Do not attempt to run `tmux` commands. No crash, no cryptic error.
   - Local session list can still display status files from Claude Code if they exist.

3. **Provider script execution on Windows** — Execute `.sh` scripts through bash:
   - Detect bash: check `PATH` for `bash`, then fall back to common Git Bash locations (`C:\Program Files\Git\bin\bash.exe`, `C:\Program Files (x86)\Git\bin\bash.exe`).
   - Execute: `exec.Command("bash", scriptPath)` instead of `exec.Command(scriptPath)`.
   - For custom providers: support `.sh` (via bash), `.bat` (via `cmd /c`), `.ps1` (via `powershell -File`), and `.exe` (direct).
   - Clear error if bash is not found: "bash not found — install Git for Windows from https://git-scm.com".

4. **Hook script execution on Windows** — Same approach as providers: execute via Git Bash.
   - Claude Code hook configuration (`hooks/config.json`) may need to specify `bash` as the interpreter on Windows.
   - Document this in Windows installation instructions.

5. **Windows build target** — Add cross-compilation:
   - `GOOS=windows GOARCH=amd64 go build -o navi.exe ./cmd/navi/`
   - CI build step to verify Windows compilation.
   - Release workflow to produce `navi-windows-amd64.exe` alongside Linux/macOS binaries.

6. **Platform-specific test handling** — For tests that use Unix-specific paths or permissions:
   - Use `t.TempDir()` instead of hardcoded `/tmp` paths.
   - Skip or adapt tests that rely on Unix file permission semantics on Windows.
   - Add `runtime.GOOS` checks where needed.

### Connecting to Windows remotes

Out of scope. Remote session management assumes the remote host is Linux/macOS with tmux. SSH commands in `internal/remote/actions.go` run on the remote host, so they use Unix utilities regardless of the client OS. If Windows remotes are needed in the future, the remote command layer would need a separate abstraction.

## UX/UI Considerations

- **Windows startup** — No error or crash. Navi launches normally, with local session features visually disabled or hidden.
- **Clear messaging** — The session panel should indicate "Remote only — local sessions require tmux (Linux/macOS)" rather than showing an empty list with no explanation.
- **Keybinding changes** — Create/attach keybindings should be no-ops on Windows with a brief flash message if pressed.
- **Config paths** — Error messages and documentation should reference Windows paths (`%APPDATA%\navi\`) when on Windows.
- **Provider errors** — If bash is not found, error should name Git for Windows specifically rather than just "bash not found".

## Acceptance Criteria

1. `GOOS=windows GOARCH=amd64 go build ./cmd/navi/` compiles successfully.
2. All hardcoded `~/` config paths are replaced with platform-detected paths using `os.UserConfigDir()` / `os.UserHomeDir()`.
3. `pathutil.ExpandPath` handles both `~/` and `~\` patterns.
4. On Windows, local session management features (create, attach, kill, rename, preview) are disabled with a clear user-facing message.
5. On Windows, provider scripts execute via Git Bash (`bash <script>`).
6. On Windows, if bash is not found, a clear error message directs the user to install Git for Windows.
7. Remote session management works from Windows (SSH to Linux hosts).
8. Task view, git integration, search, and content viewer work on Windows.
9. All existing Unix tests continue to pass.
10. Windows-specific tests are added (build tags or `runtime.GOOS` checks).
11. CI includes a Windows build step (compilation at minimum).

## Dependencies

- PBI-35 (Embed Built-in Providers) should ideally land first, as it changes the provider execution model that this PBI builds on. Specifically, PBI-35 embeds scripts in the binary, and this PBI adds the Windows execution path for those embedded scripts.

## Open Questions

- None. Claude Code stores its data in `~/.claude` on all platforms (`C:\Users\{USER}\.claude` on Windows). The `~/.claude-sessions/` directory is navi's own status directory (written by hooks), and should follow navi's platform-specific config path conventions (e.g., `%APPDATA%\navi\claude-sessions\` on Windows).

## Affected Files Summary

| Area | Files | Impact |
|------|-------|--------|
| Path constants | `internal/pathutil/pathutil.go`, `internal/session/session.go`, `internal/remote/config.go`, `internal/task/types.go` | Replace hardcoded `~/` with platform detection |
| Provider execution | `internal/task/provider.go` | Execute `.sh` via bash on Windows, support `.bat`/`.ps1`/`.exe` |
| tmux integration | `internal/tui/sessions.go`, `internal/tui/model.go` | Gate behind `runtime.GOOS` check, disable on Windows |
| Remote commands | `internal/remote/actions.go`, `internal/remote/git.go` | No changes needed (run on remote Linux host) |
| Hooks | `hooks/notify.sh`, `hooks/tool-tracker.sh` | Document Git Bash execution on Windows |
| Install | `install.sh` | Windows installation instructions/script |
| Tests | 30+ test files in `internal/tui/`, `internal/task/` | Platform-aware test paths and permissions |
| Build | CI config | Add Windows build target |

## Related Tasks

Tasks will be defined when this PBI moves to Agreed status.
