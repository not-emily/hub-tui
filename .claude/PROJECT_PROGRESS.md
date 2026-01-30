# Project Progress - hub-tui

## Plan Files
Roadmap: [v9-workflow-builder](../docs/plan/plan.md)
Current Phase: [Phase 8: Polish & Edge Cases](../docs/plan/phases/phase-8.md)
Latest Weekly Report: [weekly-2026-W05.md](../docs/reports/weekly-2026-W05.md)
Archived: [v1-initial-build](../docs/plan/_archived/v1-initial-build/), [v2-llm-profiles](../docs/plan/_archived/v2-llm-profiles/), [v3-param-collection](../docs/plan/_archived/v3-param-collection/), [v4-config-types](../docs/plan/_archived/v4-config-types/), [v5-dynamic-provider-fields](../docs/plan/_archived/v5-dynamic-provider-fields/), [v6-dependency-management](../docs/plan/_archived/v6-dependency-management/), [v7-assistant-crud](../docs/plan/_archived/v7-assistant-crud/), [v8-module-registry](../docs/plan/_archived/v8-module-registry/)

Last Updated: 2026-01-30

## Current Focus
v9-workflow-builder: Visual workflow builder for creating automations without writing JSON

## Active Tasks
- [IN PROGRESS] Phase 8: Polish & Edge Cases
  - ⏭ Nested params for object-type parameters (requires hub-core API changes)
  - ⏭ Better param documentation/examples from API
  - ⏭ Error handling and edge cases

## Open Questions/Blockers
- **Nested params**: Need hub-core to provide `properties` field for object-type params so TUI can render nested form fields instead of raw JSON input

## Completed This Week
None yet - see [weekly-2026-W05.md](../docs/reports/weekly-2026-W05.md) for last week's accomplishments

## Future Enhancements (not in current plan)
- oauth config type (requires browser redirect flow)
- email_pass config type
- Workflow enable/disable toggle (API: PUT /workflows/{name})
- Rich array editing with add/remove buttons
- Nested object forms instead of JSON textarea
- Workflow output format hints (server sends output_format: markdown/json for rendering)
- Refactor all text fields to use "[Enter to edit]" / "[Enter to confirm]" pattern (like workflow name field)

## Next Session
- Hub-core: Add `properties` field to ToolParam for nested object schemas
- Continue Phase 8: Polish & Edge Cases
