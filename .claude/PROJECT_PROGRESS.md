# Project Progress - hub-tui

## Plan Files
Roadmap: [v6-dependency-management plan](../docs/plan/plan.md)
Current Phase: [phase-4.md](../docs/plan/phases/phase-4.md)
Latest Weekly Report: [weekly-2026-W03.md](../docs/reports/weekly-2026-W03.md)
Archived: [v1-initial-build](../docs/plan/_archived/v1-initial-build/), [v2-llm-profiles](../docs/plan/_archived/v2-llm-profiles/), [v3-param-collection](../docs/plan/_archived/v3-param-collection/), [v4-config-types](../docs/plan/_archived/v4-config-types/), [v5-dynamic-provider-fields](../docs/plan/_archived/v5-dynamic-provider-fields/)

Last Updated: 2026-01-23

## Current Focus
Admin UI for CLI dependency management - installing/updating CLI tools (like sage) and hub-core self-updates

## Active Tasks
- [IN PROGRESS] Phase 4 - Dependencies Tab UI
  - ✓ Added tabs to IntegrationsModal
  - ✓ Added Dependencies tab view with table rendering
  - ✓ Added async dependency loading
  - ✓ Added message routing from app to modal
  - ✓ Removed debug println statements (were corrupting terminal display)
  - ✓ Fixed admin status not fetched on startup (added doGetUserInfo() to Init())
  - ✓ Fixed Dependency struct to match actual API (RequiredVersion, NeedsUpdate, InstalledVersion)
  - ✓ Updated table columns: Tool, Status, Installed, Required, Actions
  - ✓ Install now passes RequiredVersion instead of "latest"
  - ✓ Dynamic hint - hides [Enter] when no actions available
  - ✓ Tested: install/update sage works end-to-end
  - ⏭ Non-admin view testing (needs non-admin user)

## Open Questions/Blockers
None

## Completed This Week
- Phase 1 - User Info & Admin Status
  - UserInfo type with is_admin field
  - GetMe() client method to fetch user info from /me endpoint
  - isAdmin field cached in root app.Model
  - GetMe() called automatically after successful login
  - Error handling defaults to non-admin on failure
  - Tested: /me endpoint successfully called on login
- Phase 2 - Tab Component
  - Reusable Tabs component in internal/ui/components/tabs.go
  - Tab switching methods (Next, Previous, SetActive)
  - Styled tab bar with active/inactive highlighting
  - Width-aware rendering
  - Added Border, BorderActive, Link colors to theme
  - Created test program to verify functionality
- Phase 3 - Dependency Client Layer
  - Created internal/client/dependencies.go
  - Dependency, DependencyUpdate, HubUpdateInfo types
  - GetDependencies() fetches all CLI deps and status
  - InstallDependency() installs/updates a specific CLI tool
  - CheckDependencyUpdates() checks for available updates
  - GetHubUpdates() checks for hub-core updates (handles private repo)
  - ApplyHubUpdate() triggers hub-core self-update and restart
  - All methods follow existing client patterns with parseError()

## Future Enhancements (not in current plan)
- oauth config type (requires browser redirect flow)
- email_pass config type
- Workflow enable/disable toggle (API: PUT /workflows/{name})
- Workflow delete (API: DELETE /workflows/{name})
- Rich array editing with add/remove buttons
- Nested object forms instead of JSON textarea
- Workflow output format hints (server sends output_format: markdown/json for rendering)

## Next Session
- Test non-admin view when user available
- Phase 5: Integration Config Check (block config until deps satisfied)
- Phase 6: Hub Self-Update UI
