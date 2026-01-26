# Phase 4: Memory Management

> **Depends on:** Phase 3 (Create & Edit)
> **Enables:** Completes core assistant management
>
> See: [Full Plan](../plan.md)

## Goal

Add the ability to view and edit an assistant's core memory entries from a dedicated sub-view.

## Key Deliverables

- Memory view (viewMemory) showing all memory entries
- Inline editing of memory entry values
- Add new memory entry
- Delete memory entry
- Save changes back to API

## Files to Modify

- `internal/ui/modal/assistants.go` — Add memory view and handlers

## Dependencies

**Internal:**
- Phase 1 client methods (GetAssistantMemory, UpdateAssistantMemory)
- Phase 3 modal structure with detail view

**External:** None

## Implementation Notes

### Memory Data Structure

The API returns memory as a simple key-value map:

```go
type AssistantMemory struct {
    Entries map[string]string `json:"entries"`
}
```

Example memory:
```json
{
  "entries": {
    "user_name": "Emily",
    "preferences": "Prefers vegetarian recipes, morning workouts",
    "context": "Software engineer working on personal AI projects"
  }
}
```

### New View State

```go
const (
    // ... existing ...
    viewMemory  // NEW
)
```

### Memory View State

```go
type AssistantsModal struct {
    // ... existing fields ...

    // Memory state
    memoryEntries   []memoryEntry  // Ordered list for navigation
    memorySelected  int            // Selected entry index
    memoryEditing   bool           // Currently editing a value
    memoryEditValue string         // Value being edited
    memoryNewKey    string         // When adding new entry
    memoryNewValue  string
    memoryAddMode   bool           // In "add new entry" mode
    memoryDirty     bool           // Has unsaved changes
    memoryError     string
}

type memoryEntry struct {
    Key   string
    Value string
}
```

### Memory View Layout

```
  jarvis — Core Memory

  ┌──────────────────────────────────────────────────────┐
  │ user_name                                            │
  │ Emily                                                │
  ├──────────────────────────────────────────────────────┤
  │ preferences                                    [sel] │
  │ Prefers vegetarian recipes, morning workouts         │
  ├──────────────────────────────────────────────────────┤
  │ context                                              │
  │ Software engineer working on personal AI projects    │
  └──────────────────────────────────────────────────────┘

  [Enter] Edit  [a] Add entry  [d] Delete  [s] Save  [Esc] Back
```

**When editing:**
```
  jarvis — Core Memory

  ┌──────────────────────────────────────────────────────┐
  │ preferences                                          │
  │ ┌──────────────────────────────────────────────────┐ │
  │ │ Prefers vegetarian recipes, morning workouts,   │ │
  │ │ likes Asian cuisine█                            │ │
  │ └──────────────────────────────────────────────────┘ │
  └──────────────────────────────────────────────────────┘

  [Enter] Save  [Esc] Cancel
```

**When adding:**
```
  jarvis — Core Memory — Add Entry

  Key:   [favorite_color__]
  Value: [blue____________]

  [Enter] Add  [Esc] Cancel
```

### Key Bindings (Memory View)

| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Navigate entries |
| `Enter` | Edit selected entry value |
| `a` | Add new entry |
| `d` | Delete selected entry (with confirm) |
| `s` | Save all changes |
| `Esc` | Back (warn if dirty) |

**When editing:**
| Key | Action |
|-----|--------|
| `Enter` | Save edit |
| `Esc` | Cancel edit |

### Dirty State Warning

If user presses Esc with unsaved changes:
```
  Unsaved Changes

  You have unsaved memory changes.

  [s] Save & exit  [Enter] Discard & exit  [Esc] Cancel
```

### Message Types

```go
type AssistantMemoryLoadedMsg struct {
    Memory *client.AssistantMemory
    Error  error
}

type AssistantMemorySavedMsg struct {
    Error error
}
```

### Handling Long Values

Memory values can be long. Options:
- Truncate display with "..." and show full on edit
- Wrap text within entry box
- Show first N lines with scroll indicator

Recommended: Wrap text, show full content, allow scrolling within entry when selected.

## Validation

How do we know this phase is complete?

- [ ] [m] from detail view opens memory view
- [ ] Memory entries are displayed with keys and values
- [ ] Can navigate between entries with j/k
- [ ] Enter on entry opens inline edit for value
- [ ] Edited values are tracked as dirty
- [ ] [a] opens add entry dialog
- [ ] Can add new key-value pair
- [ ] [d] removes selected entry (with confirmation)
- [ ] [s] saves all changes to API
- [ ] Success message shown after save
- [ ] Esc with no changes returns to detail
- [ ] Esc with dirty changes shows warning dialog
- [ ] Long values wrap properly within entry box
- [ ] Error states are handled (API failures)
