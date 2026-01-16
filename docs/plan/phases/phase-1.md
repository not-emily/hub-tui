# Phase 1: Client Layer

> **Depends on:** None
> **Enables:** Phase 2 (Dynamic Provider Form)
>
> See: [Full Plan](../plan.md)

## Goal

Add types and API method to support the new provider fields endpoint and request format.

## Key Deliverables

- `ProviderFieldInfo` struct for field metadata
- `GetLLMProviderFields()` method to fetch field requirements
- Updated `AddProviderRequest` with `Fields map[string]string`

## Files to Modify

- `internal/client/integrations_llm.go` — Add type, method, update request struct

## Implementation Notes

### ProviderFieldInfo Type

```go
// ProviderFieldInfo describes a configuration field required by a provider.
type ProviderFieldInfo struct {
    Key      string `json:"key"`      // Field identifier (e.g., "api_key", "base_url")
    Label    string `json:"label"`    // Human-readable label for UI
    Required bool   `json:"required"` // Whether field must be provided
    Secret   bool   `json:"secret"`   // Whether to mask input (passwords, API keys)
    Default  string `json:"default"`  // Default value if not provided
}
```

### GetLLMProviderFields Method

```go
// providerFieldsResponse is the API response for provider fields.
type providerFieldsResponse struct {
    Fields []ProviderFieldInfo `json:"fields"`
}

// GetLLMProviderFields fetches field requirements for a provider.
func (c *Client) GetLLMProviderFields(integration, provider string) ([]ProviderFieldInfo, error) {
    resp, err := c.get("/integrations/" + integration + "/providers/" + provider + "/fields")
    if err != nil {
        return nil, fmt.Errorf("cannot connect to server: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return nil, parseError(resp)
    }

    var result providerFieldsResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("invalid response from server: %w", err)
    }

    return result.Fields, nil
}
```

### Updated AddProviderRequest

Change from:
```go
type AddProviderRequest struct {
    Provider string `json:"provider"`
    Account  string `json:"account"`
    APIKey   string `json:"api_key"`
}
```

To:
```go
type AddProviderRequest struct {
    Provider string            `json:"provider"`
    Account  string            `json:"account"`
    Fields   map[string]string `json:"fields"`
}
```

### Update saveProvider in modal

The `saveProvider()` function in `integrations_llm.go` will need to be updated in Phase 2 to build the `Fields` map from form values instead of using `APIKey`.

## Validation

- [ ] `ProviderFieldInfo` struct added with correct JSON tags
- [ ] `GetLLMProviderFields()` method added and compiles
- [ ] `AddProviderRequest` uses `Fields map[string]string` instead of `APIKey`
- [ ] Code compiles (modal will have build errors until Phase 2)
