# Parameter Collection Form: Inline forms for multi-parameter module operations

> **Status:** Planning complete | Last updated: 2026-01-02
>
> Phase files: [phases/](phases/)

## Overview

When users interact with modules via natural language (e.g., "add spaghetti carbonara to my recipes"), the API may only extract partial information. The hub-core API now supports a parameter collection flow where it returns a `needs_input` status with a schema describing required fields. hub-tui needs to render these forms inline in the chat, collect user input, and submit structured parameters back to the API.

This feature enables rich data entry for any module with multi-field data collections (recipes, workouts, reading lists, etc.) without hardcoding forms for specific modules. The form structure is entirely schema-driven from the API response.

## Core Vision

- **Schema-driven**: Form structure comes entirely from the API schema - no hardcoded forms for specific modules
- **Inline rendering**: Forms appear in the chat flow (like current modals), preserving conversation context
- **Graceful validation**: Show field-level errors from API, plus client-side required field validation
- **Type-aware**: Handle all param types appropriately (string, number, boolean, array, object)

## Requirements

### Must Have
- Handle `needs_input` response status from `/ask` endpoints
- Render dynamic form based on `schema.params`
- Support all param types:
  - `string` → text input
  - `number` → text input (validated)
  - `boolean` → checkbox
  - `array` → multi-line textarea (one item per line)
  - `object` → JSON textarea
- Display field labels, descriptions, and required indicators
- Show pre-filled values from API (extracted from natural language)
- Submit structured params: `{"target": "...", "params": {...}}`
- Display field-level validation errors from API
- Client-side required field validation:
  - Disable submit if required fields are empty
  - Show "required" error on empty required fields when submit attempted
- Cancel form (Esc) returns to normal input
- Save form (Ctrl+S) submits to API
- Reset to normal request mode after form closes (success, error, or cancel)

### Nice to Have
- Tab navigation between fields
- Inline validation before submit (e.g., number format)

### Out of Scope
- Rich array editing (add/remove buttons) — use multi-line textarea for now
- Nested object forms — JSON textarea for now
- Form history/undo
- Auto-save drafts

## Constraints

- **Tech stack**: Go, Bubble Tea, Lip Gloss (existing stack)
- **API**: hub-core `/ask` endpoints with `needs_input` status
- **Patterns**: Must follow existing modal patterns for consistency

## Success Metrics

- User can say "add spaghetti to recipes" → fill form → recipe created
- Validation errors display per-field
- Works with any module that returns `needs_input`
- Form closes and state resets properly after completion

## Architecture Decisions

### 1. Reuse Modal System
**Choice:** Extend existing modal system rather than creating parallel form system
**Rationale:** Modal system already handles inline rendering, key routing, and positioning. A param form is conceptually a modal that temporarily takes over input.
**Trade-offs:** Need to add form-specific close behavior (Esc/Ctrl+S vs q), but avoids duplicating infrastructure.

### 2. Schema-Driven Form Generation
**Choice:** Generate form fields dynamically from API schema at runtime
**Rationale:** Modules define their own parameter schemas - TUI shouldn't need updates for new modules.
**Trade-offs:** Less control over field ordering/grouping, but maximizes flexibility.

### 3. Simple Array/Object Handling
**Choice:** Multi-line textarea for arrays (one item per line), JSON textarea for objects
**Rationale:** Unknown real-world usage patterns for objects. Simple approach is easy to implement and supports paste workflows.
**Trade-offs:** Less polished UX for complex data, but can be enhanced later based on real usage.

### 4. State Reset After Form Close
**Choice:** Form modal holds `target` temporarily; closing form (any reason) resets to normal mode
**Rationale:** Prevents accidental structured param submission on next natural language request.
**Trade-offs:** User loses form data if they cancel, but this is expected behavior.

## Project Structure

```
internal/
├── client/
│   └── ask.go           # MODIFY: New response types, param submission
├── ui/
│   ├── components/
│   │   └── form.go      # MODIFY: FieldTextArea, error display, required
│   └── modal/
│       ├── modal.go     # MODIFY: Form modal close behavior
│       └── paramform.go # NEW: ParamFormModal implementation
└── app/
    └── app.go           # MODIFY: Handle needs_input, form submission
```

### Key Files
- `internal/client/ask.go` — Extended types for `needs_input` responses and structured param requests
- `internal/ui/modal/paramform.go` — Schema-to-form conversion and param form modal
- `internal/ui/components/form.go` — Extended with textarea, error display, required indicators

## Core Interfaces

### Client Types

```go
// AskRequest supports both natural language and structured params
type AskRequest struct {
    Input  string                 `json:"input,omitempty"`
    Target string                 `json:"target,omitempty"`
    Params map[string]interface{} `json:"params,omitempty"`
}

// AskResponse with status-based result
type AskResponse struct {
    Status string         `json:"status"` // needs_input, executed, error
    Target string         `json:"target"`
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
    Type        string      `json:"type"` // string, number, boolean, array, object
    Required    bool        `json:"required"`
    Description string      `json:"description"`
    Value       interface{} `json:"value"`  // Pre-filled from API
    Error       string      `json:"error"`  // Validation error from API
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

### Form Component Extensions

```go
// New field type
const (
    FieldText     FieldType = iota
    FieldSelect
    FieldButton
    FieldCheckbox
    FieldTextArea  // NEW: Multi-line text input
)

// Extended FormField
type FormField struct {
    // ... existing fields ...
    Required    bool   // NEW: Is this field required?
    Error       string // NEW: Validation error to display
    Description string // NEW: Help text below field
    ParamType   string // NEW: Original param type (for conversion)
}
```

### ParamFormModal

```go
type ParamFormModal struct {
    target      string
    schema      *client.ParamSchema
    form        *components.Form
    width       int
    submitError string
}

// Messages
type ParamFormSubmitMsg struct {
    Target string
    Params map[string]interface{}
}

type ParamFormCancelMsg struct{}
```

## Implementation Phases

| Phase | Name | Scope | Depends On | Key Outputs |
|-------|------|-------|------------|-------------|
| 1 | Client Types | New types in ask.go | — | Response types, AskDirect method |
| 2 | Form Component | Extend form.go | — | FieldTextArea, error/required display |
| 3 | ParamForm Modal | New modal type | Phase 1, 2 | paramform.go with schema→form |
| 4 | App Integration | Wire up the flow | Phase 1, 3 | Handle needs_input, submission |

### Critical Path
All phases are sequential. Phase 1 and 2 have no dependencies on each other but are cleaner done in order.

### Phase Details
- [Phase 1: Client Types](phases/phase-1.md)
- [Phase 2: Form Component](phases/phase-2.md)
- [Phase 3: ParamForm Modal](phases/phase-3.md)
- [Phase 4: App Integration](phases/phase-4.md)

## Tech Stack

| Category | Choice | Notes |
|----------|--------|-------|
| Language | Go | Existing |
| TUI Framework | Bubble Tea | Existing |
| Styling | Lip Gloss | Existing |
| HTTP Client | net/http | Existing pattern |

## Future Considerations

- Rich array editing with add/remove buttons
- Nested object forms instead of JSON textarea
- Field grouping/sections for complex schemas
- Form auto-save/drafts
- Keyboard shortcuts for common actions
