# Phase 3: ParamForm Modal

> **Depends on:** Phase 1 (Client Types), Phase 2 (Form Component)
> **Enables:** Phase 4 (App Integration)
>
> See: [Full Plan](../plan.md)

## Goal

Create a new modal type that renders a dynamic form from an API schema and handles submission of structured parameters.

## Key Deliverables

- `ParamFormModal` struct implementing `Modal` interface
- Schema-to-form conversion logic
- Form-to-params conversion logic (respecting param types)
- Client-side required field validation
- Custom close behavior (Esc to cancel, Ctrl+S to save)
- Messages for form submission and cancellation

## Files to Create

- `internal/ui/modal/paramform.go` — ParamFormModal implementation

## Files to Modify

- `internal/ui/modal/modal.go` — Add support for form modal close behavior

## Dependencies

**Internal:**
- `internal/client.ParamSchema`, `ParamField` (Phase 1)
- `internal/ui/components.Form`, `FormField`, `FieldTextArea` (Phase 2)

**External:** None

## Implementation Notes

### ParamFormModal Structure

```go
package modal

import (
    "github.com/pxp/hub-tui/internal/client"
    "github.com/pxp/hub-tui/internal/ui/components"
)

// ParamFormModal handles parameter collection for module operations.
type ParamFormModal struct {
    target string              // API target (e.g., "recipes.add_recipe")
    schema *client.ParamSchema // Schema from API
    form   *components.Form    // Generated form
    width  int
}

// Messages
type ParamFormSubmitMsg struct {
    Target string
    Params map[string]interface{}
}

type ParamFormCancelMsg struct{}
```

### Schema to Form Conversion

```go
// NewParamFormModal creates a modal from an API schema.
func NewParamFormModal(target string, schema *client.ParamSchema) *ParamFormModal {
    fields := schemaToFormFields(schema.Params)
    form := components.NewForm(schema.Title, fields)

    return &ParamFormModal{
        target: target,
        schema: schema,
        form:   form,
    }
}

func schemaToFormFields(params []client.ParamField) []components.FormField {
    var fields []components.FormField

    for _, p := range params {
        field := components.FormField{
            Label:       humanize(p.Name), // "cook_time" -> "Cook Time"
            Key:         p.Name,
            Required:    p.Required,
            Description: p.Description,
            Error:       p.Error, // Pre-populate with API errors
            ParamType:   p.Type,
        }

        // Set field type based on param type
        switch p.Type {
        case "boolean":
            field.Type = components.FieldCheckbox
            field.Checked = valueToBool(p.Value)
        case "array", "object":
            field.Type = components.FieldTextArea
            field.Value = valueToTextArea(p.Value, p.Type)
        default: // string, number
            field.Type = components.FieldText
            field.Value = valueToString(p.Value)
        }

        fields = append(fields, field)
    }

    return fields
}
```

### Form to Params Conversion

```go
// buildParams converts form values to typed params for API submission.
func (m *ParamFormModal) buildParams() map[string]interface{} {
    params := make(map[string]interface{})

    for _, field := range m.form.Fields {
        switch field.ParamType {
        case "boolean":
            params[field.Key] = field.Checked
        case "number":
            // Parse as float64, API will validate
            if val, err := strconv.ParseFloat(field.Value, 64); err == nil {
                params[field.Key] = val
            } else {
                params[field.Key] = field.Value // Send as string, let API error
            }
        case "array":
            // Split by newlines, trim each item
            params[field.Key] = textAreaToArray(field.Value)
        case "object":
            // Parse as JSON
            var obj map[string]interface{}
            if err := json.Unmarshal([]byte(field.Value), &obj); err == nil {
                params[field.Key] = obj
            } else {
                params[field.Key] = field.Value // Send as string, let API error
            }
        default: // string
            params[field.Key] = field.Value
        }
    }

    return params
}

func textAreaToArray(s string) []string {
    lines := strings.Split(s, "\n")
    var result []string
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        if trimmed != "" {
            result = append(result, trimmed)
        }
    }
    return result
}
```

### Update Handling

```go
func (m *ParamFormModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "esc":
            // Cancel - return nil to close modal
            return nil, func() tea.Msg { return ParamFormCancelMsg{} }

        case "ctrl+s":
            // Validate required fields
            m.form.ClearErrors()
            errors := m.form.ValidateRequired()
            if len(errors) > 0 {
                for key, errMsg := range errors {
                    m.form.SetFieldError(key, errMsg)
                }
                return m, nil
            }

            // Build and submit params
            params := m.buildParams()
            return nil, func() tea.Msg {
                return ParamFormSubmitMsg{
                    Target: m.target,
                    Params: params,
                }
            }
        }

        // Forward to form
        m.form.Update(msg)
    }

    return m, nil
}
```

### View Rendering

```go
func (m *ParamFormModal) View() string {
    var lines []string

    // Description if present
    if m.schema.Description != "" {
        descStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
        lines = append(lines, descStyle.Render(m.schema.Description))
        lines = append(lines, "")
    }

    // Form
    lines = append(lines, m.form.View())

    // Hints
    lines = append(lines, "")
    hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
    lines = append(lines, hintStyle.Render("Esc cancel · Ctrl+S save"))

    return strings.Join(lines, "\n")
}

func (m *ParamFormModal) Title() string {
    return m.schema.Title
}
```

### Modal System Updates

In `modal.go`, update the close behavior to NOT use 'q' for form modals:

```go
func (s *State) Update(msg tea.Msg) (bool, tea.Cmd) {
    if s.Active == nil {
        return false, nil
    }

    if keyMsg, ok := msg.(tea.KeyMsg); ok {
        // Check if this is a form modal (uses Esc/Ctrl+S, not q)
        if _, isFormModal := s.Active.(*ParamFormModal); isFormModal {
            // Let form modal handle all keys
        } else {
            // q closes non-form modals
            if keyMsg.String() == "q" {
                s.Active = nil
                return true, nil
            }
        }
    }

    // Forward to modal
    var cmd tea.Cmd
    s.Active, cmd = s.Active.Update(msg)

    if s.Active == nil {
        return true, cmd
    }
    return true, cmd
}
```

Also update the hint in `View()` to show appropriate close hint:
```go
func (s *State) View() string {
    // ...

    // Different hint for form modals
    var hint string
    if _, isFormModal := s.Active.(*ParamFormModal); isFormModal {
        hint = hintStyle.Render("Esc cancel · Ctrl+S save")
    } else {
        hint = hintStyle.Render("q to close")
    }

    // ...
}
```

Actually, the hint is already rendered by the modal itself in its View(), so we may want to remove the hint from the modal wrapper for form modals, or have the modal signal what hint to show. Let the form modal render its own hint at the bottom of its view.

## Validation

How do we know this phase is complete?

- [ ] `ParamFormModal` can be created from a `ParamSchema`
- [ ] Form fields are correctly generated for all param types
- [ ] Pre-filled values from API appear in form fields
- [ ] API validation errors appear on fields
- [ ] Required field indicator shows on required fields
- [ ] Esc closes modal and emits `ParamFormCancelMsg`
- [ ] Ctrl+S with empty required fields shows validation errors
- [ ] Ctrl+S with valid fields emits `ParamFormSubmitMsg` with typed params
- [ ] Arrays correctly convert from multi-line text to `[]string`
- [ ] Objects correctly parse from JSON text to `map[string]interface{}`
- [ ] 'q' does NOT close the form modal (only Esc)
- [ ] Form hint shows "Esc cancel · Ctrl+S save"
