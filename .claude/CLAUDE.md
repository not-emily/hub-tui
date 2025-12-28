# Claude Context - hub-tui

This file provides context for Claude Code sessions.

## Project Overview
hub-tui is a terminal-based client for interacting with hub-core, the backend API for the Hub personal AI system. It provides a conversation-first interface inspired by Claude Code's TUI, allowing users to chat with Hub, switch between assistants, trigger workflows, and manage system configuration—all from the keyboard.

## Tech Stack
- Go with Bubble Tea (TUI framework, Elm architecture)
- Lip Gloss for styling
- stdlib net/http for API client
- JSON config at `~/.config/hub-tui/config.json`

## Key Patterns & Conventions
- **API documentation:** `[../hub-core/docs/api/](../hub-core/docs/api/)`
- **Governing design doc:** `[../hub-core/docs/Design.md](../hub-core/docs/Design.md)`
- **Keyboard-only** — no mouse support
- **Client only** — no local state beyond config and cache; hub-core is source of truth
- **Command triggers:** `@assistant` (context switch), `#workflow` (run), `/command` (system)
- **Modal overlays** for management (modules, integrations, tasks), not separate views
- **Cache on startup** — assistants/workflows/modules cached for autocomplete, refresh on modal open
- **Polling** for background task status (hub-core doesn't have push yet)
- **Single root Bubble Tea model** — modals are overlays, not separate apps

## Important Context
The TUI runs on the user's local machine and connects to a remote hub-core instance (e.g., on a Mac Mini or Raspberry Pi) over Tailscale. It is a pure client with no local state beyond configuration and cached lists for autocomplete.

## Project Structure
- `cmd/hub-tui/` — entrypoint
- `internal/app/` — root Bubble Tea model, keymap, messages
- `internal/ui/chat/` — chat view, input, message rendering
- `internal/ui/modal/` — modal overlays (modules, integrations, workflows, tasks, help, settings)
- `internal/ui/status/` — status bar (connection, context, task counts)
- `internal/ui/components/` — reusable list, form components
- `internal/ui/theme/` — Lip Gloss colors and styles (dark theme, grays)
- `internal/client/` — HTTP client for hub-core API
- `internal/config/` — config loading/saving

## Helper Scripts
Scripts in `scripts/` are reusable helpers. **Before writing repetitive bash commands:**
1. Check if a script already exists in `scripts/`
2. If not, consider creating one for sequences you'll run again

This reduces permission prompts and ensures consistency.
