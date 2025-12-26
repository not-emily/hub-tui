# Phase 3: Chat Interface

> **Depends on:** Phase 2 (API Client & Connection)
> **Enables:** Phase 4 (Commands & Autocomplete)
>
> See: [Full Plan](../plan.md)

## Goal

Send messages to hub-core and see streaming responses in a scrollable chat view.

## Key Deliverables

- Chat view with message list (scrollable)
- Input component with multi-line support (Shift+Enter)
- Send message to `/ask` endpoint
- Streaming response display (chunks appear as they arrive)
- Message styling (distinguish user vs hub)
- Keyboard navigation (scroll, double Ctrl+C to exit)

## Files to Create

- `internal/ui/chat/chat.go` — Chat view container
- `internal/ui/chat/message.go` — Single message component
- `internal/ui/chat/input.go` — Text input with multi-line
- `internal/client/ask.go` — /ask endpoint with streaming

## Files to Modify

- `internal/app/app.go` — Integrate chat view
- `internal/app/messages.go` — Add chat-related messages

## Dependencies

**Internal:** API client from Phase 2

**External:** None

## Implementation Notes

### Chat View Layout

```
┌─────────────────────────────────────────┐
│                                         │
│  You: What's the weather like?          │
│                                         │
│  Hub: I don't have access to weather    │
│  data directly, but I can help you...   │
│                                         │
│  You: Can you check my calendar?        │
│                                         │
│  Hub: Looking at your calendar...       │
│  ▌ (streaming indicator)                │
│                                         │
├─────────────────────────────────────────┤
│ > Type a message...                     │
├─────────────────────────────────────────┤
│ Connected to hub                        │
└─────────────────────────────────────────┘
```

### Message Model

```go
type Message struct {
    Role      string    // "user" or "hub"
    Content   string
    Timestamp time.Time
    Streaming bool      // True while response is being received
}
```

### Streaming Implementation

The `/ask` endpoint returns streaming responses. Implementation:

```go
func (c *Client) Ask(ctx context.Context, message string, onChunk func(string)) (*Response, error) {
    // POST to /ask with message
    // Read response body incrementally
    // Call onChunk for each piece of text
    // Return complete response when done
}
```

In the Bubble Tea model:

```go
// Start streaming in a goroutine
go func() {
    err := client.Ask(ctx, message, func(chunk string) {
        p.Send(StreamChunkMsg{Content: chunk})
    })
    p.Send(StreamDoneMsg{Error: err})
}()
```

### Input Component

Features:
- Single-line by default
- Shift+Enter inserts newline (multi-line mode)
- Enter sends message
- Placeholder text when empty
- Cursor position tracking

```go
type Input struct {
    value    []string  // Lines of text
    cursor   Position  // Row, column
    focused  bool
}
```

### Scrolling

- Up/Down arrows scroll chat history
- PgUp/PgDn scroll faster (e.g., 10 lines)
- Auto-scroll to bottom on new messages
- Manual scroll disables auto-scroll until user scrolls to bottom

### Message Styling

```go
var (
    UserStyle = lipgloss.NewStyle().
        Foreground(theme.TextPrimary).
        Bold(true)

    HubStyle = lipgloss.NewStyle().
        Foreground(theme.TextSecondary)

    StreamingIndicator = "▌"
)
```

### Keyboard Handling

| Key | Action |
|-----|--------|
| Enter | Send message (if input not empty) |
| Shift+Enter | Insert newline |
| Up | Scroll chat up (when input empty) |
| Down | Scroll chat down |
| PgUp | Scroll up 10 lines |
| PgDn | Scroll down 10 lines |

## Validation

How do we know this phase is complete?

- [ ] Can type a message in the input area
- [ ] Enter sends message to hub-core
- [ ] Response streams in character by character
- [ ] Streaming indicator (▌) shows while receiving
- [ ] Messages styled differently for user vs hub
- [ ] Can scroll through chat history with Up/Down
- [ ] PgUp/PgDn scroll faster
- [ ] Shift+Enter creates newline in input
- [ ] Auto-scroll to bottom on new messages
- [ ] Empty input doesn't send
