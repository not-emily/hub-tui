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

func (m StepProfilesLoadedMsg) IsAsyncModalMessage() {}
func (m StepProfilesLoadedMsg) AuthError() error     { return m.Error }

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

func (m StepTestedMsg) IsAsyncModalMessage() {}
func (m StepTestedMsg) AuthError() error     { return m.Error }

// Step form field indices
const (
	stepFieldName = iota
	stepFieldProfile
	stepFieldSaveAs
	stepFieldParamsStart
)

// expandedField represents a single navigable field in the form,
// including nested object properties and array items.
type expandedField struct {
	param      *client.ToolParam // The param this field belongs to
	propName   string            // For nested object property (empty for top-level)
	arrayIndex int               // Array item index (-1 if not an array item)
	parentName string            // Parent param name (for nested fields)
	indent     int               // Visual indent level
}

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
	varToStepName   map[string]string // variable name -> step name mapping

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

	// Array item counts (for dynamic array editing)
	arrayItemCounts map[string]int

	// Transform mode (for data.json_transform tool)
	isTransformTool bool
	transformForm   *TransformForm
}

// NewStepForm creates a new step form.
func NewStepForm(c *client.Client, step *client.WorkflowStep, tool *client.Tool, toolType string, isNew bool) *StepForm {
	f := &StepForm{
		client:          c,
		Step:            step,
		Tool:            tool,
		ToolType:        toolType,
		IsNew:           isNew,
		arrayItemCounts: make(map[string]int),
	}

	// Initialize step from tool if new
	if isNew {
		f.Step.Type = toolType
		f.Step.Target = tool.Target
	}

	// Check if this is the data.json_transform tool - use special transform UI
	if tool.Target == "data.json_transform" {
		f.isTransformTool = true
		if isNew {
			// New step - show operation picker
			f.transformForm = NewTransformForm(c)
			f.transformForm.Name = step.Name
			f.transformForm.SaveAs = step.SaveAs
		} else {
			// Existing step - try to parse operation from params
			f.transformForm = NewTransformFormFromStep(c, step)
			if f.transformForm == nil {
				// Couldn't parse - fall back to raw params editing
				f.isTransformTool = false
			}
		}
	}

	// Initialize array item counts from existing params
	f.initArrayCounts()

	return f
}

// initArrayCounts initializes array item counts from existing step params.
func (f *StepForm) initArrayCounts() {
	if f.Step.Params == nil {
		return
	}
	for _, param := range f.Tool.Params {
		if param.Type == "array" && param.Items != nil {
			if val, ok := f.Step.Params[param.Name]; ok {
				if arr, ok := val.([]interface{}); ok {
					f.arrayItemCounts[param.Name] = len(arr)
				}
			}
		}
	}
}

// buildExpandedFields builds the list of navigable fields including nested ones.
func (f *StepForm) buildExpandedFields() []expandedField {
	var fields []expandedField

	for i := range f.Tool.Params {
		param := &f.Tool.Params[i]
		fields = append(fields, f.expandParam(param, "", 0)...)
	}

	return fields
}

// expandParam expands a single param into one or more fields.
func (f *StepForm) expandParam(param *client.ToolParam, parentName string, indent int) []expandedField {
	var fields []expandedField

	// Determine the full param key
	paramKey := param.Name
	if parentName != "" {
		paramKey = parentName + "." + param.Name
	}

	// Handle object with properties - expand into nested fields
	if param.Type == "object" && len(param.Properties) > 0 {
		// Add header field for the object (not editable, just for navigation context)
		fields = append(fields, expandedField{
			param:      param,
			parentName: parentName,
			arrayIndex: -1,
			indent:     indent,
		})
		// Add each property as a sub-field
		for i := range param.Properties {
			prop := &param.Properties[i]
			// For nested properties, recurse (but limit depth)
			if prop.Type == "object" && len(prop.Properties) > 0 && indent < 2 {
				fields = append(fields, f.expandParam(prop, paramKey, indent+1)...)
			} else {
				fields = append(fields, expandedField{
					param:      prop,
					parentName: paramKey,
					arrayIndex: -1,
					indent:     indent + 1,
				})
			}
		}
		return fields
	}

	// Handle array with items - expand into item fields
	if param.Type == "array" && param.Items != nil {
		// Add header field for the array (shows [a] Add hint)
		fields = append(fields, expandedField{
			param:      param,
			parentName: parentName,
			arrayIndex: -1,
			indent:     indent,
		})

		// Add fields for each existing item
		itemCount := f.arrayItemCounts[paramKey]
		for i := 0; i < itemCount; i++ {
			if param.Items.Type == "object" && len(param.Items.Properties) > 0 {
				// Object items - expand properties for each item
				for j := range param.Items.Properties {
					prop := &param.Items.Properties[j]
					fields = append(fields, expandedField{
						param:      prop,
						parentName: paramKey,
						arrayIndex: i,
						indent:     indent + 1,
					})
				}
			} else {
				// Simple items - one field per item
				fields = append(fields, expandedField{
					param:      param.Items,
					parentName: paramKey,
					arrayIndex: i,
					indent:     indent + 1,
				})
			}
		}
		return fields
	}

	// Simple param (or object/array without schema - JSON editor)
	fields = append(fields, expandedField{
		param:      param,
		parentName: parentName,
		arrayIndex: -1,
		indent:     indent,
	})

	return fields
}

// SetAvailableVariables sets the variables available from previous steps.
func (f *StepForm) SetAvailableVariables(vars []string) {
	f.availableVars = vars
	if f.transformForm != nil {
		f.transformForm.SetAvailableVariables(vars)
	}
}

// SetPreviousOutputs sets the outputs from previous steps for variable substitution.
func (f *StepForm) SetPreviousOutputs(outputs map[string]interface{}) {
	f.previousOutputs = outputs
	if f.transformForm != nil {
		f.transformForm.SetPreviousOutputs(outputs)
	}
}

// SetVarToStepName sets the mapping from variable names to step names.
func (f *StepForm) SetVarToStepName(mapping map[string]string) {
	f.varToStepName = mapping
	if f.transformForm != nil {
		f.transformForm.SetVarToStepName(mapping)
	}
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
	debugLog(fmt.Sprintf("StepForm.Update: msg=%T, isTransformTool=%v, transformForm=%v", msg, f.isTransformTool, f.transformForm != nil))

	// Delegate to transform form if in transform mode
	if f.isTransformTool && f.transformForm != nil {
		switch msg := msg.(type) {
		case TransformPreviewedMsg:
			form, cmd := f.transformForm.Update(msg)
			f.transformForm = form
			return f, cmd

		case TransformSavedMsg:
			// Transform saved - copy step data from transform form
			if msg.Step != nil {
				f.Step.Name = msg.Step.Name
				f.Step.SaveAs = msg.Step.SaveAs
				f.Step.Params = msg.Step.Params
				f.Step.Type = msg.Step.Type
				f.Step.Target = msg.Step.Target
			}
			f.saved = true
			return nil, nil

		case PickerTestRequestedMsg:
			// Bubble up to WorkflowBuilder
			return f, func() tea.Msg { return msg }

		case components.PickerNeedsTestMsg:
			// Forward to transform form, which will convert to PickerTestRequestedMsg
			form, cmd := f.transformForm.Update(msg)
			f.transformForm = form
			return f, cmd

		case PickerTestCompletedMsg:
			// Forward to transform form
			form, cmd := f.transformForm.Update(msg)
			f.transformForm = form
			// Also update our own previousOutputs cache
			if msg.Error == nil && f.previousOutputs != nil {
				f.previousOutputs[msg.VarName] = msg.Output
			}
			return f, cmd

		case components.FieldCancelledMsg:
			// Forward to transform form to close field picker
			form, cmd := f.transformForm.Update(msg)
			f.transformForm = form
			return f, cmd

		case tea.KeyMsg:
			form, cmd := f.transformForm.Update(msg)
			if form == nil {
				// If there's a cmd, return it so the message can be processed
				// (e.g., TransformSavedMsg from Ctrl+S)
				if cmd != nil {
					return f, cmd
				}
				// Transform form cancelled - check if we should fall back to raw mode
				if f.transformForm.pickingOperation {
					// Cancelled at operation picker - close step form
					return nil, nil
				}
				// Cancelled from form - go back to operation picker
				f.transformForm.pickingOperation = true
				return f, nil
			}
			f.transformForm = form
			return f, cmd
		}
		return f, nil
	}

	// Handle field picker if active
	if f.showFieldPicker && f.fieldPicker != nil {
		switch msg := msg.(type) {
		case components.FieldCancelledMsg:
			f.showFieldPicker = false
			f.fieldPicker = nil
			return f, nil
		case components.PickerNeedsTestMsg:
			// Bubble up to WorkflowBuilder to run the test
			return f, func() tea.Msg { return PickerTestRequestedMsg{VarName: msg.VarName} }
		case PickerTestCompletedMsg:
			// Forward the result to the picker
			f.fieldPicker.Update(components.PickerTestResultMsg{
				VarName: msg.VarName,
				Output:  msg.Output,
				Error:   msg.Error,
			})
			// Update our own previousOutputs cache
			if msg.Error == nil && f.previousOutputs != nil {
				f.previousOutputs[msg.VarName] = msg.Output
			}
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
		ef := f.getExpandedFieldAt(f.focusedField)
		if ef != nil && len(f.previousOutputs) > 0 {
			// Don't allow variable picker on array/object headers
			if !f.isArrayHeader(ef) && !f.isObjectHeader(ef) {
				f.fieldPicker = components.NewFieldPickerMulti(f.previousOutputs, f.varToStepName)
				f.showFieldPicker = true
			}
		}

	case "a":
		// Add array item (when on array header field)
		ef := f.getExpandedFieldAt(f.focusedField)
		if ef != nil && f.isArrayHeader(ef) {
			arrayKey := f.getArrayKeyForField(ef)
			f.addArrayItem(arrayKey)
			f.dirty = true
		}

	case "d":
		// Remove array item (when on an array item field)
		ef := f.getExpandedFieldAt(f.focusedField)
		if ef != nil && ef.arrayIndex >= 0 {
			arrayKey := ef.parentName
			f.removeArrayItem(arrayKey, ef.arrayIndex)
			f.dirty = true
			// Move focus up if we removed an item
			if f.focusedField > stepFieldParamsStart {
				f.focusedField = f.prevField()
			}
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
		// Param field - use expanded fields
		ef := f.getExpandedFieldAt(f.focusedField)
		if ef != nil {
			f.setFieldValue(*ef, f.editBuffer)
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
		// Param field - use expanded fields
		ef := f.getExpandedFieldAt(f.focusedField)
		if ef != nil {
			// Don't allow editing array/object headers
			if f.isArrayHeader(ef) || f.isObjectHeader(ef) {
				return f, nil
			}

			if ef.param.Type == "boolean" {
				// Toggle boolean
				current := f.getFieldValue(*ef) == "true"
				f.setFieldValue(*ef, strconv.FormatBool(!current))
				f.dirty = true
			} else {
				f.editing = true
				f.editBuffer = f.getFieldValue(*ef)
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
	// Count expanded fields instead of raw params
	expandedFields := f.buildExpandedFields()
	return stepFieldParamsStart + len(expandedFields) - 1
}

// getExpandedFieldAt returns the expanded field at a given field index.
func (f *StepForm) getExpandedFieldAt(fieldIndex int) *expandedField {
	paramIndex := fieldIndex - stepFieldParamsStart
	if paramIndex < 0 {
		return nil
	}
	expandedFields := f.buildExpandedFields()
	if paramIndex >= len(expandedFields) {
		return nil
	}
	return &expandedFields[paramIndex]
}

// isArrayHeader returns true if the expanded field is an array header (not an item).
func (f *StepForm) isArrayHeader(ef *expandedField) bool {
	return ef.param.Type == "array" && ef.param.Items != nil && ef.arrayIndex == -1
}

// isObjectHeader returns true if the expanded field is an object header (not a property).
func (f *StepForm) isObjectHeader(ef *expandedField) bool {
	return ef.param.Type == "object" && len(ef.param.Properties) > 0 && ef.parentName == ""
}

// getArrayKeyForField returns the array key for an expanded field (for array items).
func (f *StepForm) getArrayKeyForField(ef *expandedField) string {
	if ef.arrayIndex >= 0 {
		// This is an array item, parentName is the array key
		return ef.parentName
	}
	if ef.param.Type == "array" && ef.param.Items != nil {
		// This is an array header
		if ef.parentName != "" {
			return ef.parentName + "." + ef.param.Name
		}
		return ef.param.Name
	}
	return ""
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
	ef := f.getExpandedFieldAt(f.focusedField)
	if ef != nil {
		// Replace the field value with the variable path
		f.setFieldValue(*ef, path)
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

// getFieldValue gets the value for an expanded field.
func (f *StepForm) getFieldValue(ef expandedField) string {
	if f.Step.Params == nil {
		return ""
	}

	// Array item
	if ef.arrayIndex >= 0 {
		return f.getArrayItemValue(ef.parentName, ef.arrayIndex, ef.param.Name)
	}

	// Nested property (parentName is like "config" for "config.header")
	if ef.parentName != "" {
		return f.getNestedValue(ef.parentName, ef.param.Name)
	}

	// Top-level param
	return f.getParamValue(ef.param.Name)
}

// setFieldValue sets the value for an expanded field.
func (f *StepForm) setFieldValue(ef expandedField, value string) {
	if f.Step.Params == nil {
		f.Step.Params = make(map[string]interface{})
	}

	// Array item
	if ef.arrayIndex >= 0 {
		f.setArrayItemValue(ef.parentName, ef.arrayIndex, ef.param.Name, value, ef.param.Type)
		return
	}

	// Nested property
	if ef.parentName != "" {
		f.setNestedValue(ef.parentName, ef.param.Name, value, ef.param.Type)
		return
	}

	// Top-level param
	f.setParamValue(ef.param.Name, value, ef.param.Type)
}

// getNestedValue gets a nested object property value like config.header.
func (f *StepForm) getNestedValue(parentKey, propName string) string {
	val, ok := f.Step.Params[parentKey]
	if !ok {
		return ""
	}
	obj, ok := val.(map[string]interface{})
	if !ok {
		return ""
	}
	propVal, ok := obj[propName]
	if !ok {
		return ""
	}
	return f.formatValue(propVal)
}

// setNestedValue sets a nested object property value.
func (f *StepForm) setNestedValue(parentKey, propName, value, propType string) {
	// Get or create parent object
	var obj map[string]interface{}
	if existing, ok := f.Step.Params[parentKey]; ok {
		if m, ok := existing.(map[string]interface{}); ok {
			obj = m
		}
	}
	if obj == nil {
		obj = make(map[string]interface{})
	}

	// Set the property value with correct type
	obj[propName] = f.parseTypedValue(value, propType)
	f.Step.Params[parentKey] = obj
}

// getArrayItemValue gets a value from an array item.
func (f *StepForm) getArrayItemValue(arrayKey string, index int, propName string) string {
	val, ok := f.Step.Params[arrayKey]
	if !ok {
		return ""
	}
	arr, ok := val.([]interface{})
	if !ok || index >= len(arr) {
		return ""
	}

	item := arr[index]

	// If propName is empty, return the item directly (simple array)
	if propName == "" {
		return f.formatValue(item)
	}

	// Object item - get property
	if obj, ok := item.(map[string]interface{}); ok {
		if propVal, ok := obj[propName]; ok {
			return f.formatValue(propVal)
		}
	}
	return ""
}

// setArrayItemValue sets a value in an array item.
func (f *StepForm) setArrayItemValue(arrayKey string, index int, propName, value, propType string) {
	// Get or create array
	var arr []interface{}
	if existing, ok := f.Step.Params[arrayKey]; ok {
		if a, ok := existing.([]interface{}); ok {
			arr = a
		}
	}

	// Extend array if needed
	for len(arr) <= index {
		arr = append(arr, nil)
	}

	typedValue := f.parseTypedValue(value, propType)

	// If propName is empty, set item directly (simple array)
	if propName == "" {
		arr[index] = typedValue
	} else {
		// Object item - set property
		var obj map[string]interface{}
		if existing, ok := arr[index].(map[string]interface{}); ok {
			obj = existing
		} else {
			obj = make(map[string]interface{})
		}
		obj[propName] = typedValue
		arr[index] = obj
	}

	f.Step.Params[arrayKey] = arr
}

// addArrayItem adds a new item to an array parameter.
func (f *StepForm) addArrayItem(arrayKey string) {
	count := f.arrayItemCounts[arrayKey]
	f.arrayItemCounts[arrayKey] = count + 1

	// Initialize the array in params if needed
	if f.Step.Params == nil {
		f.Step.Params = make(map[string]interface{})
	}
	var arr []interface{}
	if existing, ok := f.Step.Params[arrayKey]; ok {
		if a, ok := existing.([]interface{}); ok {
			arr = a
		}
	}
	// Add empty item (will be filled in by user)
	arr = append(arr, nil)
	f.Step.Params[arrayKey] = arr
}

// removeArrayItem removes an item from an array parameter.
func (f *StepForm) removeArrayItem(arrayKey string, index int) {
	if index < 0 {
		return
	}

	if existing, ok := f.Step.Params[arrayKey]; ok {
		if arr, ok := existing.([]interface{}); ok && index < len(arr) {
			// Remove item at index
			arr = append(arr[:index], arr[index+1:]...)
			f.Step.Params[arrayKey] = arr
		}
	}

	count := f.arrayItemCounts[arrayKey]
	if count > 0 {
		f.arrayItemCounts[arrayKey] = count - 1
	}
}

// formatValue converts an interface{} to string for display.
func (f *StepForm) formatValue(val interface{}) string {
	if val == nil {
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

// parseTypedValue parses a string into the appropriate type.
func (f *StepForm) parseTypedValue(value, paramType string) interface{} {
	switch paramType {
	case "number":
		if n, err := strconv.ParseFloat(value, 64); err == nil {
			return n
		}
		return value
	case "boolean":
		return value == "true"
	case "array", "object":
		var parsed interface{}
		if err := json.Unmarshal([]byte(value), &parsed); err == nil {
			return parsed
		}
		return value
	default:
		return value
	}
}

// View renders the step form.
func (f *StepForm) View() string {
	// Delegate to transform form if in transform mode
	if f.isTransformTool && f.transformForm != nil {
		return f.transformForm.View()
	}

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

	// Available variables with color coding
	if len(f.availableVars) > 0 {
		lines = append(lines, f.renderAvailableVariables(dimStyle))
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

	// Dynamic params (using expanded fields for nested support)
	if len(f.Tool.Params) > 0 {
		lines = append(lines, "")
		lines = append(lines, dimStyle.Render("Parameters:"))
		expandedFields := f.buildExpandedFields()
		for i, ef := range expandedFields {
			fieldIndex := stepFieldParamsStart + i
			lines = append(lines, f.renderExpandedField(ef, fieldIndex, selectedStyle, dimStyle))
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

// renderExpandedField renders a single expanded field with proper indentation.
func (f *StepForm) renderExpandedField(ef expandedField, fieldIndex int, selectedStyle, dimStyle lipgloss.Style) string {
	param := ef.param
	isFocused := f.focusedField == fieldIndex

	// Build indent string
	indent := strings.Repeat("  ", ef.indent)

	// Handle array header (shows [a] Add hint)
	if f.isArrayHeader(&ef) {
		label := param.Name
		if param.Required {
			label += "*"
		}
		hint := ""
		if isFocused {
			hint = "  " + dimStyle.Render("[a] Add item")
			if param.Description != "" {
				hint += "\n" + indent + "    " + dimStyle.Render(param.Description)
			}
			// Show example if present
			if param.Example != nil {
				hint += "\n" + indent + "    " + dimStyle.Render(fmt.Sprintf("Example: %v", param.Example))
			}
		}
		line := fmt.Sprintf("%s  %-12s", indent, label+":")
		if isFocused {
			return selectedStyle.Render("> "+line[2:]) + hint
		}
		return line
	}

	// Handle object header (shows nested properties below)
	if f.isObjectHeader(&ef) {
		label := param.Name
		if param.Required {
			label += "*"
		}
		hint := ""
		if isFocused {
			if param.Description != "" {
				hint = "\n" + indent + "    " + dimStyle.Render(param.Description)
			}
		}
		line := fmt.Sprintf("%s  %-12s", indent, label+":")
		if isFocused {
			return selectedStyle.Render("> "+line[2:]) + hint
		}
		return line
	}

	// Handle array item
	if ef.arrayIndex >= 0 {
		return f.renderArrayItemField(ef, fieldIndex, selectedStyle, dimStyle)
	}

	// Handle regular field (nested property or simple param)
	return f.renderRegularField(ef, fieldIndex, selectedStyle, dimStyle)
}

// renderArrayItemField renders an array item field.
func (f *StepForm) renderArrayItemField(ef expandedField, fieldIndex int, selectedStyle, dimStyle lipgloss.Style) string {
	param := ef.param
	isFocused := f.focusedField == fieldIndex

	indent := strings.Repeat("  ", ef.indent)

	// Get the value
	value := f.getFieldValue(ef)
	display := value
	if display == "" {
		if param.Default != nil {
			display = fmt.Sprintf("%v", param.Default)
		} else {
			display = "(empty)"
		}
	}

	// Build label - either [0]: for simple items or [0].field: for object items
	var label string
	if param.Name == "" {
		// Simple array item - just show index
		label = fmt.Sprintf("[%d]", ef.arrayIndex)
	} else {
		// Object item property - show index and property name
		label = fmt.Sprintf("[%d].%s", ef.arrayIndex, param.Name)
	}

	hint := ""
	if f.editing && isFocused {
		display = f.editBuffer[:f.cursorPos] + "|" + f.editBuffer[f.cursorPos:]
		hint = "  " + dimStyle.Render("[Enter] Confirm  [←/→] Move cursor")
	} else if isFocused {
		if param.Type == "boolean" {
			hint = "  " + dimStyle.Render("[Enter] Toggle")
		} else {
			hint = "  " + dimStyle.Render("[Enter] Edit")
			if len(f.previousOutputs) > 0 {
				hint += "  " + dimStyle.Render("[v] Variable")
			}
		}
		hint += "  " + dimStyle.Render("[d] Remove")
		if param.Description != "" {
			hint += "\n" + indent + "    " + dimStyle.Render(param.Description)
		}
	}

	line := fmt.Sprintf("%s  %-12s %s", indent, label+":", display)

	if isFocused {
		return selectedStyle.Render("> "+line[2:]) + hint
	}
	return line
}

// renderRegularField renders a regular param field (nested property or simple param).
func (f *StepForm) renderRegularField(ef expandedField, fieldIndex int, selectedStyle, dimStyle lipgloss.Style) string {
	param := ef.param
	isFocused := f.focusedField == fieldIndex

	indent := strings.Repeat("  ", ef.indent)

	// Get the value
	value := f.getFieldValue(ef)
	display := value
	if display == "" {
		if param.Default != nil {
			display = fmt.Sprintf("%v", param.Default)
		} else {
			display = "(empty)"
		}
	}

	label := param.Name
	if param.Required {
		label += "*"
	}

	hint := ""
	if f.editing && isFocused {
		display = f.editBuffer[:f.cursorPos] + "|" + f.editBuffer[f.cursorPos:]
		hint = "  " + dimStyle.Render("[Enter] Confirm  [←/→] Move cursor")
	} else if isFocused {
		if param.Type == "boolean" {
			hint = "  " + dimStyle.Render("[Enter] Toggle")
		} else {
			hint = "  " + dimStyle.Render("[Enter] Edit")
			if len(f.previousOutputs) > 0 {
				hint += "  " + dimStyle.Render("[v] Variable")
			}
		}
		// Show description for focused param
		if param.Description != "" {
			hint += "\n" + indent + "    " + dimStyle.Render(param.Description)
		}
		// Show example if present
		if param.Example != nil {
			hint += "\n" + indent + "    " + dimStyle.Render(fmt.Sprintf("Example: %v", param.Example))
		}
	}

	line := fmt.Sprintf("%s  %-12s %s", indent, label+":", display)

	if isFocused {
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

// renderAvailableVariables renders available variables with color coding.
// Green = tested (has output), Yellow = not yet tested.
func (f *StepForm) renderAvailableVariables(dimStyle lipgloss.Style) string {
	if len(f.availableVars) == 0 {
		return dimStyle.Render("No variables available yet")
	}

	successStyle := lipgloss.NewStyle().Foreground(theme.Success)
	warningStyle := lipgloss.NewStyle().Foreground(theme.Warning)

	var parts []string
	for _, v := range f.availableVars {
		// Check if this variable has test output
		varName := strings.TrimPrefix(v, "$")
		if f.isVariableTested(varName) {
			parts = append(parts, successStyle.Render(v))
		} else {
			parts = append(parts, warningStyle.Render(v+"(?)"))
		}
	}

	return dimStyle.Render("Available: ") + strings.Join(parts, ", ")
}

// isVariableTested returns true if we have test output for this variable.
func (f *StepForm) isVariableTested(varName string) bool {
	if f.previousOutputs == nil {
		return false
	}
	output, exists := f.previousOutputs[varName]
	// Check that output exists AND is not nil (nil means placeholder without actual test output)
	return exists && output != nil
}
