# Phase 3.2: Builder Editing

> **Depends on:** Phase 3.1 (Builder State & Display)
> **Enables:** Phase 4 (Trigger Form), Phase 5 (Tool Picker)
>
> See: [Full Plan](../plan.md)

## Goal

Add metadata editing (name, output) and step management (add, edit, delete, reorder).

## Key Deliverables

- Name editing field
- Output variable dropdown (populated from steps' save_as)
- `[a]` to add new step
- `[e]` or `[Enter]` to edit selected step
- `[d]` to delete selected step
- `[J]`/`[K]` to reorder steps (move up/down)
- Dirty state tracking
- Save workflow (`[s]` or `Ctrl+S`)

## Files to Modify

- `internal/ui/modal/workflows_builder.go` — Add editing capabilities

## Implementation Notes

### Extended State

```go
type WorkflowBuilder struct {
    // ... existing fields ...

    // Editing state
    editingName   bool      // true when name field is focused
    nameInput     string    // current name input value
    editingOutput bool      // true when output field is focused
    outputOptions []string  // available output variables
    outputIndex   int       // selected output option

    // Focus tracking
    focusedField  int       // 0=name, 1=output, 2+=steps
}
```

### Metadata Editing

```go
func (b *WorkflowBuilder) renderMetadata() string {
    var lines []string

    // Name field
    nameLabel := "Name: "
    if b.editingName {
        nameLabel += "[" + b.nameInput + "_]"  // Show cursor
    } else {
        name := b.Name
        if name == "" {
            name = "(required)"
        }
        if b.focusedField == 0 {
            nameLabel += selectedStyle.Render(name) + " [Enter to edit]"
        } else {
            nameLabel += name
        }
    }
    lines = append(lines, nameLabel)

    // Output dropdown
    outputLabel := "Output: "
    b.outputOptions = b.buildOutputOptions()
    if len(b.outputOptions) == 0 {
        outputLabel += dimStyle.Render("(no variables defined)")
    } else if b.editingOutput {
        // Show dropdown
        for i, opt := range b.outputOptions {
            if i == b.outputIndex {
                outputLabel += selectedStyle.Render("["+opt+"]") + " "
            } else {
                outputLabel += opt + " "
            }
        }
    } else {
        output := b.Output
        if output == "" {
            output = "(none)"
        }
        if b.focusedField == 1 {
            outputLabel += selectedStyle.Render(output) + " [Enter to select]"
        } else {
            outputLabel += output
        }
    }
    lines = append(lines, outputLabel)

    return strings.Join(lines, "\n")
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
```

### Step Management Keys

```go
func (b *WorkflowBuilder) handleKeyPress(msg tea.KeyMsg) (*WorkflowBuilder, tea.Cmd) {
    // Handle name editing mode
    if b.editingName {
        return b.handleNameInput(msg)
    }

    // Handle output selection mode
    if b.editingOutput {
        return b.handleOutputSelect(msg)
    }

    switch msg.String() {
    // Navigation
    case "j", "down":
        b.moveSelectionDown()
    case "k", "up":
        b.moveSelectionUp()

    // Step reordering
    case "J":  // Shift+J
        b.moveStepDown()
    case "K":  // Shift+K
        b.moveStepUp()

    // Step management
    case "a":
        return b.addStep()
    case "e", "enter":
        if b.focusedField == 0 {
            b.editingName = true
            b.nameInput = b.Name
        } else if b.focusedField == 1 {
            b.editingOutput = true
            b.outputIndex = b.findOutputIndex()
        } else if len(b.Steps) > 0 {
            return b.editStep(b.SelectedStep)
        }
    case "d":
        if b.focusedField >= 2 && len(b.Steps) > 0 {
            return b.deleteStep(b.SelectedStep)
        }

    // Save
    case "s", "ctrl+s":
        return b.saveWorkflow()

    // Trigger editing
    case "t":
        b.View = ViewTriggerForm
        return b, nil

    // Close
    case "esc":
        if b.Dirty {
            // Could show confirmation, for now just close
        }
        return nil, nil
    }

    return b, nil
}
```

### Step Reordering

```go
func (b *WorkflowBuilder) moveStepDown() {
    if b.SelectedStep < len(b.Steps)-1 {
        // Swap with next step
        b.Steps[b.SelectedStep], b.Steps[b.SelectedStep+1] =
            b.Steps[b.SelectedStep+1], b.Steps[b.SelectedStep]
        b.SelectedStep++
        b.Dirty = true
    }
}

func (b *WorkflowBuilder) moveStepUp() {
    if b.SelectedStep > 0 {
        // Swap with previous step
        b.Steps[b.SelectedStep], b.Steps[b.SelectedStep-1] =
            b.Steps[b.SelectedStep-1], b.Steps[b.SelectedStep]
        b.SelectedStep--
        b.Dirty = true
    }
}
```

### Add/Delete Steps

```go
func (b *WorkflowBuilder) addStep() (*WorkflowBuilder, tea.Cmd) {
    // Create placeholder step
    newStep := client.WorkflowStep{
        Name: fmt.Sprintf("step_%d", len(b.Steps)+1),
    }
    b.Steps = append(b.Steps, newStep)
    b.SelectedStep = len(b.Steps) - 1
    b.Dirty = true

    // Enter tool picker to select tool for this step
    b.View = ViewToolPicker
    // Tool picker will be implemented in Phase 5

    return b, nil
}

func (b *WorkflowBuilder) deleteStep(index int) (*WorkflowBuilder, tea.Cmd) {
    if index < 0 || index >= len(b.Steps) {
        return b, nil
    }

    // Remove step
    b.Steps = append(b.Steps[:index], b.Steps[index+1:]...)

    // Adjust selection
    if b.SelectedStep >= len(b.Steps) && b.SelectedStep > 0 {
        b.SelectedStep--
    }

    b.Dirty = true
    return b, nil
}
```

### Save Workflow

```go
func (b *WorkflowBuilder) saveWorkflow() (*WorkflowBuilder, tea.Cmd) {
    // Basic validation
    if b.Name == "" {
        b.Error = "Workflow name is required"
        return b, nil
    }

    b.Loading = true
    return b, b.doSave()
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

func (b *WorkflowBuilder) ToWorkflow() *client.Workflow {
    return &client.Workflow{
        Name:        b.Name,
        Description: b.Description,
        Trigger:     b.Trigger,
        Steps:       b.Steps,
        Output:      b.Output,
    }
}
```

### Updated Hints

```go
func (b *WorkflowBuilder) renderHints() string {
    hints := []string{}

    if b.focusedField >= 2 && len(b.Steps) > 0 {
        hints = append(hints, "[a]dd", "[e]dit", "[d]elete", "[J/K]move")
    } else {
        hints = append(hints, "[a]dd step")
    }

    hints = append(hints, "[t]rigger", "[s]ave", "[Esc]cancel")

    if b.Dirty {
        return dimStyle.Render(strings.Join(hints, "  ")) + " " +
               warningStyle.Render("(unsaved)")
    }
    return dimStyle.Render(strings.Join(hints, "  "))
}
```

## Validation

- [ ] Can edit workflow name (Enter to edit, Enter to confirm, Esc to cancel)
- [ ] Output dropdown shows all `save_as` variables from steps
- [ ] Can select output variable with j/k and Enter
- [ ] `[a]` adds a new step and enters tool picker (placeholder for now)
- [ ] `[e]` or Enter on step enters step detail (placeholder for now)
- [ ] `[d]` deletes selected step
- [ ] `[J]` moves selected step down
- [ ] `[K]` moves selected step up
- [ ] Dirty state shows "(unsaved)" indicator
- [ ] `[s]` or `Ctrl+S` saves workflow
- [ ] Create mode calls `CreateWorkflow`
- [ ] Edit mode calls `UpdateWorkflow` with original name
- [ ] Save errors display appropriately
- [ ] After successful save, dirty state clears
