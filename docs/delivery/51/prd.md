# PBI-51: PM Conversational Interaction

[View in Backlog](../backlog.md)

## Overview

Add an input mechanism to the PM view so developers can ask the PM questions. Route questions through the Claude CLI invoker and display responses in the briefing zone. Turn the PM from a one-way briefing system into a conversational assistant.

## Problem Statement

In Phase 2, the PM speaks but the developer can't talk back. If the briefing mentions something and the developer wants to dig deeper ("what did Apollo do yesterday?", "which project has the most stalled tasks?"), they have to check manually. A conversational interface lets the developer interrogate the PM's knowledge.

## User Stories

- As a developer, I want to ask the PM questions about my projects so that I can get contextual answers without leaving the TUI.
- As a developer, I want the PM to answer based on its memory and current state so that responses are informed by history, not just raw data.

## Technical Approach

- Add text input widget to PM view (Zone 1 or dedicated input area).
- On submit, construct an inbox with the user's question as the trigger (new trigger type: `user_question`), include current state and recent events as context.
- Invoke PM via the existing invoker with `--resume` (same session, so PM has full conversation history).
- Display PM response in the briefing zone, replacing or augmenting the last automated briefing.
- PM can still update its memory files during conversational invocations.
- Input history (up/down arrow to recall previous questions) is a nice-to-have.

## UX/UI Considerations

- Input appears at the bottom of Zone 1 or as an overlay, triggered by a keybinding (e.g., `/` or `:`).
- While waiting for PM response, show a loading indicator.
- PM response replaces the briefing text temporarily; next automated briefing cycle restores normal output.
- Input is single-line, not multi-line. Questions should be concise.

## Acceptance Criteria

1. A keybinding in PM view activates a text input for asking questions.
2. Submitted questions are sent to the PM agent via the existing invoker.
3. PM response renders in the briefing zone.
4. The PM's `--resume` session provides conversational continuity — it remembers previous questions and context.
5. Loading state is shown while waiting for PM response.
6. Normal automated briefings resume after a conversational exchange on the next trigger cycle.

## Dependencies

- **Depends on**: PBI-49 (working briefing panel and PM agent integration)
- **External**: None

## Open Questions

- What keybinding activates the input? `/` conflicts with search. `:` or `?` may work.
- Should conversational responses persist in the briefing zone until the next automated cycle, or clear after a timeout?

## Related Tasks

_Tasks will be created when this PBI moves to Agreed via `/plan-pbi 51`._
