# Project Progress - hub-tui

## Plan Files
Roadmap: [v9-workflow-builder](../docs/plan/plan.md)
Current Phase: [Phase 8: Transform Forms](../docs/plan/phases/phase-8.md)
Latest Weekly Report: [weekly-2026-W05.md](../docs/reports/weekly-2026-W05.md)
Archived: [v1-initial-build](../docs/plan/_archived/v1-initial-build/), [v2-llm-profiles](../docs/plan/_archived/v2-llm-profiles/), [v3-param-collection](../docs/plan/_archived/v3-param-collection/), [v4-config-types](../docs/plan/_archived/v4-config-types/), [v5-dynamic-provider-fields](../docs/plan/_archived/v5-dynamic-provider-fields/), [v6-dependency-management](../docs/plan/_archived/v6-dependency-management/), [v7-assistant-crud](../docs/plan/_archived/v7-assistant-crud/), [v8-module-registry](../docs/plan/_archived/v8-module-registry/)

Last Updated: 2026-02-03

## Current Focus
v9-workflow-builder: Visual workflow builder for creating automations without writing JSON

## Active Tasks
- [IN PROGRESS] Phase 8: Transform Forms
  - ✓ Transform operation picker (Filter, Extract, Sort, First, Last, Count)
  - ✓ Filter form with field/operator/value
  - ✓ Extract form with field mappings
  - ✓ Sort form with direction toggle
  - ✓ First/Last/Count forms
  - ✓ Transform preview API integration
  - ✓ Step insertion into workflow
  - ✓ Variable picker [v] with field extraction
  - ✓ Test on demand [t] for variables without cached output
  - ✓ Bug fixes:
    - ✓ Workflow save 400 error (step type mapping: integration→module, primitive→utility)
    - ✓ [Esc] on Select Variable page (missing FieldCancelledMsg routing in app.go)
    - ✓ [t] test stuck on regular steps (missing PickerNeedsTestMsg handler in StepForm)
    - ✓ /tasks modal JSON unmarshal error (RunResult.Output: string→interface{})
  - ✓ Clean up debug logging (disabled logging to debug.log)
  - ✓ `needs_attention_on_complete` workflow toggle (Notify: Yes/No)
  - ⏭ Final validation testing

## Open Questions/Blockers
None

## Completed This Week
- Nested object params with `properties` render as nested form fields
- Array params with `items` render as dynamic list with add/remove
- Example values shown as hints in param descriptions
- Objects/arrays without schema fall back to JSON editor (unchanged)
- Variable picker UX improvements:
  - Always shows variable list first (even with 1 variable)
  - "(entire variable)" option at top of field list
  - Step name shown as description for each variable
- AsyncModalMessage interface pattern for automatic async message routing
  - Replaces ~80 individual message handlers in app.go
  - Ensures auth error handling on all async responses
  - Documented in .claude/CLAUDE.md
- Transform form bug fixes:
  - FieldPicker now properly updates view after successful [t] test
  - Ctrl+S in transform form now correctly saves and returns to step list
- `needs_attention_on_complete` toggle added to workflow builder
- Debug logging cleanup (functions made no-ops, unused os imports removed)

## Future Enhancements (not in current plan)
- oauth config type (requires browser redirect flow)
- email_pass config type
- Workflow enable/disable toggle (API: PUT /workflows/{name})
- Workflow output format hints (server sends output_format: markdown/json for rendering)
- Refactor all text fields to use "[Enter to edit]" / "[Enter to confirm]" pattern (like workflow name field)

## Next Session
- Final validation testing for Phase 8
- Commit current changes
- Phase 9: Validation & Polish
