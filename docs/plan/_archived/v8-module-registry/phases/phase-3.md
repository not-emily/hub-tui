# Phase 3: Admin Browse & Install

> **Depends on:** Phase 2 (Modal Refactor)
> **Enables:** Phase 4 (Admin Uninstall)
>
> See: [Full Plan](../plan.md)

## Goal

Add the available modules view for admins to browse the registry and install modules.

## Key Deliverables

- `moduleViewAvailable` view state
- `[b]` hotkey to browse available modules (admin only)
- Available modules list showing install status
- `[i]` hotkey to install module from detail view
- Install action with loading/success/error feedback
- Navigation between list ↔ available ↔ detail views

## Files to Modify

- `internal/ui/modal/modules.go` — Add available view, install functionality
- `internal/app/app.go` — Route new message types

## Dependencies

**Internal:**
- Phase 1: `ListAvailableModules()`, `InstallModule()`, `AvailableModule` type
- Phase 2: View state pattern, detail view

**External:** None

## Implementation Notes

### New View State

```go
const (
    moduleViewList = iota
    moduleViewDetail
    moduleViewAvailable  // NEW
)
```

### New Modal Fields

```go
type ModulesModal struct {
    // ... existing fields ...

    // Available modules (admin)
    availableModules []client.AvailableModule
    availableSelected int
    availableLoading bool

    // Track which view we came from for detail
    detailSource int  // moduleViewList or moduleViewAvailable
}
```

### New Message Types

```go
type AvailableModulesLoadedMsg struct {
    Modules []client.AvailableModule
    Error   error
}

type ModuleInstalledMsg struct {
    Name  string
    Error error
}
```

### Key Handling - List View

```go
case "b":
    if m.isAdmin {
        m.availableLoading = true
        m.view = moduleViewAvailable
        return m, m.loadAvailableModules()
    }
```

### Key Handling - Available View

```go
case "esc":
    m.view = moduleViewList
case "up", "k":
    // Navigate available modules
case "down", "j":
    // Navigate available modules
case "enter":
    if len(m.availableModules) > 0 {
        // Store source and show detail
        m.detailSource = moduleViewAvailable
        m.currentAvailable = &m.availableModules[m.availableSelected]
        m.view = moduleViewDetail
    }
```

### Key Handling - Detail View (Install)

```go
case "i":
    if m.isAdmin && m.detailSource == moduleViewAvailable {
        avail := m.currentAvailable
        if avail != nil && !avail.Installed {
            m.loading = true
            return m, m.installModule(avail.Name)
        }
    }
```

### Detail View Adaptation

The detail view needs to work for both installed modules (from list) and available modules (from browse):

```go
func (m *ModulesModal) viewDetail() string {
    if m.detailSource == moduleViewAvailable && m.currentAvailable != nil {
        return m.viewAvailableDetail()
    }
    return m.viewInstalledDetail()
}
```

**Available module detail shows:**
- Name, description, version
- Keywords
- Install status (installed version if installed)
- Update available indicator
- Hints: `[i] Install` (if not installed), `[Esc] Back`

### Available Modules List Rendering

```
  Browse Available Modules

  ● recipes        1.1.0    Manage recipes, grocery lists...
  ○ fitness        1.0.0    Track workouts and fitness goals
  ○ notes          0.9.0    Personal note-taking system

  ● installed  ○ not installed

  [Enter] View  [Esc] Back
```

### After Install Success

- Reload both module lists (installed and available)
- Could show success message briefly
- Stay in detail view or return to available list

## Validation

How do we know this phase is complete?

- [ ] `[b]` from list opens available modules view (admin only)
- [ ] `[b]` does nothing for non-admin users
- [ ] Available modules load from registry
- [ ] Available modules show installed/not-installed indicator
- [ ] `Enter` on available module opens detail view
- [ ] Detail view shows available module info
- [ ] `[i]` installs module (admin, not installed)
- [ ] `[i]` hidden/disabled for already-installed modules
- [ ] Install success refreshes module lists
- [ ] Install errors display properly
- [ ] `Esc` from available returns to list
- [ ] `Esc` from detail returns to correct source view
