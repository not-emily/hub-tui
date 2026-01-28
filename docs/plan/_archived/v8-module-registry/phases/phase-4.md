# Phase 4: Admin Uninstall

> **Depends on:** Phase 3 (Admin Browse & Install)
> **Enables:** Phase 5 (Admin Update)
>
> See: [Full Plan](../plan.md)

## Goal

Add uninstall functionality with a confirmation dialog that shows affected users.

## Key Deliverables

- `moduleViewConfirmUninstall` view state
- `[x]` hotkey to initiate uninstall from detail view
- Confirmation dialog showing affected users
- Force uninstall on confirmation
- Proper error handling and feedback

## Files to Modify

- `internal/ui/modal/modules.go` — Add uninstall confirmation view
- `internal/app/app.go` — Route new message types

## Dependencies

**Internal:**
- Phase 1: `UninstallModule()`, `UninstallModuleForce()`, `UninstallResult` type
- Phase 3: Detail view, available modules context

**External:** None

## Implementation Notes

### New View State

```go
const (
    moduleViewList = iota
    moduleViewDetail
    moduleViewAvailable
    moduleViewConfirmUninstall  // NEW
)
```

### New Modal Fields

```go
type ModulesModal struct {
    // ... existing fields ...

    // Uninstall state
    uninstallResult *client.UninstallResult  // Stores affected users
    uninstallTarget string                    // Module being uninstalled
}
```

### New Message Types

```go
type ModuleUninstallResultMsg struct {
    Result *client.UninstallResult
    Error  error
}

type ModuleUninstalledMsg struct {
    Name  string
    Error error
}
```

### Uninstall Flow

1. User presses `[x]` on installed module in detail view
2. Call `UninstallModule(name)` which returns `UninstallResult`
3. If `ConfirmRequired` is true, show confirmation with affected users
4. If user confirms, call `UninstallModuleForce(name)`
5. On success, refresh lists and return to appropriate view

### Key Handling - Detail View

```go
case "x":
    if m.isAdmin && m.isCurrentModuleInstalled() {
        m.loading = true
        name := m.getCurrentModuleName()
        return m, m.uninstallModule(name)
    }
```

### Handle Uninstall Result

```go
case ModuleUninstallResultMsg:
    m.loading = false
    if msg.Error != nil {
        m.error = msg.Error.Error()
        return m, nil
    }

    if msg.Result.ConfirmRequired {
        // Show confirmation with affected users
        m.uninstallResult = msg.Result
        m.uninstallTarget = msg.Result.Module
        m.view = moduleViewConfirmUninstall
    } else {
        // Uninstall succeeded without needing confirmation
        return m, m.refreshAfterUninstall()
    }
```

### Key Handling - Confirm Uninstall View

```go
case "enter":
    // Force uninstall
    m.loading = true
    return m, m.uninstallModuleForce(m.uninstallTarget)
case "esc":
    // Cancel
    m.view = moduleViewDetail
    m.uninstallResult = nil
    m.uninstallTarget = ""
```

### Confirmation Dialog Rendering

```go
func (m *ModulesModal) viewConfirmUninstall() string {
    // Example output:
    //
    // Uninstall "recipes"?
    //
    // This module is enabled by 2 user(s):
    //   - alice
    //   - bob
    //
    // Uninstalling will disable it for these users.
    //
    // [Enter] Uninstall  [Esc] Cancel
}
```

**Show:**
- Module name being uninstalled
- Warning message
- List of affected users (from `UninstallResult.AffectedUsers`)
- Clear consequences ("Uninstalling will disable it for these users")
- Action hints

### After Uninstall Success

```go
case ModuleUninstalledMsg:
    m.loading = false
    if msg.Error != nil {
        m.error = msg.Error.Error()
        m.view = moduleViewDetail
        return m, nil
    }

    // Clear uninstall state
    m.uninstallResult = nil
    m.uninstallTarget = ""

    // Return to available view and refresh
    m.view = moduleViewAvailable
    return m, tea.Batch(m.loadModules(), m.loadAvailableModules())
```

### Edge Cases

- Module has no users: Uninstall succeeds immediately (no confirmation needed)
- API returns error: Show error, stay in detail view
- User is in the affected list: Still show confirmation (admin may be disabling for themselves too)

## Validation

How do we know this phase is complete?

- [ ] `[x]` from detail view initiates uninstall (admin only)
- [ ] `[x]` hidden for non-admin users
- [ ] `[x]` only works on installed modules
- [ ] Confirmation dialog shows when users are affected
- [ ] Affected users are listed by name
- [ ] `Enter` on confirmation performs force uninstall
- [ ] `Esc` on confirmation cancels and returns to detail
- [ ] Successful uninstall refreshes module lists
- [ ] Uninstall errors display properly
- [ ] Module with no users uninstalls without confirmation
