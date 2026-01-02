# Phase 2: Form Component Extensions

> **Depends on:** None
> **Enables:** Phase 3 (ParamForm Modal)
>
> See: [Full Plan](../plan.md)

## Goal

Extend the reusable form component to support multi-line text areas, error display, required field indicators, and field descriptions.

## Key Deliverables

- New `FieldTextArea` type for multi-line text input
- Error display below fields
- Required field indicator (`*` or styling)
- Field description/help text display
- `ParamType` field for type conversion during submission

## Files to Modify

- `internal/ui/components/form.go` — Add new field type and display features

## Dependencies

**Internal:** None

**External:** None

## Implementation Notes

### New Field Type: FieldTextArea

```go
const (
    FieldText     FieldType = iota
    FieldSelect
    FieldButton
    FieldCheckbox
    FieldTextArea  // NEW
)
```

Multi-line text area behavior:
- Enter inserts newline (does NOT move to next field)
- Tab/Shift+Tab moves between fields
- Up/Down arrows navigate within text (not between fields) when in textarea
- Renders with a visible boundary showing the text area
- Cursor moves line by line

For simplicity, the textarea can store content as a single string with `\n` separators. The cursor position tracks both line and column.

### Extended FormField

```go
type FormField struct {
    Label           string
    Key             string
    Value           string
    Password        bool
    Type            FieldType
    Options         []string
    Selected        int
    DisabledOptions map[string]bool
    Checked         bool

    // NEW fields
    Required    bool   // Show required indicator, used for validation
    Error       string // Validation error to display below field
    Description string // Help text shown below field (when focused or always)
    ParamType   string // Original param type: "string", "number", "boolean", "array", "object"
}
```

### Rendering Updates

#### Required Indicator
Show `*` after label for required fields:
```
  Name*: [Spaghetti Carbonara    ]
  Description: [                  ]
```

Or use color/bold styling to distinguish required fields.

#### Error Display
Show error in red/warning color below the field:
```
  Servings*: [not a number]
             ⚠ servings must be a number
```

#### Description Display
Show description in muted text below field (when focused, or always for complex fields):
```
  Ingredients*:
  ┌────────────────────────────┐
  │ pasta                      │
  │ eggs                       │
  │                            │
  └────────────────────────────┘
  One ingredient per line
```

### TextArea Rendering

```
  Ingredients*:
  ┌────────────────────────────┐
  │ pasta                      │
  │ eggs                       │
  │ guanciale█                 │
  │                            │
  └────────────────────────────┘
```

The textarea should:
- Show a bordered box
- Display cursor position
- Handle scrolling if content exceeds visible lines
- Support a configurable height (e.g., 4-6 lines)

### TextArea Input Handling

```go
func (f *Form) updateTextArea(msg tea.KeyMsg) bool {
    switch msg.Type {
    case tea.KeyTab:
        // Move to next field
    case tea.KeyShiftTab:
        // Move to previous field
    case tea.KeyEnter:
        // Insert newline (NOT move to next field)
    case tea.KeyUp, tea.KeyDown:
        // Move cursor within text
    case tea.KeyLeft, tea.KeyRight:
        // Move cursor within line
    case tea.KeyBackspace, tea.KeyDelete:
        // Delete character
    case tea.KeyRunes:
        // Insert text
    }
    return false
}
```

### Validation Helper

Add a method to check if all required fields are filled:

```go
// ValidateRequired checks all required fields have values.
// Returns map of field key -> error message for empty required fields.
func (f *Form) ValidateRequired() map[string]string {
    errors := make(map[string]string)
    for _, field := range f.Fields {
        if field.Required && strings.TrimSpace(field.Value) == "" {
            errors[field.Key] = field.Label + " is required"
        }
    }
    return errors
}

// SetFieldError sets the error message for a field.
func (f *Form) SetFieldError(key, errorMsg string) {
    for i := range f.Fields {
        if f.Fields[i].Key == key {
            f.Fields[i].Error = errorMsg
            break
        }
    }
}

// ClearErrors clears all field errors.
func (f *Form) ClearErrors() {
    for i := range f.Fields {
        f.Fields[i].Error = ""
    }
}

// HasErrors returns true if any field has an error.
func (f *Form) HasErrors() bool {
    for _, field := range f.Fields {
        if field.Error != "" {
            return true
        }
    }
    return false
}
```

## Validation

How do we know this phase is complete?

- [ ] `FieldTextArea` renders with bordered box
- [ ] TextArea supports multi-line input with Enter
- [ ] TextArea cursor navigation works (up/down/left/right within text)
- [ ] Tab/Shift+Tab moves between fields (including textarea)
- [ ] Required fields show `*` indicator
- [ ] Error messages display below fields in warning style
- [ ] Description text displays for fields
- [ ] `ValidateRequired()` correctly identifies empty required fields
- [ ] `SetFieldError()` and `ClearErrors()` work correctly
- [ ] Existing field types (text, select, checkbox, button) still work
