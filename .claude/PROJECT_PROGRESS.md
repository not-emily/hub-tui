# Project Progress - hub-tui

## Plan Files
Roadmap: [v4-config-types plan](../docs/plan/plan.md)
Current Phase: [Phase 1: Client Layer Updates](../docs/plan/phases/phase-1.md)
Latest Weekly Report: [weekly-2026-W01.md](../docs/reports/weekly-2026-W01.md)
Archived: [v1-initial-build](../docs/plan/_archived/v1-initial-build/), [v2-llm-profiles](../docs/plan/_archived/v2-llm-profiles/), [v3-param-collection](../docs/plan/_archived/v3-param-collection/)

Last Updated: 2026-01-05

## Current Focus
v4-config-types: Config-type-aware Integration Configuration

## Active Tasks
- Phase 1: Client Layer Updates - NEXT
  - Update Integration type with ConfigType field
  - Create integrations_llm.go with provider/profile methods

## Open Questions/Blockers
None

## Completed This Week
- Parameter Collection Form - all phases complete (from previous week)
- v4-config-types planning complete

## Future Enhancements (not in current plan)
- oauth config type (requires browser redirect flow)
- email_pass config type
- Workflow enable/disable toggle (API: PUT /workflows/{name})
- Workflow delete (API: DELETE /workflows/{name})
- Rich array editing with add/remove buttons
- Nested object forms instead of JSON textarea

## Next Session
Start Phase 1: Client Layer Updates - update Integration type and create integrations_llm.go
