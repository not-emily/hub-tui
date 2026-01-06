# v4-config-types: Config-type-aware Integration Configuration

> **Status:** Planning complete | Last updated: 2026-01-05
>
> Phase files: [phases/](phases/)

## Overview

The hub-core API now exposes a `config_type` field on integrations that tells clients what configuration UI to render. Different config types require different UIs: `api_key` needs a simple form, while `llm` needs a two-section provider/profile management interface.

Currently, the hub-tui integrations modal treats all integrations the same (simple API key form), and has a separate `/llm` command for LLM profile management. This project consolidates everything into a single, config-type-aware integrations modal.

The implementation establishes an extensible pattern where adding future config types (like `oauth` or `email_pass`) requires minimal changes: one client file + one modal file + registration in the routing switch.

## Core Vision

- **Config-type driven UI**: The `config_type` field determines which configuration view renders, not the integration name
- **Extensible pattern**: Adding new config types requires minimal changes (new files + registration)
- **Consolidation over duplication**: One integrations modal handles all config types; remove standalone `/llm` command
- **Preserve existing UX**: Keep inline modals, form components, keyboard patterns users know

## Requirements

### Must Have
- Add `ConfigType` field to `Integration` struct in client
- Update LLM client methods to use new `/integrations/{name}/*` endpoints
- Config-type routing in integrations modal
- `api_key` config type: gate existing flow (profile selection → form)
- `llm` config type: provider accounts + profiles management in same view
- Indented provider list (provider headers non-selectable, accounts selectable)
- Continuous j/k navigation through providers and profiles sections
- Profile form with cascading dropdowns (provider → account → model)
- Model pagination in profile form
- Remove `/llm` command

### Nice to Have
- Provider account testing before saving
- Inline help text explaining provider→account→profile relationship
- Keyboard shortcut to jump between Providers and Profiles sections

### Out of Scope
- `email_pass` config type — future work
- `oauth` config type — future work (will need browser redirect flow)
- Migration of existing LLM configs — hub-core handles server-side

## Constraints

- **Tech stack**: Go, Bubble Tea, existing form component
- **UX**: Must use inline modal pattern (not separate screens)
- **Order**: Implement `api_key` before `llm` (validates extensible pattern)
- **API**: Client must work with new hub-core endpoints (`/integrations/llm/*`)

## Success Metrics

- User can configure Notion integration (api_key type) through integrations modal
- User can add provider accounts and create LLM profiles through integrations modal
- `/llm` command removed; `/integrations` is the single entry point
- Adding a new config type requires: 1 client file + 1 modal file + registration

## Architecture Decisions

### 1. Config-type based file organization
**Choice:** Separate files per config type in both client and modal layers
**Rationale:** Groups related code together, makes it easy to add/remove config types
**Trade-offs:** More files, but better organization and maintainability

### 2. Continuous navigation in LLM config view
**Choice:** Single continuous list with j/k navigation through both providers and profiles sections
**Rationale:** Simpler mental model than tabbed or multi-level navigation
**Trade-offs:** May get unwieldy with many items; can iterate to tabs/numbers if needed

### 3. Indented list with non-selectable headers
**Choice:** Provider names are visual headers only; cursor only lands on accounts
**Rationale:** Matches the data model (you operate on accounts, not providers)
**Trade-offs:** Slightly unusual pattern, but intuitive once understood

### 4. Integration name passed to all config-type methods
**Choice:** LLM methods take `integration string` parameter, not hardcoded to "llm"
**Rationale:** Multiple integrations can share a config type; future-proofs the API
**Trade-offs:** Slightly more verbose API, but more flexible

## Project Structure

```
internal/
├── client/
│   ├── client.go              # HTTP client, auth (no change)
│   ├── integrations.go        # Generic + api_key type methods
│   │                          # Types: Integration (with ConfigType)
│   └── integrations_llm.go    # NEW: llm config type methods + types
│                              # Types: ProviderAccount, LLMProfile, etc.
├── ui/
│   ├── modal/
│   │   ├── integrations.go       # Modal struct, list view, routing, api_key views
│   │   └── integrations_llm.go   # NEW: llm config type views
│   └── ...
└── app/
    └── app.go                 # Remove /llm command handler
```

### Key Files
- `client/integrations.go` — `Integration` type with `ConfigType`, generic methods, `ConfigureAPIKey`
- `client/integrations_llm.go` — `ProviderAccount`, `LLMProfile` types, provider/profile CRUD methods
- `modal/integrations.go` — Modal struct, list view, config type routing, api_key form views
- `modal/integrations_llm.go` — LLM config view, provider form, profile form

### Files to Delete
- `client/llm.go` — Replaced by `integrations_llm.go`
- `modal/llm.go` — Logic moves to `integrations_llm.go`

## Core Interfaces

### Client Methods

```go
// Generic (integrations.go)
func (c *Client) ListIntegrations() ([]Integration, error)
func (c *Client) TestIntegration(name string) error
func (c *Client) ConfigureAPIKey(integration, profile string, config map[string]string) error

// LLM config type (integrations_llm.go)
func (c *Client) ListLLMProviders(integration string) ([]ProviderAccount, error)
func (c *Client) ListAvailableLLMProviders(integration string) ([]AvailableProvider, error)
func (c *Client) AddLLMProvider(integration string, req AddProviderRequest) error
func (c *Client) DeleteLLMProvider(integration, provider, account string) error
func (c *Client) ListLLMProfiles(integration string) (*LLMProfileList, error)
func (c *Client) CreateLLMProfile(integration string, req CreateProfileRequest) error
func (c *Client) UpdateLLMProfile(integration string, req UpdateProfileRequest) error
func (c *Client) DeleteLLMProfile(integration, profile string) error
func (c *Client) TestLLMProfile(integration, profile string) (*LLMTestResult, error)
func (c *Client) SetDefaultLLMProfile(integration, profile string) error
```

### Key Types

```go
type Integration struct {
    Name           string
    DisplayName    string
    Type           string   // "api", "cli", "mcp"
    ConfigType     string   // "api_key", "llm", "oauth", etc.
    Configured     bool
    Profiles       []string // api_key type
    DefaultProfile string   // api_key type
    Fields         []string // required config fields
    ProviderCount  int      // llm type summary
    ProfileCount   int      // llm type summary
}

type ProviderAccount struct {
    Provider    string   // "openai"
    DisplayName string   // "OpenAI"
    Accounts    []string // ["default", "work"]
}

type LLMProfile struct {
    Name      string
    Provider  string
    Account   string
    Model     string
    IsDefault bool
}
```

### Modal State Integration

Each config type implements in its `integrations_{type}.go` file:

```go
// Entry point
func (m *IntegrationsModal) enterLLMConfig(integration Integration) (Modal, tea.Cmd)

// Update handler
func (m *IntegrationsModal) updateLLM(msg tea.Msg) (Modal, tea.Cmd)

// View renderer
func (m *IntegrationsModal) viewLLM() string
```

### Key Bindings

**LLM config view:**
| Key | Action |
|-----|--------|
| `j/↓` | Move down (continuous through providers + profiles) |
| `k/↑` | Move up |
| `a` | Add provider account |
| `n` | New profile |
| `Enter` | Edit selected item |
| `d` | Delete selected (double-press confirm) |
| `t` | Test selected profile |
| `s` | Set selected profile as default |
| `r` | Refresh |
| `Esc` | Back to list |

## Implementation Phases

| Phase | Name | Scope | Depends On | Key Outputs |
|-------|------|-------|------------|-------------|
| 1 | Client Layer Updates | Types + LLM methods | — | `Integration` with `ConfigType`, `integrations_llm.go` |
| 2 | Modal Routing + api_key | Config type switching | Phase 1 | api_key integrations work through routing |
| 3 | LLM Config View | Providers + profiles list | Phase 2 | Viewable LLM config with navigation |
| 4 | LLM Provider Management | Add/delete providers | Phase 3 | Full provider CRUD |
| 5.1 | LLM Profile Form | Cascading dropdowns, create/edit | Phase 4 | Working profile form |
| 5.2 | LLM Profile Operations | Delete, test, set default | Phase 5.1 | Full profile operations |
| 6 | Cleanup & Integration | Remove old code | Phase 5.2 | Clean codebase |

### Critical Path

All phases are sequential:
```
Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5.1 → Phase 5.2 → Phase 6
```

### Phase Details
- [Phase 1: Client Layer Updates](phases/phase-1.md)
- [Phase 2: Modal Routing + api_key](phases/phase-2.md)
- [Phase 3: LLM Config View](phases/phase-3.md)
- [Phase 4: LLM Provider Management](phases/phase-4.md)
- [Phase 5.1: LLM Profile Form](phases/phase-5-1.md)
- [Phase 5.2: LLM Profile Operations](phases/phase-5-2.md)
- [Phase 6: Cleanup & Integration](phases/phase-6.md)

## Tech Stack

| Category | Choice | Notes |
|----------|--------|-------|
| Language | Go | Existing |
| TUI Framework | Bubble Tea | Existing |
| Styling | Lip Gloss | Existing |
| Form Component | components/form.go | Existing, supports cascading dropdowns |

## Future Considerations

### Adding a New Config Type

When hub-core adds a new config type (e.g., `oauth`):

**1. Client layer** — Create `internal/client/integrations_oauth.go`:
```go
// Types
type OAuthToken struct { ... }

// Methods
func (c *Client) InitiateOAuth(integration string) (authURL string, error)
func (c *Client) CompleteOAuth(integration, code string) error
```

**2. Modal layer** — Create `internal/ui/modal/integrations_oauth.go`:
```go
// View constants (offset to avoid collision)
const (
    viewConfigOAuth integrationsView = iota + 200
    viewOAuthWaiting
)

// Entry, update, view methods
func (m *IntegrationsModal) enterOAuthConfig(integration Integration) (Modal, tea.Cmd)
func (m *IntegrationsModal) updateOAuth(msg tea.Msg) (Modal, tea.Cmd)
func (m *IntegrationsModal) viewOAuth() string
```

**3. Register in main modal** — Add cases to switch statements in `integrations.go`:
```go
// In enterConfigMode()
case "oauth":
    return m.enterOAuthConfig(integration)

// In Update()
case viewConfigOAuth, viewOAuthWaiting:
    return m.updateOAuth(msg)

// In View()
case viewConfigOAuth, viewOAuthWaiting:
    return m.viewOAuth()
```
