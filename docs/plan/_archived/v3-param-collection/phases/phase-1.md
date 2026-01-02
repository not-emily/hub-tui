# Phase 1: Client Types

> **Depends on:** None
> **Enables:** Phase 3 (ParamForm Modal), Phase 4 (App Integration)
>
> See: [Full Plan](../plan.md)

## Goal

Add new types and methods to the client package to support the `needs_input` response status and structured parameter submission.

## Key Deliverables

- New response types for status-based `/ask` responses
- `ParamSchema` and `ParamField` types for form schema
- `ExecuteResult` and `AskError` types for success/error responses
- `AskDirect` method for blocking requests (form submission)
- Update existing `Ask` method to handle new response format

## Files to Modify

- `internal/client/ask.go` — Add new types, update response handling

## Dependencies

**Internal:** None

**External:** None (uses existing net/http)

## Implementation Notes

### New Types

```go
// Response status constants
const (
    StatusNeedsInput = "needs_input"
    StatusExecuted   = "executed"
    StatusError      = "error"
)

// AskRequest supports both natural language and structured params
type AskRequest struct {
    Input  string                 `json:"input,omitempty"`
    Target string                 `json:"target,omitempty"`
    Params map[string]interface{} `json:"params,omitempty"`
}

// AskResponse with status-based result
type AskResponse struct {
    Status string         `json:"status"`
    Target string         `json:"target,omitempty"`
    Schema *ParamSchema   `json:"schema,omitempty"`
    Result *ExecuteResult `json:"result,omitempty"`
    Error  *AskError      `json:"error,omitempty"`
}

type ParamSchema struct {
    Title       string       `json:"title"`
    Description string       `json:"description"`
    Params      []ParamField `json:"params"`
}

type ParamField struct {
    Name        string      `json:"name"`
    Type        string      `json:"type"`
    Required    bool        `json:"required"`
    Description string      `json:"description"`
    Value       interface{} `json:"value"`
    Error       string      `json:"error"`
}

type ExecuteResult struct {
    Success bool                   `json:"success"`
    Message string                 `json:"message"`
    Data    map[string]interface{} `json:"data"`
}

type AskError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}
```

### AskDirect Method

For form submission, we need a blocking (non-streaming) request to `/ask/direct`:

```go
// AskDirect sends a blocking request to /ask/direct.
// Used for both natural language queries and structured param submission.
func (c *Client) AskDirect(req AskRequest) (*AskResponse, error) {
    // POST to /ask/direct with JSON body
    // Return parsed AskResponse
}
```

### Streaming Ask Updates

The existing `Ask` method uses SSE streaming. For the parameter collection flow:

1. Initial natural language request uses streaming `Ask` (shows route info, chunks)
2. When `needs_input` is returned, the `done` event contains the schema
3. Form submission uses blocking `AskDirect`

The streaming `Ask` method needs to:
- Parse the new response format from `done` events
- Return `AskResponse` with proper status/schema/result/error fields

### Backward Compatibility

The existing code expects `AskResponse.Message` for display. For `status: "executed"`, we should populate `Message` from `Result.Message` for backward compatibility until app.go is updated in Phase 4.

## Validation

How do we know this phase is complete?

- [ ] New types compile without errors
- [ ] `AskDirect` method successfully calls `/ask/direct` endpoint
- [ ] `AskDirect` correctly parses `needs_input` response with schema
- [ ] `AskDirect` correctly parses `executed` response with result
- [ ] `AskDirect` correctly parses `error` response
- [ ] Existing streaming `Ask` still works for assistant/workflow responses
- [ ] Unit tests pass (if applicable)
