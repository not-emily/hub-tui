# Project Progress - hub-tui

## Plan Files
Roadmap: [v4-config-types plan](../docs/plan/plan.md)
Current Phase: Complete (all phases done)
Latest Weekly Report: [weekly-2026-W01.md](../docs/reports/weekly-2026-W01.md)
Archived: [v1-initial-build](../docs/plan/_archived/v1-initial-build/), [v2-llm-profiles](../docs/plan/_archived/v2-llm-profiles/), [v3-param-collection](../docs/plan/_archived/v3-param-collection/)

Last Updated: 2026-01-07

## Current Focus
QoL improvements: Autocomplete UX and workflow cancel hint

## Active Tasks
- [IN PROGRESS] Autocomplete & Workflow UX improvements (uncommitted)
  - ✓ Auto-show autocomplete when typing /, @, #
  - ✓ Enter executes / commands and # workflows directly
  - ✓ Enter completes @ assistants (for typing message)
  - ✓ Shift+C to cancel tracked workflow with 30s hint
  - ✓ Single active hint model (new workflow clears previous)
  - ⏳ Ready to commit

## Open Questions/Blockers
- hub-core bug: dismiss endpoint returns "run not found" for valid runs

## Completed This Week
- Parameter Collection Form - all phases complete (from previous week)
- v4-config-types planning complete
- v4-config-types implementation complete (all 6 phases)
  - Phase 1: Client layer updates
  - Phase 2: Integration type routing
  - Phase 3: LLM config view
  - Phase 4: Provider management
  - Phase 5: Profile form with cascading dropdowns
  - Phase 6: Cleanup - removed standalone /llm command

## Future Enhancements (not in current plan)
- oauth config type (requires browser redirect flow)
- email_pass config type
- Workflow enable/disable toggle (API: PUT /workflows/{name})
- Workflow delete (API: DELETE /workflows/{name})
- Rich array editing with add/remove buttons
- Nested object forms instead of JSON textarea
- Soften "run not found" error for already-completed workflows on cancel

## Next Session
Commit the autocomplete/workflow UX changes, then decide on next enhancement
