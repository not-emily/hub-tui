# Phase 3.1: Builder State & Display

> **Depends on:** Phase 2 (List View Enhancement)
> **Enables:** Phase 3.2 (Builder Editing)
>
> See: [Full Plan](../plan.md)

## Goal

Create the WorkflowBuilder struct and implement read-only step list display with basic navigation.

## Key Deliverables

- `WorkflowBuilder` struct with full state management
- Step list view showing steps with type, target, and `save_as` variable
- Basic j/k navigation in step list
- Display of workflow metadata (name, trigger summary, output)
- View routing infrastructure

## Files to Create

- `internal/ui/modal/workflows_builder.go` — Builder state and step list view

## Implementation Notes

### WorkflowBuilder Struct

```go
package modal

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/pxp/hub-tui/internal/client"
)

type BuilderView int

const (
    ViewList BuilderView = iota
    ViewStepDetail
    ViewToolPicker
    ViewFieldPicker
    ViewTransformPicker
    ViewTransformForm
    ViewTriggerForm
    ViewValidation
)

type WorkflowBuilder struct {
    client *client.Client

    // Identity
    IsNew        bool
    OriginalName string

    // Workflow data
    Name        string
    Description string
    Trigger     client.TriggerConfig
    Steps       []client.WorkflowStep
    Output      string

    // Editing state
    View          BuilderView
    SelectedStep  int
    EditingStep   *client.WorkflowStep
    StepOutput    interface{}

    // Cached data
    Tools    *client.ToolsResponse
    Profiles map[string][]string  // integration -> profile names

    // UI state
    Dirty   bool
    Error   string
    Loading bool

    // Dimensions
    width  int
    height int
}
```

### Constructor and Initialization

```go
func NewWorkflowBuilder(c *client.Client, isNew bool) *WorkflowBuilder {
    return &WorkflowBuilder{
        client:   c,
        IsNew:    isNew,
        View:     ViewList,
        Steps:    []client.WorkflowStep{},
        Profiles: make(map[string][]string),
        Trigger:  client.TriggerConfig{Type: "manual"},
    }
}

func (b *WorkflowBuilder) LoadWorkflow(wf *client.Workflow) {
    b.Name = wf.Name
    b.OriginalName = wf.Name
    b.Description = wf.Description
    b.Trigger = wf.Trigger
    b.Steps = wf.Steps
    b.Output = wf.Output
    b.IsNew = false
}

func (b *WorkflowBuilder) Init() tea.Cmd {
    // Could pre-load tools here, or wait until needed
    return nil
}

func (b *WorkflowBuilder) SetSize(width, height int) {
    b.width = width
    b.height = height
}
```

### Step List View

```go
func (b *WorkflowBuilder) View() string {
    switch b.View {
    case ViewList:
        return b.renderStepList()
    // Other views will be added in later phases
    default:
        return b.renderStepList()
    }
}

func (b *WorkflowBuilder) renderStepList() string {
    var lines []string

    // Header with workflow name
    nameDisplay := b.Name
    if nameDisplay == "" {
        nameDisplay = "(unnamed)"
    }
    if b.IsNew {
        nameDisplay += " (new)"
    }
    lines = append(lines, headerStyle.Render(nameDisplay))
    lines = append(lines, "")

    // Trigger summary
    triggerInfo := b.formatTrigger()
    lines = append(lines, dimStyle.Render("Trigger: "+triggerInfo))

    // Output variable
    outputInfo := b.Output
    if outputInfo == "" {
        outputInfo = "(none)"
    }
    lines = append(lines, dimStyle.Render("Output: "+outputInfo))
    lines = append(lines, "")

    // Steps header
    lines = append(lines, "Steps:")

    if len(b.Steps) == 0 {
        lines = append(lines, dimStyle.Render("  No steps yet. Press [a] to add."))
    } else {
        for i, step := range b.Steps {
            line := b.formatStepLine(i, step)
            lines = append(lines, line)
        }
    }

    lines = append(lines, "")

    // Hints (read-only for now, editing in Phase 3.2)
    lines = append(lines, dimStyle.Render("[j/k] Navigate  [Esc] Back to list"))

    return strings.Join(lines, "\n")
}

func (b *WorkflowBuilder) formatTrigger() string {
    switch b.Trigger.Type {
    case "schedule":
        if b.Trigger.Frequency != "" {
            return fmt.Sprintf("%s at %s", b.Trigger.Frequency, b.Trigger.Time)
        }
        return "scheduled"
    case "manual":
        return "manual"
    default:
        return "manual"
    }
}

func (b *WorkflowBuilder) formatStepLine(index int, step client.WorkflowStep) string {
    // Selection indicator
    prefix := "  "
    if index == b.SelectedStep {
        prefix = "> "
    }

    // Step number and name
    name := step.Name
    if name == "" {
        name = fmt.Sprintf("step_%d", index+1)
    }

    // Type and target
    typeStr := step.Type
    target := step.Target

    // SaveAs indicator
    saveAs := ""
    if step.SaveAs != "" {
        saveAs = fmt.Sprintf(" -> $%s", step.SaveAs)
    }

    // Format: > 1. get_tasks     integration  notion.query    -> $tasks
    line := fmt.Sprintf("%s%d. %-12s  %-11s  %-16s%s",
        prefix, index+1, name, typeStr, target, saveAs)

    if index == b.SelectedStep {
        return selectedStyle.Render(line)
    }
    return line
}
```

### Basic Navigation

```go
func (b *WorkflowBuilder) Update(msg tea.Msg) (*WorkflowBuilder, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        return b.handleKeyPress(msg)
    }
    return b, nil
}

func (b *WorkflowBuilder) handleKeyPress(msg tea.KeyMsg) (*WorkflowBuilder, tea.Cmd) {
    if b.View != ViewList {
        // Other views handle their own keys (in later phases)
        return b, nil
    }

    switch msg.String() {
    case "j", "down":
        if b.SelectedStep < len(b.Steps)-1 {
            b.SelectedStep++
        }
    case "k", "up":
        if b.SelectedStep > 0 {
            b.SelectedStep--
        }
    case "esc":
        // Signal to parent to close builder
        // Return nil to indicate close, or use a message
        return nil, nil
    }

    return b, nil
}
```

### Helper: Available Variables

```go
// AvailableVariables returns variables defined by steps before the given index
func (b *WorkflowBuilder) AvailableVariables(beforeIndex int) []string {
    var vars []string
    for i := 0; i < beforeIndex && i < len(b.Steps); i++ {
        if b.Steps[i].SaveAs != "" {
            vars = append(vars, "$"+b.Steps[i].SaveAs)
        }
    }
    return vars
}
```

## Validation

- [ ] `WorkflowBuilder` struct compiles
- [ ] `NewWorkflowBuilder` creates builder with correct initial state
- [ ] `LoadWorkflow` populates all fields from existing workflow
- [ ] Step list displays with correct formatting (number, name, type, target, save_as)
- [ ] Selected step is highlighted
- [ ] j/k navigation moves selection up/down
- [ ] Selection doesn't go out of bounds
- [ ] Empty state shows "No steps yet" message
- [ ] Trigger and output display correctly
- [ ] `[Esc]` signals to close builder (returns to workflow list)
