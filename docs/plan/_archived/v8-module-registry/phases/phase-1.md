# Phase 1: Client Layer

> **Depends on:** None
> **Enables:** Phase 2 (Modal Refactor)
>
> See: [Full Plan](../plan.md)

## Goal

Add HTTP client methods for module registry browsing, install, uninstall, and update operations.

## Key Deliverables

- `AvailableModule` struct for registry data
- `UninstallResult` struct for uninstall response (with affected users)
- `UpdateResult` struct for update response
- `ListAvailableModules()` method
- `InstallModule()` method
- `UninstallModule()` and `UninstallModuleForce()` methods
- `UpdateModule()` method

## Files to Modify

- `internal/client/modules.go` — Add new types and methods

## Dependencies

**Internal:** None

**External:** None (uses existing net/http patterns)

## Implementation Notes

### API Endpoints

From hub-core API:

```
GET  /modules/available           → []AvailableModule
POST /modules/install             → {name: string}
DELETE /modules/{name}            → UninstallResult
DELETE /modules/{name}?force=true → success
POST /modules/{name}/update       → UpdateResult
```

### Type Definitions

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
    CurrentVersion  string `json:"current_version"` // Present when already_latest=true
}
```

### Method Signatures

```go
// ListAvailableModules fetches modules from the registry (admin only).
func (c *Client) ListAvailableModules() ([]AvailableModule, error)

// InstallModule installs a module from the registry (admin only).
func (c *Client) InstallModule(name string) error

// UninstallModule attempts to uninstall a module.
// Returns UninstallResult which may indicate users are affected.
func (c *Client) UninstallModule(name string) (*UninstallResult, error)

// UninstallModuleForce uninstalls a module even if users have it enabled.
func (c *Client) UninstallModuleForce(name string) error

// UpdateModule updates a module to the latest version (admin only).
func (c *Client) UpdateModule(name string) (*UpdateResult, error)
```

### Install Request Body

```go
type installModuleRequest struct {
    Name string `json:"name"`
}
```

### Error Handling

- 403 responses indicate non-admin user (should not happen if UI gates properly)
- UninstallModule returns the full result even on "failure" so UI can show affected users
- Follow existing `parseError()` pattern for other errors

## Validation

How do we know this phase is complete?

- [ ] `AvailableModule` struct matches API response
- [ ] `UninstallResult` struct captures affected users
- [ ] `UpdateResult` struct handles both update and already-latest cases
- [ ] `ListAvailableModules()` returns registry modules
- [ ] `InstallModule()` successfully installs a module
- [ ] `UninstallModule()` returns result with affected users when applicable
- [ ] `UninstallModuleForce()` bypasses user check
- [ ] `UpdateModule()` returns version info
- [ ] All methods handle errors following existing patterns
