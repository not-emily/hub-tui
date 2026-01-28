# Phase 5: Admin Update

> **Depends on:** Phase 4 (Admin Uninstall)
> **Enables:** Completes module registry feature
>
> See: [Full Plan](../plan.md)

## Goal

Add update functionality with visual indicators for available updates and update action.

## Key Deliverables

- Update available indicator in list view
- `[u]` hotkey to update from list view (on selected module)
- `[u]` hotkey to update from detail view
- Update action with loading/success feedback
- Handle "already latest" case gracefully

## Files to Modify

- `internal/ui/modal/modules.go` — Add update indicators and action
- `internal/app/app.go` — Route new message type

## Dependencies

**Internal:**
- Phase 1: `UpdateModule()`, `UpdateResult` type
- Phase 3: Available modules (which include `UpdateAvailable` field)

**External:** None

## Implementation Notes

### Tracking Update Availability

The `AvailableModule` type from Phase 1 includes:
```go
UpdateAvailable  bool   `json:"update_available"`
InstalledVersion string `json:"installed_version"`
Version          string `json:"version"`  // Latest available
```

We need to cross-reference installed modules with available modules to know which have updates. Options:

1. **Load available modules on modal open** (for admins) to get update info
2. **Add UpdateAvailable to Module struct** if the API provides it
3. **Merge data** when both lists are loaded

Recommend option 1: When admin opens modal, load both lists. Store a map of `moduleName -> AvailableModule` for quick lookup.

### New Modal Fields

```go
type ModulesModal struct {
    // ... existing fields ...

    // Update tracking (admin)
    availableByName map[string]*client.AvailableModule
    updateLoading   string  // Module currently being updated
}
```

### New Message Type

```go
type ModuleUpdatedMsg struct {
    Result *client.UpdateResult
    Error  error
}
```

### Init for Admins

```go
func (m *ModulesModal) Init() tea.Cmd {
    if m.isAdmin {
        // Load both lists for admins
        return tea.Batch(m.loadModules(), m.loadAvailableModules())
    }
    return m.loadModules()
}
```

### Build Lookup Map

```go
case AvailableModulesLoadedMsg:
    // ... existing handling ...

    // Build lookup map
    m.availableByName = make(map[string]*client.AvailableModule)
    for i := range msg.Modules {
        m.availableByName[msg.Modules[i].Name] = &msg.Modules[i]
    }
```

### Helper Function

```go
func (m *ModulesModal) hasUpdate(moduleName string) bool {
    if avail, ok := m.availableByName[moduleName]; ok {
        return avail.UpdateAvailable
    }
    return false
}

func (m *ModulesModal) getAvailableVersion(moduleName string) string {
    if avail, ok := m.availableByName[moduleName]; ok {
        return avail.Version
    }
    return ""
}
```

### List View Rendering - Update Indicator

```
  Modules

  ● recipes        1.0.0 → 1.1.0    Manage recipes...    ⬆
  ● fitness        1.0.0            Track workouts...
  ○ notes          0.9.0            Personal notes...

  ● enabled  ○ disabled  ⬆ update available

  [Space] Toggle  [Enter] Detail  [u] Update  [b] Browse  [r] Refresh
```

Show `⬆` or similar indicator next to modules with updates. Show version transition `1.0.0 → 1.1.0` if update available.

### Key Handling - List View

```go
case "u":
    if m.isAdmin && len(m.modules) > 0 {
        mod := m.modules[m.selected]
        if m.hasUpdate(mod.Name) {
            m.updateLoading = mod.Name
            return m, m.updateModule(mod.Name)
        }
    }
```

### Key Handling - Detail View

```go
case "u":
    if m.isAdmin {
        name := m.getCurrentModuleName()
        if m.hasUpdate(name) {
            m.loading = true
            return m, m.updateModule(name)
        }
    }
```

### Handle Update Result

```go
case ModuleUpdatedMsg:
    m.updateLoading = ""
    m.loading = false

    if msg.Error != nil {
        m.error = msg.Error.Error()
        return m, nil
    }

    if msg.Result.AlreadyLatest {
        // Could show brief "Already up to date" message
        // Or just silently succeed
    }

    // Refresh both lists to get new version info
    return m, tea.Batch(m.loadModules(), m.loadAvailableModules())
```

### Detail View - Update Section

For installed modules with updates available, show in detail view:

```
  recipes (installed)

  Installed:  1.0.0
  Available:  1.1.0  ⬆ Update available

  Description: Manage recipes, grocery lists...

  [Space] Disable  [u] Update  [x] Uninstall  [Esc] Back
```

### Hints Update

List view hints for admins:
```go
hints := "[Space] Toggle  [Enter] Detail  [r] Refresh"
if m.isAdmin {
    hints += "  [b] Browse"
    if m.hasUpdate(m.modules[m.selected].Name) {
        hints += "  [u] Update"
    }
}
```

Detail view hints include `[u] Update` only when update is available.

### Loading State

Show loading indicator next to module being updated:
```
  ● recipes        1.0.0 → 1.1.0    Updating...
```

Or use the global loading state if simpler.

## Validation

How do we know this phase is complete?

- [ ] Admin modal loads available modules on init (for update info)
- [ ] Update indicator (⬆) shows in list for modules with updates
- [ ] Version transition shown (e.g., `1.0.0 → 1.1.0`)
- [ ] `[u]` in list updates selected module (admin, has update)
- [ ] `[u]` in detail updates current module (admin, has update)
- [ ] `[u]` hidden/disabled when no update available
- [ ] `[u]` hidden for non-admin users
- [ ] Update success refreshes module lists
- [ ] "Already latest" case handled gracefully
- [ ] Update errors display properly
- [ ] Loading state shown during update
