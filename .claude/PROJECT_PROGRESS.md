# Project Progress - hub-tui

## Plan Files
Roadmap: None
Current Phase: None
Latest Weekly Report: [weekly-2026-W01.md](../docs/reports/weekly-2026-W01.md)
Archived: [v1-initial-build](../docs/plan/_archived/v1-initial-build/), [v2-llm-profiles](../docs/plan/_archived/v2-llm-profiles/), [v3-param-collection](../docs/plan/_archived/v3-param-collection/)

Last Updated: 2026-01-02

## Current Focus
Parameter Collection Form - COMPLETE

## Active Tasks
None

## Open Questions/Blockers
None

## Completed This Week
- Parameter Collection Form - all phases complete
  - Phase 1: Client Types (AskResponse, ParamSchema, AskDirect method)
  - Phase 2: Form Component (FieldTextArea, error/required display, validation helpers)
  - Phase 3: ParamForm Modal (schema→form conversion, Esc/Ctrl+S handling)
  - Phase 4: App Integration (message handlers, doAskWithParams)

## Future Enhancements (not in current plan)
- Workflow enable/disable toggle (API: PUT /workflows/{name})
- Workflow delete (API: DELETE /workflows/{name})
- Integration profile rename/delete (requires hub-core API additions)
- Rich array editing with add/remove buttons
- Nested object forms instead of JSON textarea
- hub-core: Pre-populate schema.params[].value from extracted route params

## Next Session
No active plan. Parameter collection tested and working.
