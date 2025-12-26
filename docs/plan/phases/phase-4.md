# Phase 4: Commands & Autocomplete

> **Depends on:** Phase 3 (Chat Interface)
> **Enables:** Phase 5 (Assistant Context), Phase 6.1 (Modal Framework)
>
> See: [Full Plan](../plan.md)

## Goal

Implement slash commands and tab completion for assistants, workflows, and commands.

## Key Deliverables

- Input parser detects `@`, `#`, `/` prefixes
- Slash commands: `/exit`, `/clear`, `/help`, `/refresh`
- Cache assistants, workflows, modules on startup
- Refresh cache on `/refresh`
- Tab completion for all prefix types
- `/help` shows inline help text

## Files to Create

- `internal/ui/chat/parser.go` — Parse prefixes and commands
- `internal/ui/chat/autocomplete.go` — Autocomplete logic
- `internal/client/assistants.go` — List assistants endpoint
- `internal/client/workflows.go` — List workflows endpoint
- `internal/client/modules.go` — List modules endpoint

## Files to Modify

- `internal/app/app.go` — Add cache, command dispatch
- `internal/app/messages.go` — Add command-related messages
- `internal/ui/chat/input.go` — Integrate parser and autocomplete

## Dependencies

**Internal:** Chat interface from Phase 3

**External:** None

## Implementation Notes

### Prefix Detection

When the user types, detect prefixes at the start of input:

```go
type InputPrefix int

const (
    PrefixNone InputPrefix = iota
    PrefixAssistant  // @
    PrefixWorkflow   // #
    PrefixCommand    // /
)

func DetectPrefix(input string) (InputPrefix, string) {
    input = strings.TrimSpace(input)
    if strings.HasPrefix(input, "@") {
        return PrefixAssistant, input[1:]
    }
    if strings.HasPrefix(input, "#") {
        return PrefixWorkflow, input[1:]
    }
    if strings.HasPrefix(input, "/") {
        return PrefixCommand, input[1:]
    }
    return PrefixNone, input
}
```

### Slash Commands

| Command | Action |
|---------|--------|
| `/exit` | Exit TUI (same as double Ctrl+C) |
| `/clear` | Clear chat history |
| `/help` | Show help text inline |
| `/refresh` | Refresh cached data |

For now, `/help` can just print help text as a "hub" message. Modal version comes in Phase 6.1.

### Caching

On startup (after successful auth):
1. Fetch `GET /assistants` → cache list
2. Fetch `GET /workflows` → cache list
3. Fetch `GET /modules` → cache list

Store in app model:

```go
type Cache struct {
    Assistants []Assistant
    Workflows  []Workflow
    Modules    []Module
    LastUpdate time.Time
}
```

`/refresh` re-fetches all three.

### Autocomplete

When Tab is pressed:

1. Detect prefix
2. Get current partial text after prefix
3. Filter cached list by partial match
4. If one match → complete it
5. If multiple matches → show completion menu

Completion menu (simple list below input):

```
┌─────────────────────────────────────────┐
│ > @fit                                  │
├─────────────────────────────────────────┤
│   fitness_trainer                       │
│   fitness_log                           │
└─────────────────────────────────────────┘
```

Arrow keys navigate, Enter selects, Esc cancels.

### Tab Completion Sources

| Prefix | Source |
|--------|--------|
| `@` | Cached assistants + `hub` (to return to main) |
| `#` | Cached workflows |
| `/` | Hardcoded command list |

### Help Text

```
hub-tui Commands:

  @{assistant}  Switch to an assistant context
  @hub          Return to main hub context
  #{workflow}   Trigger a workflow

  /modules      Manage modules
  /integrations Configure integrations
  /workflows    Browse workflows
  /tasks        View background tasks
  /settings     Open settings
  /help         Show this help
  /clear        Clear chat
  /refresh      Refresh cached data
  /exit         Exit hub-tui

Keyboard:
  Enter         Send message
  Shift+Enter   New line
  Tab           Autocomplete
  Ctrl+C        Exit (press twice)
  Esc           Cancel/close
```

## Validation

How do we know this phase is complete?

- [ ] `/exit` exits the TUI
- [ ] `/clear` clears chat history
- [ ] `/help` shows help text
- [ ] `/refresh` refreshes cache (log message confirms)
- [ ] Type `@` + Tab shows assistant list
- [ ] Type `#` + Tab shows workflow list
- [ ] Type `/` + Tab shows command list
- [ ] Partial match filters list (e.g., `@fit` → `fitness_trainer`)
- [ ] Enter completes selection
- [ ] Esc cancels autocomplete
- [ ] Unknown commands show error message
