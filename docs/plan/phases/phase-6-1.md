# Phase 6.1: Modal Framework + Simple Modals

> **Depends on:** Phase 4 (Commands & Autocomplete)
> **Enables:** Phase 6.2 (Resource Modals)
>
> See: [Full Plan](../plan.md)

## Goal

Establish the modal pattern with simpler read-only modals before building interactive ones.

## Key Deliverables

- Modal framework (overlay rendering, keyboard handling)
- Help modal (command reference, keyboard shortcuts)
- Settings modal (display current config)
- j/k navigation in modal lists
- Esc to close modals

## Files to Create

- `internal/ui/modal/modal.go` — Modal container/manager
- `internal/ui/modal/help.go` — Help modal
- `internal/ui/modal/settings.go` — Settings modal
- `internal/ui/components/list.go` — Reusable list component

## Files to Modify

- `internal/app/app.go` — Modal state, keyboard routing
- `internal/app/messages.go` — Modal-related messages

## Dependencies

**Internal:** Commands from Phase 4

**External:** None

## Implementation Notes

### Modal Framework

The modal is an overlay that renders on top of the chat view:

```go
type Modal interface {
    Init() tea.Cmd
    Update(msg tea.Msg) (Modal, tea.Cmd)
    View() string
    Title() string
}

type ModalState struct {
    Active Modal  // nil when no modal open
}
```

### Rendering

```go
func (m Model) View() string {
    base := m.chatView()

    if m.modal.Active != nil {
        overlay := m.renderModal(m.modal.Active)
        return m.overlayOnBase(base, overlay)
    }
    return base
}

func (m Model) renderModal(modal Modal) string {
    content := modal.View()

    return lipgloss.NewStyle().
        Width(60).
        Height(20).
        Border(lipgloss.RoundedBorder()).
        BorderForeground(theme.Accent).
        Padding(1, 2).
        Render(
            lipgloss.JoinVertical(
                lipgloss.Left,
                lipgloss.NewStyle().Bold(true).Render(modal.Title()),
                "",
                content,
            ),
        )
}
```

### Keyboard Routing

When a modal is open:
1. All keys go to the modal first
2. Esc closes the modal
3. Other keys handled by modal's Update

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if m.modal.Active != nil {
        switch msg := msg.(type) {
        case tea.KeyMsg:
            if msg.String() == "esc" {
                m.modal.Active = nil
                return m, nil
            }
        }
        var cmd tea.Cmd
        m.modal.Active, cmd = m.modal.Active.Update(msg)
        return m, cmd
    }
    // Normal handling...
}
```

### Help Modal

Content (formatted nicely):

```
Commands:
  @{assistant}  Switch to assistant
  @hub          Return to hub
  #{workflow}   Run workflow

  /modules      Manage modules
  /integrations Configure integrations
  /workflows    Browse workflows
  /tasks        View tasks
  /settings     Settings
  /help         This help
  /clear        Clear chat
  /refresh      Refresh cache
  /exit         Exit

Keyboard:
  Enter         Send / Select
  Shift+Enter   New line
  Tab           Autocomplete
  Ctrl+C        Exit (×2)
  Esc           Close / Cancel
  j/k           Navigate lists
```

### Settings Modal

Display current configuration:

```
Settings

Server URL:  http://192.168.1.100:8787
Username:    emily
Connected:   Yes

Token expires: 2025-12-25 10:00

(Settings are stored in ~/.config/hub-tui/config.json)
```

Read-only for now. Editing could be added later.

### List Component

Reusable for modals that have selectable lists:

```go
type List struct {
    items    []string
    selected int
    height   int
}

func (l *List) Update(msg tea.Msg) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "up", "k":
            l.selected = max(0, l.selected-1)
        case "down", "j":
            l.selected = min(len(l.items)-1, l.selected+1)
        }
    }
}
```

### Opening Modals

From command dispatch:

```go
func (m *Model) handleCommand(cmd string) tea.Cmd {
    switch cmd {
    case "help":
        m.modal.Active = NewHelpModal()
    case "settings":
        m.modal.Active = NewSettingsModal(m.config)
    // ...
    }
    return nil
}
```

## Validation

How do we know this phase is complete?

- [ ] `/help` opens help modal
- [ ] Help modal shows all commands and keyboard shortcuts
- [ ] `/settings` opens settings modal
- [ ] Settings modal shows server URL, username, connection status
- [ ] Esc closes any open modal
- [ ] j/k navigate within modals (if applicable)
- [ ] Modal renders centered over chat view
- [ ] Modal has visible border
- [ ] Only one modal can be open at a time
- [ ] Keys don't leak to chat when modal is open
