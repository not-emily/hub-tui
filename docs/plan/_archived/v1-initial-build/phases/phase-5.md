# Phase 5: Assistant Context

> **Depends on:** Phase 4 (Commands & Autocomplete)
> **Enables:** Independent (can be done in parallel with Phase 6)
>
> See: [Full Plan](../plan.md)

## Goal

Switch to assistant contexts and chat with them, with clear visual indication of the current context.

## Key Deliverables

- `@{assistant}` switches context to that assistant
- `@hub` returns to main hub context
- Colored border when in assistant mode
- Colored hint below input showing current assistant
- Chat endpoint for assistants (streaming)
- Context persists until explicitly switched

## Files to Create

- `internal/client/assistants.go` — Chat endpoint (extend from Phase 4)

## Files to Modify

- `internal/app/app.go` — Add context state, border styling
- `internal/ui/chat/chat.go` — Context indicator below input
- `internal/ui/chat/input.go` — Handle @assistant completion as context switch
- `internal/ui/theme/theme.go` — Add assistant context colors

## Dependencies

**Internal:** Commands & autocomplete from Phase 4

**External:** None

## Implementation Notes

### Context State

```go
type Context struct {
    Mode      ContextMode  // ModeHub or ModeAssistant
    Assistant string       // Name of current assistant (if in assistant mode)
}

type ContextMode int

const (
    ModeHub ContextMode = iota
    ModeAssistant
)
```

### Switching Context

When user sends `@fitness_trainer`:
1. Parse as assistant switch (not a message)
2. Validate assistant exists in cache
3. Update context state
4. Show confirmation in chat ("Switched to fitness_trainer")
5. Apply visual indicator

When user sends `@hub`:
1. Clear assistant context
2. Return to main mode
3. Show confirmation ("Returned to hub")

### Visual Indicator: Colored Border

When in assistant mode, the entire chat area gets a colored border:

```go
var AssistantBorderColor = lipgloss.Color("#7c9fc7")  // Soft blue

func (m Model) View() string {
    content := m.chatView()

    if m.context.Mode == ModeAssistant {
        return lipgloss.NewStyle().
            Border(lipgloss.RoundedBorder()).
            BorderForeground(AssistantBorderColor).
            Render(content)
    }
    return content
}
```

### Visual Indicator: Context Hint

Below the input area, show current context:

```
┌─────────────────────────────────────────┐
│ > Type a message...                     │
│ Talking to: fitness_trainer             │  ← Colored hint
├─────────────────────────────────────────┤
│ Connected to hub                        │
└─────────────────────────────────────────┘
```

The hint line:
- Only visible when in assistant mode
- Uses accent color
- Shows assistant name

### Chat Endpoint

```go
func (c *Client) Chat(
    ctx context.Context,
    assistant string,
    message string,
    onChunk func(string),
) (*Response, error) {
    // POST to /assistants/{assistant}/chat
    // Stream response chunks via onChunk
}
```

### Message Routing

When sending a message:

```go
if m.context.Mode == ModeAssistant {
    // Send to /assistants/{name}/chat
    go m.client.Chat(ctx, m.context.Assistant, message, onChunk)
} else {
    // Send to /ask
    go m.client.Ask(ctx, message, onChunk)
}
```

### Context Persistence

Context stays until:
- User types `@hub`
- User types `@{different_assistant}`
- User exits TUI

Context is NOT persisted to config file—each session starts at hub.

### Error Cases

- `@unknown_assistant` → "Assistant 'unknown_assistant' not found"
- Assistant API error → Show error in chat, stay in current context

## Validation

How do we know this phase is complete?

- [ ] `@fitness_trainer` switches to that assistant
- [ ] Chat view gets colored border in assistant mode
- [ ] "Talking to: fitness_trainer" hint appears below input
- [ ] Messages sent go to assistant chat endpoint
- [ ] Responses stream correctly from assistant
- [ ] `@hub` returns to main context
- [ ] Border and hint disappear when back to hub
- [ ] `@different_assistant` switches directly
- [ ] Invalid assistant shows error message
- [ ] New session starts at hub (no persistence)
