# Phase 2: Modal Refactor

> **Depends on:** Phase 1 (Client Layer)
> **Enables:** Phase 3 (Admin Browse & Install)
>
> See: [Full Plan](../plan.md)

## Goal

Refactor ModulesModal to support view states, add detail view, pass isAdmin flag, and implement double-press confirmation for disable.

## Key Deliverables

- View state constants (list, detail)
- `isAdmin` flag passed to modal
- Module detail view showing full info
- Double-press confirmation for disable using `Confirmation` component
- Updated key bindings (Space to toggle, Enter for detail)
- Message routing in app.go for new message types

## Files to Modify

- `internal/ui/modal/modules.go` — Add view states, detail view, confirmation
- `internal/app/app.go` — Pass isAdmin to ModulesModal, route new messages

## Dependencies

**Internal:**
- Phase 1 client methods (for type definitions)
- `internal/ui/components/confirm.go` (existing Confirmation component)

**External:** None

## Implementation Notes

### View State Constants

```go
const (
    moduleViewList = iota
    moduleViewDetail
    // Phase 3 will add: moduleViewAvailable
    // Phase 4 will add: moduleViewConfirmUninstall
)
```

### Modal Struct Updates

```go
type ModulesModal struct {
    client   *client.Client
    isAdmin  bool              // NEW: admin status
    modules  []client.Module
    selected int
    loading  bool
    error    string

    // View state
    view    int               // NEW: current view
    current *client.Module    // NEW: selected module for detail view

    // Confirmation
    confirm *components.Confirmation  // NEW: for disable confirmation
}
```

### Constructor Update

```go
func NewModulesModal(c *client.Client, isAdmin bool) *ModulesModal {
    return &ModulesModal{
        client:  c,
        isAdmin: isAdmin,
        loading: true,
        view:    moduleViewList,
        confirm: components.NewConfirmation(),
    }
}
```

### Key Handling Updates

**List View:**
```go
case "space", " ":
    if len(m.modules) > 0 {
        mod := m.modules[m.selected]
        if mod.Enabled {
            // Disable requires confirmation
            if execute, cmd := m.confirm.Check("disable", mod.Name); execute {
                return m, m.disableModule()
            } else if cmd != nil {
                return m, cmd
            }
        } else {
            // Enable is immediate
            return m, m.enableModule()
        }
    }
case "enter":
    if len(m.modules) > 0 {
        m.current = &m.modules[m.selected]
        m.view = moduleViewDetail
        m.confirm.Clear()
    }
```

**Detail View:**
```go
case "esc":
    m.view = moduleViewList
    m.current = nil
case "space", " ":
    // Same toggle logic as list view
```

### Handle Confirmation Expiry

```go
case components.ConfirmationExpiredMsg:
    m.confirm.HandleExpired(msg)
    return m, nil
```

### Detail View Rendering

Show:
- Name (title)
- Description
- Version
- Tools list
- Enabled status
- Hints based on role:
  - All users: `[Space] Enable/Disable  [Esc] Back`
  - Admins will get more in later phases

### Update app.go

```go
// In handleCommand for "modules":
return m, m.modal.Open(modal.NewModulesModal(m.client, m.isAdmin))

// Add message routing for ConfirmationExpiredMsg if not already present
```

### Hint Line Logic

When confirmation is pending:
```go
if m.confirm.IsPending("disable", m.modules[m.selected].Name) {
    hints = "Press Space again to disable"
} else {
    hints = "[Space] Toggle  [Enter] Detail  [r] Refresh"
}
```

## Validation

How do we know this phase is complete?

- [ ] Modal receives and stores `isAdmin` flag
- [ ] `Enter` on list opens detail view
- [ ] `Esc` on detail returns to list
- [ ] Detail view shows name, description, version, tools, status
- [ ] `Space` on enabled module shows "Press Space again to disable"
- [ ] Second `Space` within 2s disables the module
- [ ] `Space` on disabled module enables immediately (no confirmation)
- [ ] Confirmation expires after 2 seconds
- [ ] Navigation clears pending confirmation
- [ ] app.go passes isAdmin to ModulesModal
- [ ] ConfirmationExpiredMsg is routed to modal
