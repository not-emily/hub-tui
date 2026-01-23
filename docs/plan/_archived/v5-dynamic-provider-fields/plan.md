# v5-dynamic-provider-fields: Dynamic Provider Field Configuration

> **Status:** Planning complete | Last updated: 2026-01-16
>
> Phase files: [phases/](phases/)

## Overview

The hub-core API has changed how LLM provider accounts are configured. Instead of a hardcoded `api_key` field, providers now declare their required configuration fields dynamically. Different providers need different fields: OpenAI/Anthropic need `api_key`, Ollama needs `base_url`, Azure needs `endpoint`, `developer_id`, and `api_key`.

This project updates hub-tui to:
1. Query provider field requirements before showing the provider form
2. Dynamically render form fields based on the API response
3. Submit provider configuration using the new `{"fields": {...}}` format

This builds on the v4-config-types work and follows the same pattern of letting the server define what the client needs to render.

## Core Vision

- **Server-driven UI**: The API tells us what fields to render, not hardcoded assumptions
- **Minimal changes**: Focused update to provider form only; profiles unchanged
- **Future-proof**: Any new provider with any fields will work without client changes

## Requirements

### Must Have
- Add `ProviderFieldInfo` type to client
- Add `GetLLMProviderFields(integration, provider)` client method
- Update `AddProviderRequest` to use `Fields map[string]string` instead of `APIKey`
- Fetch field requirements when user selects a provider in the form
- Dynamically build form fields from API response
- Handle field properties: `key`, `label`, `required`, `secret`, `default`
- Pre-populate fields with `default` values
- Client-side validation for `required` fields before submit

### Nice to Have
- Show field descriptions/help text if API provides them (not in current spec)

### Out of Scope
- Changes to profile form (profiles don't use the fields API)
- Editing existing provider accounts (API doesn't support this)

## Constraints

- **API compatibility**: Must work with the new hub-core endpoint format
- **Minimal disruption**: Keep existing UX patterns (provider dropdown, account name field)
- **Existing form component**: Use the existing `components.Form` for rendering

## Success Metrics

- User can add an OpenAI provider (api_key field)
- User can add an Ollama provider (base_url field)
- User can add an Azure provider (endpoint, developer_id, api_key fields)
- Adding a new provider type in hub-core requires zero hub-tui changes

## Architecture Decisions

### 1. Fetch fields on provider selection change
**Choice:** When user changes the provider dropdown, fetch fields and rebuild form
**Rationale:** Same pattern as profile form's provider→account→model cascade
**Trade-offs:** Extra API call per provider change, but fields are small and fast

### 2. Keep provider and account as static fields
**Choice:** Provider dropdown and Account name remain hardcoded; only credential fields are dynamic
**Rationale:** Every provider needs these two fields; they're not provider-specific
**Trade-offs:** Slightly mixed approach, but simpler than making everything dynamic

### 3. Rebuild form on provider change
**Choice:** Reconstruct the entire form with new fields when provider changes
**Rationale:** Simpler than trying to add/remove individual fields from existing form
**Trade-offs:** Loses any typed values in dynamic fields, but account name is preserved

## Project Structure

No new files. Changes to existing files:

```
internal/
├── client/
│   └── integrations_llm.go    # Add ProviderFieldInfo, GetLLMProviderFields, update AddProviderRequest
└── ui/
    └── modal/
        └── integrations_llm.go # Dynamic form building, field fetching on provider change
```

## Core Interfaces

### New Client Types

```go
// ProviderFieldInfo describes a configuration field required by a provider.
type ProviderFieldInfo struct {
    Key      string `json:"key"`      // e.g., "api_key", "base_url"
    Label    string `json:"label"`    // e.g., "API Key", "Base URL"
    Required bool   `json:"required"` // Whether field must be provided
    Secret   bool   `json:"secret"`   // Whether to mask input
    Default  string `json:"default"`  // Default value if not provided
}
```

### Updated Client Types

```go
// AddProviderRequest - CHANGED
type AddProviderRequest struct {
    Provider string            `json:"provider"`
    Account  string            `json:"account"`
    Fields   map[string]string `json:"fields"` // Was: APIKey string
}
```

### New Client Method

```go
// GetLLMProviderFields fetches field requirements for a provider.
func (c *Client) GetLLMProviderFields(integration, provider string) ([]ProviderFieldInfo, error)
```

### New Message Type

```go
// LLMProviderFieldsMsg is sent when provider field requirements are loaded.
type LLMProviderFieldsMsg struct {
    Provider string
    Fields   []client.ProviderFieldInfo
    Err      error
}
```

## Implementation Phases

| Phase | Name | Scope | Depends On | Key Outputs |
|-------|------|-------|------------|-------------|
| 1 | Client Layer | Types + API method | — | `ProviderFieldInfo`, `GetLLMProviderFields`, updated `AddProviderRequest` |
| 2 | Dynamic Provider Form | Fetch fields, rebuild form | Phase 1 | Working dynamic form for any provider |

### Critical Path

Sequential: Phase 1 → Phase 2

### Phase Details
- [Phase 1: Client Layer](phases/phase-1.md)
- [Phase 2: Dynamic Provider Form](phases/phase-2.md)

## Tech Stack

| Category | Choice | Notes |
|----------|--------|-------|
| Language | Go | Existing |
| TUI Framework | Bubble Tea | Existing |
| Form Component | components/form.go | Existing, supports dynamic field building |

## API Reference

### GET /integrations/llm/providers/{provider}/fields

Returns field requirements for a provider.

Response:
```json
{
  "fields": [
    {"key": "api_key", "label": "API Key", "required": true, "secret": true, "default": ""},
    {"key": "base_url", "label": "Base URL", "required": false, "secret": false, "default": "https://api.openai.com/v1"}
  ]
}
```

### POST /integrations/llm/providers (CHANGED)

Request body format changed from:
```json
{"provider": "openai", "account": "default", "api_key": "sk-..."}
```

To:
```json
{"provider": "openai", "account": "default", "fields": {"api_key": "sk-..."}}
```
