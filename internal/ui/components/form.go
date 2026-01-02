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
	FieldText     FieldType = iota // Text input field
	FieldSelect                    // Selection field with options
	FieldButton                    // Button (e.g., Save, Cancel)
	FieldCheckbox                  // Checkbox (toggle with space or enter)
	FieldTextArea                  // Multi-line text input field
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
	Checked         bool            // For checkbox fields: whether the checkbox is checked

	// Extended fields for parameter forms
	Required    bool   // Show required indicator, used for validation
	Error       string // Validation error to display below field
	Description string // Help text shown below field
	ParamType   string // Original param type: "string", "number", "boolean", "array", "object"
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
	case FieldCheckbox:
		return f.updateCheckbox(msg)
	case FieldTextArea:
		return f.updateTextArea(msg)
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
	case tea.KeySpace:
		// Insert space at cursor position
		val := f.Fields[f.focused].Value
		f.Fields[f.focused].Value = val[:f.cursor] + " " + val[f.cursor:]
		f.cursor++
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

// updateCheckbox handles input for checkbox fields.
func (f *Form) updateCheckbox(msg tea.KeyMsg) bool {
	field := &f.Fields[f.focused]

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
	case tea.KeySpace, tea.KeyEnter:
		field.Checked = !field.Checked
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

// updateTextArea handles input for multi-line text area fields.
// Enter inserts newlines, Tab/Shift+Tab moves between fields.
func (f *Form) updateTextArea(msg tea.KeyMsg) bool {
	field := &f.Fields[f.focused]

	switch msg.Type {
	case tea.KeyTab:
		f.focused = (f.focused + 1) % len(f.Fields)
		f.cursor = len(f.Fields[f.focused].Value)
	case tea.KeyShiftTab:
		f.focused = (f.focused - 1 + len(f.Fields)) % len(f.Fields)
		f.cursor = len(f.Fields[f.focused].Value)
	case tea.KeyEnter:
		// Insert newline at cursor position
		val := field.Value
		field.Value = val[:f.cursor] + "\n" + val[f.cursor:]
		f.cursor++
	case tea.KeyUp:
		// Move cursor up one line
		f.moveCursorVertical(field, -1)
	case tea.KeyDown:
		// Move cursor down one line
		f.moveCursorVertical(field, 1)
	case tea.KeyLeft:
		if f.cursor > 0 {
			f.cursor--
		}
	case tea.KeyRight:
		if f.cursor < len(field.Value) {
			f.cursor++
		}
	case tea.KeyHome, tea.KeyCtrlA:
		// Move to start of current line
		f.cursor = f.findLineStart(field.Value, f.cursor)
	case tea.KeyEnd, tea.KeyCtrlE:
		// Move to end of current line
		f.cursor = f.findLineEnd(field.Value, f.cursor)
	case tea.KeyBackspace:
		if f.cursor > 0 {
			val := field.Value
			field.Value = val[:f.cursor-1] + val[f.cursor:]
			f.cursor--
		}
	case tea.KeyDelete:
		val := field.Value
		if f.cursor < len(val) {
			field.Value = val[:f.cursor] + val[f.cursor+1:]
		}
	case tea.KeySpace:
		// Insert space at cursor position
		val := field.Value
		field.Value = val[:f.cursor] + " " + val[f.cursor:]
		f.cursor++
	case tea.KeyRunes:
		// Insert runes at cursor position (handles both typing and paste)
		text := string(msg.Runes)
		val := field.Value
		field.Value = val[:f.cursor] + text + val[f.cursor:]
		f.cursor += len(text)
	}
	return false
}

// moveCursorVertical moves the cursor up or down by the given number of lines.
func (f *Form) moveCursorVertical(field *FormField, direction int) {
	lines := strings.Split(field.Value, "\n")
	if len(lines) <= 1 {
		return
	}

	// Find current line and column
	currentLine := 0
	currentCol := f.cursor
	pos := 0
	for i, line := range lines {
		lineLen := len(line)
		if i < len(lines)-1 {
			lineLen++ // Account for newline
		}
		if pos+lineLen > f.cursor {
			currentLine = i
			currentCol = f.cursor - pos
			break
		}
		pos += lineLen
	}

	// Calculate new line
	newLine := currentLine + direction
	if newLine < 0 {
		newLine = 0
	}
	if newLine >= len(lines) {
		newLine = len(lines) - 1
	}

	// Calculate new cursor position
	newCol := currentCol
	if newCol > len(lines[newLine]) {
		newCol = len(lines[newLine])
	}

	// Calculate absolute position
	newPos := 0
	for i := 0; i < newLine; i++ {
		newPos += len(lines[i]) + 1 // +1 for newline
	}
	newPos += newCol
	f.cursor = newPos
}

// findLineStart finds the start position of the line containing the cursor.
func (f *Form) findLineStart(value string, cursor int) int {
	for i := cursor - 1; i >= 0; i-- {
		if value[i] == '\n' {
			return i + 1
		}
	}
	return 0
}

// findLineEnd finds the end position of the line containing the cursor.
func (f *Form) findLineEnd(value string, cursor int) int {
	for i := cursor; i < len(value); i++ {
		if value[i] == '\n' {
			return i
		}
	}
	return len(value)
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

// GetFieldChecked returns whether a checkbox field is checked.
func (f *Form) GetFieldChecked(key string) bool {
	for _, field := range f.Fields {
		if field.Key == key {
			return field.Checked
		}
	}
	return false
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

// ValidateRequired checks all required fields have values.
// Returns map of field key -> error message for empty required fields.
func (f *Form) ValidateRequired() map[string]string {
	errors := make(map[string]string)
	for _, field := range f.Fields {
		if field.Required && strings.TrimSpace(field.Value) == "" && !field.Checked {
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

// SetFieldValue sets the value for a field by key.
func (f *Form) SetFieldValue(key, value string) {
	for i := range f.Fields {
		if f.Fields[i].Key == key {
			f.Fields[i].Value = value
			break
		}
	}
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
	errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
	descStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary).Faint(true)
	requiredStyle := lipgloss.NewStyle().Foreground(theme.Error)

	for i, field := range f.Fields {
		isFocused := i == f.focused

		switch field.Type {
		case FieldSelect:
			lines = append(lines, f.renderSelectField(field, isFocused, labelStyle, valueStyle, focusedValueStyle, optionStyle, selectedOptionStyle, disabledStyle)...)
		case FieldButton:
			lines = append(lines, f.renderButtonField(field, isFocused, focusedValueStyle, labelStyle))
		case FieldCheckbox:
			lines = append(lines, f.renderCheckboxField(field, isFocused, labelStyle, focusedValueStyle, requiredStyle, errorStyle)...)
		case FieldTextArea:
			lines = append(lines, f.renderTextAreaField(field, isFocused, labelStyle, valueStyle, focusedValueStyle, cursorStyle, requiredStyle, errorStyle, descStyle)...)
		default:
			lines = append(lines, f.renderTextField(field, isFocused, labelStyle, valueStyle, focusedValueStyle, cursorStyle, requiredStyle, errorStyle, descStyle)...)
		}
	}

	return strings.Join(lines, "\n")
}

// renderTextField renders a text input field.
func (f *Form) renderTextField(field FormField, isFocused bool, labelStyle, valueStyle, focusedValueStyle, cursorStyle, requiredStyle, errorStyle, descStyle lipgloss.Style) []string {
	var lines []string

	// Build label with required indicator
	labelText := field.Label
	if field.Required {
		labelText += requiredStyle.Render("*")
	}
	label := labelStyle.Render(labelText + ":")

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

	lines = append(lines, "  "+label+" "+renderedValue)

	// Show error if present
	if field.Error != "" {
		lines = append(lines, "    "+errorStyle.Render("! "+field.Error))
	}

	// Show description if focused and no error
	if isFocused && field.Description != "" && field.Error == "" {
		lines = append(lines, "    "+descStyle.Render(field.Description))
	}

	return lines
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

// renderCheckboxField renders a checkbox field.
func (f *Form) renderCheckboxField(field FormField, isFocused bool, labelStyle, focusedStyle, requiredStyle, errorStyle lipgloss.Style) []string {
	var lines []string

	// Checkbox indicator: [x] for checked, [ ] for unchecked
	var checkbox string
	if field.Checked {
		checkbox = "[x]"
	} else {
		checkbox = "[ ]"
	}

	// Build label with required indicator
	labelText := field.Label
	if field.Required {
		labelText += requiredStyle.Render("*")
	}

	if isFocused {
		lines = append(lines, "  "+focusedStyle.Render(checkbox+" "+labelText))
	} else {
		lines = append(lines, "  "+labelStyle.Render(checkbox+" "+labelText))
	}

	// Show error if present
	if field.Error != "" {
		lines = append(lines, "    "+errorStyle.Render("! "+field.Error))
	}

	return lines
}

// renderTextAreaField renders a multi-line text area field.
func (f *Form) renderTextAreaField(field FormField, isFocused bool, labelStyle, valueStyle, focusedValueStyle, cursorStyle, requiredStyle, errorStyle, descStyle lipgloss.Style) []string {
	var lines []string

	// Build label with required indicator
	labelText := field.Label
	if field.Required {
		labelText += requiredStyle.Render("*")
	}
	label := labelStyle.Render(labelText + ":")
	lines = append(lines, "  "+label)

	// Split value into lines for rendering
	textLines := strings.Split(field.Value, "\n")
	if len(textLines) == 0 {
		textLines = []string{""}
	}

	// Calculate cursor position (line and column)
	cursorLine := 0
	cursorCol := f.cursor
	pos := 0
	for i, line := range textLines {
		lineLen := len(line)
		if i < len(textLines)-1 {
			lineLen++ // Account for newline
		}
		if pos+lineLen > f.cursor || i == len(textLines)-1 {
			cursorLine = i
			cursorCol = f.cursor - pos
			if cursorCol > len(line) {
				cursorCol = len(line)
			}
			break
		}
		pos += lineLen
	}

	// Render each line of the textarea
	const maxVisibleLines = 5
	borderStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	// Calculate visible range
	startLine := 0
	if cursorLine >= maxVisibleLines {
		startLine = cursorLine - maxVisibleLines + 1
	}
	endLine := startLine + maxVisibleLines
	if endLine > len(textLines) {
		endLine = len(textLines)
	}

	// Top border
	lines = append(lines, "    "+borderStyle.Render("┌────────────────────────────────────────┐"))

	// Render visible lines
	for i := startLine; i < endLine; i++ {
		line := textLines[i]
		if len(line) > 38 {
			line = line[:38]
		}

		var renderedLine string
		if isFocused && i == cursorLine {
			// Show cursor on this line
			if cursorCol < len(line) {
				before := line[:cursorCol]
				cursorChar := string(line[cursorCol])
				after := ""
				if cursorCol+1 < len(line) {
					after = line[cursorCol+1:]
				}
				renderedLine = focusedValueStyle.Render(before) +
					cursorStyle.Render(cursorChar) +
					focusedValueStyle.Render(after)
			} else {
				renderedLine = focusedValueStyle.Render(line) + cursorStyle.Render(" ")
			}
		} else if isFocused {
			renderedLine = focusedValueStyle.Render(line)
		} else {
			renderedLine = valueStyle.Render(line)
		}

		// Pad line to fill the box
		padding := 38 - lipgloss.Width(line)
		if padding < 0 {
			padding = 0
		}
		renderedLine += strings.Repeat(" ", padding)

		lines = append(lines, "    "+borderStyle.Render("│ ")+renderedLine+borderStyle.Render(" │"))
	}

	// Fill remaining lines if less than maxVisibleLines
	for i := endLine - startLine; i < maxVisibleLines; i++ {
		emptyLine := strings.Repeat(" ", 38)
		if isFocused && len(textLines) == 1 && textLines[0] == "" && i == 0 {
			emptyLine = cursorStyle.Render(" ") + strings.Repeat(" ", 37)
		}
		lines = append(lines, "    "+borderStyle.Render("│ ")+emptyLine+borderStyle.Render(" │"))
	}

	// Bottom border
	lines = append(lines, "    "+borderStyle.Render("└────────────────────────────────────────┘"))

	// Show error if present
	if field.Error != "" {
		lines = append(lines, "    "+errorStyle.Render("! "+field.Error))
	}

	// Show description
	if field.Description != "" && field.Error == "" {
		lines = append(lines, "    "+descStyle.Render(field.Description))
	}

	return lines
}
