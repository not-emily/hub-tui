package modal

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/client"
	"github.com/pxp/hub-tui/internal/ui/components"
	"github.com/pxp/hub-tui/internal/ui/theme"
)

// BuilderView represents the current view within the builder.
type BuilderView int

const (
	builderViewList BuilderView = iota
	builderViewStepDetail
	builderViewToolPicker
	builderViewFieldPicker
	builderViewTriggerForm
	builderViewValidation
	builderViewHelp
)

// Focus field constants
const (
	focusName      = 0
	focusOutput    = 1
	focusAttention = 2
	focusTrigger   = 3
	focusSteps     = 4 // focusSteps + index for specific step; focusAddStep is after all steps
)

// WorkflowBuilder handles creating and editing workflows.
type WorkflowBuilder struct {
	client *client.Client

	// Identity
	IsNew        bool
	OriginalName string

	// Workflow data
	Name                     string
	Description              string
	Trigger                  client.TriggerConfig
	Steps                    []client.WorkflowStep
	Output                   string
	NeedsAttentionOnComplete bool

	// Editing state
	viewState     BuilderView
	SelectedStep  int
	EditingStep   *client.WorkflowStep
	StepOutput    interface{}
	editingName   bool
	nameInput     string
	editingOutput bool
	outputOptions []string
	outputIndex   int
	focusedField  int // 0=name, 1=output, 2+=steps

	// Sub-forms
	scheduleForm *ScheduleForm
	toolBrowser  *ToolBrowser
	stepForm     *StepForm

	// Cached data
	Tools       *client.ToolsResponse
	Profiles    map[string][]string    // integration -> profile names
	StepOutputs map[string]interface{} // test outputs keyed by step save_as name

	// Temporarily stored tool for new step (before step form opens)
	pendingTool     *client.Tool
	pendingToolType string

	// Flag to open step form after tools load
	pendingStepEdit bool

	// UI state
	Dirty   bool
	Error   string
	Loading bool

	// Validation state
	validationResult  *client.ValidationResult
	validationLoading bool
	pendingSave       bool // true if validation was triggered by save attempt

	// Close confirmation
	showCloseConfirm bool

	// Dimensions
	width  int
	height int
}

// NewWorkflowBuilder creates a new workflow builder.
func NewWorkflowBuilder(c *client.Client, isNew bool) *WorkflowBuilder {
	return &WorkflowBuilder{
		client:       c,
		IsNew:        isNew,
		viewState:    builderViewList,
		Steps:        []client.WorkflowStep{},
		Profiles:     make(map[string][]string),
		StepOutputs:  make(map[string]interface{}),
		Trigger:      client.TriggerConfig{Type: "manual"},
		focusedField: focusName, // Start focused on name for new workflows
	}
}

// LoadWorkflow populates the builder from an existing workflow.
func (b *WorkflowBuilder) LoadWorkflow(wf *client.Workflow) {
	b.Name = wf.Name
	b.OriginalName = wf.Name
	b.Description = wf.Description
	b.Trigger = wf.Trigger
	b.Steps = wf.Steps
	b.Output = wf.Output
	b.NeedsAttentionOnComplete = wf.NeedsAttentionOnComplete
	b.IsNew = false
	// For existing workflows, start focused on steps if any
	if len(wf.Steps) > 0 {
		b.focusedField = focusSteps
	}
}

// Init initializes the builder.
func (b *WorkflowBuilder) Init() tea.Cmd {
	return nil
}

// SetSize sets the available dimensions.
func (b *WorkflowBuilder) SetSize(width, height int) {
	b.width = width
	b.height = height
}

// BuilderCloseMsg signals the builder should close.
type BuilderCloseMsg struct {
	Saved bool
}

// Update handles input for the builder.
func (b *WorkflowBuilder) Update(msg tea.Msg) (*WorkflowBuilder, tea.Cmd) {
	debugLog(fmt.Sprintf("WorkflowBuilder.Update: msg=%T, viewState=%v, stepForm=%v", msg, b.viewState, b.stepForm != nil))

	// Handle async messages first (before view-specific routing)
	switch msg := msg.(type) {
	case BuilderToolsLoadedMsg:
		b.Loading = false
		if msg.Error != nil {
			if b.toolBrowser != nil {
				b.toolBrowser.SetError(msg.Error)
			} else {
				b.Error = msg.Error.Error()
			}
			b.pendingStepEdit = false
			return b, nil
		}

		// Cache tools
		b.Tools = msg.Tools

		// If tool browser is active, set tools on it
		if b.toolBrowser != nil {
			b.toolBrowser.SetTools(msg.Tools)
		}

		// If we were waiting to edit a step, open the form now
		if b.pendingStepEdit {
			b.pendingStepEdit = false
			cmd := b.openStepForm(false)
			return b, cmd
		}

		return b, nil

	case StepProfilesLoadedMsg:
		// Forward to step form if active
		if b.stepForm != nil {
			form, cmd := b.stepForm.Update(msg)
			b.stepForm = form
			return b, cmd
		}
		return b, nil

	case StepTestedMsg:
		// Forward to step form and store output
		if b.stepForm != nil {
			form, cmd := b.stepForm.Update(msg)
			b.stepForm = form
			// Store output in builder cache
			if msg.Result != nil && msg.Result.Success && b.stepForm.Step.SaveAs != "" {
				b.StepOutputs[b.stepForm.Step.SaveAs] = msg.Result.Output
			}
			return b, cmd
		}
		return b, nil

	case TransformPreviewedMsg:
		// Forward to step form (which delegates to transform form)
		if b.stepForm != nil {
			form, cmd := b.stepForm.Update(msg)
			b.stepForm = form
			return b, cmd
		}
		return b, nil

	case PickerTestRequestedMsg:
		// Field picker needs test output for a variable - find and test that step
		debugLog(fmt.Sprintf("WorkflowBuilder: handling PickerTestRequestedMsg for %s", msg.VarName))
		return b, b.testStepForPicker(msg.VarName)

	case PickerTestCompletedMsg:
		// Forward to step form which will forward to transform form
		if b.stepForm != nil {
			// Update our cache
			if msg.Error == nil {
				b.StepOutputs[msg.VarName] = msg.Output
			}
			form, cmd := b.stepForm.Update(msg)
			b.stepForm = form
			return b, cmd
		}
		return b, nil

	case SchedulePreviewedMsg:
		// Forward to schedule form if active
		if b.scheduleForm != nil {
			form, cmd := b.scheduleForm.Update(msg)
			b.scheduleForm = form
			return b, cmd
		}
		return b, nil

	case WorkflowValidatedMsg:
		b.validationLoading = false
		if msg.Error != nil {
			b.Error = msg.Error.Error()
			b.pendingSave = false
			return b, nil
		}
		b.validationResult = &client.ValidationResult{
			Valid:  msg.Valid,
			Errors: msg.Errors,
		}

		// If this was a save attempt and valid, proceed with save
		if b.pendingSave && msg.Valid {
			b.pendingSave = false
			return b, b.doSave()
		} else if b.pendingSave && !msg.Valid {
			// Show validation view with errors
			b.viewState = builderViewValidation
			b.pendingSave = false
		} else {
			// Manual validation request - show results
			b.viewState = builderViewValidation
		}
		return b, nil

	case components.TreeCancelledMsg:
		// Tool picker cancelled - remove the placeholder step and go back to list
		if b.viewState == builderViewToolPicker {
			// Remove the placeholder step that was added
			if len(b.Steps) > 0 {
				b.Steps = b.Steps[:len(b.Steps)-1]
				if len(b.Steps) == 0 {
					b.SelectedStep = 0
					b.focusedField = focusTrigger
				} else {
					b.SelectedStep = len(b.Steps) - 1
					b.focusedField = focusSteps + b.SelectedStep
				}
			}
			b.toolBrowser = nil
			b.viewState = builderViewList
			return b, nil
		}
	}

	// Route to schedule form if active
	if b.viewState == builderViewTriggerForm && b.scheduleForm != nil {
		form, cmd := b.scheduleForm.Update(msg)
		if form == nil {
			// Form closed
			if b.scheduleForm.Saved() {
				b.Trigger = b.scheduleForm.ToTriggerConfig()
				b.Dirty = true
			}
			b.scheduleForm = nil
			b.viewState = builderViewList
			return b, nil
		}
		b.scheduleForm = form
		return b, cmd
	}

	// Route to step form if active
	if b.viewState == builderViewStepDetail && b.stepForm != nil {
		form, cmd := b.stepForm.Update(msg)
		if form == nil {
			// Form closed - store test output if any
			if b.stepForm.Step.SaveAs != "" && b.stepForm.TestOutput() != nil {
				b.StepOutputs[b.stepForm.Step.SaveAs] = b.stepForm.TestOutput()
			}
			if b.stepForm.Saved() {
				b.Dirty = true
			}
			b.stepForm = nil
			b.viewState = builderViewList
			return b, nil
		}
		b.stepForm = form
		return b, cmd
	}

	// Route to tool browser if active
	if b.viewState == builderViewToolPicker && b.toolBrowser != nil {
		tool, toolType, cmd := b.toolBrowser.Update(msg)
		if tool != nil {
			// Tool selected - configure step and open step form
			b.toolBrowser = nil
			stepCmd := b.configureStepWithTool(*tool, toolType)
			return b, stepCmd
		}
		return b, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return b.handleKeyPress(msg)
	}
	return b, nil
}

func (b *WorkflowBuilder) handleKeyPress(msg tea.KeyMsg) (*WorkflowBuilder, tea.Cmd) {
	// Handle close confirmation dialog
	if b.showCloseConfirm {
		return b.handleCloseConfirm(msg)
	}

	// Handle name editing mode
	if b.editingName {
		return b.handleNameInput(msg)
	}

	// Handle output selection mode
	if b.editingOutput {
		return b.handleOutputSelect(msg)
	}

	// Handle help view
	if b.viewState == builderViewHelp {
		// Any key closes help
		b.viewState = builderViewList
		return b, nil
	}

	// Handle validation view
	if b.viewState == builderViewValidation {
		switch msg.String() {
		case "esc":
			b.viewState = builderViewList
		case "s", "ctrl+s":
			// Try save again if valid
			if b.validationResult != nil && b.validationResult.Valid {
				return b, b.doSave()
			}
		}
		return b, nil
	}

	// For now, only handle list view. Other views will be added in later phases.
	if b.viewState != builderViewList {
		// Allow escape to go back to list from other views
		if msg.String() == "esc" {
			b.viewState = builderViewList
		}
		return b, nil
	}

	switch msg.String() {
	// Navigation
	case "j", "down":
		b.moveSelectionDown()
	case "k", "up":
		b.moveSelectionUp()

	// Step reordering (Shift+J/K)
	case "J":
		b.moveStepDown()
	case "K":
		b.moveStepUp()

	// Enter to edit/select
	case "e", "enter":
		if b.focusedField == focusName {
			if b.IsNew {
				b.editingName = true
				b.nameInput = b.Name
			}
			// Name is not editable for existing workflows (it's the identifier)
		} else if b.focusedField == focusOutput {
			b.editingOutput = true
			b.outputOptions = b.buildOutputOptions()
			b.outputIndex = b.findOutputIndex()
		} else if b.focusedField == focusAttention {
			b.NeedsAttentionOnComplete = !b.NeedsAttentionOnComplete
			b.Dirty = true
		} else if b.focusedField == focusTrigger {
			b.scheduleForm = NewScheduleForm(b.client, b.Trigger)
			b.viewState = builderViewTriggerForm
			return b, b.scheduleForm.Init()
		} else if b.focusedField == b.focusAddStep() {
			return b.addStep()
		} else if b.focusedField >= focusSteps && b.focusedField < b.focusAddStep() {
			// Edit existing step
			return b.editStep()
		}

	// Delete step
	case "d":
		if b.focusedField >= focusSteps && b.focusedField < b.focusAddStep() && len(b.Steps) > 0 {
			return b.deleteStep(b.SelectedStep)
		}

	// Validate
	case "v":
		return b.validateWorkflow()

	// Help
	case "?":
		b.viewState = builderViewHelp

	// Save
	case "ctrl+s":
		return b.saveWorkflow()

	// Close
	case "esc", "q":
		return b.handleClose()
	}

	return b, nil
}

func (b *WorkflowBuilder) handleClose() (*WorkflowBuilder, tea.Cmd) {
	if b.Dirty {
		b.showCloseConfirm = true
		return b, nil
	}
	return nil, nil // Close without confirmation
}

func (b *WorkflowBuilder) handleCloseConfirm(msg tea.KeyMsg) (*WorkflowBuilder, tea.Cmd) {
	switch msg.String() {
	case "y":
		// Discard and close
		return nil, nil
	case "n", "esc":
		// Cancel - keep editing
		b.showCloseConfirm = false
	case "s":
		// Save and close
		b.showCloseConfirm = false
		return b.saveWorkflow()
	}
	return b, nil
}

func (b *WorkflowBuilder) handleNameInput(msg tea.KeyMsg) (*WorkflowBuilder, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Confirm name change
		if b.nameInput != b.Name {
			b.Name = b.nameInput
			b.Dirty = true
		}
		b.editingName = false
	case "esc":
		// Cancel name change
		b.editingName = false
		b.nameInput = ""
	case "backspace":
		if len(b.nameInput) > 0 {
			b.nameInput = b.nameInput[:len(b.nameInput)-1]
		}
	default:
		// Add character if printable
		if len(msg.String()) == 1 {
			b.nameInput += msg.String()
		}
	}
	return b, nil
}

func (b *WorkflowBuilder) handleOutputSelect(msg tea.KeyMsg) (*WorkflowBuilder, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Confirm output selection
		if b.outputIndex >= 0 && b.outputIndex < len(b.outputOptions) {
			selected := b.outputOptions[b.outputIndex]
			if selected == "(none)" {
				selected = ""
			}
			if selected != b.Output {
				b.Output = selected
				b.Dirty = true
			}
		}
		b.editingOutput = false
	case "esc":
		// Cancel output selection
		b.editingOutput = false
	case "j", "down":
		if b.outputIndex < len(b.outputOptions)-1 {
			b.outputIndex++
		}
	case "k", "up":
		if b.outputIndex > 0 {
			b.outputIndex--
		}
	}
	return b, nil
}

// focusAddStep returns the focus index for the "+ Add step" button.
func (b *WorkflowBuilder) focusAddStep() int {
	return focusSteps + len(b.Steps)
}

func (b *WorkflowBuilder) moveSelectionDown() {
	maxField := b.focusAddStep() // Can go to "+ Add step"

	if b.focusedField < maxField {
		b.focusedField++
		if b.focusedField >= focusSteps && b.focusedField < b.focusAddStep() {
			b.SelectedStep = b.focusedField - focusSteps
		}
	}
}

func (b *WorkflowBuilder) moveSelectionUp() {
	if b.focusedField > focusName {
		b.focusedField--
		if b.focusedField >= focusSteps && b.focusedField < b.focusAddStep() {
			b.SelectedStep = b.focusedField - focusSteps
		}
	}
}

func (b *WorkflowBuilder) moveStepDown() {
	// Only works when focused on an actual step (not Add Step button)
	if b.focusedField < focusSteps || b.focusedField >= b.focusAddStep() {
		return
	}
	if b.SelectedStep < len(b.Steps)-1 {
		// Swap with next step
		b.Steps[b.SelectedStep], b.Steps[b.SelectedStep+1] =
			b.Steps[b.SelectedStep+1], b.Steps[b.SelectedStep]
		b.SelectedStep++
		b.focusedField++
		b.Dirty = true
	}
}

func (b *WorkflowBuilder) moveStepUp() {
	// Only works when focused on an actual step (not Add Step button)
	if b.focusedField < focusSteps || b.focusedField >= b.focusAddStep() {
		return
	}
	if b.SelectedStep > 0 {
		// Swap with previous step
		b.Steps[b.SelectedStep], b.Steps[b.SelectedStep-1] =
			b.Steps[b.SelectedStep-1], b.Steps[b.SelectedStep]
		b.SelectedStep--
		b.focusedField--
		b.Dirty = true
	}
}

func (b *WorkflowBuilder) addStep() (*WorkflowBuilder, tea.Cmd) {
	// Create placeholder step
	newStep := client.WorkflowStep{
		Name: fmt.Sprintf("step_%d", len(b.Steps)+1),
	}
	b.Steps = append(b.Steps, newStep)
	b.SelectedStep = len(b.Steps) - 1
	b.focusedField = focusSteps + b.SelectedStep
	// Don't mark dirty yet - wait until tool is selected

	// Enter tool picker to select tool for this step
	b.viewState = builderViewToolPicker
	b.toolBrowser = NewToolBrowser()

	// Use cached tools if available, otherwise fetch
	if b.Tools != nil {
		b.toolBrowser.SetTools(b.Tools)
		return b, nil
	}
	return b, b.toolBrowser.Init(b.client)
}

// editStep opens the step form for the currently selected step.
func (b *WorkflowBuilder) editStep() (*WorkflowBuilder, tea.Cmd) {
	if b.SelectedStep < 0 || b.SelectedStep >= len(b.Steps) {
		return b, nil
	}

	// Need tools loaded to find tool schema
	if b.Tools == nil {
		// Load tools first, then open step form
		b.Loading = true
		b.pendingStepEdit = true
		return b, func() tea.Msg {
			tools, err := b.client.GetBuilderTools()
			return BuilderToolsLoadedMsg{Tools: tools, Error: err}
		}
	}

	cmd := b.openStepForm(false)
	return b, cmd
}

// configureStepWithTool sets up a step with the selected tool and opens the step form.
func (b *WorkflowBuilder) configureStepWithTool(tool client.Tool, toolType string) tea.Cmd {
	if b.SelectedStep < 0 || b.SelectedStep >= len(b.Steps) {
		return nil
	}

	step := &b.Steps[b.SelectedStep]
	step.Type = toolType
	step.Target = tool.Target

	// Generate a friendly save_as name from the tool name
	// e.g., "notion.query_database" -> "query_database_result"
	parts := strings.Split(tool.Target, ".")
	baseName := parts[len(parts)-1]
	step.SaveAs = baseName + "_result"

	// Store tool info and open step form
	b.pendingTool = &tool
	b.pendingToolType = toolType

	return b.openStepForm(true)
}

// openStepForm opens the step form for the selected step.
func (b *WorkflowBuilder) openStepForm(isNew bool) tea.Cmd {
	if b.SelectedStep < 0 || b.SelectedStep >= len(b.Steps) {
		return nil
	}

	step := &b.Steps[b.SelectedStep]

	// Find the tool - either from pending (new step) or lookup (existing step)
	var tool *client.Tool
	var toolType string

	if b.pendingTool != nil {
		tool = b.pendingTool
		toolType = b.pendingToolType
	} else {
		// Look up tool from cached tools
		tool, toolType = b.findTool(step.Target)
		if tool == nil {
			b.Error = "Tool not found: " + step.Target
			return nil
		}
	}

	b.stepForm = NewStepForm(b.client, step, tool, toolType, isNew)
	b.stepForm.SetAvailableVariables(b.AvailableVariables(b.SelectedStep))
	b.stepForm.SetPreviousOutputs(b.buildPreviousOutputs(b.SelectedStep))
	b.stepForm.SetVarToStepName(b.buildVarToStepName(b.SelectedStep))
	b.viewState = builderViewStepDetail

	// Clear pending tool
	b.pendingTool = nil
	b.pendingToolType = ""

	return b.stepForm.Init()
}

// buildPreviousOutputs returns test outputs from steps before the given index.
func (b *WorkflowBuilder) buildPreviousOutputs(beforeIndex int) map[string]interface{} {
	outputs := make(map[string]interface{})
	for i := 0; i < beforeIndex && i < len(b.Steps); i++ {
		if b.Steps[i].SaveAs != "" {
			// Use test output if available, otherwise use nil placeholder
			// This allows variable picker to show variables even without test output
			if output, ok := b.StepOutputs[b.Steps[i].SaveAs]; ok {
				outputs[b.Steps[i].SaveAs] = output
			} else {
				outputs[b.Steps[i].SaveAs] = nil
			}
		}
	}
	return outputs
}

// buildVarToStepName returns a mapping from variable names to step names.
func (b *WorkflowBuilder) buildVarToStepName(beforeIndex int) map[string]string {
	mapping := make(map[string]string)
	for i := 0; i < beforeIndex && i < len(b.Steps); i++ {
		if b.Steps[i].SaveAs != "" {
			mapping[b.Steps[i].SaveAs] = b.Steps[i].Name
		}
	}
	return mapping
}

// testStepForPicker runs a test for a step to get output for the field picker.
func (b *WorkflowBuilder) testStepForPicker(varName string) tea.Cmd {
	// Find the step with this SaveAs
	var step *client.WorkflowStep
	for i := range b.Steps {
		if b.Steps[i].SaveAs == varName {
			step = &b.Steps[i]
			break
		}
	}

	if step == nil {
		return func() tea.Msg {
			return PickerTestCompletedMsg{
				VarName: varName,
				Error:   fmt.Errorf("step not found for variable $%s", varName),
			}
		}
	}

	// Build variables map from outputs of steps before this one
	variables := make(map[string]interface{})
	for i := range b.Steps {
		if b.Steps[i].SaveAs == varName {
			break // Stop at the step we're testing
		}
		if b.Steps[i].SaveAs != "" {
			if output, ok := b.StepOutputs[b.Steps[i].SaveAs]; ok {
				variables[b.Steps[i].SaveAs] = output
			}
		}
	}

	return func() tea.Msg {
		result, err := b.client.TestStep(&client.StepTestRequest{
			Step:      *step,
			Variables: variables,
		})

		if err != nil {
			return PickerTestCompletedMsg{
				VarName: varName,
				Error:   err,
			}
		}

		if !result.Success {
			return PickerTestCompletedMsg{
				VarName: varName,
				Error:   fmt.Errorf("%s", result.Error),
			}
		}

		return PickerTestCompletedMsg{
			VarName: varName,
			Output:  result.Output,
		}
	}
}

// findTool looks up a tool by target in the cached tools.
func (b *WorkflowBuilder) findTool(target string) (*client.Tool, string) {
	if b.Tools == nil {
		return nil, ""
	}

	// Search all categories
	for _, tool := range b.searchCategory(b.Tools.Tools.Modules) {
		if tool.Target == target {
			return &tool, "module"
		}
	}
	for _, tool := range b.searchCategory(b.Tools.Tools.Integrations) {
		if tool.Target == target {
			return &tool, "integration"
		}
	}
	for _, tool := range b.searchCategory(b.Tools.Tools.Utilities) {
		if tool.Target == target {
			return &tool, "utility"
		}
	}
	for _, tool := range b.searchCategory(b.Tools.Tools.Primitives) {
		if tool.Target == target {
			return &tool, "primitive"
		}
	}
	return nil, ""
}

func (b *WorkflowBuilder) searchCategory(category map[string][]client.Tool) []client.Tool {
	var tools []client.Tool
	for _, ts := range category {
		tools = append(tools, ts...)
	}
	return tools
}

func (b *WorkflowBuilder) deleteStep(index int) (*WorkflowBuilder, tea.Cmd) {
	if index < 0 || index >= len(b.Steps) {
		return b, nil
	}

	// Remove step
	b.Steps = append(b.Steps[:index], b.Steps[index+1:]...)

	// Adjust selection
	if len(b.Steps) == 0 {
		b.SelectedStep = 0
		b.focusedField = focusOutput
	} else if b.SelectedStep >= len(b.Steps) {
		b.SelectedStep = len(b.Steps) - 1
		b.focusedField = focusSteps + b.SelectedStep
	}

	b.Dirty = true
	return b, nil
}

func (b *WorkflowBuilder) validateWorkflow() (*WorkflowBuilder, tea.Cmd) {
	// Basic local validation
	if b.Name == "" {
		b.Error = "Workflow name is required"
		return b, nil
	}

	b.validationLoading = true
	b.validationResult = nil
	b.Error = ""

	originalName := b.OriginalName
	return b, func() tea.Msg {
		wf := b.ToWorkflow()
		result, err := b.client.ValidateWorkflow(wf, originalName)
		if err != nil {
			return WorkflowValidatedMsg{Error: err}
		}
		return WorkflowValidatedMsg{
			Valid:  result.Valid,
			Errors: result.Errors,
		}
	}
}

func (b *WorkflowBuilder) saveWorkflow() (*WorkflowBuilder, tea.Cmd) {
	// Basic validation
	if b.Name == "" {
		b.Error = "Workflow name is required"
		return b, nil
	}

	// Validate before save
	b.validationLoading = true
	b.validationResult = nil
	b.pendingSave = true
	b.Error = ""

	originalName := b.OriginalName
	return b, func() tea.Msg {
		wf := b.ToWorkflow()
		result, err := b.client.ValidateWorkflow(wf, originalName)
		if err != nil {
			return WorkflowValidatedMsg{Error: err}
		}
		return WorkflowValidatedMsg{
			Valid:  result.Valid,
			Errors: result.Errors,
		}
	}
}

func (b *WorkflowBuilder) doSave() tea.Cmd {
	return func() tea.Msg {
		wf := b.ToWorkflow()

		var err error
		if b.IsNew {
			err = b.client.CreateWorkflow(wf)
		} else {
			err = b.client.UpdateWorkflow(b.OriginalName, wf)
		}

		return WorkflowSavedMsg{
			Name:  wf.Name,
			IsNew: b.IsNew,
			Error: err,
		}
	}
}

// ToWorkflow converts the builder state to a Workflow.
func (b *WorkflowBuilder) ToWorkflow() *client.Workflow {
	return &client.Workflow{
		Name:                     b.Name,
		Description:              b.Description,
		Trigger:                  b.Trigger,
		Steps:                    b.Steps,
		Output:                   b.Output,
		NeedsAttentionOnComplete: b.NeedsAttentionOnComplete,
		Enabled:                  true,
	}
}

func (b *WorkflowBuilder) buildOutputOptions() []string {
	options := []string{"(none)"}
	for _, step := range b.Steps {
		if step.SaveAs != "" {
			options = append(options, "$"+step.SaveAs)
		}
	}
	return options
}

func (b *WorkflowBuilder) findOutputIndex() int {
	if b.Output == "" {
		return 0
	}
	for i, opt := range b.outputOptions {
		if opt == b.Output {
			return i
		}
	}
	return 0
}

// View renders the builder.
func (b *WorkflowBuilder) View() string {
	// Show close confirmation overlay if active
	if b.showCloseConfirm {
		return b.renderCloseConfirm()
	}

	switch b.viewState {
	case builderViewList:
		return b.renderStepList()
	case builderViewToolPicker:
		if b.toolBrowser != nil {
			return b.toolBrowser.View()
		}
		return b.renderPlaceholder("Tool Picker", "Select a tool for this step", "[Esc] Back")
	case builderViewStepDetail:
		if b.stepForm != nil {
			return b.stepForm.View()
		}
		return b.renderPlaceholder("Step Detail", "Edit step parameters", "[Esc] Back")
	case builderViewTriggerForm:
		if b.scheduleForm != nil {
			return b.scheduleForm.View()
		}
		return b.renderPlaceholder("Trigger Settings", "Configure workflow trigger", "[Esc] Back")
	case builderViewValidation:
		return b.renderValidation()
	case builderViewHelp:
		return b.renderHelp()
	default:
		return b.renderStepList()
	}
}

func (b *WorkflowBuilder) renderPlaceholder(title, desc, hint string) string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	var lines []string
	lines = append(lines, headerStyle.Render(title))
	lines = append(lines, "")
	lines = append(lines, dimStyle.Render(desc))
	lines = append(lines, dimStyle.Render("(Coming in a later phase)"))
	lines = append(lines, "")
	lines = append(lines, dimStyle.Render(hint))
	return strings.Join(lines, "\n")
}

// renderStepList renders the main step list view.
func (b *WorkflowBuilder) renderStepList() string {
	var lines []string

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	labelStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
	warningStyle := lipgloss.NewStyle().Foreground(theme.Warning)

	// Header
	header := "Edit Workflow"
	if b.IsNew {
		header = "Create Workflow"
	}
	lines = append(lines, headerStyle.Render(header))
	lines = append(lines, "")

	// Error display
	if b.Error != "" {
		lines = append(lines, errorStyle.Render("Error: "+b.Error))
		lines = append(lines, "")
	}

	// Name field
	nameLabel := "Name:    "
	if b.editingName {
		cursor := "_"
		lines = append(lines, labelStyle.Render(nameLabel)+selectedStyle.Render(b.nameInput+cursor)+" "+dimStyle.Render("[Enter to confirm]"))
	} else {
		nameValue := b.Name
		if nameValue == "" {
			nameValue = "(required)"
		}
		if b.focusedField == focusName {
			hint := "[Enter to edit]"
			if !b.IsNew {
				hint = "(read-only)"
			}
			lines = append(lines, labelStyle.Render(nameLabel)+selectedStyle.Render(nameValue)+" "+dimStyle.Render(hint))
		} else {
			lines = append(lines, labelStyle.Render(nameLabel)+nameValue)
		}
	}

	// Output field
	outputLabel := "Output:  "
	if b.editingOutput {
		// Show dropdown
		var opts []string
		for i, opt := range b.outputOptions {
			if i == b.outputIndex {
				opts = append(opts, selectedStyle.Render("["+opt+"]"))
			} else {
				opts = append(opts, dimStyle.Render(opt))
			}
		}
		lines = append(lines, labelStyle.Render(outputLabel)+strings.Join(opts, " "))
	} else {
		outputValue := b.Output
		if outputValue == "" {
			outputValue = "(none)"
		}
		if b.focusedField == focusOutput {
			lines = append(lines, labelStyle.Render(outputLabel)+selectedStyle.Render(outputValue)+" "+dimStyle.Render("[Enter to select]"))
		} else {
			lines = append(lines, labelStyle.Render(outputLabel)+outputValue)
		}
	}

	// Attention toggle field
	attentionLabel := "Notify:  "
	attentionValue := "No"
	if b.NeedsAttentionOnComplete {
		attentionValue = "Yes"
	}
	if b.focusedField == focusAttention {
		lines = append(lines, labelStyle.Render(attentionLabel)+selectedStyle.Render(attentionValue)+" "+dimStyle.Render("[Enter to toggle]"))
	} else {
		lines = append(lines, labelStyle.Render(attentionLabel)+attentionValue)
	}

	// Trigger field (focusable)
	triggerLabel := "Trigger: "
	triggerInfo := b.formatTrigger()
	if b.focusedField == focusTrigger {
		lines = append(lines, labelStyle.Render(triggerLabel)+selectedStyle.Render(triggerInfo)+" "+dimStyle.Render("[Enter to edit]"))
	} else {
		lines = append(lines, labelStyle.Render(triggerLabel)+triggerInfo)
	}

	lines = append(lines, "")

	// Steps section
	lines = append(lines, labelStyle.Render("Steps:"))

	if len(b.Steps) == 0 {
		// No steps yet - just show the add button
	} else {
		for i, step := range b.Steps {
			line := b.formatStepLine(i, step)
			lines = append(lines, line)
		}
	}

	// "+ Add step" button with space before it
	lines = append(lines, "")
	addStepLabel := "+ Add step"
	if b.focusedField == b.focusAddStep() {
		lines = append(lines, selectedStyle.Render("> "+addStepLabel))
	} else {
		lines = append(lines, "  "+addStepLabel)
	}

	lines = append(lines, "")

	// Hints
	lines = append(lines, b.renderHints(dimStyle, warningStyle))

	return strings.Join(lines, "\n")
}

func (b *WorkflowBuilder) renderHints(dimStyle, warningStyle lipgloss.Style) string {
	var hints []string

	// Show contextual hints based on focus
	if b.focusedField >= focusSteps && b.focusedField < b.focusAddStep() && len(b.Steps) > 0 {
		hints = append(hints, "[Enter]edit", "[d]elete", "[J/K]move")
	} else {
		hints = append(hints, "[Enter]select")
	}

	hints = append(hints, "[v]alidate", "[Ctrl+s]save", "[?]help")

	hintStr := dimStyle.Render(strings.Join(hints, "  "))
	if b.Dirty {
		hintStr += " " + warningStyle.Render("(unsaved)")
	}
	if b.validationLoading {
		hintStr += " " + dimStyle.Render("(validating...)")
	}
	return hintStr
}

// formatTrigger returns a human-readable trigger description.
func (b *WorkflowBuilder) formatTrigger() string {
	switch b.Trigger.Type {
	case "schedule":
		if b.Trigger.Frequency != "" && b.Trigger.Time != "" {
			return fmt.Sprintf("%s at %s", b.Trigger.Frequency, b.Trigger.Time)
		}
		if b.Trigger.Frequency != "" {
			return b.Trigger.Frequency
		}
		return "scheduled"
	case "manual":
		return "manual"
	case "webhook":
		return "webhook"
	case "condition":
		return "condition"
	default:
		return "manual"
	}
}

// formatStepLine formats a single step for display in the list.
func (b *WorkflowBuilder) formatStepLine(index int, step client.WorkflowStep) string {
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	errorStyle := lipgloss.NewStyle().Foreground(theme.Error)

	isSelected := b.focusedField == focusSteps+index

	// Selection indicator
	prefix := "  "
	if isSelected {
		prefix = "> "
	}

	// Step name (or default)
	name := step.Name
	if name == "" {
		name = fmt.Sprintf("step_%d", index+1)
	}

	// Truncate name if too long
	maxNameLen := 12
	if len(name) > maxNameLen {
		name = name[:maxNameLen-1] + "…"
	}

	// Type indicator
	typeStr := step.Type
	if typeStr == "" {
		typeStr = "(no tool)"
	}
	if len(typeStr) > 11 {
		typeStr = typeStr[:10] + "…"
	}

	// Target
	target := step.Target
	if target == "" {
		target = ""
	}
	maxTargetLen := 20
	if len(target) > maxTargetLen {
		target = target[:maxTargetLen-1] + "…"
	}

	// SaveAs indicator
	saveAs := ""
	if step.SaveAs != "" {
		saveAs = " → $" + step.SaveAs
	}

	// Error indicator for steps with validation errors
	errorIndicator := ""
	if b.hasStepValidationError(index) {
		errorIndicator = " " + errorStyle.Render("!")
	}

	// Format the line
	line := fmt.Sprintf("%s%d. %-*s  %-11s  %-*s%s",
		prefix, index+1, maxNameLen, name, typeStr, maxTargetLen, target, saveAs)

	if isSelected {
		return selectedStyle.Render(line) + errorIndicator
	}

	// Dim the metadata parts for unselected rows
	namePart := fmt.Sprintf("%s%d. %-*s", prefix, index+1, maxNameLen, name)
	metaPart := fmt.Sprintf("  %-11s  %-*s%s", typeStr, maxTargetLen, target, saveAs)
	return normalStyle.Render(namePart) + dimStyle.Render(metaPart) + errorIndicator
}

// hasStepValidationError returns true if the step has validation errors.
func (b *WorkflowBuilder) hasStepValidationError(stepIndex int) bool {
	if b.validationResult == nil || b.validationResult.Valid {
		return false
	}
	for _, err := range b.validationResult.Errors {
		if err.Step != nil && *err.Step == stepIndex {
			return true
		}
	}
	return false
}

// AvailableVariables returns variables defined by steps before the given index.
func (b *WorkflowBuilder) AvailableVariables(beforeIndex int) []string {
	var vars []string
	for i := 0; i < beforeIndex && i < len(b.Steps); i++ {
		if b.Steps[i].SaveAs != "" {
			vars = append(vars, "$"+b.Steps[i].SaveAs)
		}
	}
	return vars
}

// renderValidation renders the validation results view.
func (b *WorkflowBuilder) renderValidation() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	successStyle := lipgloss.NewStyle().Foreground(theme.Success)
	errorStyle := lipgloss.NewStyle().Foreground(theme.Error)

	var lines []string

	lines = append(lines, headerStyle.Render("Workflow Validation"))
	lines = append(lines, "")

	if b.validationLoading {
		lines = append(lines, dimStyle.Render("Validating..."))
		return strings.Join(lines, "\n")
	}

	if b.validationResult == nil {
		lines = append(lines, dimStyle.Render("Press [v] to validate"))
		return strings.Join(lines, "\n")
	}

	if b.validationResult.Valid {
		lines = append(lines, successStyle.Render("✓ Workflow is valid"))
		lines = append(lines, "")
		lines = append(lines, dimStyle.Render("[s] Save  [Esc] Continue editing"))
	} else {
		lines = append(lines, errorStyle.Render("✗ Validation failed"))
		lines = append(lines, "")

		// Group errors by step
		var workflowErrors []client.ValidationError
		stepErrors := make(map[int][]client.ValidationError)

		for _, err := range b.validationResult.Errors {
			if err.Step == nil {
				workflowErrors = append(workflowErrors, err)
			} else {
				stepErrors[*err.Step] = append(stepErrors[*err.Step], err)
			}
		}

		// Workflow-level errors
		if len(workflowErrors) > 0 {
			lines = append(lines, "Workflow:")
			for _, err := range workflowErrors {
				msg := err.Message
				if err.Field != "" {
					msg = err.Field + ": " + msg
				}
				lines = append(lines, "  • "+msg)
			}
			lines = append(lines, "")
		}

		// Step errors
		for stepIdx, errors := range stepErrors {
			stepName := fmt.Sprintf("Step %d", stepIdx+1)
			if stepIdx < len(b.Steps) && b.Steps[stepIdx].Name != "" {
				stepName = b.Steps[stepIdx].Name
			}
			lines = append(lines, stepName+":")
			for _, err := range errors {
				msg := err.Message
				if err.Field != "" {
					msg = err.Field + ": " + msg
				}
				lines = append(lines, "  • "+msg)
			}
			lines = append(lines, "")
		}

		lines = append(lines, dimStyle.Render("[Esc] Continue editing"))
	}

	return strings.Join(lines, "\n")
}

// renderHelp renders the help overlay.
func (b *WorkflowBuilder) renderHelp() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	labelStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary).Width(12)

	var lines []string

	lines = append(lines, headerStyle.Render("Workflow Builder Help"))
	lines = append(lines, "")

	lines = append(lines, dimStyle.Render("Navigation:"))
	lines = append(lines, labelStyle.Render("  j/k")+"  Move selection up/down")
	lines = append(lines, labelStyle.Render("  J/K")+"  Move step up/down (reorder)")
	lines = append(lines, labelStyle.Render("  Enter")+"  Edit selected item")
	lines = append(lines, labelStyle.Render("  Esc")+"  Go back / Cancel")
	lines = append(lines, "")

	lines = append(lines, dimStyle.Render("Actions:"))
	lines = append(lines, labelStyle.Render("  e")+"  Edit selected step")
	lines = append(lines, labelStyle.Render("  d")+"  Delete selected step")
	lines = append(lines, labelStyle.Render("  v")+"  Validate workflow")
	lines = append(lines, labelStyle.Render("  Ctrl+s")+"  Save workflow")
	lines = append(lines, "")

	lines = append(lines, dimStyle.Render("In Step Editor:"))
	lines = append(lines, labelStyle.Render("  t")+"  Test step")
	lines = append(lines, labelStyle.Render("  v")+"  Pick variable from previous step")
	lines = append(lines, labelStyle.Render("  a")+"  Add array item (on array fields)")
	lines = append(lines, labelStyle.Render("  d")+"  Remove array item")
	lines = append(lines, "")

	lines = append(lines, dimStyle.Render("Press any key to close help"))

	return strings.Join(lines, "\n")
}

// renderCloseConfirm renders the close confirmation dialog.
func (b *WorkflowBuilder) renderCloseConfirm() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Warning)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	var lines []string

	lines = append(lines, headerStyle.Render("Unsaved Changes"))
	lines = append(lines, "")
	lines = append(lines, "You have unsaved changes.")
	lines = append(lines, "")
	lines = append(lines, "[y] Discard and close")
	lines = append(lines, "[n] Cancel and keep editing")
	lines = append(lines, "[s] Save and close")
	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("Press y, n, or s"))

	return strings.Join(lines, "\n")
}
