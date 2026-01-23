# Project Progress - hub-tui

## Plan Files
Roadmap: None
Current Phase: None
Latest Weekly Report: [weekly-2026-W03.md](../docs/reports/weekly-2026-W03.md)
Archived: [v1-initial-build](../docs/plan/_archived/v1-initial-build/), [v2-llm-profiles](../docs/plan/_archived/v2-llm-profiles/), [v3-param-collection](../docs/plan/_archived/v3-param-collection/), [v4-config-types](../docs/plan/_archived/v4-config-types/), [v5-dynamic-provider-fields](../docs/plan/_archived/v5-dynamic-provider-fields/), [v6-dependency-management](../docs/plan/_archived/v6-dependency-management/)

Last Updated: 2026-01-23

## Current Focus
No active plan - ready for new work

## Active Tasks
None

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
- Phase 4 - Dependencies Tab UI
  - Added tabs to IntegrationsModal
  - Added Dependencies tab view with table rendering
  - Added async dependency loading and message routing
  - Fixed admin status not fetched on startup (added doGetUserInfo() to Init())
  - Fixed Dependency struct to match actual API (RequiredVersion, NeedsUpdate, InstalledVersion)
  - Table columns: Tool, Status, Installed, Required, Actions
  - Install passes RequiredVersion instead of "latest"
  - Dynamic hint - hides [Enter] when no actions available
  - Tested: install/update sage works end-to-end
  - Note: Non-admin view testing deferred (needs non-admin user)
- Phase 5 - Integration Config Dependency Check
  - Check dependencies before allowing integration configuration
  - Block config flow if deps unsatisfied, show "Dependencies Required" view
  - Admin can install via [Enter], auto-proceeds to config after success
  - Non-admin sees "Contact your administrator" message
  - Added GetDependencies() integration filter support (?integration=llm)
  - Added DependencyCheckMsg routing from app to modal
  - Tested: outdated sage blocks AI config, install proceeds to config
- Phase 6 - Hub Self-Update UI
  - Added Hub Version section to Dependencies tab
  - Shows current version and update availability
  - [c] to check for updates, [u] to apply update
  - Confirmation prompt before applying update
  - Auto-loads hub updates when switching to Dependencies tab
  - Handles private repo gracefully (404 = updates not available)
  - Note: Full testing deferred until hub-core repo is public

## Future Enhancements (not in current plan)
- oauth config type (requires browser redirect flow)
- email_pass config type
- Workflow enable/disable toggle (API: PUT /workflows/{name})
- Workflow delete (API: DELETE /workflows/{name})
- Rich array editing with add/remove buttons
- Nested object forms instead of JSON textarea
- Workflow output format hints (server sends output_format: markdown/json for rendering)

## Next Session
Ready for new work - no active plan
