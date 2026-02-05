# Phase 8: Transform Forms

> **Depends on:** Phase 7.2 (Field Picker)
> **Enables:** Phase 9 (Validation & Polish)
>
> See: [Full Plan](../plan.md)

## Goal

Implement preset transform forms for data manipulation without requiring jq knowledge.

## Key Deliverables

- Transform type picker (Filter, Extract, Sort, First, Last, Count)
- Dedicated form for each transform operation
- Integration with transform preview API
- Generated step insertion into workflow

## Files to Create

- `internal/ui/modal/workflows_transform.go` — Transform forms

## Implementation Notes

### Transform Types

```go
package modal

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/pxp/hub-tui/internal/client"
)

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
    {"greater_than", "greater than"},
    {"less_than", "less than"},
    {"greater_or_equal", "greater or equal"},
    {"less_or_equal", "less or equal"},
}
```

### TransformForm Struct

```go
type TransformForm struct {
    client *client.Client

    // Current view
    pickingOperation bool
    operationIndex   int

    // Selected operation
    Operation TransformOperation

    // Common fields
    Input  string  // variable reference
    SaveAs string
    Name   string  // step name

    // Filter-specific
    FilterField    string
    FilterOperator string
    FilterValue    string

    // Extract-specific
    ExtractFields []ExtractFieldMapping

    // Sort-specific
    SortField     string
    SortDirection string  // "asc", "desc"

    // First/Last-specific
    Count int

    // Preview
    PreviewStep *client.WorkflowStep
    PreviewJQ   string

    // UI state
    focusedField int
    editing      bool
    editBuffer   string
    loading      bool
    error        string

    // Available variables
    availableVars []string
}

type ExtractFieldMapping struct {
    Source string  // path from input
    As     string  // output field name
}
```

### Operation Picker

```go
func NewTransformForm(c *client.Client) *TransformForm {
    return &TransformForm{
        client:           c,
        pickingOperation: true,
        operationIndex:   0,
        FilterOperator:   "equals",
        SortDirection:    "asc",
        Count:            1,
    }
}

func (f *TransformForm) SetAvailableVariables(vars []string) {
    f.availableVars = vars
}

func (f *TransformForm) View() string {
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
```

### Filter Form

```go
func (f *TransformForm) renderFilterForm() string {
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
    lines = append(lines, f.renderField("Input", f.Input, 0, "Variable to filter (e.g., $tasks.results)"))

    // Field to filter on
    lines = append(lines, f.renderField("Field", f.FilterField, 1, "Property to check (e.g., status)"))

    // Operator
    lines = append(lines, f.renderOperatorField())

    // Value
    lines = append(lines, f.renderField("Value", f.FilterValue, 3, "Value to compare against"))

    lines = append(lines, "")

    // Name and SaveAs
    lines = append(lines, f.renderField("Step name", f.Name, 4, ""))
    lines = append(lines, f.renderField("Save as", f.SaveAs, 5, "Variable name for output"))

    lines = append(lines, "")

    // Preview
    lines = append(lines, f.renderPreview())

    lines = append(lines, "")
    lines = append(lines, f.renderHints())

    if f.error != "" {
        lines = append(lines, errorStyle.Render("Error: "+f.error))
    }

    return strings.Join(lines, "\n")
}

func (f *TransformForm) renderOperatorField() string {
    line := "Operator: "
    for i, op := range filterOperators {
        if op.Op == f.FilterOperator {
            line += "[" + op.Label + "] "
        } else {
            line += op.Label + " "
        }
        if i < len(filterOperators)-1 {
            line += "| "
        }
    }

    if f.focusedField == 2 {
        return selectedStyle.Render(line)
    }
    return line
}
```

### Extract Form

```go
func (f *TransformForm) renderExtractForm() string {
    var lines []string

    lines = append(lines, headerStyle.Render("Extract"))
    lines = append(lines, dimStyle.Render("Pull out specific fields from each item"))
    lines = append(lines, "")

    // Input
    lines = append(lines, f.renderField("Input", f.Input, 0, "Array to extract from"))
    lines = append(lines, "")

    // Field mappings
    lines = append(lines, "Fields to extract:")
    for i, mapping := range f.ExtractFields {
        fieldIndex := 1 + i*2
        lines = append(lines, f.renderField(fmt.Sprintf("  Source %d", i+1), mapping.Source, fieldIndex, "Path in input"))
        lines = append(lines, f.renderField(fmt.Sprintf("  As %d", i+1), mapping.As, fieldIndex+1, "Output field name"))
    }

    // Add field hint
    lines = append(lines, dimStyle.Render("  [a] Add field  [d] Remove last"))
    lines = append(lines, "")

    // Name and SaveAs
    baseField := 1 + len(f.ExtractFields)*2
    lines = append(lines, f.renderField("Step name", f.Name, baseField, ""))
    lines = append(lines, f.renderField("Save as", f.SaveAs, baseField+1, ""))

    lines = append(lines, "")
    lines = append(lines, f.renderPreview())
    lines = append(lines, "")
    lines = append(lines, f.renderHints())

    return strings.Join(lines, "\n")
}
```

### Sort Form

```go
func (f *TransformForm) renderSortForm() string {
    var lines []string

    lines = append(lines, headerStyle.Render("Sort"))
    lines = append(lines, dimStyle.Render("Order items by a field"))
    lines = append(lines, "")

    lines = append(lines, f.renderField("Input", f.Input, 0, "Array to sort"))
    lines = append(lines, f.renderField("Sort by", f.SortField, 1, "Field to sort on"))

    // Direction
    dirLine := "Direction: "
    if f.SortDirection == "asc" {
        dirLine += "[Ascending] Descending"
    } else {
        dirLine += "Ascending [Descending]"
    }
    if f.focusedField == 2 {
        lines = append(lines, selectedStyle.Render(dirLine))
    } else {
        lines = append(lines, dirLine)
    }

    lines = append(lines, "")
    lines = append(lines, f.renderField("Step name", f.Name, 3, ""))
    lines = append(lines, f.renderField("Save as", f.SaveAs, 4, ""))

    lines = append(lines, "")
    lines = append(lines, f.renderPreview())
    lines = append(lines, "")
    lines = append(lines, f.renderHints())

    return strings.Join(lines, "\n")
}
```

### First/Last Form

```go
func (f *TransformForm) renderCountForm() string {
    var lines []string

    title := "First"
    desc := "Take first N items"
    if f.Operation == TransformLast {
        title = "Last"
        desc = "Take last N items"
    }

    lines = append(lines, headerStyle.Render(title))
    lines = append(lines, dimStyle.Render(desc))
    lines = append(lines, "")

    lines = append(lines, f.renderField("Input", f.Input, 0, "Array to slice"))
    lines = append(lines, f.renderField("Count", fmt.Sprintf("%d", f.Count), 1, "Number of items"))
    lines = append(lines, "")
    lines = append(lines, f.renderField("Step name", f.Name, 2, ""))
    lines = append(lines, f.renderField("Save as", f.SaveAs, 3, ""))

    lines = append(lines, "")
    lines = append(lines, f.renderPreview())
    lines = append(lines, "")
    lines = append(lines, f.renderHints())

    return strings.Join(lines, "\n")
}
```

### Preview

```go
func (f *TransformForm) renderPreview() string {
    if f.loading {
        return dimStyle.Render("Generating preview...")
    }

    if f.PreviewStep == nil {
        return dimStyle.Render("Press [p] to preview generated step")
    }

    var lines []string
    lines = append(lines, "─── Generated Step ───")

    // Show the jq query
    if params, ok := f.PreviewStep.Params["query"]; ok {
        lines = append(lines, "jq: "+dimStyle.Render(fmt.Sprintf("%v", params)))
    }

    return strings.Join(lines, "\n")
}

func (f *TransformForm) fetchPreview() tea.Cmd {
    f.loading = true
    return func() tea.Msg {
        req := f.buildTransformRequest()
        preview, err := f.client.PreviewTransform(req)
        return TransformPreviewedMsg{
            Step:  &preview.Step,
            Error: err,
        }
    }
}

func (f *TransformForm) buildTransformRequest() *client.TransformRequest {
    params := make(map[string]interface{})
    params["input"] = f.Input

    switch f.Operation {
    case TransformFilter:
        params["field"] = f.FilterField
        params["operator"] = f.FilterOperator
        params["value"] = f.FilterValue

    case TransformExtract:
        fields := make([]map[string]string, len(f.ExtractFields))
        for i, m := range f.ExtractFields {
            fields[i] = map[string]string{"source": m.Source, "as": m.As}
        }
        params["fields"] = fields

    case TransformSort:
        params["field"] = f.SortField
        params["direction"] = f.SortDirection

    case TransformFirst, TransformLast:
        params["count"] = f.Count
    }

    return &client.TransformRequest{
        Operation: string(f.Operation),
        Params:    params,
    }
}
```

### Update Handling

```go
func (f *TransformForm) Update(msg tea.Msg) (*TransformForm, tea.Cmd) {
    switch msg := msg.(type) {
    case TransformPreviewedMsg:
        f.loading = false
        if msg.Error != nil {
            f.error = msg.Error.Error()
        } else {
            f.PreviewStep = msg.Step
            f.error = ""
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
    case "esc":
        return nil, nil  // Cancel
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
    case "k", "up":
        if f.focusedField > 0 {
            f.focusedField--
        }
    case "enter":
        f.startEditing()
    case "h", "left", "l", "right":
        f.handleLeftRight(msg.String())
    case "p":
        return f, f.fetchPreview()
    case "s", "ctrl+s":
        return f.saveTransform()
    case "esc":
        return nil, nil
    }
    return f, nil
}

func (f *TransformForm) saveTransform() (*TransformForm, tea.Cmd) {
    // Validate
    if f.Input == "" {
        f.error = "Input is required"
        return f, nil
    }
    if f.Name == "" {
        f.error = "Step name is required"
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

    // Use preview step params if available
    if f.PreviewStep != nil {
        step.Params = f.PreviewStep.Params
    }

    return step
}
```

## Validation

- [ ] Operation picker shows all 6 transform types
- [ ] Each operation shows description
- [ ] j/k navigates, Enter selects operation
- [ ] Filter form has Input, Field, Operator, Value fields
- [ ] Filter operator cycles with left/right
- [ ] Extract form allows multiple field mappings
- [ ] [a] adds field mapping, [d] removes last
- [ ] Sort form has direction toggle
- [ ] First/Last form has count input
- [ ] Count form only needs Input
- [ ] [p] fetches preview from API
- [ ] Preview shows generated jq query
- [ ] [s] validates and saves step
- [ ] Generated step has correct type/target
- [ ] Esc cancels at any point
