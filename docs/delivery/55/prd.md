# PBI-55: PM View Refresh UX and Provider Performance

[View in Backlog](../backlog.md)

## Overview

Improve PM view responsiveness so users get immediate feedback when opening the PM screen and faster project status updates. Add clear loading visibility and reduce provider-driven latency in the PM data path.

## Problem Statement

Today, pressing `P` can feel slow and opaque because PM updates rely on periodic ticks and provider refresh timing. Users can interpret the delay as a hang because there is no clear loading state, and provider execution across many projects can add noticeable wait time.

## User Stories

- As a user, I want PM project status to refresh quickly when I open PM view so I can trust the view is current.
- As a user, I want a visible loading indicator while PM data is refreshing so I know the app is working.
- As a user, I want provider refreshes to complete faster across multiple projects so PM context appears without long delays.

## Technical Approach

Add an immediate PM run trigger on PM view entry and expose in-flight PM refresh state in the PM rendering path. Introduce an explicit loading indicator/spinner tied to PM and/or task refresh in-flight states.

Optimize provider refresh latency by improving execution strategy in task refresh (for example, bounded concurrency while preserving cache behavior and deterministic result assembly). Keep PM output correctness unchanged and preserve existing resolver precedence from PBI-54.

## UX/UI Considerations

- Show a clear PM loading indicator when PM refresh is in flight.
- Avoid flicker by preserving last successful PM output until new data arrives.
- Ensure loading treatment works in narrow terminal widths.

## Acceptance Criteria

1. Opening PM view triggers an immediate PM refresh without waiting for the periodic PM tick.
2. PM view displays a visible loading indicator while PM refresh is in flight.
3. Manual refresh interaction in PM context is explicit and consistent with user expectations.
4. Provider refresh path is optimized so multi-project refresh latency is reduced compared with sequential execution.
5. Existing PM project snapshot correctness and current-PBI resolution behavior remain unchanged.
6. Automated tests cover PM loading-state visibility and refresh behavior regressions.

## Dependencies

- **Depends on**: PBI-47, PBI-54
- **Blocks**: None
- **External**: None

## Open Questions

- Should PM loading indicator be text-only, spinner glyph-based, or both?

## Related Tasks

[View Tasks](./tasks.md)
