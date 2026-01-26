# Phase 2: List & Detail Views

> **Depends on:** Phase 1 (Client Layer)
> **Enables:** Phase 3 (Create & Edit), Phase 4 (Memory), Phase 5 (Templates)
>
> See: [Full Plan](../plan.md)

## Goal

Create the AssistantsModal with list view and read-only detail view, establishing the navigation foundation for later phases.

## Key Deliverables

- AssistantsModal implementing Modal interface
- List view with assistant names and status indicators
- Detail view showing all assistant fields (read-only)
- Navigation between list and detail
- Message routing from app.go to modal
- Basic key bindings (j/k, Enter, Esc, r)

## Files to Create/Modify

- `internal/ui/modal/assistants.go` — New modal implementation
- `internal/app/app.go` — Add message routing for assistant messages

## Dependencies

**Internal:**
- Phase 1 client methods (GetAssistant, ListAssistants)
- Existing Modal interface pattern

**External:** None

## Implementation Notes

### View States (Phase 2 scope)

```go
const (
    viewList = iota
    viewDetail
    // viewCreate, viewEdit, viewMemory added in later phases
)
```

### AssistantsModal Structure

```go
type AssistantsModal struct {
    client     *client.Client

    // List state
    assistants []client.Assistant
    selected   int
    loading    bool
    error      string

    // Detail state
    view       int // viewList, viewDetail, etc.
    current    *client.Assistant // Currently viewed assistant
}
```

### Message Types

```go
// AssistantsLoadedMsg is sent when assistant list is loaded
type AssistantsLoadedMsg struct {
    Assistants []client.Assistant
    Error      error
}

// AssistantDetailMsg is sent when single assistant is loaded
type AssistantDetailMsg struct {
    Assistant *client.Assistant
    Error     error
}
```

### List View Layout

```
  Assistants

  ● jarvis          default    general assistant
  ● chef            local      cooking & recipes
  ○ fitness         default    (disabled)

  ● enabled  ○ disabled

  [n] New  [t] From template  [r] Refresh  [Enter] View
```

### Detail View Layout

```
  jarvis

  Display Name:  Jarvis
  LLM Profile:   default
  Keywords:      jarvis, hey jarvis, j

  Modules:       recipes, calendar
  Gather:        recipes → get_meal_plan
                 calendar → get_today_events

  Persona:
  ┌──────────────────────────────────────────────────────┐
  │ You are Jarvis, a helpful AI assistant who helps     │
  │ with daily tasks, meal planning, and scheduling.     │
  │ You have access to recipes and calendar modules.     │
  └──────────────────────────────────────────────────────┘

  Memory: 3 entries (use [m] to view)

  [e] Edit  [m] Memory  [h] Clear history  [d] Delete  [Esc] Back
```

### Key Bindings

**List view:**
- `j/k` or `↑/↓` — Navigate list
- `Enter` — View selected assistant detail
- `n` — New assistant (Phase 3)
- `t` — Create from template (Phase 5)
- `r` — Refresh list
- `Esc` — Close modal

**Detail view:**
- `e` — Edit assistant (Phase 3)
- `m` — View/edit memory (Phase 4)
- `h` — Clear history (with confirmation)
- `d` — Delete assistant (with confirmation)
- `Esc` — Back to list

For Phase 2, `n`, `t`, `e`, `m` can show "Coming soon" or be no-ops.

### Message Routing in app.go

Add cases for assistant messages in the main Update function:

```go
case modal.AssistantsLoadedMsg:
    if m.modal != nil {
        var cmd tea.Cmd
        m.modal, cmd = m.modal.Update(msg)
        return m, cmd
    }

case modal.AssistantDetailMsg:
    // Same pattern
```

## Validation

How do we know this phase is complete?

- [ ] AssistantsModal can be opened from main app
- [ ] List view shows all assistants with enabled/disabled indicators
- [ ] Can navigate list with j/k or arrow keys
- [ ] Pressing Enter on assistant opens detail view
- [ ] Detail view shows all assistant fields formatted nicely
- [ ] Persona is displayed in a bordered box (may be truncated with scroll hint)
- [ ] Esc from detail returns to list
- [ ] Esc from list closes modal
- [ ] [r] refreshes the assistant list
- [ ] Loading and error states are handled
- [ ] Messages are properly routed from app.go to modal
