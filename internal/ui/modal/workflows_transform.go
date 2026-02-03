package modal

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/client"
	"github.com/pxp/hub-tui/internal/ui/components"
	"github.com/pxp/hub-tui/internal/ui/theme"
)

func debugLog(msg string) {
	// Debug logging disabled - uncomment to enable
	// f, _ := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	// if f != nil {
	// 	f.WriteString(msg + "\n")
	// 	f.Close()
	// }
	_ = msg
}

// TransformPreviewedMsg is sent when a transform preview completes.
type TransformPreviewedMsg struct {
	Step  *client.WorkflowStep
	Error error
}

func (m TransformPreviewedMsg) IsAsyncModalMessage() {}
func (m TransformPreviewedMsg) AuthError() error     { return m.Error }

// TransformSavedMsg is sent when a transform is saved.
type TransformSavedMsg struct {
	Step *client.WorkflowStep
}

// PickerTestRequestedMsg is sent when the field picker needs to test a previous step.
type PickerTestRequestedMsg struct {
	VarName string
}

func (m PickerTestRequestedMsg) IsAsyncModalMessage() {}
func (m PickerTestRequestedMsg) AuthError() error     { return nil }

// PickerTestCompletedMsg is sent when the picker's test request completes.
type PickerTestCompletedMsg struct {
	VarName string
	Output  interface{}
	Error   error
}

func (m PickerTestCompletedMsg) IsAsyncModalMessage() {}
func (m PickerTestCompletedMsg) AuthError() error     { return m.Error }

// TransformOperation represents a transform type.
type TransformOperation string

const (
	TransformFilter  TransformOperation = "filter"
	TransformExtract TransformOperation = "pick"
	TransformSort    TransformOperation = "sort"
	TransformFirst   TransformOperation = "first"
	TransformLast    TransformOperation = "last"
	TransformCount   TransformOperation = "count"
)

var transformOperations = []struct {
	Op          TransformOperation
	Label       string
	Description string
}{
	{TransformFilter, "Filter", "Keep items matching a condition"},
	{TransformExtract, "Extract", "Pull out specific fields"},
	{TransformSort, "Sort", "Order items by a field"},
	{TransformFirst, "First", "Take first N items"},
	{TransformLast, "Last", "Take last N items"},
	{TransformCount, "Count", "Get number of items"},
}

var filterOperators = []struct {
	Op    string
	Label string
}{
	{"equals", "equals"},
	{"not_equals", "not equals"},
	{"contains", "contains"},
	{"not_contains", "not contains"},
	{"greater_than", ">"},
	{"less_than", "<"},
	{"greater_or_equal", ">="},
	{"less_or_equal", "<="},
}

// ExtractFieldMapping represents a field to extract.
type ExtractFieldMapping struct {
	Source string // path from input
	As     string // output field name
}

// TransformForm handles creating transform steps.
type TransformForm struct {
	client *client.Client

	// Current view
	pickingOperation bool
	operationIndex   int

	// Selected operation
	Operation TransformOperation

	// Common fields
	Input  string // variable reference
	SaveAs string
	Name   string // step name

	// Filter-specific
	FilterField    string
	FilterOperator int // index into filterOperators
	FilterValue    string

	// Extract-specific
	ExtractFields []ExtractFieldMapping

	// Sort-specific
	SortField     string
	SortAscending bool

	// First/Last-specific
	Count int

	// Preview
	PreviewStep *client.WorkflowStep

	// UI state
	focusedField int
	editing      bool
	editBuffer   string
	cursorPos    int
	loading      bool
	err          string

	// Available variables
	availableVars   []string
	previousOutputs map[string]interface{}
	varToStepName   map[string]string

	// Field picker
	fieldPicker     *components.FieldPicker
	showFieldPicker bool
}

// NewTransformForm creates a new transform form.
func NewTransformForm(c *client.Client) *TransformForm {
	return &TransformForm{
		client:           c,
		pickingOperation: true,
		operationIndex:   0,
		FilterOperator:   0, // equals
		SortAscending:    true,
		Count:            1,
		ExtractFields:    []ExtractFieldMapping{{Source: "", As: ""}},
	}
}

// NewTransformFormFromStep creates a transform form pre-populated from an existing step.
// Returns nil if the step params can't be parsed as a known transform operation.
func NewTransformFormFromStep(c *client.Client, step *client.WorkflowStep) *TransformForm {
	if step.Params == nil {
		return nil
	}

	f := &TransformForm{
		client:         c,
		FilterOperator: 0,
		SortAscending:  true,
		Count:          1,
		ExtractFields:  []ExtractFieldMapping{},
		Name:           step.Name,
		SaveAs:         step.SaveAs,
	}

	// Get input
	if input, ok := step.Params["input"].(string); ok {
		f.Input = input
	}

	// Try to detect operation from query pattern
	query, hasQuery := step.Params["query"].(string)
	if !hasQuery {
		return nil // Can't parse without query
	}

	// Detect operation type from query pattern
	if op := f.detectOperationFromQuery(query); op != "" {
		f.Operation = op
		f.pickingOperation = false
		return f
	}

	return nil // Unknown pattern - fall back to raw editor
}

// detectOperationFromQuery attempts to identify the transform operation from a jq query.
func (f *TransformForm) detectOperationFromQuery(query string) TransformOperation {
	query = strings.TrimSpace(query)

	// Filter: [.[] | select(.field == "value")]
	if strings.HasPrefix(query, "[.[] | select(") {
		f.parseFilterQuery(query)
		return TransformFilter
	}

	// Sort: sort_by(.field) or sort_by(.field) | reverse
	if strings.HasPrefix(query, "sort_by(") {
		f.parseSortQuery(query)
		return TransformSort
	}

	// First: .[:N] or first or limit(N)
	if strings.HasPrefix(query, ".[:") || query == "first" || strings.HasPrefix(query, "limit(") {
		f.parseFirstQuery(query)
		return TransformFirst
	}

	// Last: .[-N:] or last
	if strings.HasPrefix(query, ".[-") || query == "last" {
		f.parseLastQuery(query)
		return TransformLast
	}

	// Count: length
	if query == "length" || query == "| length" {
		return TransformCount
	}

	// Pick/Extract: [.[] | {field1, field2}] or map({...})
	if strings.Contains(query, "| {") || strings.HasPrefix(query, "map({") {
		f.parseExtractQuery(query)
		return TransformExtract
	}

	return "" // Unknown
}

// parseFilterQuery extracts filter params from a query like [.[] | select(.status == "active")]
func (f *TransformForm) parseFilterQuery(query string) {
	// Extract the select condition
	// Pattern: [.[] | select(.field op "value")]
	re := regexp.MustCompile(`select\(\.(\w+)\s*(==|!=|>|<|>=|<=)\s*"?([^")\]]+)"?\)`)
	matches := re.FindStringSubmatch(query)
	if len(matches) >= 4 {
		f.FilterField = matches[1]
		f.FilterValue = strings.Trim(matches[3], `"`)

		// Map operator
		switch matches[2] {
		case "==":
			f.FilterOperator = 0 // equals
		case "!=":
			f.FilterOperator = 1 // not_equals
		case ">":
			f.FilterOperator = 4 // greater_than
		case "<":
			f.FilterOperator = 5 // less_than
		case ">=":
			f.FilterOperator = 6 // greater_or_equal
		case "<=":
			f.FilterOperator = 7 // less_or_equal
		}
	}

	// Check for contains
	reContains := regexp.MustCompile(`select\(\.(\w+)\s*\|\s*contains\("([^"]+)"\)\)`)
	matchesContains := reContains.FindStringSubmatch(query)
	if len(matchesContains) >= 3 {
		f.FilterField = matchesContains[1]
		f.FilterValue = matchesContains[2]
		f.FilterOperator = 2 // contains
	}
}

// parseSortQuery extracts sort params from a query like sort_by(.field) | reverse
func (f *TransformForm) parseSortQuery(query string) {
	re := regexp.MustCompile(`sort_by\(\.(\w+)\)`)
	matches := re.FindStringSubmatch(query)
	if len(matches) >= 2 {
		f.SortField = matches[1]
	}
	f.SortAscending = !strings.Contains(query, "reverse")
}

// parseFirstQuery extracts first/limit params
func (f *TransformForm) parseFirstQuery(query string) {
	re := regexp.MustCompile(`\.\[:(\d+)\]|limit\((\d+)\)`)
	matches := re.FindStringSubmatch(query)
	if len(matches) >= 2 {
		for _, m := range matches[1:] {
			if m != "" {
				if n, err := strconv.Atoi(m); err == nil {
					f.Count = n
					return
				}
			}
		}
	}
	f.Count = 1
}

// parseLastQuery extracts last params
func (f *TransformForm) parseLastQuery(query string) {
	re := regexp.MustCompile(`\.\[-(\d+):\]`)
	matches := re.FindStringSubmatch(query)
	if len(matches) >= 2 {
		if n, err := strconv.Atoi(matches[1]); err == nil {
			f.Count = n
			return
		}
	}
	f.Count = 1
}

// parseExtractQuery extracts pick/extract field mappings
func (f *TransformForm) parseExtractQuery(query string) {
	// Pattern: [.[] | {name: .title, id}] or map({name: .title, id})
	// This is complex - just extract field names for now
	re := regexp.MustCompile(`(\w+):\s*\.(\w+)|\.(\w+)`)
	matches := re.FindAllStringSubmatch(query, -1)

	f.ExtractFields = []ExtractFieldMapping{}
	for _, m := range matches {
		if m[1] != "" && m[2] != "" {
			// Renamed field: name: .title
			f.ExtractFields = append(f.ExtractFields, ExtractFieldMapping{
				Source: m[2],
				As:     m[1],
			})
		} else if m[3] != "" {
			// Same name: .id -> id
			f.ExtractFields = append(f.ExtractFields, ExtractFieldMapping{
				Source: m[3],
				As:     m[3],
			})
		}
	}

	if len(f.ExtractFields) == 0 {
		f.ExtractFields = []ExtractFieldMapping{{Source: "", As: ""}}
	}
}

// SetAvailableVariables sets the variables available from previous steps.
func (f *TransformForm) SetAvailableVariables(vars []string) {
	f.availableVars = vars
}

// SetPreviousOutputs sets the outputs from previous steps for the field picker.
func (f *TransformForm) SetPreviousOutputs(outputs map[string]interface{}) {
	f.previousOutputs = outputs
}

// SetVarToStepName sets the mapping from variable names to step names.
func (f *TransformForm) SetVarToStepName(mapping map[string]string) {
	f.varToStepName = mapping
}

// Update handles input.
func (f *TransformForm) Update(msg tea.Msg) (*TransformForm, tea.Cmd) {
	debugLog(fmt.Sprintf("TransformForm.Update: msg=%T, showFieldPicker=%v, fieldPicker=%v", msg, f.showFieldPicker, f.fieldPicker != nil))

	// Handle field picker if active
	if f.showFieldPicker && f.fieldPicker != nil {
		switch msg := msg.(type) {
		case components.FieldCancelledMsg:
			debugLog("TransformForm: handling FieldCancelledMsg")
			f.showFieldPicker = false
			f.fieldPicker = nil
			return f, nil
		case components.PickerNeedsTestMsg:
			// Bubble up the test request - will be handled by WorkflowBuilder
			debugLog(fmt.Sprintf("TransformForm: handling PickerNeedsTestMsg, converting to PickerTestRequestedMsg for %s", msg.VarName))
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
			debugLog(fmt.Sprintf("TransformForm: handling KeyMsg %s", msg.String()))
			selectedPath, cmd := f.fieldPicker.Update(msg)
			debugLog(fmt.Sprintf("TransformForm: fieldPicker returned path=%q, cmd=%v", selectedPath, cmd != nil))
			if selectedPath != "" {
				// Field selected - set as Input value
				f.Input = selectedPath
				f.showFieldPicker = false
				f.fieldPicker = nil
				return f, nil
			}
			if cmd != nil {
				debugLog("TransformForm: returning cmd from fieldPicker")
				return f, cmd
			}
			return f, nil
		}
		return f, nil
	}

	switch msg := msg.(type) {
	case TransformPreviewedMsg:
		f.loading = false
		if msg.Error != nil {
			f.err = msg.Error.Error()
		} else {
			f.PreviewStep = msg.Step
			f.err = ""
		}
		return f, nil

	case tea.KeyMsg:
		if f.pickingOperation {
			return f.handleOperationPicker(msg)
		}
		return f.handleFormInput(msg)
	}
	return f, nil
}

func (f *TransformForm) handleOperationPicker(msg tea.KeyMsg) (*TransformForm, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if f.operationIndex < len(transformOperations)-1 {
			f.operationIndex++
		}
	case "k", "up":
		if f.operationIndex > 0 {
			f.operationIndex--
		}
	case "enter":
		f.Operation = transformOperations[f.operationIndex].Op
		f.pickingOperation = false
		f.Name = string(f.Operation) + "_result"
		f.SaveAs = string(f.Operation) + "_output"
	case "esc":
		return nil, nil // Cancel
	}
	return f, nil
}

func (f *TransformForm) handleFormInput(msg tea.KeyMsg) (*TransformForm, tea.Cmd) {
	if f.editing {
		return f.handleTextEdit(msg)
	}

	switch msg.String() {
	case "j", "down":
		f.focusedField++
		f.clampFocusedField()
	case "k", "up":
		if f.focusedField > 0 {
			f.focusedField--
		}
	case "enter":
		return f.handleEnter()
	case "h", "left":
		f.handleLeft()
	case "l", "right":
		f.handleRight()
	case "v":
		// Variable picker - only for Input field (field 0)
		if f.focusedField == 0 && len(f.previousOutputs) > 0 {
			f.fieldPicker = components.NewFieldPickerMulti(f.previousOutputs, f.varToStepName)
			f.showFieldPicker = true
		}
	case "a":
		// Add extract field
		if f.Operation == TransformExtract {
			f.ExtractFields = append(f.ExtractFields, ExtractFieldMapping{})
		}
	case "d":
		// Remove last extract field
		if f.Operation == TransformExtract && len(f.ExtractFields) > 1 {
			f.ExtractFields = f.ExtractFields[:len(f.ExtractFields)-1]
			f.clampFocusedField()
		}
	case "p":
		return f, f.fetchPreview()
	case "ctrl+s":
		return f.saveTransform()
	case "esc":
		// Go back to operation picker
		f.pickingOperation = true
		f.focusedField = 0
	}
	return f, nil
}

func (f *TransformForm) handleTextEdit(msg tea.KeyMsg) (*TransformForm, tea.Cmd) {
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
			f.editBuffer = f.editBuffer[:f.cursorPos] + msg.String() + f.editBuffer[f.cursorPos:]
			f.cursorPos++
		}
	}
	return f, nil
}

func (f *TransformForm) handleEnter() (*TransformForm, tea.Cmd) {
	switch f.Operation {
	case TransformFilter:
		return f.handleFilterEnter()
	case TransformExtract:
		return f.handleExtractEnter()
	case TransformSort:
		return f.handleSortEnter()
	case TransformFirst, TransformLast:
		return f.handleCountEnter()
	case TransformCount:
		return f.handleCountOnlyEnter()
	}
	return f, nil
}

func (f *TransformForm) handleLeft() {
	switch f.Operation {
	case TransformFilter:
		if f.focusedField == 2 && f.FilterOperator > 0 {
			f.FilterOperator--
		}
	case TransformSort:
		if f.focusedField == 2 {
			f.SortAscending = true
		}
	}
}

func (f *TransformForm) handleRight() {
	switch f.Operation {
	case TransformFilter:
		if f.focusedField == 2 && f.FilterOperator < len(filterOperators)-1 {
			f.FilterOperator++
		}
	case TransformSort:
		if f.focusedField == 2 {
			f.SortAscending = false
		}
	}
}

func (f *TransformForm) clampFocusedField() {
	max := f.maxField()
	if f.focusedField > max {
		f.focusedField = max
	}
}

func (f *TransformForm) maxField() int {
	switch f.Operation {
	case TransformFilter:
		return 5 // input, field, operator, value, name, save_as
	case TransformExtract:
		return len(f.ExtractFields)*2 + 2 // input, (source, as)*N, name, save_as
	case TransformSort:
		return 4 // input, field, direction, name, save_as
	case TransformFirst, TransformLast:
		return 3 // input, count, name, save_as
	case TransformCount:
		return 2 // input, name, save_as
	}
	return 0
}

// Filter form handlers
func (f *TransformForm) handleFilterEnter() (*TransformForm, tea.Cmd) {
	switch f.focusedField {
	case 0: // Input
		f.editing = true
		f.editBuffer = f.Input
		f.cursorPos = len(f.editBuffer)
	case 1: // Field
		f.editing = true
		f.editBuffer = f.FilterField
		f.cursorPos = len(f.editBuffer)
	case 2: // Operator - use left/right
	case 3: // Value
		f.editing = true
		f.editBuffer = f.FilterValue
		f.cursorPos = len(f.editBuffer)
	case 4: // Name
		f.editing = true
		f.editBuffer = f.Name
		f.cursorPos = len(f.editBuffer)
	case 5: // SaveAs
		f.editing = true
		f.editBuffer = f.SaveAs
		f.cursorPos = len(f.editBuffer)
	}
	return f, nil
}

// Extract form handlers
func (f *TransformForm) handleExtractEnter() (*TransformForm, tea.Cmd) {
	if f.focusedField == 0 {
		// Input
		f.editing = true
		f.editBuffer = f.Input
		f.cursorPos = len(f.editBuffer)
		return f, nil
	}

	// Field mappings
	baseField := 1
	for i := range f.ExtractFields {
		if f.focusedField == baseField+i*2 {
			f.editing = true
			f.editBuffer = f.ExtractFields[i].Source
			f.cursorPos = len(f.editBuffer)
			return f, nil
		}
		if f.focusedField == baseField+i*2+1 {
			f.editing = true
			f.editBuffer = f.ExtractFields[i].As
			f.cursorPos = len(f.editBuffer)
			return f, nil
		}
	}

	// Name and SaveAs
	afterFields := baseField + len(f.ExtractFields)*2
	if f.focusedField == afterFields {
		f.editing = true
		f.editBuffer = f.Name
		f.cursorPos = len(f.editBuffer)
	} else if f.focusedField == afterFields+1 {
		f.editing = true
		f.editBuffer = f.SaveAs
		f.cursorPos = len(f.editBuffer)
	}
	return f, nil
}

// Sort form handlers
func (f *TransformForm) handleSortEnter() (*TransformForm, tea.Cmd) {
	switch f.focusedField {
	case 0: // Input
		f.editing = true
		f.editBuffer = f.Input
		f.cursorPos = len(f.editBuffer)
	case 1: // Field
		f.editing = true
		f.editBuffer = f.SortField
		f.cursorPos = len(f.editBuffer)
	case 2: // Direction - use left/right
	case 3: // Name
		f.editing = true
		f.editBuffer = f.Name
		f.cursorPos = len(f.editBuffer)
	case 4: // SaveAs
		f.editing = true
		f.editBuffer = f.SaveAs
		f.cursorPos = len(f.editBuffer)
	}
	return f, nil
}

// Count form handlers (First/Last)
func (f *TransformForm) handleCountEnter() (*TransformForm, tea.Cmd) {
	switch f.focusedField {
	case 0: // Input
		f.editing = true
		f.editBuffer = f.Input
		f.cursorPos = len(f.editBuffer)
	case 1: // Count
		f.editing = true
		f.editBuffer = strconv.Itoa(f.Count)
		f.cursorPos = len(f.editBuffer)
	case 2: // Name
		f.editing = true
		f.editBuffer = f.Name
		f.cursorPos = len(f.editBuffer)
	case 3: // SaveAs
		f.editing = true
		f.editBuffer = f.SaveAs
		f.cursorPos = len(f.editBuffer)
	}
	return f, nil
}

// Count-only form handlers (Count operation)
func (f *TransformForm) handleCountOnlyEnter() (*TransformForm, tea.Cmd) {
	switch f.focusedField {
	case 0: // Input
		f.editing = true
		f.editBuffer = f.Input
		f.cursorPos = len(f.editBuffer)
	case 1: // Name
		f.editing = true
		f.editBuffer = f.Name
		f.cursorPos = len(f.editBuffer)
	case 2: // SaveAs
		f.editing = true
		f.editBuffer = f.SaveAs
		f.cursorPos = len(f.editBuffer)
	}
	return f, nil
}

func (f *TransformForm) applyEdit() {
	switch f.Operation {
	case TransformFilter:
		f.applyFilterEdit()
	case TransformExtract:
		f.applyExtractEdit()
	case TransformSort:
		f.applySortEdit()
	case TransformFirst, TransformLast:
		f.applyCountEdit()
	case TransformCount:
		f.applyCountOnlyEdit()
	}
	f.editBuffer = ""
}

func (f *TransformForm) applyFilterEdit() {
	switch f.focusedField {
	case 0:
		f.Input = f.editBuffer
	case 1:
		f.FilterField = f.editBuffer
	case 3:
		f.FilterValue = f.editBuffer
	case 4:
		f.Name = f.editBuffer
	case 5:
		f.SaveAs = f.editBuffer
	}
}

func (f *TransformForm) applyExtractEdit() {
	if f.focusedField == 0 {
		f.Input = f.editBuffer
		return
	}

	baseField := 1
	for i := range f.ExtractFields {
		if f.focusedField == baseField+i*2 {
			f.ExtractFields[i].Source = f.editBuffer
			return
		}
		if f.focusedField == baseField+i*2+1 {
			f.ExtractFields[i].As = f.editBuffer
			return
		}
	}

	afterFields := baseField + len(f.ExtractFields)*2
	if f.focusedField == afterFields {
		f.Name = f.editBuffer
	} else if f.focusedField == afterFields+1 {
		f.SaveAs = f.editBuffer
	}
}

func (f *TransformForm) applySortEdit() {
	switch f.focusedField {
	case 0:
		f.Input = f.editBuffer
	case 1:
		f.SortField = f.editBuffer
	case 3:
		f.Name = f.editBuffer
	case 4:
		f.SaveAs = f.editBuffer
	}
}

func (f *TransformForm) applyCountEdit() {
	switch f.focusedField {
	case 0:
		f.Input = f.editBuffer
	case 1:
		if n, err := strconv.Atoi(f.editBuffer); err == nil && n > 0 {
			f.Count = n
		}
	case 2:
		f.Name = f.editBuffer
	case 3:
		f.SaveAs = f.editBuffer
	}
}

func (f *TransformForm) applyCountOnlyEdit() {
	switch f.focusedField {
	case 0:
		f.Input = f.editBuffer
	case 1:
		f.Name = f.editBuffer
	case 2:
		f.SaveAs = f.editBuffer
	}
}

func (f *TransformForm) fetchPreview() tea.Cmd {
	f.loading = true
	f.err = ""
	return func() tea.Msg {
		req := f.buildTransformRequest()
		preview, err := f.client.PreviewTransform(req)
		if err != nil {
			return TransformPreviewedMsg{Error: err}
		}
		return TransformPreviewedMsg{Step: &preview.Step}
	}
}

func (f *TransformForm) buildTransformRequest() *client.TransformRequest {
	params := make(map[string]interface{})
	params["input"] = f.Input

	switch f.Operation {
	case TransformFilter:
		params["field"] = f.FilterField
		params["operator"] = filterOperators[f.FilterOperator].Op
		params["value"] = f.FilterValue

	case TransformExtract:
		fields := make([]map[string]string, len(f.ExtractFields))
		for i, m := range f.ExtractFields {
			fields[i] = map[string]string{"source": m.Source, "as": m.As}
		}
		params["fields"] = fields

	case TransformSort:
		params["field"] = f.SortField
		if f.SortAscending {
			params["direction"] = "asc"
		} else {
			params["direction"] = "desc"
		}

	case TransformFirst, TransformLast:
		params["count"] = f.Count
	}

	return &client.TransformRequest{
		Operation: string(f.Operation),
		Params:    params,
	}
}

func (f *TransformForm) saveTransform() (*TransformForm, tea.Cmd) {
	// Validate
	if f.Input == "" {
		f.err = "Input is required"
		return f, nil
	}
	if f.Name == "" {
		f.err = "Step name is required"
		return f, nil
	}

	// Build the step
	step := f.buildStep()
	return nil, func() tea.Msg { return TransformSavedMsg{Step: step} }
}

func (f *TransformForm) buildStep() *client.WorkflowStep {
	step := &client.WorkflowStep{
		Name:   f.Name,
		Type:   "utility",
		Target: "data.json_transform",
		SaveAs: f.SaveAs,
	}

	// Use preview step params if available, otherwise build from form
	if f.PreviewStep != nil {
		step.Params = f.PreviewStep.Params
	} else {
		// Build params manually
		req := f.buildTransformRequest()
		step.Params = map[string]interface{}{
			"operation": req.Operation,
		}
		for k, v := range req.Params {
			step.Params[k] = v
		}
	}

	return step
}

// View renders the transform form.
func (f *TransformForm) View() string {
	// Show field picker if active
	if f.showFieldPicker && f.fieldPicker != nil {
		return f.fieldPicker.View()
	}

	if f.pickingOperation {
		return f.renderOperationPicker()
	}

	switch f.Operation {
	case TransformFilter:
		return f.renderFilterForm()
	case TransformExtract:
		return f.renderExtractForm()
	case TransformSort:
		return f.renderSortForm()
	case TransformFirst, TransformLast:
		return f.renderCountForm()
	case TransformCount:
		return f.renderCountOnlyForm()
	default:
		return "Unknown operation"
	}
}

func (f *TransformForm) renderOperationPicker() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)

	var lines []string

	lines = append(lines, headerStyle.Render("Add Transform Step"))
	lines = append(lines, "")
	lines = append(lines, "What do you want to do?")
	lines = append(lines, "")

	for i, op := range transformOperations {
		if i == f.operationIndex {
			lines = append(lines, selectedStyle.Render("> "+op.Label))
			lines = append(lines, "  "+dimStyle.Render(op.Description))
		} else {
			lines = append(lines, "  "+op.Label)
		}
	}

	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("[j/k] Navigate  [Enter] Select  [Esc] Cancel"))

	return strings.Join(lines, "\n")
}

func (f *TransformForm) renderFilterForm() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	errorStyle := lipgloss.NewStyle().Foreground(theme.Error)

	var lines []string

	lines = append(lines, headerStyle.Render("Filter"))
	lines = append(lines, dimStyle.Render("Keep items matching a condition"))
	lines = append(lines, "")

	// Available variables
	if len(f.availableVars) > 0 {
		lines = append(lines, dimStyle.Render("Available: "+strings.Join(f.availableVars, ", ")))
		lines = append(lines, "")
	}

	// Input field
	lines = append(lines, f.renderTextField("Input", f.Input, 0, "Variable to filter (e.g., $tasks)", selectedStyle, dimStyle))

	// Field to filter on
	lines = append(lines, f.renderTextField("Field", f.FilterField, 1, "Property to check (e.g., status)", selectedStyle, dimStyle))

	// Operator
	lines = append(lines, f.renderOperatorField(selectedStyle, dimStyle))

	// Value
	lines = append(lines, f.renderTextField("Value", f.FilterValue, 3, "Value to compare against", selectedStyle, dimStyle))

	lines = append(lines, "")

	// Name and SaveAs
	lines = append(lines, f.renderTextField("Step name", f.Name, 4, "", selectedStyle, dimStyle))
	lines = append(lines, f.renderTextField("Save as", f.SaveAs, 5, "Variable name for output", selectedStyle, dimStyle))

	lines = append(lines, "")

	// Preview
	lines = append(lines, f.renderPreview(dimStyle))

	lines = append(lines, "")

	// Error
	if f.err != "" {
		lines = append(lines, errorStyle.Render("Error: "+f.err))
		lines = append(lines, "")
	}

	// Hints
	lines = append(lines, f.renderHints(dimStyle))

	return strings.Join(lines, "\n")
}

func (f *TransformForm) renderExtractForm() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	errorStyle := lipgloss.NewStyle().Foreground(theme.Error)

	var lines []string

	lines = append(lines, headerStyle.Render("Extract"))
	lines = append(lines, dimStyle.Render("Pull out specific fields from each item"))
	lines = append(lines, "")

	if len(f.availableVars) > 0 {
		lines = append(lines, dimStyle.Render("Available: "+strings.Join(f.availableVars, ", ")))
		lines = append(lines, "")
	}

	// Input
	lines = append(lines, f.renderTextField("Input", f.Input, 0, "Array to extract from", selectedStyle, dimStyle))
	lines = append(lines, "")

	// Field mappings
	lines = append(lines, "Fields to extract:")
	baseField := 1
	for i, mapping := range f.ExtractFields {
		lines = append(lines, f.renderTextField(fmt.Sprintf("  Source %d", i+1), mapping.Source, baseField+i*2, "Path in input (e.g., .name)", selectedStyle, dimStyle))
		lines = append(lines, f.renderTextField(fmt.Sprintf("  As %d", i+1), mapping.As, baseField+i*2+1, "Output field name", selectedStyle, dimStyle))
	}

	lines = append(lines, dimStyle.Render("  [a] Add field  [d] Remove last"))
	lines = append(lines, "")

	// Name and SaveAs
	afterFields := baseField + len(f.ExtractFields)*2
	lines = append(lines, f.renderTextField("Step name", f.Name, afterFields, "", selectedStyle, dimStyle))
	lines = append(lines, f.renderTextField("Save as", f.SaveAs, afterFields+1, "", selectedStyle, dimStyle))

	lines = append(lines, "")
	lines = append(lines, f.renderPreview(dimStyle))
	lines = append(lines, "")

	if f.err != "" {
		lines = append(lines, errorStyle.Render("Error: "+f.err))
		lines = append(lines, "")
	}

	lines = append(lines, f.renderHints(dimStyle))

	return strings.Join(lines, "\n")
}

func (f *TransformForm) renderSortForm() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	errorStyle := lipgloss.NewStyle().Foreground(theme.Error)

	var lines []string

	lines = append(lines, headerStyle.Render("Sort"))
	lines = append(lines, dimStyle.Render("Order items by a field"))
	lines = append(lines, "")

	if len(f.availableVars) > 0 {
		lines = append(lines, dimStyle.Render("Available: "+strings.Join(f.availableVars, ", ")))
		lines = append(lines, "")
	}

	lines = append(lines, f.renderTextField("Input", f.Input, 0, "Array to sort", selectedStyle, dimStyle))
	lines = append(lines, f.renderTextField("Sort by", f.SortField, 1, "Field to sort on", selectedStyle, dimStyle))

	// Direction
	var dirLine string
	if f.SortAscending {
		dirLine = "Direction:   [Ascending] Descending"
	} else {
		dirLine = "Direction:   Ascending [Descending]"
	}
	hint := ""
	if f.focusedField == 2 {
		hint = "  " + dimStyle.Render("[←/→] Toggle")
		lines = append(lines, selectedStyle.Render("> "+dirLine)+hint)
	} else {
		lines = append(lines, "  "+dirLine)
	}

	lines = append(lines, "")
	lines = append(lines, f.renderTextField("Step name", f.Name, 3, "", selectedStyle, dimStyle))
	lines = append(lines, f.renderTextField("Save as", f.SaveAs, 4, "", selectedStyle, dimStyle))

	lines = append(lines, "")
	lines = append(lines, f.renderPreview(dimStyle))
	lines = append(lines, "")

	if f.err != "" {
		lines = append(lines, errorStyle.Render("Error: "+f.err))
		lines = append(lines, "")
	}

	lines = append(lines, f.renderHints(dimStyle))

	return strings.Join(lines, "\n")
}

func (f *TransformForm) renderCountForm() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	errorStyle := lipgloss.NewStyle().Foreground(theme.Error)

	title := "First"
	desc := "Take first N items"
	if f.Operation == TransformLast {
		title = "Last"
		desc = "Take last N items"
	}

	var lines []string

	lines = append(lines, headerStyle.Render(title))
	lines = append(lines, dimStyle.Render(desc))
	lines = append(lines, "")

	if len(f.availableVars) > 0 {
		lines = append(lines, dimStyle.Render("Available: "+strings.Join(f.availableVars, ", ")))
		lines = append(lines, "")
	}

	lines = append(lines, f.renderTextField("Input", f.Input, 0, "Array to slice", selectedStyle, dimStyle))
	lines = append(lines, f.renderTextField("Count", strconv.Itoa(f.Count), 1, "Number of items", selectedStyle, dimStyle))
	lines = append(lines, "")
	lines = append(lines, f.renderTextField("Step name", f.Name, 2, "", selectedStyle, dimStyle))
	lines = append(lines, f.renderTextField("Save as", f.SaveAs, 3, "", selectedStyle, dimStyle))

	lines = append(lines, "")
	lines = append(lines, f.renderPreview(dimStyle))
	lines = append(lines, "")

	if f.err != "" {
		lines = append(lines, errorStyle.Render("Error: "+f.err))
		lines = append(lines, "")
	}

	lines = append(lines, f.renderHints(dimStyle))

	return strings.Join(lines, "\n")
}

func (f *TransformForm) renderCountOnlyForm() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	errorStyle := lipgloss.NewStyle().Foreground(theme.Error)

	var lines []string

	lines = append(lines, headerStyle.Render("Count"))
	lines = append(lines, dimStyle.Render("Get number of items"))
	lines = append(lines, "")

	if len(f.availableVars) > 0 {
		lines = append(lines, dimStyle.Render("Available: "+strings.Join(f.availableVars, ", ")))
		lines = append(lines, "")
	}

	lines = append(lines, f.renderTextField("Input", f.Input, 0, "Array to count", selectedStyle, dimStyle))
	lines = append(lines, "")
	lines = append(lines, f.renderTextField("Step name", f.Name, 1, "", selectedStyle, dimStyle))
	lines = append(lines, f.renderTextField("Save as", f.SaveAs, 2, "", selectedStyle, dimStyle))

	lines = append(lines, "")
	lines = append(lines, f.renderPreview(dimStyle))
	lines = append(lines, "")

	if f.err != "" {
		lines = append(lines, errorStyle.Render("Error: "+f.err))
		lines = append(lines, "")
	}

	lines = append(lines, f.renderHints(dimStyle))

	return strings.Join(lines, "\n")
}

func (f *TransformForm) renderTextField(label, value string, fieldIndex int, placeholder string, selectedStyle, dimStyle lipgloss.Style) string {
	display := value
	if display == "" {
		display = "(empty)"
	}

	hint := ""
	if f.editing && f.focusedField == fieldIndex {
		display = f.editBuffer[:f.cursorPos] + "|" + f.editBuffer[f.cursorPos:]
		hint = "  " + dimStyle.Render("[Enter] Confirm")
	} else if f.focusedField == fieldIndex {
		hint = "  " + dimStyle.Render("[Enter] Edit")
		// Show [v] Variable hint for Input field (field 0)
		if fieldIndex == 0 && len(f.previousOutputs) > 0 {
			hint += "  " + dimStyle.Render("[v] Variable")
		}
		if placeholder != "" {
			hint += "\n    " + dimStyle.Render(placeholder)
		}
	}

	line := fmt.Sprintf("%-12s %s", label+":", display)

	if f.focusedField == fieldIndex {
		return selectedStyle.Render("> "+line) + hint
	}
	return "  " + line
}

func (f *TransformForm) renderOperatorField(selectedStyle, dimStyle lipgloss.Style) string {
	var parts []string
	for i, op := range filterOperators {
		if i == f.FilterOperator {
			parts = append(parts, "["+op.Label+"]")
		} else {
			parts = append(parts, op.Label)
		}
	}
	line := "Operator:    " + strings.Join(parts, " ")

	hint := ""
	if f.focusedField == 2 {
		hint = "  " + dimStyle.Render("[←/→] Select")
		return selectedStyle.Render("> "+line) + hint
	}
	return "  " + line
}

func (f *TransformForm) renderPreview(dimStyle lipgloss.Style) string {
	if f.loading {
		return dimStyle.Render("Generating preview...")
	}

	if f.PreviewStep == nil {
		return dimStyle.Render("Press [p] to preview generated step")
	}

	var lines []string
	lines = append(lines, "─── Generated Step ───")

	// Show the jq query if present
	if params, ok := f.PreviewStep.Params["query"]; ok {
		lines = append(lines, "jq: "+dimStyle.Render(fmt.Sprintf("%v", params)))
	}

	return strings.Join(lines, "\n")
}

func (f *TransformForm) renderHints(dimStyle lipgloss.Style) string {
	if f.editing {
		return dimStyle.Render("[Enter] Confirm  [Esc] Cancel")
	}

	hints := "[j/k] Navigate  [Enter] Edit"

	if f.Operation == TransformExtract {
		hints += "  [a/d] Add/Remove field"
	}

	hints += "  [p] Preview  [Ctrl+s] Save  [Esc] Back"

	return dimStyle.Render(hints)
}
