# Phase 1: Client Layer Updates

> **Depends on:** None
> **Enables:** Phase 2 (Modal Routing)
>
> See: [Full Plan](../plan.md)

## Goal

Update client types and methods to match the new hub-core API structure for config-type-aware integrations.

## Key Deliverables

- `Integration` struct updated with `ConfigType` field and LLM summary fields
- New `integrations_llm.go` with all LLM config type methods and types
- All methods use new `/integrations/{name}/*` endpoint structure

## Files to Create

- `internal/client/integrations_llm.go` — LLM config type client methods and types

## Files to Modify

- `internal/client/integrations.go` — Update `Integration` struct, add `ConfigureAPIKey` method

## Dependencies

**Internal:** None

**External:** None (uses existing net/http client)

## Implementation Notes

### Integration Type Updates

Update the `Integration` struct to include:

```go
type Integration struct {
    Name           string   `json:"name"`
    DisplayName    string   `json:"display_name"`
    Type           string   `json:"type"`        // "api", "cli", "mcp"
    ConfigType     string   `json:"config_type"` // "api_key", "llm", etc.
    Configured     bool     `json:"configured"`
    Profiles       []string `json:"profiles,omitempty"`
    DefaultProfile string   `json:"default_profile,omitempty"`
    Fields         []string `json:"fields,omitempty"`
    // LLM type summary fields (for list display)
    ProviderCount  int      `json:"provider_count,omitempty"`
    ProfileCount   int      `json:"profile_count,omitempty"`
}
```

### LLM Types (integrations_llm.go)

```go
type ProviderAccount struct {
    Provider    string   `json:"provider"`
    DisplayName string   `json:"display_name"`
    Accounts    []string `json:"accounts"`
}

type AvailableProvider struct {
    Name        string `json:"name"`
    DisplayName string `json:"display_name"`
}

type LLMProfile struct {
    Name      string `json:"name"`
    Provider  string `json:"provider"`
    Account   string `json:"account"`
    Model     string `json:"model"`
    IsDefault bool   `json:"is_default"`
}

type LLMProfileList struct {
    Profiles []LLMProfile `json:"profiles"`
}

type AddProviderRequest struct {
    Provider string `json:"provider"`
    Account  string `json:"account"`
    APIKey   string `json:"api_key"`
}

type CreateProfileRequest struct {
    Name     string `json:"name"`
    Provider string `json:"provider"`
    Account  string `json:"account"`
    Model    string `json:"model"`
}

type UpdateProfileRequest struct {
    Name     string `json:"name,omitempty"` // for rename
    Provider string `json:"provider"`
    Account  string `json:"account"`
    Model    string `json:"model"`
}

type LLMTestResult struct {
    Success   bool   `json:"success"`
    Model     string `json:"model"`
    LatencyMs int    `json:"latency_ms"`
    Error     string `json:"error,omitempty"`
}
```

### LLM Methods (integrations_llm.go)

All methods take `integration string` as first parameter for future-proofing:

```go
// Providers
func (c *Client) ListLLMProviders(integration string) ([]ProviderAccount, error)
// GET /integrations/{integration}/providers

func (c *Client) ListAvailableLLMProviders(integration string) ([]AvailableProvider, error)
// GET /integrations/{integration}/providers/available

func (c *Client) AddLLMProvider(integration string, req AddProviderRequest) error
// POST /integrations/{integration}/providers

func (c *Client) DeleteLLMProvider(integration, provider, account string) error
// DELETE /integrations/{integration}/providers/{provider}/{account}

// Profiles
func (c *Client) ListLLMProfiles(integration string) (*LLMProfileList, error)
// GET /integrations/{integration}/profiles

func (c *Client) CreateLLMProfile(integration string, req CreateProfileRequest) error
// POST /integrations/{integration}/profiles

func (c *Client) UpdateLLMProfile(integration, name string, req UpdateProfileRequest) error
// PUT /integrations/{integration}/profiles/{name} (if update exists, else use Create)

func (c *Client) DeleteLLMProfile(integration, profile string) error
// DELETE /integrations/{integration}/profiles/{profile}

func (c *Client) TestLLMProfile(integration, profile string) (*LLMTestResult, error)
// POST /integrations/{integration}/profiles/{profile}/test

func (c *Client) SetDefaultLLMProfile(integration, profile string) error
// PUT /integrations/{integration}/profiles/set-default with {"profile": "..."}
```

### API Endpoint Reference

Refer to hub-core API docs at `../hub-core/docs/api/README.md` for exact request/response formats.

Key endpoints:
- `GET /integrations` — Returns integrations with `config_type` field
- `GET /integrations/llm/providers` — List configured providers
- `GET /integrations/llm/providers/available` — List all supported providers
- `POST /integrations/llm/providers` — Add provider account
- `DELETE /integrations/llm/providers/{provider}/{account}` — Remove provider account
- `GET /integrations/llm/profiles` — List LLM profiles
- `POST /integrations/llm/profiles` — Create profile
- `DELETE /integrations/llm/profiles/{name}` — Delete profile
- `POST /integrations/llm/profiles/{name}/test` — Test profile
- `PUT /integrations/llm/profiles/set-default` — Set default profile

## Validation

- [ ] `go build` succeeds with no errors
- [ ] `Integration` struct has `ConfigType` field
- [ ] All LLM types defined in `integrations_llm.go`
- [ ] All LLM methods implemented with correct endpoints
- [ ] Methods return appropriate errors for non-2xx responses
- [ ] Manual test: `ListIntegrations()` returns integrations with `config_type` populated
- [ ] Manual test: `ListLLMProviders("llm")` returns provider accounts
