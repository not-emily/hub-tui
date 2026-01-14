# Weekly Report - hub-tui - Week 2 (Jan 6-12, 2026)

## Week Overview

This week focused on quality-of-life improvements and UX polish. Major accomplishments include Claude Code-style autocomplete behavior, workflow cancel functionality with hint tracking, fixing the tasks modal stale cache bug, and simplifying workflow output display.

## Key Accomplishments

### Autocomplete & Input UX
- Implemented auto-show autocomplete when typing `/`, `@`, or `#` prefixes
- Changed Enter key behavior: executes `/commands` and `#workflows` directly, completes `@assistants` for continued typing
- Added `AutocompletePrefix()`, `MessageCount()`, `UpdateMessageContent()` methods to chat model

### Workflow Management
- Added Shift+C to cancel the most recently started workflow
- Implemented 30-second cancel hint that clears on timeout or workflow completion
- Single active hint model - new workflow clears previous hint
- Added `WorkflowHintExpiredMsg` message type and hint tracking state

### Tasks Modal Improvements
- Fixed stale cache bug - always fetch fresh data from API when opening /tasks
- Removed `NewTasksModalWithState` and related conversion helpers
- Added `[h]` History hint to empty "No tasks today" state
- Added `[r]` refresh hotkey to list, history, and detail views
- Added `Output` field to `RunResult` struct for simplified workflow output
- Simplified `formatRunOutput()` to use `result.Output` directly instead of parsing step outputs

## Decisions This Week

No new architectural decisions logged this week.

## Challenges Encountered

- **hub-core dismiss endpoint bug**: Returns "run not found" for valid runs visible in /tasks list. Determined to be a race condition for fast-completing workflows - the run completes before the cancel request arrives. This is expected behavior and could be handled with a softer error message in the future.

## Metrics

- **Commits:** 3
- **Files changed:** 7
- **Lines added:** ~254
- **Lines removed:** ~163

## Next Week Priorities

1. Decide on next enhancement (oauth config type, workflow management, settings improvements)
2. Consider archiving the completed v4-config-types plan
3. Address any UX issues discovered through usage
