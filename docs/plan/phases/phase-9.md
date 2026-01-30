# Phase 9: Validation & Polish

> **Depends on:** Phase 8 (Transform Forms)
> **Enables:** Production-ready workflow builder
>
> See: [Full Plan](../plan.md)

## Goal

Add workflow validation, error display, and UX polish for a production-ready builder.

## Key Deliverables

- `[v]` to validate workflow via API
- Validation error display with step/field highlighting
- Block save if invalid
- Dirty state warning on close
- Available variables in step detail header with color coding
- Overall UX improvements

## Files to Modify

- `internal/ui/modal/workflows_builder.go` — Validation view, dirty handling
- `internal/ui/modal/workflows_step.go` — Variable color coding

## Implementation Notes

### Validation View

```go
// Add to BuilderView enum
const (
    // ... existing views ...
    ViewValidation
)

// Validation state in builder
type WorkflowBuilder struct {
    // ... existing fields ...

    // Validation
    validationResult *client.ValidationResult
    validationLoading bool
}

func (b *WorkflowBuilder) validateWorkflow() tea.Cmd {
    b.validationLoading = true
    return func() tea.Msg {
        wf := b.ToWorkflow()
        result, err := b.client.ValidateWorkflow(wf)
        if err != nil {
            return WorkflowValidatedMsg{
                Valid:  false,
                Errors: []client.ValidationError{{Message: err.Error()}},
            }
        }
        return WorkflowValidatedMsg{
            Valid:  result.Valid,
            Errors: result.Errors,
        }
    }
}
```

### Validation Error Display

```go
func (b *WorkflowBuilder) renderValidation() string {
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
        lines = append(lines, dimStyle.Render("Press [s] to save or [Esc] to continue editing"))
    } else {
        lines = append(lines, errorStyle.Render("✗ Validation failed"))
        lines = append(lines, "")

        // Group errors by step
        workflowErrors := []client.ValidationError{}
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
                lines = append(lines, "  • "+err.Message)
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

        lines = append(lines, dimStyle.Render("Press [Esc] to continue editing"))
    }

    return strings.Join(lines, "\n")
}
```

### Validate Before Save

```go
func (b *WorkflowBuilder) saveWorkflow() (*WorkflowBuilder, tea.Cmd) {
    // Always validate before save
    return b, tea.Batch(
        b.validateWorkflow(),
        func() tea.Msg { return validateThenSaveMsg{} },
    )
}

type validateThenSaveMsg struct{}

// In Update:
case WorkflowValidatedMsg:
    b.validationLoading = false
    b.validationResult = &client.ValidationResult{
        Valid:  msg.Valid,
        Errors: msg.Errors,
    }

    // If this was a save attempt and valid, proceed with save
    if b.pendingSave && msg.Valid {
        b.pendingSave = false
        return b, b.doSave()
    } else if !msg.Valid {
        b.View = ViewValidation
        b.pendingSave = false
    }
    return b, nil
```

### Dirty State Warning

```go
func (b *WorkflowBuilder) handleClose() (*WorkflowBuilder, tea.Cmd) {
    if b.Dirty {
        // Show confirmation
        b.showCloseConfirm = true
        return b, nil
    }
    return nil, nil  // Close
}

func (b *WorkflowBuilder) renderCloseConfirm() string {
    return `You have unsaved changes.

[y] Discard and close
[n] Cancel and keep editing
[s] Save and close`
}

// In key handling:
if b.showCloseConfirm {
    switch msg.String() {
    case "y":
        return nil, nil  // Close without saving
    case "n", "esc":
        b.showCloseConfirm = false
    case "s":
        b.showCloseConfirm = false
        return b.saveWorkflow()
    }
    return b, nil
}
```

### Variable Color Coding

In step detail, show variables with color coding:

```go
func (f *StepForm) renderAvailableVariables() string {
    if len(f.availableVars) == 0 {
        return dimStyle.Render("No variables available yet")
    }

    var parts []string
    for _, v := range f.availableVars {
        // Check if this variable is actually defined (has test output)
        if f.isVariableDefined(v) {
            parts = append(parts, successStyle.Render(v))  // green
        } else {
            parts = append(parts, warningStyle.Render(v+"(?)"))  // yellow
        }
    }

    return "Available: " + strings.Join(parts, ", ")
}

func (f *StepForm) isVariableDefined(varName string) bool {
    // Check if we have test output for this variable
    name := strings.TrimPrefix(varName, "$")
    _, exists := f.previousOutputs[name]
    return exists
}
```

### Step List Error Indicators

Show validation errors in step list:

```go
func (b *WorkflowBuilder) formatStepLine(index int, step client.WorkflowStep) string {
    // ... existing formatting ...

    // Add error indicator if validation failed for this step
    if b.validationResult != nil && !b.validationResult.Valid {
        for _, err := range b.validationResult.Errors {
            if err.Step != nil && *err.Step == index {
                line += " " + errorStyle.Render("!")
                break
            }
        }
    }

    // ... rest of formatting ...
}
```

### UX Improvements

```go
// 1. Auto-focus first empty required field when entering step detail
func (f *StepForm) focusFirstEmpty() {
    // Check name
    if f.Step.Name == "" {
        f.focusedField = stepFieldName
        return
    }

    // Check profile
    if f.Tool.RequiresProfile && f.Step.Profile == "" {
        f.focusedField = stepFieldProfile
        return
    }

    // Check params
    for i, param := range f.Tool.Params {
        if param.Required && f.getParamValue(param.Name) == "" {
            f.focusedField = stepFieldParamsStart + i
            return
        }
    }
}

// 2. Show step count and progress
func (b *WorkflowBuilder) renderHeader() string {
    title := b.Name
    if title == "" {
        title = "(unnamed workflow)"
    }

    if b.Dirty {
        title += " *"
    }

    stepCount := fmt.Sprintf("(%d steps)", len(b.Steps))

    return headerStyle.Render(title) + " " + dimStyle.Render(stepCount)
}

// 3. Keyboard shortcut reminder
func (b *WorkflowBuilder) renderQuickHelp() string {
    return dimStyle.Render("? for help")
}

// 4. Help overlay
func (b *WorkflowBuilder) renderHelp() string {
    return `Workflow Builder Help

Navigation:
  j/k       Move selection up/down
  J/K       Move step up/down (reorder)
  Enter     Edit selected item
  Esc       Go back / Cancel

Actions:
  a         Add new step
  e         Edit selected step
  d         Delete selected step
  t         Edit trigger
  v         Validate workflow
  s         Save workflow

In Step Editor:
  t         Test step
  p         Pick field from output

Press any key to close help`
}
```

### Final Hints

```go
func (b *WorkflowBuilder) renderHints() string {
    var hints []string

    if len(b.Steps) > 0 {
        hints = append(hints, "[a]dd", "[e]dit", "[d]el", "[J/K]move")
    } else {
        hints = append(hints, "[a]dd step")
    }

    hints = append(hints, "[t]rigger", "[v]alidate", "[s]ave", "[?]help")

    base := strings.Join(hints, "  ")

    if b.Dirty {
        return dimStyle.Render(base) + " " + warningStyle.Render("(unsaved)")
    }
    return dimStyle.Render(base)
}
```

## Validation Checklist

- [ ] `[v]` triggers workflow validation
- [ ] Validation view shows success or errors
- [ ] Errors grouped by workflow vs step
- [ ] Step errors show step name and field
- [ ] Save validates first, shows errors if invalid
- [ ] Valid workflow saves successfully
- [ ] Invalid workflow blocks save, shows errors
- [ ] Dirty state tracked on all changes
- [ ] Close with dirty state shows confirmation
- [ ] Discard/Cancel/Save options in confirmation
- [ ] Variables show green if defined (tested)
- [ ] Variables show yellow if not yet tested
- [ ] Step list shows error indicator for invalid steps
- [ ] Help overlay shows all keyboard shortcuts
- [ ] `[?]` opens help overlay
- [ ] Any key closes help overlay

## End-to-End Test Scenarios

1. **Create simple workflow**
   - Open /workflows, press [n]
   - Set name "test-workflow"
   - Set trigger to Manual
   - Add step: primitive > time > now
   - Set save_as to "now"
   - Set output to "$now"
   - Validate (should pass)
   - Save

2. **Create workflow with data flow**
   - Create workflow with scheduled trigger
   - Add step 1: integration > notion > query_database
   - Configure params, set save_as "tasks"
   - Test step, see output
   - Add step 2: transform > filter
   - Pick input from step 1 output
   - Configure filter
   - Add step 3: integration > llm > complete
   - Reference $filtered in prompt
   - Validate and save

3. **Edit existing workflow**
   - Select workflow in list
   - Press [e]
   - Modify a step
   - Validate
   - Save

4. **Validation errors**
   - Create workflow with missing required params
   - Try to save
   - See validation errors
   - Fix errors
   - Save successfully

5. **Dirty state handling**
   - Make changes
   - Press Esc
   - See confirmation
   - Choose discard/cancel/save appropriately
