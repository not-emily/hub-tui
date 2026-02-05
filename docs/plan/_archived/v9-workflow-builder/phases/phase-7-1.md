# Phase 7.1: Step Testing

> **Depends on:** Phase 6 (Step Detail Form)
> **Enables:** Phase 7.2 (Field Picker)
>
> See: [Full Plan](../plan.md)

## Goal

Add ability to test individual steps and display their output.

## Key Deliverables

- `[t]` hotkey in step detail to test current step
- Build variables map from previous steps' test results
- Display test output (success/error)
- Store output for field picker

## Files to Modify

- `internal/ui/modal/workflows_step.go` — Add test functionality

## Implementation Notes

### Test State

Add to `StepForm`:

```go
type StepForm struct {
    // ... existing fields ...

    // Test state
    testOutput  interface{}
    testError   string
    testLoading bool
    showOutput  bool  // toggle to show/hide output

    // Previous step outputs (for building variables map)
    previousOutputs map[string]interface{}
}

func (f *StepForm) SetPreviousOutputs(outputs map[string]interface{}) {
    f.previousOutputs = outputs
}
```

### Test Execution

```go
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
            Output: result,
            Error:  err,
        }
    }
}
```

### Handle Test Result

```go
func (f *StepForm) Update(msg tea.Msg) (*StepForm, tea.Cmd) {
    switch msg := msg.(type) {
    case StepTestedMsg:
        f.testLoading = false
        if msg.Error != nil {
            f.testError = msg.Error.Error()
            f.testOutput = nil
        } else if msg.Output != nil {
            if msg.Output.Success {
                f.testOutput = msg.Output.Output
                f.testError = ""
            } else {
                f.testError = msg.Output.Error
                f.testOutput = nil
            }
        }
        return f, nil

    // ... existing cases ...
    }
}

// In handleKeyPress:
case "t":
    // Validate step has required fields before testing
    if !f.canTest() {
        f.error = "Fill required fields before testing"
        return f, nil
    }
    return f, f.testStep()

case "o":
    // Toggle output visibility
    f.showOutput = !f.showOutput
```

### Render Test Output

```go
func (f *StepForm) View() string {
    var lines []string

    // ... existing rendering ...

    // Test output section
    if f.showOutput {
        lines = append(lines, "")
        lines = append(lines, f.renderTestOutput())
    }

    // ... hints and errors ...
}

func (f *StepForm) renderTestOutput() string {
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

    // Hint about field picker
    lines = append(lines, "")
    lines = append(lines, dimStyle.Render("[p] Pick field from output"))

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
        return truncated + "\n" + dimStyle.Render("... (truncated, "+strconv.Itoa(len(lines)-maxLines)+" more lines)")
    }

    return s
}
```

### Validation Before Test

```go
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
```

### Update Hints

```go
func (f *StepForm) renderHints() string {
    if f.editing {
        return dimStyle.Render("[Enter] Confirm  [Esc] Cancel")
    }

    hints := "[j/k] Navigate  [Enter] Edit"

    if f.canTest() {
        hints += "  [t] Test"
    }

    if f.testOutput != nil {
        hints += "  [p] Pick field"
        hints += "  [o] Toggle output"
    }

    hints += "  [s] Save  [Esc] Cancel"

    return dimStyle.Render(hints)
}
```

### Store Output in Builder

The builder needs to track test outputs from all steps:

```go
// In WorkflowBuilder
type WorkflowBuilder struct {
    // ... existing fields ...

    // Test outputs keyed by step save_as name
    StepOutputs map[string]interface{}
}

// When entering step form, pass previous outputs
func (b *WorkflowBuilder) editStep(index int) (*WorkflowBuilder, tea.Cmd) {
    step := &b.Steps[index]
    tool := b.findToolForStep(step)

    b.stepForm = NewStepForm(b.client, step, tool, false)
    b.stepForm.SetAvailableVariables(b.AvailableVariables(index))
    b.stepForm.SetPreviousOutputs(b.buildPreviousOutputs(index))
    b.View = ViewStepDetail

    return b, b.stepForm.Init()
}

func (b *WorkflowBuilder) buildPreviousOutputs(beforeIndex int) map[string]interface{} {
    outputs := make(map[string]interface{})
    for i := 0; i < beforeIndex; i++ {
        if b.Steps[i].SaveAs != "" {
            if output, ok := b.StepOutputs[b.Steps[i].SaveAs]; ok {
                outputs[b.Steps[i].SaveAs] = output
            }
        }
    }
    return outputs
}

// After step test, store the output
case StepTestedMsg:
    if msg.Output != nil && msg.Output.Success && b.stepForm != nil {
        if b.stepForm.Step.SaveAs != "" {
            b.StepOutputs[b.stepForm.Step.SaveAs] = msg.Output.Output
        }
    }
```

## Validation

- [ ] `[t]` triggers step test when required fields are filled
- [ ] `[t]` shows error if required fields missing
- [ ] Loading state shows "Running..."
- [ ] Success shows formatted JSON output
- [ ] Error shows error message in red
- [ ] Output truncates if > 15 lines
- [ ] `[o]` toggles output visibility
- [ ] `[p]` hint appears when output exists
- [ ] Variables from previous steps passed to test
- [ ] Test output stored for field picker use
- [ ] Test output keyed by step's `save_as` name
