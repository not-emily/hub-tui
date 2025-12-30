package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/ui/theme"
)

// FieldType indicates the type of form field.
type FieldType int

const (
	FieldText   FieldType = iota // Text input field
	FieldSelect                  // Selection field with options
	FieldButton                  // Button (e.g., Save, Cancel)
)

// FormField represents a single form field.
type FormField struct {
	Label           string
	Key             string
	Value           string          // For text fields: the text value. For select fields: the selected option value.
	Password        bool            // Mask input with asterisks (text fields only)
	Type            FieldType
	Options         []string        // For select fields: available options
	Selected        int             // For select fields: currently selected index
	DisabledOptions map[string]bool // For select fields: options that are disabled (grayed out)
}

// Form is a reusable form component.
type Form struct {
	Title   string
	Fields  []FormField
	focused int
	cursor  int // Cursor position in current field (text fields only)
}

// NewForm creates a new form with the given title and fields.
func NewForm(title string, fields []FormField) *Form {
	// Initialize select field selected index based on Value
	for i := range fields {
		if fields[i].Type == FieldSelect && fields[i].Value != "" {
			for j, opt := range fields[i].Options {
				if opt == fields[i].Value {
					fields[i].Selected = j
					break
				}
			}
		}
	}
	return &Form{
		Title:  title,
		Fields: fields,
	}
}

// Update handles input for the form.
// Returns true if Enter was pressed on a button field (submit).
func (f *Form) Update(msg tea.KeyMsg) bool {
	field := &f.Fields[f.focused]

	switch field.Type {
	case FieldSelect:
		return f.updateSelect(msg)
	case FieldButton:
		return f.updateButton(msg)
	default:
		return f.updateText(msg)
	}
}

// updateText handles input for text fields.
func (f *Form) updateText(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyTab, tea.KeyEnter:
		f.focused = (f.focused + 1) % len(f.Fields)
		f.cursor = len(f.Fields[f.focused].Value)
	case tea.KeyShiftTab:
		f.focused = (f.focused - 1 + len(f.Fields)) % len(f.Fields)
		f.cursor = len(f.Fields[f.focused].Value)
	case tea.KeyUp:
		f.focused = (f.focused - 1 + len(f.Fields)) % len(f.Fields)
		f.cursor = len(f.Fields[f.focused].Value)
	case tea.KeyDown:
		f.focused = (f.focused + 1) % len(f.Fields)
		f.cursor = len(f.Fields[f.focused].Value)
	case tea.KeyLeft:
		if f.cursor > 0 {
			f.cursor--
		}
	case tea.KeyRight:
		if f.cursor < len(f.Fields[f.focused].Value) {
			f.cursor++
		}
	case tea.KeyHome, tea.KeyCtrlA:
		f.cursor = 0
	case tea.KeyEnd, tea.KeyCtrlE:
		f.cursor = len(f.Fields[f.focused].Value)
	case tea.KeyBackspace:
		if f.cursor > 0 {
			val := f.Fields[f.focused].Value
			f.Fields[f.focused].Value = val[:f.cursor-1] + val[f.cursor:]
			f.cursor--
		}
	case tea.KeyDelete:
		val := f.Fields[f.focused].Value
		if f.cursor < len(val) {
			f.Fields[f.focused].Value = val[:f.cursor] + val[f.cursor+1:]
		}
	case tea.KeyRunes:
		// Insert runes at cursor position (handles both typing and paste)
		text := string(msg.Runes)
		val := f.Fields[f.focused].Value
		f.Fields[f.focused].Value = val[:f.cursor] + text + val[f.cursor:]
		f.cursor += len(text)
	}
	return false
}

// updateButton handles input for button fields.
func (f *Form) updateButton(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyTab:
		f.focused = (f.focused + 1) % len(f.Fields)
		f.cursor = len(f.Fields[f.focused].Value)
	case tea.KeyShiftTab, tea.KeyUp:
		f.focused = (f.focused - 1 + len(f.Fields)) % len(f.Fields)
		f.cursor = len(f.Fields[f.focused].Value)
	case tea.KeyDown:
		f.focused = (f.focused + 1) % len(f.Fields)
		f.cursor = len(f.Fields[f.focused].Value)
	case tea.KeyEnter:
		return true
	}
	return false
}

// updateSelect handles input for select fields.
func (f *Form) updateSelect(msg tea.KeyMsg) bool {
	field := &f.Fields[f.focused]

	switch msg.Type {
	case tea.KeyTab, tea.KeyEnter:
		// Move to next field (don't submit)
		f.focused = (f.focused + 1) % len(f.Fields)
		f.cursor = len(f.Fields[f.focused].Value)
	case tea.KeyShiftTab:
		f.focused = (f.focused - 1 + len(f.Fields)) % len(f.Fields)
		f.cursor = len(f.Fields[f.focused].Value)
	case tea.KeyUp, tea.KeyLeft:
		// Navigate options up
		if len(field.Options) > 0 {
			field.Selected = (field.Selected - 1 + len(field.Options)) % len(field.Options)
			field.Value = field.Options[field.Selected]
		}
	case tea.KeyDown, tea.KeyRight:
		// Navigate options down
		if len(field.Options) > 0 {
			field.Selected = (field.Selected + 1) % len(field.Options)
			field.Value = field.Options[field.Selected]
		}
	case tea.KeyRunes:
		// Handle j/k for vim-style navigation
		switch msg.String() {
		case "k":
			if len(field.Options) > 0 {
				field.Selected = (field.Selected - 1 + len(field.Options)) % len(field.Options)
				field.Value = field.Options[field.Selected]
			}
		case "j":
			if len(field.Options) > 0 {
				field.Selected = (field.Selected + 1) % len(field.Options)
				field.Value = field.Options[field.Selected]
			}
		}
	}
	return false
}

// SetFieldOptions updates the options for a select field and resets selection.
func (f *Form) SetFieldOptions(key string, options []string, defaultValue string) {
	for i := range f.Fields {
		if f.Fields[i].Key == key {
			f.Fields[i].Options = options
			f.Fields[i].Selected = 0
			f.Fields[i].Value = ""
			// Try to select the default value
			for j, opt := range options {
				if opt == defaultValue {
					f.Fields[i].Selected = j
					f.Fields[i].Value = opt
					break
				}
			}
			// If no default matched but we have options, select first
			if f.Fields[i].Value == "" && len(options) > 0 {
				f.Fields[i].Value = options[0]
			}
			break
		}
	}
}

// GetFieldValue returns the current value of a field.
func (f *Form) GetFieldValue(key string) string {
	for _, field := range f.Fields {
		if field.Key == key {
			return field.Value
		}
	}
	return ""
}

// SetFieldDisabledOptions sets which options are disabled for a select field.
func (f *Form) SetFieldDisabledOptions(key string, disabled map[string]bool) {
	for i := range f.Fields {
		if f.Fields[i].Key == key {
			f.Fields[i].DisabledOptions = disabled
			break
		}
	}
}

// IsSelectedDisabled returns true if the currently selected option is disabled.
func (f *Form) IsSelectedDisabled(key string) bool {
	for _, field := range f.Fields {
		if field.Key == key {
			if field.DisabledOptions == nil {
				return false
			}
			return field.DisabledOptions[field.Value]
		}
	}
	return false
}

// IsFieldFocused returns true if the field with the given key is currently focused.
func (f *Form) IsFieldFocused(key string) bool {
	if f.focused < 0 || f.focused >= len(f.Fields) {
		return false
	}
	return f.Fields[f.focused].Key == key
}

// Values returns all field values as a map (trimmed of whitespace).
func (f *Form) Values() map[string]string {
	result := make(map[string]string)
	for _, field := range f.Fields {
		result[field.Key] = strings.TrimSpace(field.Value)
	}
	return result
}

// View renders the form.
func (f *Form) View() string {
	var lines []string

	labelStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	valueStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	focusedValueStyle := lipgloss.NewStyle().Foreground(theme.Accent)
	cursorStyle := lipgloss.NewStyle().Foreground(theme.Accent).Underline(true)
	optionStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	selectedOptionStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	disabledStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary).Faint(true)

	for i, field := range f.Fields {
		isFocused := i == f.focused

		switch field.Type {
		case FieldSelect:
			lines = append(lines, f.renderSelectField(field, isFocused, labelStyle, valueStyle, focusedValueStyle, optionStyle, selectedOptionStyle, disabledStyle)...)
		case FieldButton:
			lines = append(lines, f.renderButtonField(field, isFocused, focusedValueStyle, labelStyle))
		default:
			lines = append(lines, f.renderTextField(field, isFocused, labelStyle, valueStyle, focusedValueStyle, cursorStyle))
		}
	}

	return strings.Join(lines, "\n")
}

// renderTextField renders a text input field.
func (f *Form) renderTextField(field FormField, isFocused bool, labelStyle, valueStyle, focusedValueStyle, cursorStyle lipgloss.Style) string {
	label := labelStyle.Render(field.Label + ":")

	val := field.Value
	if field.Password && val != "" {
		val = strings.Repeat("*", len(val))
	}

	var renderedValue string
	if isFocused {
		// Show cursor
		if f.cursor < len(val) {
			before := val[:f.cursor]
			cursorChar := string(val[f.cursor])
			after := val[f.cursor+1:]
			if field.Password {
				cursorChar = "*"
			}
			renderedValue = focusedValueStyle.Render(before) +
				cursorStyle.Render(cursorChar) +
				focusedValueStyle.Render(after)
		} else {
			renderedValue = focusedValueStyle.Render(val) + cursorStyle.Render(" ")
		}
	} else {
		renderedValue = valueStyle.Render(val)
		if val == "" {
			renderedValue = labelStyle.Render("(empty)")
		}
	}

	return "  " + label + " " + renderedValue
}

// renderSelectField renders a selection field with options.
func (f *Form) renderSelectField(field FormField, isFocused bool, labelStyle, valueStyle, focusedValueStyle, optionStyle, selectedOptionStyle, disabledStyle lipgloss.Style) []string {
	var lines []string

	label := labelStyle.Render(field.Label + ":")

	// Check if current value is disabled
	isDisabled := field.DisabledOptions != nil && field.DisabledOptions[field.Value]

	if !isFocused {
		// When not focused, show label and current value on one line
		val := field.Value
		if val == "" {
			val = "(none)"
		}
		if isDisabled {
			lines = append(lines, "  "+label+" "+disabledStyle.Render(val+" (not configured)"))
		} else {
			lines = append(lines, "  "+label+" "+valueStyle.Render(val))
		}
	} else {
		// When focused, show label and options below
		lines = append(lines, "  "+label)

		if len(field.Options) == 0 {
			lines = append(lines, "    "+optionStyle.Render("(no options available)"))
		} else {
			for j, opt := range field.Options {
				optDisabled := field.DisabledOptions != nil && field.DisabledOptions[opt]
				displayOpt := opt
				if optDisabled {
					displayOpt = opt + " (not configured)"
				}

				if j == field.Selected {
					if optDisabled {
						lines = append(lines, "    › "+disabledStyle.Render(displayOpt))
					} else {
						lines = append(lines, "    › "+selectedOptionStyle.Render(displayOpt))
					}
				} else {
					if optDisabled {
						lines = append(lines, "      "+disabledStyle.Render(displayOpt))
					} else {
						lines = append(lines, "      "+optionStyle.Render(opt))
					}
				}
			}
		}
	}

	return lines
}

// renderButtonField renders a button field.
func (f *Form) renderButtonField(field FormField, isFocused bool, focusedStyle, normalStyle lipgloss.Style) string {
	label := field.Label
	if isFocused {
		return "  " + focusedStyle.Render("[ "+label+" ]")
	}
	return "  " + normalStyle.Render("[ "+label+" ]")
}
