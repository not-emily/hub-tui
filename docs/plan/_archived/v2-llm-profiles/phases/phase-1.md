# Phase 1: Client Layer

> **Depends on:** None
> **Enables:** Phase 2 (Modal List View)
>
> See: [Full Plan](../plan.md)

## Goal

Create the API client layer for LLM profile management, following established patterns from `integrations.go`.

## Key Deliverables

- `LLMProfile`, `LLMProfileList`, `LLMTestResult`, `LLMProfileConfig` types
- Client methods for all LLM profile operations
- Proper error handling matching existing patterns

## Files to Create

- `internal/client/llm.go` — LLM profile API client

## Dependencies

**Internal:** None (first phase)

**External:** None (uses existing `net/http` patterns)

## Implementation Notes

### API Endpoints to Implement

| Method | Endpoint | Client Method |
|--------|----------|---------------|
| GET | `/llm/profiles` | `ListLLMProfiles()` |
| PUT | `/llm/profiles/{name}` | `CreateLLMProfile()` / `UpdateLLMProfile()` |
| DELETE | `/llm/profiles/{name}` | `DeleteLLMProfile()` |
| POST | `/llm/profiles/{name}/test` | `TestLLMProfile()` |
| PUT | `/llm/default` | `SetDefaultLLMProfile()` |

### Type Definitions

```go
// LLMProfile represents an LLM profile configuration.
type LLMProfile struct {
    Integration string `json:"integration"`
    Profile     string `json:"profile,omitempty"`
    Model       string `json:"model"`
}

// LLMProfileList is the response from GET /llm/profiles.
type LLMProfileList struct {
    Profiles       map[string]LLMProfile `json:"profiles"`
    DefaultProfile string                `json:"default_profile"`
}

// LLMTestResult is the response from POST /llm/profiles/{name}/test.
type LLMTestResult struct {
    Success   bool   `json:"success"`
    Model     string `json:"model"`
    LatencyMs int    `json:"latency_ms"`
    Error     string `json:"error,omitempty"`
}

// LLMProfileConfig is the request body for PUT /llm/profiles/{name}.
type LLMProfileConfig struct {
    Name        string `json:"name,omitempty"`  // for rename
    Integration string `json:"integration"`
    Profile     string `json:"profile,omitempty"`
    Model       string `json:"model"`
}
```

### Pattern Reference

Follow `internal/client/integrations.go` for:
- Error handling with `parseError(resp)`
- Request body encoding with `json.Marshal`
- Response decoding with `json.NewDecoder`
- Using `c.get()`, `c.post()`, `c.put()`, `c.delete()` helpers

### Note on Create vs Update

Both create and update use `PUT /llm/profiles/{name}`. The client can have separate methods for clarity, but they call the same endpoint. Include `Name` in config body only when renaming.

## Validation

How do we know this phase is complete?

- [ ] `internal/client/llm.go` exists with all type definitions
- [ ] `ListLLMProfiles()` returns profile list from API
- [ ] `CreateLLMProfile()` creates a new profile
- [ ] `UpdateLLMProfile()` updates existing profile (including rename when Name is set)
- [ ] `DeleteLLMProfile()` deletes a profile
- [ ] `TestLLMProfile()` returns test result with success/latency/error
- [ ] `SetDefaultLLMProfile()` sets the default profile
- [ ] Code compiles with `go build ./...`
- [ ] Code passes `go vet ./...`
