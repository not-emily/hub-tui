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

// StepProfilesLoadedMsg is sent when profiles are loaded for an integration.
type StepProfilesLoadedMsg struct {
	Integration string
	Profiles    []string
	Error       error
}

// StepSavedMsg is sent when a step is saved.
type StepSavedMsg struct {
	Step  *client.WorkflowStep
	IsNew bool
}

// StepTestedMsg is sent when a step test completes.
type StepTestedMsg struct {
	Result *client.StepTestResult
	Error  error
}

// Step form field indices
const (
	stepFieldName = iota
	stepFieldProfile
	stepFieldSaveAs
	stepFieldParamsStart
)

// StepForm handles editing a workflow step.
type StepForm struct {
	client *client.Client

	// Step being edited
	Step     *client.WorkflowStep
	Tool     *client.Tool
	ToolType string // "module", "integration", "utility", "primitive"
	IsNew    bool

	// Form state
	focusedField int
	editing      bool
	editBuffer   string
	cursorPos    int // cursor position within editBuffer

	// Profile options (for integration tools)
	profiles       []string
	profileIndex   int
	loadingProfile bool

	// Available variables (from previous steps)
	availableVars []string

	// Previous step outputs (for building variables map)
	previousOutputs map[string]interface{}

	// Test state
	testOutput  interface{}
	testError   string
	testLoading bool
	showOutput  bool

	// Field picker
	fieldPicker     *components.FieldPicker
	showFieldPicker bool

	// UI state
	err   string
	dirty bool
	saved bool
}

// NewStepForm creates a new step form.
func NewStepForm(c *client.Client, step *client.WorkflowStep, tool *client.Tool, toolType string, isNew bool) *StepForm {
	f := &StepForm{
		client:   c,
		Step:     step,
		Tool:     tool,
		ToolType: toolType,
		IsNew:    isNew,
	}

	// Initialize step from tool if new
	if isNew {
		f.Step.Type = toolType
		f.Step.Target = tool.Target
	}

	return f
}

// SetAvailableVariables sets the variables available from previous steps.
func (f *StepForm) SetAvailableVariables(vars []string) {
	f.availableVars = vars
}

// SetPreviousOutputs sets the outputs from previous steps for variable substitution.
func (f *StepForm) SetPreviousOutputs(outputs map[string]interface{}) {
	f.previousOutputs = outputs
}

// TestOutput returns the test output (for storing in builder).
func (f *StepForm) TestOutput() interface{} {
	return f.testOutput
}

// Saved returns true if the form was saved (not cancelled).
func (f *StepForm) Saved() bool {
	return f.saved
}

// Init initializes the form.
func (f *StepForm) Init() tea.Cmd {
	// If integration tool, load profiles
	if f.Tool.RequiresProfile {
		return f.loadProfiles()
	}
	return nil
}

func (f *StepForm) loadProfiles() tea.Cmd {
	f.loadingProfile = true
	integration := extractIntegration(f.Step.Target)
	return func() tea.Msg {
		profiles, err := f.client.GetIntegrationProfiles(integration)
		return StepProfilesLoadedMsg{
			Integration: integration,
			Profiles:    profiles,
			Error:       err,
		}
	}
}

// extractIntegration gets the integration name from a target like "notion.query_database"
func extractIntegration(target string) string {
	parts := strings.Split(target, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return target
}

// Update handles input.
func (f *StepForm) Update(msg tea.Msg) (*StepForm, tea.Cmd) {
	// Handle field picker if active
	if f.showFieldPicker && f.fieldPicker != nil {
		switch msg := msg.(type) {
		case components.FieldCancelledMsg:
			f.showFieldPicker = false
			f.fieldPicker = nil
			return f, nil
		case tea.KeyMsg:
			selectedPath, cmd := f.fieldPicker.Update(msg)
			if selectedPath != "" {
				// Field selected - insert into current param if editing, otherwise show it
				f.handleFieldSelected(selectedPath)
				f.showFieldPicker = false
				f.fieldPicker = nil
				return f, nil
			}
			return f, cmd
		}
		return f, nil
	}

	switch msg := msg.(type) {
	case StepProfilesLoadedMsg:
		f.loadingProfile = false
		if msg.Error != nil {
			f.err = msg.Error.Error()
		} else {
			f.profiles = msg.Profiles
			f.profileIndex = f.findProfileIndex(f.Step.Profile)
			// Set profile if not already set
			if f.Step.Profile == "" && len(f.profiles) > 0 {
				f.Step.Profile = f.profiles[0]
			}
		}
		return f, nil

	case StepTestedMsg:
		f.testLoading = false
		if msg.Error != nil {
			f.testError = msg.Error.Error()
			f.testOutput = nil
		} else if msg.Result != nil {
			if msg.Result.Success {
				f.testOutput = msg.Result.Output
				f.testError = ""
			} else {
				f.testError = msg.Result.Error
				f.testOutput = nil
			}
		}
		return f, nil

	case components.FieldCancelledMsg:
		f.showFieldPicker = false
		f.fieldPicker = nil
		return f, nil

	case tea.KeyMsg:
		return f.handleKeyPress(msg)
	}
	return f, nil
}

func (f *StepForm) findProfileIndex(profile string) int {
	for i, p := range f.profiles {
		if p == profile {
			return i
		}
	}
	return 0
}

func (f *StepForm) handleKeyPress(msg tea.KeyMsg) (*StepForm, tea.Cmd) {
	if f.editing {
		return f.handleEditInput(msg)
	}

	switch msg.String() {
	case "j", "down":
		f.focusedField = f.nextField()
	case "k", "up":
		f.focusedField = f.prevField()

	case "enter":
		return f.handleEnter()

	case "h", "left":
		if f.focusedField == stepFieldProfile && len(f.profiles) > 0 {
			if f.profileIndex > 0 {
				f.profileIndex--
				f.Step.Profile = f.profiles[f.profileIndex]
				f.dirty = true
			}
		}
	case "l", "right":
		if f.focusedField == stepFieldProfile && len(f.profiles) > 0 {
			if f.profileIndex < len(f.profiles)-1 {
				f.profileIndex++
				f.Step.Profile = f.profiles[f.profileIndex]
				f.dirty = true
			}
		}

	case "t":
		// Test step
		if !f.canTest() {
			f.err = "Fill required fields before testing"
			return f, nil
		}
		return f, f.testStep()

	case "o":
		// Toggle output visibility
		if f.testOutput != nil || f.testError != "" {
			f.showOutput = !f.showOutput
		}

	case "v":
		// Pick variable from previous steps (only when on a param field)
		paramIndex := f.focusedField - stepFieldParamsStart
		if paramIndex >= 0 && paramIndex < len(f.Tool.Params) && len(f.previousOutputs) > 0 {
			// Build combined output from all previous steps
			f.fieldPicker = components.NewFieldPickerMulti(f.previousOutputs)
			f.showFieldPicker = true
		}

	case "ctrl+s":
		return f.saveStep()

	case "esc":
		return nil, nil // Cancel
	}

	return f, nil
}

func (f *StepForm) handleEditInput(msg tea.KeyMsg) (*StepForm, tea.Cmd) {
	switch msg.String() {
	case "enter":
		f.applyEdit()
		f.editing = false
		f.cursorPos = 0
	case "esc":
		f.editing = false
		f.editBuffer = ""
		f.cursorPos = 0
	case "left":
		if f.cursorPos > 0 {
			f.cursorPos--
		}
	case "right":
		if f.cursorPos < len(f.editBuffer) {
			f.cursorPos++
		}
	case "home", "ctrl+a":
		f.cursorPos = 0
	case "end", "ctrl+e":
		f.cursorPos = len(f.editBuffer)
	case "backspace":
		if f.cursorPos > 0 {
			f.editBuffer = f.editBuffer[:f.cursorPos-1] + f.editBuffer[f.cursorPos:]
			f.cursorPos--
		}
	case "delete":
		if f.cursorPos < len(f.editBuffer) {
			f.editBuffer = f.editBuffer[:f.cursorPos] + f.editBuffer[f.cursorPos+1:]
		}
	default:
		if len(msg.String()) == 1 {
			// Insert at cursor position
			f.editBuffer = f.editBuffer[:f.cursorPos] + msg.String() + f.editBuffer[f.cursorPos:]
			f.cursorPos++
		}
	}
	return f, nil
}

func (f *StepForm) applyEdit() {
	switch f.focusedField {
	case stepFieldName:
		if f.editBuffer != f.Step.Name {
			f.Step.Name = f.editBuffer
			f.dirty = true
		}
	case stepFieldSaveAs:
		if f.editBuffer != f.Step.SaveAs {
			f.Step.SaveAs = f.editBuffer
			f.dirty = true
		}
	default:
		// Param field
		paramIndex := f.focusedField - stepFieldParamsStart
		if paramIndex >= 0 && paramIndex < len(f.Tool.Params) {
			param := f.Tool.Params[paramIndex]
			f.setParamValue(param.Name, f.editBuffer, param.Type)
			f.dirty = true
		}
	}
	f.editBuffer = ""
}

func (f *StepForm) handleEnter() (*StepForm, tea.Cmd) {
	switch f.focusedField {
	case stepFieldName:
		f.editing = true
		f.editBuffer = f.Step.Name
		f.cursorPos = len(f.editBuffer)
	case stepFieldSaveAs:
		f.editing = true
		f.editBuffer = f.Step.SaveAs
		f.cursorPos = len(f.editBuffer)
	case stepFieldProfile:
		// Profile uses left/right, not enter
	default:
		// Param field
		paramIndex := f.focusedField - stepFieldParamsStart
		if paramIndex >= 0 && paramIndex < len(f.Tool.Params) {
			param := f.Tool.Params[paramIndex]
			if param.Type == "boolean" {
				// Toggle boolean
				current := f.getParamValue(param.Name) == "true"
				f.setParamValue(param.Name, strconv.FormatBool(!current), "boolean")
				f.dirty = true
			} else {
				f.editing = true
				f.editBuffer = f.getParamValue(param.Name)
				f.cursorPos = len(f.editBuffer)
			}
		}
	}
	return f, nil
}

func (f *StepForm) nextField() int {
	maxField := f.maxField()
	next := f.focusedField + 1

	// Skip profile field if not an integration
	if next == stepFieldProfile && !f.Tool.RequiresProfile {
		next++
	}

	if next > maxField {
		return maxField
	}
	return next
}

func (f *StepForm) prevField() int {
	prev := f.focusedField - 1

	// Skip profile field if not an integration
	if prev == stepFieldProfile && !f.Tool.RequiresProfile {
		prev--
	}

	if prev < stepFieldName {
		return stepFieldName
	}
	return prev
}

func (f *StepForm) maxField() int {
	return stepFieldParamsStart + len(f.Tool.Params)
}

func (f *StepForm) saveStep() (*StepForm, tea.Cmd) {
	// Validate required fields
	for _, param := range f.Tool.Params {
		if param.Required {
			val := f.getParamValue(param.Name)
			if val == "" {
				f.err = param.Name + " is required"
				return f, nil
			}
		}
	}

	// Profile required for integration tools
	if f.Tool.RequiresProfile && f.Step.Profile == "" {
		f.err = "Profile is required"
		return f, nil
	}

	f.saved = true
	return nil, nil
}

// canTest returns true if the step has all required fields filled.
func (f *StepForm) canTest() bool {
	// Check required params
	for _, param := range f.Tool.Params {
		if param.Required {
			val := f.getParamValue(param.Name)
			if val == "" {
				return false
			}
		}
	}

	// Check profile for integrations
	if f.Tool.RequiresProfile && f.Step.Profile == "" {
		return false
	}

	return true
}

// testStep executes the step and returns the test command.
func (f *StepForm) testStep() tea.Cmd {
	f.testLoading = true
	f.testError = ""
	f.showOutput = true

	return func() tea.Msg {
		// Build variables map from previous steps
		variables := make(map[string]interface{})
		for name, output := range f.previousOutputs {
			variables[name] = output
		}

		result, err := f.client.TestStep(&client.StepTestRequest{
			Step:      *f.Step,
			Variables: variables,
		})

		return StepTestedMsg{
			Result: result,
			Error:  err,
		}
	}
}

// handleFieldSelected inserts the selected variable path into the focused param.
func (f *StepForm) handleFieldSelected(path string) {
	paramIndex := f.focusedField - stepFieldParamsStart
	if paramIndex >= 0 && paramIndex < len(f.Tool.Params) {
		param := f.Tool.Params[paramIndex]
		// Replace the param value with the variable path
		f.setParamValue(param.Name, path, param.Type)
		f.dirty = true
	}
}

func (f *StepForm) getParamValue(name string) string {
	if f.Step.Params == nil {
		return ""
	}
	val, ok := f.Step.Params[name]
	if !ok {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return fmt.Sprintf("%v", v)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func (f *StepForm) setParamValue(name, value string, paramType string) {
	if f.Step.Params == nil {
		f.Step.Params = make(map[string]interface{})
	}

	switch paramType {
	case "number":
		if n, err := strconv.ParseFloat(value, 64); err == nil {
			f.Step.Params[name] = n
		} else {
			f.Step.Params[name] = value
		}
	case "boolean":
		f.Step.Params[name] = value == "true"
	case "array", "object":
		var parsed interface{}
		if err := json.Unmarshal([]byte(value), &parsed); err == nil {
			f.Step.Params[name] = parsed
		} else {
			f.Step.Params[name] = value
		}
	default:
		f.Step.Params[name] = value
	}
}

// View renders the step form.
func (f *StepForm) View() string {
	// Show field picker if active
	if f.showFieldPicker && f.fieldPicker != nil {
		return f.fieldPicker.View()
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	errorStyle := lipgloss.NewStyle().Foreground(theme.Error)

	var lines []string

	// Header
	title := "Edit Step"
	if f.IsNew {
		title = "New Step"
	}
	lines = append(lines, headerStyle.Render(title))
	lines = append(lines, "")

	// Tool info
	lines = append(lines, "Tool: "+f.Tool.Target)
	if f.Tool.Description != "" {
		lines = append(lines, dimStyle.Render(f.Tool.Description))
	}
	lines = append(lines, "")

	// Available variables
	if len(f.availableVars) > 0 {
		vars := strings.Join(f.availableVars, ", ")
		lines = append(lines, dimStyle.Render("Available: "+vars))
		lines = append(lines, "")
	}

	// Name field
	lines = append(lines, f.renderField("Name", f.Step.Name, stepFieldName, selectedStyle, dimStyle))

	// Profile dropdown (for integrations)
	if f.Tool.RequiresProfile {
		lines = append(lines, f.renderProfileField(selectedStyle, dimStyle, errorStyle))
	}

	// SaveAs field
	lines = append(lines, f.renderField("Save as", f.Step.SaveAs, stepFieldSaveAs, selectedStyle, dimStyle))

	// Dynamic params
	if len(f.Tool.Params) > 0 {
		lines = append(lines, "")
		lines = append(lines, dimStyle.Render("Parameters:"))
		for i, param := range f.Tool.Params {
			fieldIndex := stepFieldParamsStart + i
			value := f.getParamValue(param.Name)
			lines = append(lines, f.renderParamField(param, value, fieldIndex, selectedStyle, dimStyle))
		}
	}

	lines = append(lines, "")

	// Error display
	if f.err != "" {
		lines = append(lines, errorStyle.Render("Error: "+f.err))
		lines = append(lines, "")
	}

	// Test output section
	if f.showOutput || f.testLoading {
		lines = append(lines, f.renderTestOutput(dimStyle, errorStyle))
		lines = append(lines, "")
	}

	// Hints
	lines = append(lines, f.renderHints(dimStyle))

	return strings.Join(lines, "\n")
}

func (f *StepForm) renderField(label, value string, fieldIndex int, selectedStyle, dimStyle lipgloss.Style) string {
	display := value
	if display == "" {
		display = "(empty)"
	}

	hint := ""
	if f.editing && f.focusedField == fieldIndex {
		// Show cursor at position
		display = f.editBuffer[:f.cursorPos] + "|" + f.editBuffer[f.cursorPos:]
		hint = "  " + dimStyle.Render("[Enter] Confirm")
	} else if f.focusedField == fieldIndex {
		hint = "  " + dimStyle.Render("[Enter] Edit")
	}

	line := fmt.Sprintf("%-10s %s", label+":", display)

	if f.focusedField == fieldIndex {
		return selectedStyle.Render("> "+line) + hint
	}
	return "  " + line
}

func (f *StepForm) renderParamField(param client.ToolParam, value string, fieldIndex int, selectedStyle, dimStyle lipgloss.Style) string {
	label := param.Name
	if param.Required {
		label += "*"
	}

	display := value
	if display == "" {
		if param.Default != nil {
			display = fmt.Sprintf("%v", param.Default)
		} else {
			display = "(empty)"
		}
	}

	hint := ""
	if f.editing && f.focusedField == fieldIndex {
		// Show cursor at position
		display = f.editBuffer[:f.cursorPos] + "|" + f.editBuffer[f.cursorPos:]
		hint = "  " + dimStyle.Render("[Enter] Confirm  [←/→] Move cursor")
	} else if f.focusedField == fieldIndex {
		if param.Type == "boolean" {
			hint = "  " + dimStyle.Render("[Enter] Toggle")
		} else {
			hint = "  " + dimStyle.Render("[Enter] Edit")
			// Show variable option if previous outputs exist
			if len(f.previousOutputs) > 0 {
				hint += "  " + dimStyle.Render("[v] Variable")
			}
		}
		// Show description for focused param
		if param.Description != "" {
			hint += "\n    " + dimStyle.Render(param.Description)
		}
	}

	line := fmt.Sprintf("  %-12s %s", label+":", display)

	if f.focusedField == fieldIndex {
		return selectedStyle.Render("> "+line[2:]) + hint
	}
	return line
}

func (f *StepForm) renderProfileField(selectedStyle, dimStyle, errorStyle lipgloss.Style) string {
	label := "Profile:"

	if f.loadingProfile {
		line := fmt.Sprintf("%-10s %s", label, dimStyle.Render("loading..."))
		if f.focusedField == stepFieldProfile {
			return selectedStyle.Render("> " + line)
		}
		return "  " + line
	}

	if len(f.profiles) == 0 {
		line := fmt.Sprintf("%-10s %s", label, errorStyle.Render("(no profiles configured)"))
		if f.focusedField == stepFieldProfile {
			return selectedStyle.Render("> " + line)
		}
		return "  " + line
	}

	// Show current profile with arrow indicators
	current := f.profiles[f.profileIndex]
	leftArrow := " "
	rightArrow := " "
	if f.profileIndex > 0 {
		leftArrow = "◀"
	}
	if f.profileIndex < len(f.profiles)-1 {
		rightArrow = "▶"
	}

	display := fmt.Sprintf("%s %s %s", leftArrow, current, rightArrow)

	line := fmt.Sprintf("%-10s %s", label, display)
	if f.focusedField == stepFieldProfile {
		return selectedStyle.Render("> " + line)
	}
	return "  " + line
}

func (f *StepForm) renderTestOutput(dimStyle, errorStyle lipgloss.Style) string {
	var lines []string

	lines = append(lines, "─── Test Output ───")

	if f.testLoading {
		lines = append(lines, dimStyle.Render("Running..."))
		return strings.Join(lines, "\n")
	}

	if f.testError != "" {
		lines = append(lines, errorStyle.Render("Error: "+f.testError))
		return strings.Join(lines, "\n")
	}

	if f.testOutput == nil {
		lines = append(lines, dimStyle.Render("Press [t] to test"))
		return strings.Join(lines, "\n")
	}

	// Format output nicely
	output := f.formatOutput(f.testOutput)
	lines = append(lines, output)

	return strings.Join(lines, "\n")
}

func (f *StepForm) formatOutput(output interface{}) string {
	// Pretty print JSON with indentation
	b, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", output)
	}

	// Truncate if too long
	s := string(b)
	maxLines := 15
	lines := strings.Split(s, "\n")
	if len(lines) > maxLines {
		truncated := strings.Join(lines[:maxLines], "\n")
		return truncated + "\n... (" + strconv.Itoa(len(lines)-maxLines) + " more lines)"
	}

	return s
}

func (f *StepForm) renderHints(dimStyle lipgloss.Style) string {
	if f.editing {
		return dimStyle.Render("[Enter] Confirm  [Esc] Cancel")
	}

	hints := "[j/k] Navigate  [Enter] Edit"

	if f.canTest() {
		hints += "  [t] Test"
	}

	if f.testOutput != nil || f.testError != "" {
		hints += "  [o] Toggle output"
	}

	hints += "  [Ctrl+s] Save  [Esc] Back"

	return dimStyle.Render(hints)
}
