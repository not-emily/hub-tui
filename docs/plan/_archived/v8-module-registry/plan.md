# v8-module-registry: Module Registry UI

> **Status:** Planning complete | Last updated: 2026-01-27
>
> Phase files: [phases/](phases/)

## Overview

Add module registry browsing and management capabilities to the hub-tui `/modules` modal. Currently, users can only view installed modules and toggle enable/disable. This feature adds admin functionality to browse the module registry, install new modules, uninstall modules (with user impact warnings), and update modules to newer versions.

The feature builds on the existing ModulesModal pattern and reuses established UI patterns like the `Confirmation` component for double-press confirmations and view state machines.

## Core Vision

- **Admin-aware UI**: Show additional functionality to admins without cluttering the user experience
- **Safety first**: Destructive actions (disable, uninstall) require confirmation; uninstall shows affected users
- **Consistent patterns**: Reuse existing modal patterns (view states, confirmation component, message routing)
- **Progressive disclosure**: List → Detail view flow for deeper information

## Requirements

### Must Have
- Users can view installed modules with enable/disable status
- Users can enable modules (single press)
- Users can disable modules (double-press confirmation)
- Users can view module detail (name, description, version, tools)
- Admins can browse available modules from registry (`[b]` hotkey)
- Admins can install modules from registry
- Admins can uninstall modules with confirmation showing affected users
- Admins can update modules when updates are available
- Update available indicator in list view

### Nice to Have
- Search/filter in available modules view
- Batch operations (install/update multiple)

### Out of Scope
- Module configuration (modules don't have user-configurable settings yet)
- Local module install (`/modules/install-local` - dev-only feature)
- Module dependencies visualization

## Constraints

- **Tech stack**: Go, Bubble Tea, Lip Gloss (existing stack)
- **Patterns**: Must follow existing Modal interface, tea.Cmd, message routing
- **Keyboard-only**: No mouse support
- **API**: All endpoints already exist in hub-core

## Success Metrics

- Users can enable/disable modules from list or detail view
- Admins can browse registry and install new modules
- Admins can uninstall modules and see affected users before confirming
- Admins can update modules when updates are available
- No regressions in existing module enable/disable functionality

## Architecture Decisions

### 1. Single Modal with View States
**Choice:** Expand ModulesModal with view states (like AssistantsModal)
**Rationale:** Keeps context, easier navigation, consistent with other modals
**Trade-offs:** Single file may get larger, but keeps related code together

### 2. Shared Detail View
**Choice:** One detail view that adapts based on module state and user role
**Rationale:** Reduces duplication, consistent UX whether coming from list or available view
**Trade-offs:** Detail view logic needs to handle multiple contexts

### 3. Double-Press for Disable, Dialog for Uninstall
**Choice:** Use `Confirmation` component for disable; full dialog for uninstall
**Rationale:** Disable is reversible and quick; uninstall needs to show affected users
**Trade-offs:** Two different confirmation patterns, but matches the severity of each action

### 4. Pass isAdmin to Modal
**Choice:** Follow IntegrationsModal pattern of passing isAdmin flag
**Rationale:** Proven pattern, keeps admin check centralized in app.go
**Trade-offs:** None significant

## Project Structure

```
internal/
├── client/
│   └── modules.go          # Expand with registry/install/uninstall/update methods
├── ui/modal/
│   └── modules.go          # Expand with view states, detail view, admin features
└── app/
    └── app.go              # Pass isAdmin to ModulesModal, route new messages
```

### Key Files
- `internal/client/modules.go` — HTTP client methods for module API
- `internal/ui/modal/modules.go` — Main modal implementation
- `internal/ui/components/confirm.go` — Reusable double-press confirmation (existing)

## Core Interfaces

### View States
```
moduleViewList (default - installed modules)
  └── [Enter] → moduleViewDetail
  └── [b] (admin) → moduleViewAvailable

moduleViewDetail
  ├── [Space] → toggle enable/disable
  ├── [i] (admin, not installed) → install
  ├── [x] (admin, installed) → moduleViewConfirmUninstall
  └── [u] (admin, update available) → update

moduleViewAvailable (admin only)
  └── [Enter] → moduleViewDetail

moduleViewConfirmUninstall
  └── shows affected users, [Enter] to force uninstall
```

### Key Bindings

| Context | Key | Action | Who |
|---------|-----|--------|-----|
| List | `j/k` | Navigate | All |
| List | `Space` | Toggle enable/disable (double-press for disable) | All |
| List | `Enter` | View detail | All |
| List | `r` | Refresh | All |
| List | `b` | Browse available | Admin |
| List | `u` | Update selected (if available) | Admin |
| Available | `j/k` | Navigate | Admin |
| Available | `Enter` | View detail | Admin |
| Available | `Esc` | Back to list | Admin |
| Detail | `Space` | Toggle enable/disable | All (installed) |
| Detail | `i` | Install | Admin (not installed) |
| Detail | `x` | Uninstall | Admin (installed) |
| Detail | `u` | Update | Admin (update available) |
| Detail | `Esc` | Back | All |
| Confirm Uninstall | `Enter` | Confirm (force) | Admin |
| Confirm Uninstall | `Esc` | Cancel | Admin |

### Client Methods to Add

```go
// Registry browsing (admin)
ListAvailableModules() ([]AvailableModule, error)

// Install/Uninstall (admin)
InstallModule(name string) error
UninstallModule(name string) (*UninstallResult, error)
UninstallModuleForce(name string) error

// Update (admin)
UpdateModule(name string) (*UpdateResult, error)
```

### New Types

```go
// AvailableModule from GET /modules/available
type AvailableModule struct {
    Name             string   `json:"name"`
    Version          string   `json:"version"`
    ReleaseTag       string   `json:"release_tag"`
    Description      string   `json:"description"`
    Keywords         []string `json:"keywords"`
    Installed        bool     `json:"installed"`
    InstalledVersion string   `json:"installed_version"`
    UpdateAvailable  bool     `json:"update_available"`
}

// UninstallResult for DELETE /modules/{name}
type UninstallResult struct {
    Success         bool     `json:"success"`
    Module          string   `json:"module"`
    Error           string   `json:"error"`
    Message         string   `json:"message"`
    AffectedUsers   []string `json:"affected_users"`
    ConfirmRequired bool     `json:"confirm_required"`
}

// UpdateResult for POST /modules/{name}/update
type UpdateResult struct {
    Success         bool   `json:"success"`
    Module          string `json:"module"`
    PreviousVersion string `json:"previous_version"`
    NewVersion      string `json:"new_version"`
    AlreadyLatest   bool   `json:"already_latest"`
}
```

### Message Types

```go
// Existing
ModulesLoadedMsg
ModuleToggledMsg

// New
AvailableModulesLoadedMsg { Modules []AvailableModule; Error error }
ModuleInstalledMsg { Name string; Error error }
ModuleUninstallResultMsg { Result *UninstallResult; Error error }
ModuleUninstalledMsg { Name string; Error error }
ModuleUpdatedMsg { Result *UpdateResult; Error error }
```

## Implementation Phases

| Phase | Name | Scope | Depends On | Key Outputs |
|-------|------|-------|------------|-------------|
| 1 | Client Layer | Add registry/install/uninstall/update methods | — | `modules.go` client expanded |
| 2 | Modal Refactor | Add isAdmin, view states, detail view | Phase 1 | Basic navigation + detail |
| 3 | Admin: Browse & Install | Available modules view, install action | Phase 2 | Registry browsing |
| 4 | Admin: Uninstall | Uninstall with confirmation, affected users | Phase 3 | Full uninstall flow |
| 5 | Admin: Update | Update indicator in list, update action | Phase 4 | Complete feature |

### Critical Path
All phases are sequential. Each builds on the previous.

### Phase Details
- [Phase 1: Client Layer](phases/phase-1.md)
- [Phase 2: Modal Refactor](phases/phase-2.md)
- [Phase 3: Admin Browse & Install](phases/phase-3.md)
- [Phase 4: Admin Uninstall](phases/phase-4.md)
- [Phase 5: Admin Update](phases/phase-5.md)

## Tech Stack

| Category | Choice | Notes |
|----------|--------|-------|
| Language | Go | Existing |
| TUI Framework | Bubble Tea | Existing |
| Styling | Lip Gloss | Existing |
| HTTP Client | net/http | Existing patterns in client/ |

## Future Considerations

- **Module search**: Filter available modules by name/keyword
- **Batch operations**: Install or update multiple modules at once
- **Module dependencies**: Show what a module depends on or what depends on it
- **Local install UI**: Support `/modules/install-local` for development
