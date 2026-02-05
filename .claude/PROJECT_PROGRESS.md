# Project Progress - hub-tui

## Plan Files
Roadmap: None
Current Phase: None
Latest Weekly Report: [weekly-2026-W05.md](../docs/reports/weekly-2026-W05.md)
Archived: [v1-initial-build](../docs/plan/_archived/v1-initial-build/), [v2-llm-profiles](../docs/plan/_archived/v2-llm-profiles/), [v3-param-collection](../docs/plan/_archived/v3-param-collection/), [v4-config-types](../docs/plan/_archived/v4-config-types/), [v5-dynamic-provider-fields](../docs/plan/_archived/v5-dynamic-provider-fields/), [v6-dependency-management](../docs/plan/_archived/v6-dependency-management/), [v7-assistant-crud](../docs/plan/_archived/v7-assistant-crud/), [v8-module-registry](../docs/plan/_archived/v8-module-registry/), [v9-workflow-builder](../docs/plan/_archived/v9-workflow-builder/)

Last Updated: 2026-02-05

## Current Focus
v9-workflow-builder plan complete — all 9 phases implemented and tested.

## Active Tasks
None

## Open Questions/Blockers
None

## Completed This Week
- Phase 9: Validation & Polish
  - [v] keybinding to validate workflow via API (dry_run on create/update endpoints)
  - Validation view showing success or grouped errors (by workflow/step)
  - Validate-before-save: blocks save if invalid, auto-saves if valid
  - Dirty state confirmation on close (y to discard, n to cancel, s to save)
  - Variable color coding: green for tested, yellow(?) for untested
  - Step error indicators in list view (! for steps with validation errors)
  - Help overlay with [?] showing all keyboard shortcuts
  - Workflow name field read-only for existing workflows (name is API identifier)
  - Removed unnecessary client-side step type mapping (types pass through from API)
  - Fixed 400 errors not displaying in builder (route errors from modal to builder)
  - End-to-end testing complete
- Phase 8 (previous session):
  - Transform forms, field picker navigation, async message pattern

## hub-core issues found during testing
- Variable parser doesn't support hyphens in variable names ($recipes-mod parsed as $recipes)
- "operation" field required for integration steps but not surfaced in builder tools API

## Future Enhancements (not in current plan)
- oauth config type (requires browser redirect flow)
- email_pass config type
- Workflow enable/disable toggle (API: PUT /workflows/{name})
- Workflow output format hints (server sends output_format: markdown/json for rendering)
- Refactor all text fields to use "[Enter to edit]" / "[Enter to confirm]" pattern (like workflow name field)

## Next Session
- Archive v9-workflow-builder plan
- Consider next project/plan direction
