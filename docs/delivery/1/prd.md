# PBI-1: Project Foundation & Core Types

[View in Backlog](../backlog.md)

## Overview

Set up the Go project with proper module structure, dependencies (Bubble Tea, Lip Gloss), and define the core data structures that will be used throughout the application.

## Problem Statement

Before any features can be implemented, the project needs a solid foundation including Go module initialization, dependency management, and core type definitions that all other components will rely upon.

## User Stories

- As a developer, I want a properly initialized Go module so that dependencies are managed correctly
- As a developer, I want core types defined so that all components share consistent data structures

## Technical Approach

1. Initialize Go module with `go mod init`
2. Add dependencies: Bubble Tea, Lip Gloss
3. Define `SessionInfo` struct for session data
4. Define `Model` struct for Bubble Tea application state
5. Define message types for Bubble Tea communication
6. Create the file structure as specified in the PRD

### Core Types (from PRD)

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

### Message Types

```go
type tickMsg time.Time
type sessionsMsg []SessionInfo
type attachDoneMsg struct{}
```

## UX/UI Considerations

N/A - This PBI is purely foundational infrastructure.

## Acceptance Criteria

1. Go module is initialized and builds successfully
2. Bubble Tea and Lip Gloss dependencies are added to go.mod
3. `SessionInfo` struct is defined with JSON tags
4. `Model` struct is defined with required fields
5. Message types are defined for Bubble Tea
6. Project compiles with `go build`
7. File structure matches PRD specification

## Dependencies

None - This is the foundational PBI.

## Open Questions

None

## Related Tasks

See [Tasks](./tasks.md)
