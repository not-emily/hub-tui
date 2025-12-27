# Project Progress - hub-tui

## Plan Files
Roadmap: [plan.md](../docs/plan/plan.md)
Current Phase: [phase-6-2.md](../docs/plan/phases/phase-6-2.md)
Latest Weekly Report: None

Last Updated: 2025-12-27

## Current Focus
Phase 6.2: Resource Modals - Modules, integrations, workflows, tasks modals

## Active Tasks
- [NEXT] Phase 6.2: Resource Modals
  - ⏭ Implement /modules modal
  - ⏭ Implement /integrations modal
  - ⏭ Implement /workflows modal
  - ⏭ Implement /tasks modal

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
  - Created modal framework (modal.go) with overlay rendering
  - Built reusable list component with j/k navigation
  - Implemented /help modal with commands and keyboard shortcuts
  - Implemented /settings modal displaying config and connection status
  - Integrated modals into app.go with keyboard routing
  - Esc to close modals, keys don't leak to chat when modal open

## Next Session
Start Phase 6.2: Resource Modals
