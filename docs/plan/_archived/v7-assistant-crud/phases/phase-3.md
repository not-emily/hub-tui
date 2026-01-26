# Phase 3: Create & Edit Forms

> **Depends on:** Phase 2 (List & Detail Views)
> **Enables:** Phase 4 (Memory), Phase 5 (Templates) can start in parallel
>
> See: [Full Plan](../plan.md)

## Goal

Implement create and edit forms for assistants, including the module selection and gather tool configuration interface.

## Key Deliverables

- Create assistant form (viewCreate)
- Edit assistant form (viewEdit)
- Module multi-select with nested gather tool checkboxes
- LLM profile dropdown (loaded from API)
- Multi-line textarea for persona
- Form validation and error display
- Delete confirmation dialog
- Clear history confirmation dialog

## Files to Modify

- `internal/ui/modal/assistants.go` — Add form views and handlers

## Dependencies

**Internal:**
- Phase 1 client methods (Create, Update, Delete, ClearHistory)
- Phase 2 modal structure and navigation
- LLM profiles from integrations client (`ListLLMProfiles`)

**External:** None

## Implementation Notes

### New View States

```go
const (
    viewList = iota
    viewDetail
    viewCreate      // NEW
    viewEdit        // NEW
    viewConfirmDelete    // NEW
    viewConfirmClearHistory // NEW
)
```

### Form State

```go
type AssistantsModal struct {
    // ... existing fields ...

    // Form state
    formFocused    int      // Which field is focused
    formName       string   // name field
    formDisplayName string
    formPersona    string   // Multi-line
    formProfile    string   // Selected LLM profile
    formKeywords   string   // Comma-separated
    formModules    map[string]bool           // Module name → enabled
    formGather     map[string]map[string]bool // Module → tool → enabled
    formError      string   // Validation error

    // Loaded data for form
    availableProfiles []client.LLMProfile
    availableModules  []client.Module
}
```

### Form Layout

```
┌─ Create Assistant ────────────────────────────────────┐
│                                                       │
│  Name:          [my-assistant____]                    │
│  Display Name:  [My Assistant____]                    │
│  LLM Profile:   [default        ▾]                    │
│                                                       │
│  Persona:                                             │
│  ┌─────────────────────────────────────────────────┐  │
│  │ You are a helpful assistant...                  │  │
│  │                                                 │  │
│  │                                                 │  │
│  │                                                 │  │
│  │                                                 │  │
│  └─────────────────────────────────────────────────┘  │
│  (↑↓ to scroll when focused)                          │
│                                                       │
│  Keywords:      [assistant, help_]                    │
│                 (comma-separated, optional)           │
│                                                       │
│  Modules & Gather Tools:                              │
│  [x] recipes                                          │
│      [ ] add_item  [ ] get_items  [x] get_meal_plan   │
│  [ ] notes                                            │
│  [x] calendar                                         │
│      [x] get_today  [ ] get_week                      │
│                                                       │
│  [Tab] Next  [Enter] Create  [Esc] Cancel             │
└───────────────────────────────────────────────────────┘
```

### Form Fields Order

1. Name (text input) — only in create mode, read-only in edit
2. Display Name (text input)
3. LLM Profile (dropdown)
4. Persona (multi-line textarea)
5. Keywords (text input)
6. Modules section (multi-select with nested gather)

### Validation Rules

| Field | Required | Validation |
|-------|----------|------------|
| name | Yes (create only) | Lowercase alphanumeric + hyphens, no spaces |
| display_name | Yes | Non-empty |
| llm_profile | Yes | Must be valid profile from dropdown |
| persona | Yes | Min 10 characters |
| keywords | No | Split on comma, trim whitespace |
| modules | No | Valid module names |
| gather | No | Tools must belong to selected modules |

### Pre-flight Check (Create)

Before showing create form, check if LLM profiles exist:
- If no profiles: Show message "Configure an LLM profile first" with hint to open Integrations [i]
- If profiles exist: Proceed to form

### Module/Gather Selection UX

```
Modules & Gather Tools:

[x] recipes                           <- Space toggles module
    [ ] add_item  [x] get_meal_plan   <- Space toggles gather tool
[ ] notes
[x] calendar
    [x] get_today  [ ] get_week

Navigation:
- Tab moves between module rows
- When on a module row, Space toggles it
- When a module is selected, its tools appear indented below
- Tab into the tools row, Space toggles individual tools
```

### Confirmation Dialogs

**Delete confirmation:**
```
  Delete "jarvis"?

  This will permanently delete this assistant
  and all its conversation history.

  [Enter] Delete  [Esc] Cancel
```

**Clear history confirmation:**
```
  Clear history for "jarvis"?

  This will delete all conversation history.
  Core memory will be preserved.

  [Enter] Clear  [Esc] Cancel
```

### Message Types

```go
type AssistantCreatedMsg struct {
    Assistant *client.Assistant
    Error     error
}

type AssistantUpdatedMsg struct {
    Assistant *client.Assistant
    Error     error
}

type AssistantDeletedMsg struct {
    Name  string
    Error error
}

type AssistantHistoryClearedMsg struct {
    Name  string
    Error error
}

type LLMProfilesLoadedMsg struct {
    Profiles []client.LLMProfile
    Error    error
}

type ModulesLoadedMsg struct {
    Modules []client.Module
    Error   error
}
```

## Validation

How do we know this phase is complete?

- [ ] [n] from list opens create form
- [ ] Create form shows all fields with proper layout
- [ ] LLM profile dropdown is populated from API
- [ ] If no LLM profiles exist, shows blocking message
- [ ] Persona textarea supports multi-line input with scrolling
- [ ] Keywords accepts comma-separated input
- [ ] Module selection works with checkboxes
- [ ] Selecting a module reveals its gather tools
- [ ] Gather tools can be toggled independently
- [ ] Tab navigates through all form fields
- [ ] Enter on last field submits the form
- [ ] Validation errors are shown inline
- [ ] Successful create returns to list with new assistant
- [ ] [e] from detail opens edit form (pre-populated)
- [ ] Edit form has name field read-only
- [ ] Successful edit returns to detail with updated data
- [ ] [d] shows delete confirmation
- [ ] Confirming delete removes assistant and returns to list
- [ ] [h] shows clear history confirmation
- [ ] Confirming clear history shows success and stays on detail
- [ ] Esc cancels any form/dialog without saving
