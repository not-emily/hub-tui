# Phase 4: App Integration

> **Depends on:** Phase 1 (Client Types), Phase 3 (ParamForm Modal)
> **Enables:** Feature complete
>
> See: [Full Plan](../plan.md)

## Goal

Wire up the parameter collection flow in the main app, handling `needs_input` responses, form display, and structured parameter submission.

## Key Deliverables

- Handle `needs_input` status from ask responses
- Open ParamFormModal when parameters needed
- Handle form submission and cancellation
- Submit structured params via `AskDirect`
- Handle submission results (success, validation errors, system errors)
- Display success/error messages in chat
- Proper state reset after form closes

## Files to Modify

- `internal/app/app.go` — Handle new messages, update ask flow

## Dependencies

**Internal:**
- `internal/client.AskDirect`, `AskResponse` (Phase 1)
- `internal/ui/modal.ParamFormModal`, `ParamFormSubmitMsg`, `ParamFormCancelMsg` (Phase 3)

**External:** None

## Implementation Notes

### New Messages

```go
// AskNeedsInputMsg indicates the API needs more input.
type AskNeedsInputMsg struct {
    Target string
    Schema *client.ParamSchema
}

// AskExecutedMsg indicates the API executed successfully.
type AskExecutedMsg struct {
    Target string
    Result *client.ExecuteResult
}

// AskErrorMsg indicates an API error.
type AskErrorMsg struct {
    Target string
    Error  *client.AskError
}
```

### Update doAsk Flow

The current `doAsk` uses streaming SSE. For the parameter collection flow:

1. User sends natural language → streaming `Ask`
2. If route is to a module that needs params, `done` event contains `needs_input`
3. Parse the new response format and emit appropriate message

```go
func (m *Model) doAsk(message string) tea.Cmd {
    return func() tea.Msg {
        ctx, cancel := context.WithCancel(context.Background())
        m.cancelAsk = cancel

        callbacks := client.AskCallbacks{
            OnRoute: func(route client.RouteInfo) {
                // Update status bar with route info
                m.program.Send(RouteInfoMsg{Route: route})
            },
            OnChunk: func(chunk string) {
                // Stream content to chat
                m.program.Send(StreamChunkMsg{Content: chunk})
            },
        }

        resp, err := m.client.Ask(ctx, message, callbacks)
        if err != nil {
            return AskErrorMsg{Error: &client.AskError{
                Code:    "request_failed",
                Message: err.Error(),
            }}
        }

        // Handle status-based response
        switch resp.Status {
        case client.StatusNeedsInput:
            return AskNeedsInputMsg{
                Target: resp.Target,
                Schema: resp.Schema,
            }
        case client.StatusExecuted:
            return AskExecutedMsg{
                Target: resp.Target,
                Result: resp.Result,
            }
        case client.StatusError:
            return AskErrorMsg{
                Target: resp.Target,
                Error:  resp.Error,
            }
        default:
            // Legacy response format (assistant chat, etc.)
            return StreamDoneMsg{Response: resp}
        }
    }
}
```

### Handle AskNeedsInputMsg

```go
case AskNeedsInputMsg:
    // Open parameter form modal
    formModal := modal.NewParamFormModal(msg.Target, msg.Schema)
    cmd := m.modal.Open(formModal)
    return m, cmd
```

### Handle ParamFormSubmitMsg

```go
case modal.ParamFormSubmitMsg:
    // Close modal first
    m.modal.Close()

    // Submit structured params
    return m, m.doAskWithParams(msg.Target, msg.Params)

func (m *Model) doAskWithParams(target string, params map[string]interface{}) tea.Cmd {
    return func() tea.Msg {
        req := client.AskRequest{
            Target: target,
            Params: params,
        }

        resp, err := m.client.AskDirect(req)
        if err != nil {
            return AskErrorMsg{Error: &client.AskError{
                Code:    "request_failed",
                Message: err.Error(),
            }}
        }

        switch resp.Status {
        case client.StatusNeedsInput:
            // Validation errors - reopen form with errors
            return AskNeedsInputMsg{
                Target: resp.Target,
                Schema: resp.Schema,
            }
        case client.StatusExecuted:
            return AskExecutedMsg{
                Target: resp.Target,
                Result: resp.Result,
            }
        case client.StatusError:
            return AskErrorMsg{
                Target: resp.Target,
                Error:  resp.Error,
            }
        }

        return nil
    }
}
```

### Handle ParamFormCancelMsg

```go
case modal.ParamFormCancelMsg:
    // User cancelled - just close modal, don't submit
    m.modal.Close()
    // Optionally show a message that form was cancelled
    return m, nil
```

### Handle AskExecutedMsg

```go
case AskExecutedMsg:
    // Show success message in chat
    m.chat.AddAssistantMessage(msg.Result.Message)
    return m, nil
```

### Handle AskErrorMsg

```go
case AskErrorMsg:
    // Show error message in chat
    errMsg := fmt.Sprintf("Error: %s", msg.Error.Message)
    m.chat.AddSystemMessage(errMsg)
    return m, nil
```

### State Flow Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                        NORMAL STATE                          │
│  User input → doAsk() → streaming response                  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │ Response Status │
                    └─────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
   needs_input            executed               error
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│ Open Form     │    │ Show Success  │    │ Show Error    │
│ Modal         │    │ Message       │    │ Message       │
└───────────────┘    └───────────────┘    └───────────────┘
        │                     │                     │
        ▼                     ▼                     ▼
┌─────────────────────────────────────────────────────────────┐
│                     FORM MODAL OPEN                          │
│  User fills form, validates required fields                  │
└─────────────────────────────────────────────────────────────┘
        │                                           │
        ▼                                           ▼
   Ctrl+S Save                                  Esc Cancel
        │                                           │
        ▼                                           ▼
┌───────────────┐                          ┌───────────────┐
│ doAskWithParams│                         │ Close Modal   │
│ (blocking)     │                         │ Return to     │
└───────────────┘                          │ Normal State  │
        │                                  └───────────────┘
        ▼
┌─────────────────┐
│ Response Status │
└─────────────────┘
        │
        ├─── needs_input ──→ Reopen form with errors
        │
        ├─── executed ────→ Show success, close form, NORMAL STATE
        │
        └─── error ───────→ Show error, close form, NORMAL STATE
```

### Important: State Reset

After the form closes (success, error, or cancel), the next user input should use normal natural language mode:

```go
// doAsk always uses natural language
func (m *Model) doAsk(message string) tea.Cmd {
    req := client.AskRequest{Input: message}
    // ...
}

// doAskWithParams is only called from form submission
func (m *Model) doAskWithParams(target string, params map[string]interface{}) tea.Cmd {
    req := client.AskRequest{Target: target, Params: params}
    // ...
}
```

The state is implicitly managed by which function is called - no explicit state variable needed.

## Validation

How do we know this phase is complete?

- [ ] Natural language "add spaghetti to recipes" triggers `needs_input` → form opens
- [ ] Form displays with correct fields from schema
- [ ] Pre-filled values appear in form
- [ ] Esc closes form without submission
- [ ] Ctrl+S with empty required fields shows client-side errors
- [ ] Ctrl+S with valid data submits to API
- [ ] API validation errors reopen form with field errors
- [ ] Successful execution shows message in chat
- [ ] API errors show error message in chat
- [ ] After form closes, next request uses normal `{input: "..."}` format
- [ ] Streaming responses (assistants, workflows) still work normally
