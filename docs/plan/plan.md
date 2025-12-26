# hub-tui: Terminal client for the Hub personal AI system

> **Status:** Planning complete | Last updated: 2025-12-24
>
> Phase files: [phases/](phases/)

## Overview

hub-tui is a terminal-based client for interacting with hub-core, the backend API for the Hub personal AI system. It provides a conversation-first interface inspired by Claude Code's TUI, allowing users to chat with Hub, switch between assistants, trigger workflows, and manage system configuration—all from the keyboard.

The TUI runs on the user's local machine and connects to a remote hub-core instance (e.g., on a Mac Mini or Raspberry Pi) over Tailscale. It is a pure client with no local state beyond configuration and cached lists for autocomplete.

## Core Vision

- **Conversation-first** — Chat is the primary mode, not menus. Type and Hub responds.
- **Minimal chrome** — Most of the screen is content. Claude Code-inspired simplicity.
- **Keyboard-only** — Everything via keyboard. No mouse support.
- **Modal for management** — Config and settings via overlays, not navigation.

## Requirements

### Must Have
- Connection to hub-core with token auth
- Main chat interface with streaming responses
- Assistant context switching (`@assistant`) with visual indicator (colored border)
- Workflow triggering (`#workflow`) with background execution
- Slash commands (`/modules`, `/integrations`, `/tasks`, `/help`, `/exit`, etc.)
- Tab completion for assistants, workflows, and commands
- Module management modal (list, enable/disable)
- Integration configuration modal (configure, test)
- Workflow browsing modal
- Background task tracking with status indicator
- Dark theme (grays, not black)

### Nice to Have
- Workflow run monitoring (view output)
- Assistant memory viewer
- Conversation history browser
- Vim keybindings for input area

### Out of Scope
- Admin features (user management, module installation)
- Builder UIs (create modules/workflows/assistants)
- Offline mode
- Mouse support
- Light theme
- Multiple server profiles

## Constraints

- **Tech stack:** Go + Bubble Tea + Lip Gloss (consistent with hub-core)
- **Config location:** `~/.config/hub-tui/config.json`
- **Network:** Requires Tailscale connection to hub-core
- **Single server:** Connects to one hub-core instance only

## Success Metrics

- Can chat with Hub and receive streaming responses
- Can switch to an assistant and maintain context
- Can trigger workflows and see their status
- Can manage modules and integrations via modals
- Feels responsive and keyboard-friendly

## Architecture Decisions

### 1. Go + Bubble Tea
**Choice:** Use Go with Bubble Tea framework and Lip Gloss for styling.
**Rationale:** Consistent with hub-core (Go). Bubble Tea is well-maintained with good patterns.
**Trade-offs:** Less flexibility than raw terminal handling, but much faster development.

### 2. Single Root Model
**Choice:** One Bubble Tea model owns all state (chat, modals, status).
**Rationale:** Simpler shared state management. Modals are overlays, not separate apps.
**Trade-offs:** Model file may get large; mitigate with clear separation of concerns.

### 3. Polling for Background Tasks
**Choice:** Poll `/runs` endpoint for task status updates.
**Rationale:** hub-core doesn't have WebSocket/SSE for push updates (yet).
**Trade-offs:** Slight delay in status updates; can add push later.

### 4. Cache on Startup
**Choice:** Fetch assistants/workflows/modules on startup, refresh on modal open.
**Rationale:** Instant autocomplete, minimal staleness risk for single-user system.
**Trade-offs:** Data could be stale mid-session; `/refresh` command available.

### 5. Streaming via Goroutine
**Choice:** API client runs streaming requests in goroutine, sends chunks via `tea.Program.Send()`.
**Rationale:** Bubble Tea's standard pattern for async operations.
**Trade-offs:** Need careful error handling for cancelled requests.

### 6. XDG Config Location
**Choice:** Store config at `~/.config/hub-tui/config.json`.
**Rationale:** Follows XDG Base Directory Specification. TUI config is separate from hub data.
**Trade-offs:** None significant.

## Project Structure

```
hub-tui/
├── cmd/
│   └── hub-tui/
│       └── main.go              # Entry point
│
├── internal/
│   ├── app/
│   │   ├── app.go               # Root Bubble Tea model
│   │   ├── keymap.go            # Key bindings
│   │   └── messages.go          # Custom tea.Msg types
│   │
│   ├── ui/
│   │   ├── chat/
│   │   │   ├── chat.go          # Chat view (messages + input)
│   │   │   ├── message.go       # Single message component
│   │   │   └── input.go         # Input with @/#/ prefix handling
│   │   │
│   │   ├── status/
│   │   │   └── status.go        # Status bar (connection, context, tasks)
│   │   │
│   │   ├── modal/
│   │   │   ├── modal.go         # Modal container/manager
│   │   │   ├── modules.go       # Modules list modal
│   │   │   ├── integrations.go  # Integrations config modal
│   │   │   ├── workflows.go     # Workflows list modal
│   │   │   ├── tasks.go         # Running/failed tasks modal
│   │   │   ├── settings.go      # Settings modal
│   │   │   └── help.go          # Help modal
│   │   │
│   │   ├── components/
│   │   │   ├── list.go          # Reusable list component
│   │   │   ├── form.go          # Reusable form component
│   │   │   └── confirm.go       # Confirmation dialog
│   │   │
│   │   └── theme/
│   │       └── theme.go         # Colors, styles (Lip Gloss)
│   │
│   ├── client/
│   │   ├── client.go            # API client (auth, requests)
│   │   ├── auth.go              # Login, token management
│   │   ├── ask.go               # /ask endpoint (streaming)
│   │   ├── assistants.go        # Assistant endpoints
│   │   ├── workflows.go         # Workflow endpoints
│   │   ├── modules.go           # Module endpoints
│   │   ├── integrations.go      # Integration endpoints
│   │   └── runs.go              # Task/run endpoints
│   │
│   └── config/
│       └── config.go            # Load/save config, defaults
│
├── scripts/
│   ├── build.sh                 # Build binary
│   └── run.sh                   # Run for development
│
├── go.mod
├── go.sum
└── README.md
```

### Key Files
- `internal/app/app.go` — Root Bubble Tea model, owns all state
- `internal/ui/chat/input.go` — Parses `@/#/` prefixes, manages autocomplete
- `internal/client/client.go` — HTTP client with auth header injection
- `internal/ui/theme/theme.go` — Dark theme colors (grays)

## Core Interfaces

### Command Structure

| Trigger | Command | Action |
|---------|---------|--------|
| `@` | `@{assistant}` | Switch context to assistant |
| `@` | `@hub` | Switch back to main hub context |
| `#` | `#{workflow}` | Trigger workflow in background |
| `/` | `/modules` | Open modules modal |
| `/` | `/integrations` | Open integrations modal |
| `/` | `/workflows` | Open workflows modal |
| `/` | `/tasks` | Open tasks modal |
| `/` | `/settings` | Open settings modal |
| `/` | `/help` | Open help modal |
| `/` | `/clear` | Clear chat history |
| `/` | `/refresh` | Refresh cached data |
| `/` | `/exit` | Exit TUI |

### Key Bindings

**Global:**
| Key | Action |
|-----|--------|
| `Ctrl+C` | Quit (double-tap, shows hint on first press) |
| `Ctrl+L` | Clear screen / redraw |
| `Esc` | Close modal / cancel current action |

**Chat view:**
| Key | Action |
|-----|--------|
| `Enter` | Send message |
| `Shift+Enter` | Newline in input |
| `Tab` | Autocomplete (when `@`, `#`, `/` prefix) |
| `Up/Down` | Scroll chat history |
| `PgUp/PgDn` | Scroll chat faster |

**Modal view:**
| Key | Action |
|-----|--------|
| `Up/Down` or `j/k` | Navigate list |
| `Enter` | Select / confirm |
| `Esc` | Close modal |
| `/` | Filter/search (within modal) |

### API Client Interface

```go
type Client interface {
    // Auth
    Login(username, password string) (*Token, error)

    // Ask (streaming)
    Ask(ctx context.Context, message string, onChunk func(string)) (*Response, error)

    // Assistants
    ListAssistants() ([]Assistant, error)
    Chat(ctx context.Context, assistant, message string, onChunk func(string)) (*Response, error)

    // Workflows
    ListWorkflows() ([]Workflow, error)
    RunWorkflow(name string) (runID string, error)

    // Runs/Tasks
    ListRuns() (*RunsResponse, error)
    GetRun(id string) (*Run, error)
    CancelRun(id string) error

    // Modules
    ListModules() ([]Module, error)
    EnableModule(name string) error
    DisableModule(name string) error

    // Integrations
    ListIntegrations() ([]Integration, error)
    ConfigureIntegration(name string, config map[string]string) error
    TestIntegration(name string) (*TestResult, error)
}
```

## Implementation Phases

| Phase | Name | Scope | Depends On | Key Outputs |
|-------|------|-------|------------|-------------|
| 1 | Foundation | Project setup, config, basic app | — | Compiling app, config loading |
| 2 | API Client & Connection | Auth, login flow, health check | Phase 1 | Connected status, token storage |
| 3 | Chat Interface | Messages, streaming, keyboard nav | Phase 2 | Working chat with /ask |
| 4 | Commands & Autocomplete | Slash commands, caching, tab complete | Phase 3 | /exit, /clear, tab completion |
| 5 | Assistant Context | @switching, colored border | Phase 4 | Assistant chat with indicator |
| 6.1 | Modal Framework | Modal pattern, help, settings | Phase 4 | /help, /settings modals |
| 6.2 | Resource Modals | Modules, integrations, workflows | Phase 6.1 | All management modals |
| 7 | Background Tasks | #workflows, polling, /tasks | Phase 6.2 | Task indicator, notifications |

### Critical Path
Phases 1→2→3→4 are strictly sequential.

After Phase 4:
- Phase 5 (Assistant Context) is independent
- Phase 6.1 (Modal Framework) is independent
- Phase 6.2 requires 6.1
- Phase 7 requires 6.2 (for /tasks modal)

### Phase Details
- [Phase 1: Foundation](phases/phase-1.md)
- [Phase 2: API Client & Connection](phases/phase-2.md)
- [Phase 3: Chat Interface](phases/phase-3.md)
- [Phase 4: Commands & Autocomplete](phases/phase-4.md)
- [Phase 5: Assistant Context](phases/phase-5.md)
- [Phase 6.1: Modal Framework](phases/phase-6-1.md)
- [Phase 6.2: Resource Modals](phases/phase-6-2.md)
- [Phase 7: Background Tasks](phases/phase-7.md)

## Tech Stack

| Category | Choice | Notes |
|----------|--------|-------|
| Language | Go | Consistent with hub-core |
| TUI Framework | Bubble Tea | Elm architecture, well-maintained |
| Styling | Lip Gloss | Pairs with Bubble Tea |
| HTTP Client | stdlib net/http | No extra deps |
| Config Format | JSON | Consistent with hub-core |

## Future Considerations

Items explicitly deferred but architecturally supported:

- **WebSocket/SSE for push updates** — Currently polling; can add push when hub-core supports it
- **Vim keybindings** — Input area could support vim mode later
- **Conversation history browser** — API supports it, just needs UI
- **Multiple server profiles** — Config structure could support it, but not needed now
