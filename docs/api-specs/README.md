# API Specifications

This directory contains concise API reference documentation for navi's internal packages.

## Available Specs

| System | File | Description |
|--------|------|-------------|
| audio | [audio/audio-api.md](./audio/audio-api.md) | Audio config loading, backend detection, notifier orchestration, and TUI integration |
| cli | [cli/cli-api.md](./cli/cli-api.md) | One-shot CLI subcommands, including `navi status` output and flags |
| git | [git/git-pr-api.md](./git/git-pr-api.md) | PR detail fetching, comment fetching, and related types |
| monitor | [monitor/monitor-api.md](./monitor/monitor-api.md) | Background attach monitor lifecycle and state handoff API |
| pm | [pm/pm-api.md](./pm/pm-api.md) | PM agent invoker, briefing types, recovery, caching, and TUI integration |
| session | [session/session-api.md](./session/session-api.md) | Session status model, sorting/aggregation helpers, and status file IO |
