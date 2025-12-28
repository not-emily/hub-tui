# Project Progress - hub-tui

## Plan Files
Roadmap: None
Current Phase: None
Latest Weekly Report: None
Archived: [v1-initial-build](../docs/plan/_archived/v1-initial-build/)

Last Updated: 2025-12-28

## Current Focus
Plan complete! All phases finished.

## Active Tasks
None

## Open Questions/Blockers
None

## Completed This Week
- Phase 1: Foundation
  - Initialized Go module with Bubble Tea & Lip Gloss
  - Created project directory structure
  - Implemented config loading/saving (~/.config/hub-tui/config.json)
  - Created dark theme with gray color palette
  - Built root Bubble Tea app with double Ctrl+C quit
  - Added keymap and message types
  - Created build.sh and run.sh scripts
  - Verified build with go vet and go build
- Phase 2: API Client & Connection
  - Created HTTP client with auth header injection
  - Implemented login endpoint and token validation
  - Built login flow UI (server URL, username, password)
  - Added JWT token expiry detection
  - Created status bar component (connected/disconnected)
  - Integrated login flow into app startup
  - Health check on startup verifies connection
- UI styling fixes
  - Fixed gray bar artifacts using WithWhitespaceBackground in Place()
  - Established pattern: explicit Background(theme.Background) on text elements
  - Avoid JoinVertical with centering; use string concatenation instead
  - Added dynamic Ctrl+C hint to login page
- Phase 3: Chat Interface
  - Created chat view with scrollable message list
  - Built input component with multi-line support (Ctrl+J for newline)
  - Implemented /ask endpoint with SSE streaming
  - Added streaming response display with ▌ indicator
  - Message styling differentiates user vs hub
  - Keyboard navigation: Up/Down scroll, PgUp/PgDn for faster scroll
  - Auto-scroll to bottom on new messages
  - Cancel streaming with Ctrl+C
- Phase 4: Commands & Autocomplete
  - Created input parser for @, #, / prefix detection
  - Implemented slash commands: /exit, /clear, /help, /refresh
  - Added API client endpoints for assistants, workflows, modules
  - Built cache system with startup fetch and /refresh support
  - Implemented tab autocomplete with popup menu
  - Arrow key navigation and Enter to complete
- Phase 5: Assistant Context
  - Updated SSE parser to handle route/chunk/done events
  - Added RouteMsg to capture routing info from hub-core
  - Track current context (type, target) in app model
  - Status bar shows @assistant indicator when in context
  - Colored border around chat when in assistant mode
- Phase 5 Enhancements (UI polish & routing)
  - Markdown rendering for hub responses using glamour with custom zero-margin style
  - Changed message indicators from "You:"/"Hub:" to "›"/"●" with colors
  - Hub indicator changed from green to white for cleaner look
  - Sticky target routing: messages without @ prefix go to current assistant
  - Added /hub slash command to return to main hub context
  - Input area shows colored top/bottom lines when in assistant context
  - Fixed non-streaming response handling (utility, module, workflow, unknown route types now display correctly)
  - AssistantChat client method for direct /assistants/{name}/chat endpoint
- Phase 6.1: Modal Framework
  - Created inline modal framework (renders between chat and input)
  - Built reusable list component with j/k navigation
  - Implemented /help modal with commands and keyboard shortcuts
  - Implemented /settings modal displaying config and connection status
  - Modal has rounded border with title bar and "Esc to close" hint
  - Integrated modals into app.go with keyboard routing
  - Decision: Use inline modals instead of overlay (see DECISIONS.md)
- Phase 6.2: Resource Modals
  - Implemented /modules modal with list display and enable/disable toggle
  - Added EnableModule/DisableModule client endpoints
  - Implemented /workflows modal with list display and descriptions
  - Created reusable form component with password masking and cursor support
  - Created integrations client with configure and test endpoints
  - Implemented /integrations modal with 3-level view (list → profiles → configure)
  - Profile support: multiple accounts per integration, new profile creation
  - Fixed paste handling: use tea.KeyRunes instead of msg.String() for proper paste
  - Fixed modal navigation: q closes modal, Esc goes back within modal
  - Auth expiry handling: detect 401 errors, redirect to login automatically
  - Wired up all modals in app.go with async message handling
  - Tested all resource modals (modules, workflows, integrations)
- Phase 7: Background Tasks
  - Created runs client with ListRuns, GetRun, RunWorkflow, CancelRun
  - Added task-related messages (WorkflowStartedMsg, TaskStatusMsg, etc.)
  - Status bar shows running/failed task counts
  - #workflow triggers workflow in background
  - Polling every 3 seconds while tasks running
  - Completion/failure notifications in chat
  - /tasks modal with Running, Completed, Failed sections
  - Task detail view with step outputs
  - Fixed API parsing (active + history arrays)
  - Task categorization by status AND result.success
  - Load task history from hub-core on startup

## Future Enhancements (not in current plan)
- Workflow enable/disable toggle (API: PUT /workflows/{name})
- Workflow delete (API: DELETE /workflows/{name})
- Integration profile rename/delete (requires hub-core API additions)

## Next Session
All planned phases complete!
