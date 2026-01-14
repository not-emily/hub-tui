# Project Progress - hub-tui

## Plan Files
Roadmap: [v4-config-types plan](../docs/plan/plan.md)
Current Phase: Complete (all phases done)
Latest Weekly Report: [weekly-2026-W02.md](../docs/reports/weekly-2026-W02.md)
Archived: [v1-initial-build](../docs/plan/_archived/v1-initial-build/), [v2-llm-profiles](../docs/plan/_archived/v2-llm-profiles/), [v3-param-collection](../docs/plan/_archived/v3-param-collection/)

Last Updated: 2026-01-14

## Current Focus
QoL improvements and bug fixes

## Active Tasks
None currently

## Open Questions/Blockers
None

## Completed This Week
- Settings modal edit mode
  - Added [e] to enter edit mode from /settings
  - Server URL editable with form component
  - Ctrl+S to save, Esc to cancel
  - On save: updates client, clears token, returns to login for new server
- Fix status bar missing server URL on auto-connect
  - Added SetServerURL call when reconnecting with saved token

## Future Enhancements (not in current plan)
- oauth config type (requires browser redirect flow)
- email_pass config type
- Workflow enable/disable toggle (API: PUT /workflows/{name})
- Workflow delete (API: DELETE /workflows/{name})
- Rich array editing with add/remove buttons
- Nested object forms instead of JSON textarea
- Workflow output format hints (server sends output_format: markdown/json for rendering)

## Next Session
Decide on next enhancement or QoL improvement
