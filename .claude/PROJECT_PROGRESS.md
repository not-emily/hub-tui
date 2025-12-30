# Project Progress - hub-tui

## Plan Files
Roadmap: [plan.md](../docs/plan/plan.md)
Current Phase: [phase-4.md](../docs/plan/phases/phase-4.md) ✓ Complete
Latest Weekly Report: [weekly-2025-W52.md](../docs/reports/weekly-2025-W52.md)
Archived: [v1-initial-build](../docs/plan/_archived/v1-initial-build/)

Last Updated: 2025-12-30

## Current Focus
LLM Profile Management - All phases complete with enhancements.

## Active Tasks
None - all planned phases complete!

## Open Questions/Blockers
None

## Completed This Week
- Phase 1: Client Layer
  - Created internal/client/llm.go
  - Added LLMProfile, LLMProfileList, LLMTestResult types
  - Implemented ListLLMProfiles, Create/Update/Delete, Test, SetDefault methods
  - Added put() and delete() helper methods to client.go
- Phase 2: Modal List View
  - Created internal/ui/modal/llm.go with list view
  - Name (★ default), Model, Provider columns
  - Test, delete, set-default, refresh functionality
- Phase 3: Modal Edit/Create View
  - Added edit/create forms with validation
  - "+ New Profile" menu option
- Phase 4: App Integration
  - Wired /llm command into app.go
  - Added message handlers with auth expiry handling
  - Updated help modal and autocomplete
- Enhancement: Form Select Fields
  - Extended components/form.go with FieldSelect type
  - Added SetFieldOptions, GetFieldValue methods
  - LLM modal uses select fields for Integration and Integration Profile
  - Dynamic profile updates when integration selection changes

## Future Enhancements (not in current plan)
- Workflow enable/disable toggle (API: PUT /workflows/{name})
- Workflow delete (API: DELETE /workflows/{name})
- Integration profile rename/delete (requires hub-core API additions)

## Next Session
LLM Profile Management feature complete! Ready for testing or next feature.
