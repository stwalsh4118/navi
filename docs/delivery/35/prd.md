# PBI-35: Embed Built-in Providers and Consolidate Config to ~/.config/navi/

## Overview

Built-in task provider scripts (markdown-tasks, github-issues) are currently resolved relative to the CWD of the navi process, meaning they only work when navi is run from the repo root. Custom provider scripts use an inconsistent mix of relative/absolute path resolution. Additionally, navi config is split across two locations (`~/.navi/` and `~/.config/navi/`). This PBI fixes all three issues: embed built-in scripts in the binary, standardize custom providers to `~/.config/navi/providers/`, and consolidate all config under `~/.config/navi/`.

[View in Backlog](../backlog.md#user-content-35)

## Problem Statement

### Provider resolution is broken for installed binaries

The current `ResolveProvider` function in `internal/task/provider.go` has three resolution paths:

1. **Built-in names** (e.g., `"markdown-tasks"`) resolve to `ProvidersDir/filename` where `ProvidersDir = "providers"` (relative). `filepath.Abs()` then resolves this relative to the **CWD of the navi process**, not the project or binary directory. This means built-in providers only work if navi is launched from the repo root — completely broken for installed binaries.

2. **Relative paths** resolve relative to the project directory (where `.navi.yaml` lives). This couples provider scripts to project repos.

3. **Absolute paths** are used as-is.

### Config locations are inconsistent

- Global config: `~/.navi/config.yaml` (`internal/task/types.go:118`)
- Remote config: `~/.config/navi/remotes.yaml` (`internal/remote/config.go:14`)

All config should live under `~/.config/navi/` following XDG conventions on Linux.

## User Stories

- As a user, I want built-in providers to work regardless of where I run navi so that `go install` and binary distribution work out of the box.
- As a user, I want a single well-known location for custom provider scripts so that I can install and manage them predictably.
- As a user, I want all navi configuration in one directory (`~/.config/navi/`) so that config is easy to find, back up, and manage.

## Technical Approach

### 1. Embed built-in provider scripts via `//go:embed`

- Use Go's `embed` package to embed `providers/*.sh` files directly into the compiled binary.
- When a built-in provider name is requested, write the embedded script to a temp file (or use `bash -c` with the script content) and execute it.
- Remove the `ProvidersDir` variable entirely — it's no longer needed.

### 2. Custom providers resolve from `~/.config/navi/providers/`

- Non-built-in provider names resolve to `~/.config/navi/providers/<name>`.
- This is the **only** resolution path for custom scripts — no relative paths, no absolute paths, no project-dir resolution.
- The `.navi.yaml` `provider` field specifies just the script filename (e.g., `provider: "my-custom-provider.sh"`), which resolves to `~/.config/navi/providers/my-custom-provider.sh`.
- The directory must exist and the script must be executable.

### 3. Resolution order summary

1. Check if name matches a built-in provider key → use embedded script
2. Otherwise → resolve to `~/.config/navi/providers/<name>`

### 4. Migrate global config to `~/.config/navi/config.yaml`

- Change `GlobalConfigPath` from `~/.navi/config.yaml` to `~/.config/navi/config.yaml`.
- Add a one-time migration: if `~/.config/navi/config.yaml` doesn't exist but `~/.navi/config.yaml` does, read from the old location and log a warning suggesting the user move it.
- Update all references and documentation.

### Config directory layout after this PBI

```
~/.config/navi/
  config.yaml          # global config (migrated from ~/.navi/config.yaml)
  remotes.yaml         # remote machine config (already here)
  providers/           # custom provider scripts (new)
    my-custom-provider.sh
```

## UX/UI Considerations

- Clear error messages when a custom provider script is not found at the expected path (e.g., "provider script 'foo.sh' not found at ~/.config/navi/providers/foo.sh").
- No changes to `.navi.yaml` format — the `provider` field continues to accept a simple name/filename.
- Deprecation warning when old config location `~/.navi/config.yaml` is detected, guiding the user to move it.

## Acceptance Criteria

1. Built-in providers (`markdown-tasks`, `github-issues`) work correctly when navi is installed via `go install` and run from any directory.
2. The `ProvidersDir` variable and CWD-relative resolution logic are removed.
3. Built-in provider scripts are embedded in the binary via `//go:embed`.
4. Custom (non-built-in) provider names resolve exclusively to `~/.config/navi/providers/<name>`.
5. Relative-to-project-dir and absolute path resolution paths are removed from `ResolveProvider`.
6. Error messages clearly indicate where custom providers should be placed.
7. `GlobalConfigPath` is changed to `~/.config/navi/config.yaml`.
8. Fallback migration reads from `~/.navi/config.yaml` if new path doesn't exist, with a deprecation warning.
9. All existing tests pass; new tests cover embedded providers, `~/.config/navi/providers/` resolution, and config migration fallback.
10. The `providers/` directory remains in the repo (as the source for embedding) but is no longer needed at runtime.

## Dependencies

- None. Self-contained refactor of `internal/task/provider.go`, `internal/task/types.go`, `internal/task/config.go`, and related test files.

## Open Questions

- None.

## Related Tasks

Tasks will be defined when this PBI moves to Agreed status.
