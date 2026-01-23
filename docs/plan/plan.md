# v6-dependency-management: Admin UI for CLI Dependency Management

> **Status:** Planning complete | Last updated: 2026-01-22
>
> Phase files: [phases/](phases/)

## Overview

Hub-core now supports client-first dependency management for CLI integrations. Previously, CLI tools required by integrations (like `sage` for AI) had to be manually installed via SSH on the server. Admins had no visibility into what's installed, missing, or outdated, and couldn't fix dependency issues without server access.

This project adds admin UI to hub-tui for managing CLI integration dependencies directly from the terminal interface. Admins can view dependency status, install missing tools, update outdated versions, and apply hub-core self-updates—all without leaving the TUI or requiring SSH access.

This unblocks AI integration adoption and provides a foundation for future CLI-based integrations.

## Core Vision

- **Admin-first, user-aware** — Dependency management is admin-only, but regular users see clear feedback when dependencies are missing ("Contact admin to install Sage")
- **Integration-aware** — Check dependencies at the point of need (when configuring an integration), not just in a separate admin section
- **Progressive disclosure** — Surface dependency issues when they matter, with clear paths to resolution
- **Resilient to failure** — Installation can fail for many reasons (network, permissions, platform support); handle errors gracefully with clear messaging

## Requirements

### Must Have
- User info API to check admin status
- Admin status cached on login and available to all UI components
- Reusable tab component for modal views
- Integrations modal with tabs: "Integrations" and "Dependencies"
- Dependencies tab showing: Integration | Tool | Status | Current Version | Required Version | Actions
- Install flow for missing dependencies (admin only)
- Update flow for outdated dependencies (admin only)
- "Contact admin" messaging for non-admin users
- Dependency check before entering integration config
- Block integration config until dependencies are satisfied
- Success/error feedback after install/update operations
- Hub self-update check and apply from Dependencies tab
- Error handling for all failure scenarios (network, permissions, unsupported platform, private repo)

### Nice to Have
- Installation progress tracking (API doesn't currently support)
- Bulk dependency update
- Auto-check dependencies on startup with admin notification
- Read-only dependency status for regular users

### Out of Scope
- **Dependency rollback** — API doesn't support uninstalling or downgrading
- **Custom version selection UI** — We'll install "latest" by default; API supports version param but no picker UI
- **Dependency logs/history** — No audit trail of who installed what when
- **Platform compatibility pre-check** — Server handles this; we show error if it fails

## Constraints

- **API compatibility** — Must work with existing hub-core `/admin/dependencies` and `/admin/hub/updates` endpoints
- **Admin-only access** — All dependency operations require admin token (user in `hubadmin` group)
- **No progress tracking** — Install is synchronous API call, can take 10-30 seconds with no progress updates
- **Modal-based UI** — Consistent with existing TUI patterns (modules, integrations, settings modals)
- **Hub update depends on public repo** — Self-update only works when hub-core repo is public (currently private but will be public soon)

## Success Metrics

- Admin can view all CLI dependencies and their status from Dependencies tab
- Admin can install sage CLI from TUI without SSH access
- AI integration setup blocked until sage is installed
- Clear error messages when installation fails (network, permissions, platform)
- Non-admin users see helpful "Contact admin" messaging when dependencies are missing
- Admin can check for and apply hub-core updates from Dependencies tab
- Hub update gracefully handles 404 when repo is private

## Architecture Decisions

### 1. Reusable Tab Component
**Choice:** Generic `components.Tabs` that manages tab state and rendering, parent owns content switching
**Rationale:** Simple state machine—tab component only handles tab bar rendering and selection. Parent modal owns content rendering for each tab, providing maximum flexibility. Can be reused in any modal (settings, modules, etc.).
**Trade-offs:** Parent modal must handle key routing (Tab key to switch tabs) and content switching logic, but this provides clearer separation of concerns.

### 2. Admin Status Caching
**Choice:** Cache `is_admin` in root app model, fetch on login via `/me` endpoint
**Rationale:** Admin status rarely changes during a session. Avoids repeated API calls to check permissions. Available to all UI components via app model.
**Trade-offs:** Extra API call on login (minimal cost). Stale if admin status changes mid-session (acceptable, requires re-login).

### 3. Dependency Check Flow
**Choice:** Check dependencies when entering integration config, not on modal open
**Rationale:** Just-in-time checking—only check when needed. Keeps Dependencies tab separate (manual refresh). Consistent with existing pattern (profile form checks models on provider selection).
**Trade-offs:** User might select integration and see "dependency required" blocking message (acceptable, clearer than silent failure).

### 4. Dependencies Tab in Integrations Modal
**Choice:** Add Dependencies as a second tab in the existing integrations modal, not a standalone modal
**Rationale:** Logical grouping—dependencies are tightly coupled to integrations. Reuses existing modal infrastructure. Tabs only visible in list view; config views remain full-screen.
**Trade-offs:** Integration modal grows in complexity (acceptable, still manageable with clear view switching).

### 5. Tab Switching Keys
**Choice:** Tab/Shift+Tab to switch between tabs in list view
**Rationale:** Standard convention across many UIs. Tab is already a "switch focus" mental model. When in config forms (LLM, API key), tabs aren't visible anyway, so Tab continues to work for form field navigation.
**Trade-offs:** No conflict since tabs only appear in list view. Could add arrow keys as secondary shortcuts later if needed.

### 6. Integration Modal Refactoring
**Choice:** Add tab state to existing IntegrationsModal, switch content based on active tab
**Rationale:** Minimal changes to existing structure. Tabs only visible in list view. Config views remain full-screen (better UX for multi-step forms).
**Trade-offs:** Modal state grows with dependencies list and tab component (acceptable, still manageable).

## Project Structure

```
internal/
├── client/
│   ├── auth.go             # UPDATED - Add GetMe() method for user info
│   ├── dependencies.go     # NEW - All dependency management API methods
│   ├── integrations.go     # Existing - no changes
│   └── integrations_llm.go # UPDATED - Check dependencies before LLM config
├── app/
│   └── app.go              # UPDATED - Cache is_admin, route dependency messages
└── ui/
    ├── components/
    │   ├── tabs.go         # NEW - Reusable tab component for modals
    │   ├── form.go         # Existing - reuse for confirmations
    │   ├── list.go         # Existing - reuse for dependencies table
    │   └── confirmation.go # Existing - reuse for update confirmations
    └── modal/
        ├── integrations.go     # UPDATED - Add tabs, dependencies view, install/update flows
        └── integrations_llm.go # UPDATED - Check dependencies before entering config
```

### Key Files

- `internal/client/dependencies.go` — All dependency management API methods: GetDependencies, InstallDependency, CheckDependencyUpdates, GetHubUpdates, ApplyHubUpdate
- `internal/ui/components/tabs.go` — Reusable tab component for modal views
- `internal/client/auth.go` — Add GetMe() to fetch user info including is_admin flag
- `internal/app/app.go` — Cache is_admin on login, route UserInfoLoadedMsg and dependency messages
- `internal/ui/modal/integrations.go` — Add tabs component, Dependencies tab with table view and install/update actions
- `internal/ui/modal/integrations_llm.go` — Check dependencies before entering LLM config, block if unsatisfied

## Core Interfaces

### Client API - User Info

```go
// client/auth.go
type UserInfo struct {
    Username       string   `json:"username"`
    HomeDir        string   `json:"home_dir"`
    HubDir         string   `json:"hub_dir"`
    IsAdmin        bool     `json:"is_admin"`
    Groups         []string `json:"groups"`
    EnabledModules int      `json:"enabled_modules"`
    Workflows      int      `json:"workflows"`
    Assistants     int      `json:"assistants"`
}

func (c *Client) GetMe() (*UserInfo, error)
```

### Client API - Dependency Management

```go
// client/dependencies.go
type Dependency struct {
    Integration    string `json:"integration"`
    Name           string `json:"name"`
    Installed      bool   `json:"installed"`
    CurrentVersion string `json:"current_version"`
    MinVersion     string `json:"min_version"`
    LatestVersion  string `json:"latest_version"`
    UpToDate       bool   `json:"up_to_date"`
}

type DependencyUpdate struct {
    Integration    string `json:"integration"`
    Name           string `json:"name"`
    CurrentVersion string `json:"current_version"`
    LatestVersion  string `json:"latest_version"`
}

type DependencyInstallRequest struct {
    Version string `json:"version"`
}

type DependencyInstallResponse struct {
    Success bool       `json:"success"`
    Status  Dependency `json:"status"`
}

type DependencyUpdatesResponse struct {
    UpdatesAvailable bool               `json:"updates_available"`
    Updates          []DependencyUpdate `json:"updates"`
}

type HubUpdateInfo struct {
    CurrentVersion  string `json:"current_version"`
    LatestVersion   string `json:"latest_version"`
    UpdateAvailable bool   `json:"update_available"`
    ReleaseURL      string `json:"release_url"`
    DownloadURL     string `json:"download_url"`
}

type HubUpdateResponse struct {
    Success bool   `json:"success"`
    Message string `json:"message"`
}

func (c *Client) GetDependencies() ([]Dependency, error)
func (c *Client) CheckDependencyUpdates() (*DependencyUpdatesResponse, error)
func (c *Client) InstallDependency(name, version string) (*DependencyInstallResponse, error)
func (c *Client) GetHubUpdates() (*HubUpdateInfo, error)
func (c *Client) ApplyHubUpdate() (*HubUpdateResponse, error)
```

### Tab Component

```go
// components/tabs.go
type Tabs struct {
    tabs        []string
    activeIndex int
    width       int
}

func NewTabs(labels []string) *Tabs
func (t *Tabs) SetActive(index int)
func (t *Tabs) ActiveIndex() int
func (t *Tabs) Next()
func (t *Tabs) Previous()
func (t *Tabs) SetWidth(width int)
func (t *Tabs) View() string
```

**Contract:**
- Parent modal owns tabs instance
- Parent handles Tab/Shift+Tab key events, calls Next()/Previous()
- Parent renders tabs.View() at top of modal
- Parent switches content based on tabs.ActiveIndex()

### Message Types

```go
// App-level messages (app/app.go)
type UserInfoLoadedMsg struct {
    UserInfo *client.UserInfo
    Err      error
}

// Modal messages (modal/integrations.go)
type DependenciesLoadedMsg struct {
    Dependencies []client.Dependency
    Err          error
}

type DependencyInstalledMsg struct {
    Name   string
    Status *client.Dependency
    Err    error
}

type DependencyUpdatesCheckedMsg struct {
    Updates *client.DependencyUpdatesResponse
    Err     error
}

type HubUpdatesLoadedMsg struct {
    UpdateInfo *client.HubUpdateInfo
    Err        error
}

type HubUpdateAppliedMsg struct {
    Success bool
    Message string
    Err     error
}
```

### Key Bindings

**Integrations modal (list view with tabs):**
- `Tab` — Next tab
- `Shift+Tab` — Previous tab
- `↑/↓` — Navigate list items
- `Enter` — Select integration/dependency
- `r` — Refresh dependencies (when on Dependencies tab)
- `Esc` — Close modal

**Integrations modal (config views - no tabs visible):**
- `Tab` — Next form field (existing behavior)
- `↑/↓` — Navigate form fields (existing behavior)
- `Esc` — Back to list

## Implementation Phases

| Phase | Name | Scope | Depends On | Key Outputs |
|-------|------|-------|------------|-------------|
| 1 | User Info & Admin Status | GetMe() API, cache is_admin | — | Admin status available to all UI |
| 2 | Tab Component | Reusable tabs for modals | — | components.Tabs ready for use |
| 3 | Dependency Client Layer | All dependency API methods | — | Client can fetch/install/update deps |
| 4 | Dependencies Tab UI | Tab integration, table view, install/update actions | Phases 1, 2, 3 | Working dependencies management in integrations modal |
| 5 | Integration Config Check | Dependency verification before config | Phases 1, 3, 4 | Integration config blocked until deps satisfied |
| 6 | Hub Self-Update | Hub version check and update UI | Phase 3 | Self-update from dependencies tab |

### Critical Path

**Sequential dependencies:**
- Phases 1, 2, 3 are independent (can be parallelized)
- Phase 4 depends on Phases 1, 2, 3 (needs admin status, tabs component, API methods)
- Phase 5 depends on Phases 1, 3, 4 (reuses dependency check and install logic)
- Phase 6 depends on Phase 3 (needs API methods)

**Recommended order:** 1→2→3→4→5→6 (or parallelize 1,2,3 then proceed sequentially)

### Phase Details
- [Phase 1: User Info & Admin Status](phases/phase-1.md)
- [Phase 2: Tab Component](phases/phase-2.md)
- [Phase 3: Dependency Client Layer](phases/phase-3.md)
- [Phase 4: Dependencies Tab UI](phases/phase-4.md)
- [Phase 5: Integration Config Check](phases/phase-5.md)
- [Phase 6: Hub Self-Update](phases/phase-6.md)

## Tech Stack

| Category | Choice | Notes |
|----------|--------|-------|
| Language | Go | Existing |
| TUI Framework | Bubble Tea | Existing, Elm architecture |
| HTTP Client | stdlib net/http | Existing pattern in client/ |
| Styling | Lip Gloss | Existing |
| Components | Custom | Reusing form, list, confirmation; adding tabs |

## Future Considerations

Items explicitly deferred from scope but architecturally supported:

- **Installation progress tracking** — Would require hub-core API changes to support streaming progress or polling status
- **Bulk dependency update** — "Update All" button for multiple outdated dependencies
- **Auto-check on startup** — Show admin notification "1 dependency needs installation"
- **Dependency audit logs** — Track who installed what and when
- **Custom version picker** — UI for selecting specific versions instead of always using "latest"
- **Read-only dependency status for users** — Let non-admins see what's installed without install buttons
