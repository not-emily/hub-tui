package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/ui/theme"
)

// FormField represents a single form field.
type FormField struct {
	Label    string
	Key      string
	Value    string
	Password bool // Mask input with asterisks
}

// Form is a reusable form component.
type Form struct {
	Title   string
	Fields  []FormField
	focused int
	cursor  int // Cursor position in current field
}

// NewForm creates a new form with the given title and fields.
func NewForm(title string, fields []FormField) *Form {
	return &Form{
		Title:  title,
		Fields: fields,
	}
}

// Update handles input for the form.
// Returns true if Enter was pressed (submit).
func (f *Form) Update(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyTab:
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
	case tea.KeyEnter:
		return true
	case tea.KeyRunes:
		// Insert runes at cursor position (handles both typing and paste)
		text := string(msg.Runes)
		val := f.Fields[f.focused].Value
		f.Fields[f.focused].Value = val[:f.cursor] + text + val[f.cursor:]
		f.cursor += len(text)
	}
	return false
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

	for i, field := range f.Fields {
		// Render label
		label := labelStyle.Render(field.Label + ":")

		// Render value
		val := field.Value
		if field.Password && val != "" {
			val = strings.Repeat("*", len(val))
		}

		var renderedValue string
		if i == f.focused {
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

		lines = append(lines, "  "+label+" "+renderedValue)
	}

	return strings.Join(lines, "\n")
}
