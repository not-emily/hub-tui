# Project Progress - hub-tui

## Plan Files
Roadmap: None
Current Phase: None
Latest Weekly Report: [weekly-2026-W04.md](../docs/reports/weekly-2026-W04.md)
Archived: [v1-initial-build](../docs/plan/_archived/v1-initial-build/), [v2-llm-profiles](../docs/plan/_archived/v2-llm-profiles/), [v3-param-collection](../docs/plan/_archived/v3-param-collection/), [v4-config-types](../docs/plan/_archived/v4-config-types/), [v5-dynamic-provider-fields](../docs/plan/_archived/v5-dynamic-provider-fields/), [v6-dependency-management](../docs/plan/_archived/v6-dependency-management/), [v7-assistant-crud](../docs/plan/_archived/v7-assistant-crud/), [v8-module-registry](../docs/plan/_archived/v8-module-registry/)

Last Updated: 2026-01-27

## Current Focus
None - ready for next feature

## Active Tasks
None

## Open Questions/Blockers
None

## Completed This Week
- Phase 5: Module Templates (v7)
  - Template list view grouped by module
  - Template preview with name/display_name overrides
  - Create assistant from template
  - All validation criteria passed
- v8-module-registry (all phases)
  - Phase 1: Client layer with registry API methods
  - Phase 2: Modal refactor with view states, detail view, double-press disable
  - Phase 3: Admin browse available modules, install from registry
  - Phase 4: Admin uninstall with affected users confirmation
  - Phase 5: Admin update with version indicators

## Future Enhancements (not in current plan)
- oauth config type (requires browser redirect flow)
- email_pass config type
- Workflow enable/disable toggle (API: PUT /workflows/{name})
- Workflow delete (API: DELETE /workflows/{name})
- Rich array editing with add/remove buttons
- Nested object forms instead of JSON textarea
- Workflow output format hints (server sends output_format: markdown/json for rendering)

## Next Session
- Pick next feature from Future Enhancements or start new plan
