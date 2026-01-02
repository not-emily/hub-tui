package modal

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/client"
	"github.com/pxp/hub-tui/internal/ui/components"
	"github.com/pxp/hub-tui/internal/ui/theme"
)

// ParamFormSubmitMsg is sent when the user submits the form.
type ParamFormSubmitMsg struct {
	Target string
	Params map[string]interface{}
}

// ParamFormCancelMsg is sent when the user cancels the form.
type ParamFormCancelMsg struct{}

// ParamFormModal handles parameter collection for module operations.
type ParamFormModal struct {
	target string
	schema *client.ParamSchema
	form   *components.Form
	width  int
}

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

// schemaToFormFields converts API param fields to form fields.
func schemaToFormFields(params []client.ParamField) []components.FormField {
	var fields []components.FormField

	for _, p := range params {
		field := components.FormField{
			Label:       humanize(p.Name),
			Key:         p.Name,
			Required:    p.Required,
			Description: p.Description,
			Error:       p.Error,
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

// humanize converts snake_case to Title Case.
func humanize(s string) string {
	words := strings.Split(s, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

// valueToBool converts an interface{} to bool.
func valueToBool(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val == "true" || val == "1"
	default:
		return false
	}
}

// valueToString converts an interface{} to string.
func valueToString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		// JSON numbers are float64
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// valueToTextArea converts an interface{} to multi-line text.
func valueToTextArea(v interface{}, paramType string) string {
	if v == nil {
		return ""
	}

	switch paramType {
	case "array":
		// Convert array to one item per line
		if arr, ok := v.([]interface{}); ok {
			var lines []string
			for _, item := range arr {
				lines = append(lines, valueToString(item))
			}
			return strings.Join(lines, "\n")
		}
		if arr, ok := v.([]string); ok {
			return strings.Join(arr, "\n")
		}
		return valueToString(v)

	case "object":
		// Convert object to JSON
		if obj, ok := v.(map[string]interface{}); ok {
			bytes, err := json.MarshalIndent(obj, "", "  ")
			if err == nil {
				return string(bytes)
			}
		}
		return valueToString(v)

	default:
		return valueToString(v)
	}
}

// Init implements Modal.
func (m *ParamFormModal) Init() tea.Cmd {
	return nil
}

// Update implements Modal.
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

// buildParams converts form values to typed params for API submission.
func (m *ParamFormModal) buildParams() map[string]interface{} {
	params := make(map[string]interface{})

	for _, field := range m.form.Fields {
		switch field.ParamType {
		case "boolean":
			params[field.Key] = field.Checked
		case "number":
			// Parse as float64, API will validate
			trimmed := strings.TrimSpace(field.Value)
			if trimmed == "" {
				params[field.Key] = nil
			} else if val, err := strconv.ParseFloat(trimmed, 64); err == nil {
				params[field.Key] = val
			} else {
				params[field.Key] = field.Value // Send as string, let API error
			}
		case "array":
			// Split by newlines, trim each item
			params[field.Key] = textAreaToArray(field.Value)
		case "object":
			// Parse as JSON
			trimmed := strings.TrimSpace(field.Value)
			if trimmed == "" {
				params[field.Key] = nil
			} else {
				var obj map[string]interface{}
				if err := json.Unmarshal([]byte(trimmed), &obj); err == nil {
					params[field.Key] = obj
				} else {
					params[field.Key] = field.Value // Send as string, let API error
				}
			}
		default: // string
			params[field.Key] = strings.TrimSpace(field.Value)
		}
	}

	return params
}

// textAreaToArray splits text by newlines into an array.
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

// View implements Modal.
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

	return strings.Join(lines, "\n")
}

// Title implements Modal.
func (m *ParamFormModal) Title() string {
	return m.schema.Title
}

// IsFormModal returns true to indicate this modal uses form-style keybindings.
func (m *ParamFormModal) IsFormModal() bool {
	return true
}

// SetWidth sets the available width for the modal.
func (m *ParamFormModal) SetWidth(width int) {
	m.width = width
}
