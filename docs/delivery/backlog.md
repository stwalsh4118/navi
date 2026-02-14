# Product Backlog

This document contains all Product Backlog Items (PBIs) for the claude-sessions project, ordered by priority.

## Backlog

| ID | Actor | User Story | Status | Conditions of Satisfaction (CoS) |
|----|-------|-----------|--------|----------------------------------|
| 1 | User | As a user, I want the project foundation set up with Go modules and core types so that development can proceed with a solid base | Done | [View Details](./1/prd.md) |
| 2 | User | As a user, I want Claude Code hooks that write status updates so that the TUI can monitor session states | Done | [View Details](./2/prd.md) |
| 3 | User | As a user, I want session polling and state management so that the TUI displays current session information | Done | [View Details](./3/prd.md) |
| 4 | User | As a user, I want a styled TUI that displays all Claude sessions with status icons and messages so that I can see at a glance which sessions need attention | Done | [View Details](./4/prd.md) |
| 5 | User | As a user, I want to attach to and detach from tmux sessions directly from the TUI so that I can interact with Claude and return to the dashboard seamlessly | Done | [View Details](./5/prd.md) |
| 6 | User | As a user, I want an installation script so that I can easily deploy the hooks and TUI binary | Done | [View Details](./6/prd.md) |
| 7 | User | As a user, I want to create, kill, and rename sessions directly from the TUI so that I can manage my Claude sessions without leaving the dashboard | Done | [View Details](./7/prd.md) |
| 8 | User | As a user, I want to organize sessions with groups and tags so that I can manage many sessions across different projects | Proposed | [View Details](./8/prd.md) |
| 9 | User | As a user, I want desktop notifications when sessions need attention so that I don't have to constantly watch the TUI | Proposed | [View Details](./9/prd.md) |
| 10 | User | As a user, I want webhook integrations (Slack, Discord) so that I can receive session alerts in my team communication tools | Proposed | [View Details](./10/prd.md) |
| 11 | User | As a user, I want a preview pane showing recent session output so that I can see what's happening without fully attaching | Done | [View Details](./11/prd.md) |
| 12 | User | As a user, I want to see session metrics (token usage, time tracking, tool activity) so that I can understand resource consumption | Done | [View Details](./12/prd.md) |
| 13 | User | As a user, I want fuzzy search and filtering so that I can quickly find sessions when I have many running | Done | [View Details](./13/prd.md) |
| 14 | User | As a user, I want to select multiple sessions and perform bulk actions so that I can efficiently manage many sessions at once | Proposed | [View Details](./14/prd.md) |
| 15 | User | As a user, I want a split view to compare two sessions side-by-side so that I can monitor related work simultaneously | Proposed | [View Details](./15/prd.md) |
| 16 | User | As a user, I want git integration showing branch, status, and diffs so that I have project context without leaving the TUI | Done | [View Details](./16/prd.md) |
| 17 | User | As a user, I want session history and bookmarks so that I can track what sessions accomplished and mark important ones | Proposed | [View Details](./17/prd.md) |
| 18 | User | As a user, I want to export and replay session logs so that I can review past Claude interactions | Proposed | [View Details](./18/prd.md) |
| 19 | User | As a user, I want to aggregate sessions from remote machines via SSH so that I can manage Claude across multiple servers | Done | [View Details](./19/prd.md) |
| 20 | User | As a user, I want a team dashboard to share session visibility so that my team can see each other's Claude sessions | Proposed | [View Details](./20/prd.md) |
| 21 | User | As a user, I want auto-start configuration and custom hooks so that I can automate session setup and respond to status changes | Proposed | [View Details](./21/prd.md) |
| 22 | User | As a user, I want permission rules to auto-approve certain tool calls so that low-risk operations don't interrupt my workflow | Proposed | [View Details](./22/prd.md) |
| 23 | User | As a user, I want a CLI mode with non-interactive commands so that I can script navi into my automation workflows | Proposed | [View Details](./23/prd.md) |
| 24 | User | As a user, I want customizable themes and keybindings so that I can personalize the TUI to my preferences | Proposed | [View Details](./24/prd.md) |
| 25 | User | As a user, I want mouse support so that I can click to select sessions and scroll the list | Proposed | [View Details](./25/prd.md) |
| 26 | User | As a user, I want to see token usage per session by reading Claude's transcript files so that I can track API costs | Done | [View Details](./26/prd.md) |
| 27 | User | As a user, I want full management capabilities for remote sessions (git info, preview, kill, rename, dismiss) so that remote sessions have feature parity with local sessions | Done | [View Details](./27/prd.md) |
| 28 | User | As a user, I want a task view with pluggable providers so that I can see my project tasks from any system (GitHub Issues, Linear, markdown, etc.) directly in Navi | Done | [View Details](./28/prd.md) |
| 29 | User | As a user, I want an in-app content viewer so that I can view files, diffs, and task details without leaving the TUI | Done | [View Details](./29/prd.md) |
| 30 | User | As a user, I want task providers to supply file paths so that the content viewer can open task detail files from any provider without hardcoded path assumptions | Proposed | [View Details](./30/prd.md) |
| 31 | User | As a user, I want vim-style exact-match search with next/previous cycling so that I can quickly locate specific sessions and tasks without fuzzy matching | Done | [View Details](./31/prd.md) |
| 32 | User | As a user, I want all TUI panels to scroll their content so that I can see all items even when the panel is too short to display everything | Done | [View Details](./32/prd.md) |
| 33 | User | As a user, I want the TUI to correctly handle agent team sessions and show inline team status so that I can monitor multi-agent workflows and the dashboard remains accurate | Done | [View Details](./33/prd.md) |
| 34 | User | As a user, I want enhanced session creation with shell-only mode, directory tab-completion, clone session, and attach-after-create so that session setup is faster and more flexible | Proposed | [View Details](./34/prd.md) |
| 35 | User | As a user, I want built-in provider scripts embedded in the binary, custom providers loaded from ~/.config/navi/providers/, and all config consolidated under ~/.config/navi/ so that navi works reliably from anywhere with a single config directory | Agreed | [View Details](./35/prd.md) |
| 36 | User | As a user, I want navi to build and run on Windows with platform-appropriate paths and graceful degradation for Unix-only features so that I can use navi across operating systems | Proposed | [View Details](./36/prd.md) |
| 37 | User | As a user, I want sorting, filtering, progress indicators, summary stats, and improved navigation in the task view so that I can efficiently manage a large backlog | Done | [View Details](./37/prd.md) |
| 38 | User | As a user, I want enhanced GitHub PR integration with CI/CD check statuses, review state, comments, labels, draft/merge status, and change stats so that I have full PR context without leaving the TUI | Agreed | [View Details](./38/prd.md) |
| 39 | User | As a user, I want remote sessions to have full lifecycle parity with local sessions (stale cleanup, TUI creation, responsive status updates) so that managing remote sessions feels identical to local ones | Proposed | [View Details](./39/prd.md) |
| 40 | User | As a user, I want navi to automatically detect which AI agents (Claude Code, Codex, etc.) are running in each tmux session's panes so that I can track multi-agent workflows | Proposed | [View Details](./40/prd.md) |
| 41 | User | As a user, I want the TUI to show per-agent status indicators for each session so that I can see both Claude Code and Codex state at a glance | Proposed | [View Details](./41/prd.md) |
| 42 | User | As a user, I want configurable audio notifications with custom sound files and TTS session name announcements when session statuses change so that I have audio awareness without watching the TUI | Proposed | [View Details](./42/prd.md) |

## History Log

| Timestamp | PBI_ID | Event_Type | Details | User |
|-----------|--------|------------|---------|------|
| 2026-02-05 00:00:00 | 1 | Created | PBI created from PRD breakdown | AI_Agent |
| 2026-02-05 00:00:00 | 2 | Created | PBI created from PRD breakdown | AI_Agent |
| 2026-02-05 00:00:00 | 3 | Created | PBI created from PRD breakdown | AI_Agent |
| 2026-02-05 00:00:00 | 4 | Created | PBI created from PRD breakdown | AI_Agent |
| 2026-02-05 00:00:00 | 5 | Created | PBI created from PRD breakdown | AI_Agent |
| 2026-02-05 00:00:00 | 6 | Created | PBI created from PRD breakdown | AI_Agent |
| 2026-02-05 05:45:42 | 6 | Updated | Added conflict detection and user prompting requirements for settings.json merge | AI_Agent |
| 2026-02-05 12:00:00 | 7 | Created | Session Management Actions - create, kill, rename sessions from TUI | AI_Agent |
| 2026-02-05 12:00:00 | 8 | Created | Session Organization - groups, tags, templates | AI_Agent |
| 2026-02-05 12:00:00 | 9 | Created | Desktop Notifications - native OS notifications with cooldown | AI_Agent |
| 2026-02-05 12:00:00 | 10 | Created | Webhook Integrations - Slack, Discord, custom HTTP endpoints | AI_Agent |
| 2026-02-05 12:00:00 | 11 | Created | Session Preview Pane - see output without attaching | AI_Agent |
| 2026-02-05 12:00:00 | 12 | Created | Session Metrics - token usage, time tracking, tool activity | AI_Agent |
| 2026-02-05 12:00:00 | 13 | Created | Search & Filter - fuzzy search, status filters, sort options | AI_Agent |
| 2026-02-05 12:00:00 | 14 | Created | Multi-Session Operations - bulk select, batch actions | AI_Agent |
| 2026-02-05 12:00:00 | 15 | Created | Split View - side-by-side session comparison | AI_Agent |
| 2026-02-05 12:00:00 | 16 | Created | Git Integration - branch, status, diff preview, GitHub links | AI_Agent |
| 2026-02-05 12:00:00 | 17 | Created | Session History & Bookmarks - track accomplishments, mark favorites | AI_Agent |
| 2026-02-05 12:00:00 | 18 | Created | Session Logs & Export - replay, export to markdown | AI_Agent |
| 2026-02-05 12:00:00 | 19 | Created | Remote Sessions - SSH tunneling, multi-machine aggregation | AI_Agent |
| 2026-02-05 12:00:00 | 20 | Created | Team Dashboard - shared session visibility | AI_Agent |
| 2026-02-05 12:00:00 | 21 | Created | Auto-Start & Custom Hooks - config-based launch, status change scripts | AI_Agent |
| 2026-02-05 12:00:00 | 22 | Created | Permission Rules - auto-approve certain tool calls | AI_Agent |
| 2026-02-05 12:00:00 | 23 | Created | CLI Mode - non-interactive commands for scripting | AI_Agent |
| 2026-02-05 12:00:00 | 24 | Created | Themes & Keybindings - customization, compact mode, help overlay | AI_Agent |
| 2026-02-05 12:00:00 | 25 | Created | Mouse Support - click, scroll interactions | AI_Agent |
| 2026-02-05 08:02:16 | 7 | Status Change | Proposed → Agreed - Approved for task breakdown | User |
| 2026-02-05 12:00:00 | 11 | Status Change | Proposed → Agreed - Approved for task breakdown | User |
| 2026-02-05 13:58:48 | 19 | Status Change | Proposed → Agreed - Approved for task breakdown | User |
| 2026-02-05 16:24:09 | 12 | Status Change | Proposed → Agreed - Approved for task breakdown | User |
| 2026-02-05 17:35:00 | 26 | Created | Token Metrics from Session Transcripts - parse ~/.claude transcript files | AI_Agent |
| 2026-02-05 17:35:00 | 26 | Status Change | Proposed → Agreed - Approved for implementation | User |
| 2026-02-06 00:00:00 | 27 | Created | Enhanced Remote Session Management - git, preview, kill, rename, dismiss for remote sessions | User |
| 2026-02-06 00:00:00 | 27 | Status Change | Proposed → Agreed - PBI reviewed, open questions resolved, approved for task breakdown | User |
| 2026-02-06 17:38:38 | 27 | Status Change | Agreed → InReview - All 7 tasks completed, all tests pass, ready for user review | AI_Agent |
| 2026-02-06 18:00:00 | 13 | Status Change | Proposed → Agreed - Approved for task breakdown | User |
| 2026-02-06 18:41:08 | 13 | Status Change | Agreed → InReview - All 6 tasks completed, all tests pass, ready for user review | AI_Agent |
| 2026-02-06 19:00:00 | 28 | Created | Task view with pluggable providers - display project tasks from any source via script-based provider system | User |
| 2026-02-06 19:15:00 | 28 | Status Change | Proposed → Agreed - PBI reviewed, open questions resolved, approved for task breakdown | User |
| 2026-02-06 20:00:00 | 29 | Created | In-App Content Viewer - view files, diffs, and task details within the TUI | User |
| 2026-02-06 21:00:00 | 1 | Status Change | Proposed → Done - All 4 tasks Done, backlog status corrected | AI_Agent |
| 2026-02-06 21:00:00 | 2 | Status Change | Proposed → Done - All 3 tasks Done, backlog status corrected | AI_Agent |
| 2026-02-06 21:00:00 | 3 | Status Change | Proposed → Done - All 5 tasks Done, backlog status corrected | AI_Agent |
| 2026-02-06 21:00:00 | 4 | Status Change | Proposed → Done - All 6 tasks Done, backlog status corrected | AI_Agent |
| 2026-02-06 21:00:00 | 5 | Status Change | Proposed → Done - All 5 tasks Done, backlog status corrected | AI_Agent |
| 2026-02-06 21:00:00 | 6 | Status Change | Proposed → Done - All 5 tasks Done, backlog status corrected | AI_Agent |
| 2026-02-06 21:00:00 | 7 | Status Change | Agreed → Done - All 6 tasks Done, backlog status corrected | AI_Agent |
| 2026-02-06 21:00:00 | 11 | Status Change | Agreed → Done - All 10 tasks Done, backlog status corrected | AI_Agent |
| 2026-02-06 21:00:00 | 12 | Status Change | Agreed → Done - All 10 tasks Done, backlog status corrected | AI_Agent |
| 2026-02-06 21:00:00 | 13 | Status Change | InReview → Done - All 6 tasks Done, backlog status corrected | AI_Agent |
| 2026-02-06 21:00:00 | 16 | Status Change | Proposed → Done - All 11 tasks Done, backlog status corrected | AI_Agent |
| 2026-02-06 21:00:00 | 19 | Status Change | Agreed → Done - All 8 tasks Done, backlog status corrected | AI_Agent |
| 2026-02-06 21:00:00 | 26 | Status Change | Agreed → Done - All 4 tasks Done, backlog status corrected | AI_Agent |
| 2026-02-06 21:00:00 | 27 | Status Change | InReview → Done - All 7 tasks Done, backlog status corrected | AI_Agent |
| 2026-02-06 21:00:00 | 28 | Status Change | Agreed → InReview - All 8 tasks in Review, backlog status corrected | AI_Agent |
| 2026-02-06 21:47:53 | 28 | Status Change | InReview → Done - All 8 tasks Done, all tests pass with -race, go vet clean | AI_Agent |
| 2026-02-07 14:10:17 | 29 | Status Change | Proposed → Agreed - Approved for task breakdown | User |
| 2026-02-07 15:41:24 | 29 | Status Change | Agreed → Done - All 5 tasks Done, all tests pass with -race, go vet clean | AI_Agent |
| 2026-02-07 15:50:00 | 30 | Created | Provider-supplied file paths for content viewer - identified during PBI-29 PR review | User |
| 2026-02-07 16:00:00 | 31 | Created | Vim-style exact-match search with next/previous cycling | User |
| 2026-02-07 16:10:00 | 31 | Status Change | Proposed → Agreed - Approved for task breakdown | User |
| 2026-02-09 16:12:00 | 31 | Status Change | Agreed → Done - All 6 tasks Done, all tests pass with -race, go vet clean | AI_Agent |
| 2026-02-09 17:00:00 | 32 | Created | Scrollable Panels - add scrolling to task panel, preview pane, session list, and dialogs | User |
| 2026-02-09 17:05:00 | 32 | Status Change | Proposed → Agreed - Approved for task breakdown | User |
| 2026-02-09 18:49:31 | 32 | Status Change | Agreed → Done - All 5 tasks Done, all tests pass with -race, go vet clean | AI_Agent |
| 2026-02-09 19:00:00 | 33 | Created | Agent Team Awareness and Hook Robustness - fix stale permission status, prevent teammate hooks from corrupting main session, show inline agent team info | User |
| 2026-02-09 19:15:00 | 33 | Status Change | Proposed → Agreed - Open questions resolved, approved for task breakdown | User |
| 2026-02-09 21:45:49 | 33 | Status Change | Agreed → Done - All 5 tasks Done, all tests pass with -race, go vet clean, live E2E verified with 3-agent team | AI_Agent |
| 2026-02-10 00:00:00 | 34 | Created | Enhanced Session Creation - shell-only mode, directory tab-completion, clone session, attach-after-create | User |
| 2026-02-12 00:00:00 | 35 | Created | Embed Built-in Providers and Standardize Custom Provider Location | User |
| 2026-02-12 00:00:00 | 35 | Status Change | Proposed → Agreed - Scope expanded to include global config migration to ~/.config/navi/, approved for task breakdown | User |
| 2026-02-12 00:00:00 | 36 | Created | Windows Build Support and Cross-Platform Compatibility | User |
| 2026-02-12 00:00:00 | 37 | Created | Task View Enhancements — Sorting, Filtering, Progress, and Navigation | User |
| 2026-02-12 07:37:46 | 37 | Status Change | Proposed → Agreed - Approved for task breakdown | User |
| 2026-02-12 08:49:09 | 37 | Status Change | InProgress → Done - All 7 tasks Done, all tests pass with -race, go vet clean | AI_Agent |
| 2026-02-12 12:00:00 | 38 | Created | Enhanced GitHub PR Integration — CI/CD checks, review status, comments, labels, draft/merge state, change stats | User |
| 2026-02-12 12:05:00 | 38 | Status Change | Proposed → Agreed - Approved for task breakdown | User |
| 2026-02-13 00:00:00 | 39 | Created | Remote Session Lifecycle Parity — stale cleanup, TUI creation, responsive status updates for remote sessions | User |
| 2026-02-14 00:00:00 | 40 | Created | Tmux Pane Agent Detection — auto-detect Claude Code, Codex, and other agents in tmux panes via process tree inspection | User |
| 2026-02-14 00:00:00 | 41 | Created | Multi-Agent TUI Display — show per-agent status indicators for sessions with multiple detected agents | User |
| 20260214-083634 | 42 | Created | PBI created from feature request: Audio notifications with custom sound files and TTS session announcements on status changes | AI_Agent |
