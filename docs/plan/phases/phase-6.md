# Phase 6: Step Detail Form

> **Depends on:** Phase 5 (Tool Picker)
> **Enables:** Phase 7.1 (Step Testing)
>
> See: [Full Plan](../plan.md)

## Goal

Implement dynamic parameter forms for step configuration based on tool schemas.

## Key Deliverables

- Step detail view showing tool info and parameters
- Dynamic form fields based on tool param schema
- Profile dropdown for integration tools
- `save_as` field for naming step output
- Step name editing
- Save/cancel step editing

## Files to Create

- `internal/ui/modal/workflows_step.go` — Step detail form

## Implementation Notes

### StepForm Struct

```go
package modal

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/pxp/hub-tui/internal/client"
)

type StepForm struct {
    client *client.Client

    // Step being edited
    Step    *client.WorkflowStep
    Tool    *client.Tool  // tool schema
    IsNew   bool

    // Form state
    focusedField int
    editing      bool
    editBuffer   string

    // Profile options (for integration tools)
    profiles       []string
    profileIndex   int
    loadingProfile bool

    // Available variables (from previous steps)
    availableVars []string

    // UI state
    error   string
    loading bool
}

const (
    stepFieldName = iota
    stepFieldProfile  // only shown for integrations
    stepFieldSaveAs
    stepFieldParamsStart  // params start here
)
```

### Constructor

```go
func NewStepForm(c *client.Client, step *client.WorkflowStep, tool *client.Tool, isNew bool) *StepForm {
    f := &StepForm{
        client: c,
        Step:   step,
        Tool:   tool,
        IsNew:  isNew,
    }

    // Initialize step from tool if new
    if isNew {
        f.Step.Type = toolTypeFromCategory(tool)  // determine from how tool was found
        f.Step.Target = tool.Target
        if f.Step.Name == "" {
            f.Step.Name = tool.Name
        }
    }

    return f
}

func (f *StepForm) SetAvailableVariables(vars []string) {
    f.availableVars = vars
}

func (f *StepForm) Init() tea.Cmd {
    // If integration tool, load profiles
    if f.Tool.RequiresProfile {
        return f.loadProfiles()
    }
    return nil
}

func (f *StepForm) loadProfiles() tea.Cmd {
    f.loadingProfile = true
    integration := extractIntegration(f.Step.Target)  // e.g., "notion" from "notion.query_database"
    return func() tea.Msg {
        profiles, err := f.client.GetIntegrationProfiles(integration)
        return StepProfilesLoadedMsg{
            Integration: integration,
            Profiles:    profiles,
            Error:       err,
        }
    }
}
```

### View Rendering

```go
func (f *StepForm) View() string {
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
    lines = append(lines, f.renderField("Name", f.Step.Name, stepFieldName))

    // Profile dropdown (for integrations)
    if f.Tool.RequiresProfile {
        lines = append(lines, f.renderProfileField())
    }

    // Dynamic params
    for i, param := range f.Tool.Params {
        fieldIndex := stepFieldParamsStart + i
        value := f.getParamValue(param.Name)
        lines = append(lines, f.renderParamField(param, value, fieldIndex))
    }

    // SaveAs field
    lines = append(lines, "")
    lines = append(lines, f.renderField("Save as", f.Step.SaveAs, stepFieldSaveAs))

    lines = append(lines, "")
    lines = append(lines, f.renderHints())

    if f.error != "" {
        lines = append(lines, errorStyle.Render("Error: "+f.error))
    }

    return strings.Join(lines, "\n")
}

func (f *StepForm) renderField(label, value string, fieldIndex int) string {
    display := value
    if display == "" {
        display = "(empty)"
    }

    if f.editing && f.focusedField == fieldIndex {
        display = "[" + f.editBuffer + "_]"
    }

    line := label + ": " + display

    if f.focusedField == fieldIndex {
        return selectedStyle.Render(line)
    }
    return line
}

func (f *StepForm) renderParamField(param client.ToolParam, value string, fieldIndex int) string {
    // Label with required indicator
    label := param.Name
    if param.Required {
        label += "*"
    }

    // Description as hint
    hint := ""
    if param.Description != "" && f.focusedField == fieldIndex {
        hint = "\n  " + dimStyle.Render(param.Description)
    }

    return f.renderField(label, value, fieldIndex) + hint
}

func (f *StepForm) renderProfileField() string {
    if f.loadingProfile {
        return "Profile: " + dimStyle.Render("loading...")
    }

    if len(f.profiles) == 0 {
        return "Profile: " + errorStyle.Render("(no profiles configured)")
    }

    // Show dropdown
    line := "Profile: "
    for i, p := range f.profiles {
        if i == f.profileIndex {
            if f.focusedField == stepFieldProfile {
                line += selectedStyle.Render("["+p+"]")
            } else {
                line += "[" + p + "]"
            }
        } else {
            line += " " + p + " "
        }
    }

    return line
}

func (f *StepForm) renderHints() string {
    if f.editing {
        return dimStyle.Render("[Enter] Confirm  [Esc] Cancel")
    }
    return dimStyle.Render("[j/k] Navigate  [Enter] Edit  [t] Test  [s] Save  [Esc] Cancel")
}
```

### Param Value Handling

```go
func (f *StepForm) getParamValue(name string) string {
    if f.Step.Params == nil {
        return ""
    }
    val, ok := f.Step.Params[name]
    if !ok {
        return ""
    }
    // Convert to string for display
    switch v := val.(type) {
    case string:
        return v
    case float64:
        return fmt.Sprintf("%v", v)
    case bool:
        return fmt.Sprintf("%v", v)
    default:
        // Arrays and objects - show as JSON
        b, _ := json.Marshal(v)
        return string(b)
    }
}

func (f *StepForm) setParamValue(name, value string, paramType string) {
    if f.Step.Params == nil {
        f.Step.Params = make(map[string]interface{})
    }

    // Parse based on type
    switch paramType {
    case "number":
        if n, err := strconv.ParseFloat(value, 64); err == nil {
            f.Step.Params[name] = n
        }
    case "boolean":
        f.Step.Params[name] = value == "true"
    case "array", "object":
        var parsed interface{}
        if err := json.Unmarshal([]byte(value), &parsed); err == nil {
            f.Step.Params[name] = parsed
        }
    default:
        f.Step.Params[name] = value
    }
}
```

### Update Handling

```go
func (f *StepForm) Update(msg tea.Msg) (*StepForm, tea.Cmd) {
    switch msg := msg.(type) {
    case StepProfilesLoadedMsg:
        f.loadingProfile = false
        if msg.Error != nil {
            f.error = msg.Error.Error()
        } else {
            f.profiles = msg.Profiles
            // Set initial selection to current profile or default
            f.profileIndex = f.findProfileIndex(f.Step.Profile)
        }
        return f, nil

    case tea.KeyMsg:
        return f.handleKeyPress(msg)
    }
    return f, nil
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
            }
        }
    case "l", "right":
        if f.focusedField == stepFieldProfile && len(f.profiles) > 0 {
            if f.profileIndex < len(f.profiles)-1 {
                f.profileIndex++
                f.Step.Profile = f.profiles[f.profileIndex]
            }
        }

    case "t":
        // Test step - will be implemented in Phase 7
        return f, nil

    case "s", "ctrl+s":
        return f.saveStep()

    case "esc":
        return nil, nil  // Cancel
    }

    return f, nil
}

func (f *StepForm) handleEnter() (*StepForm, tea.Cmd) {
    switch f.focusedField {
    case stepFieldName:
        f.editing = true
        f.editBuffer = f.Step.Name
    case stepFieldSaveAs:
        f.editing = true
        f.editBuffer = f.Step.SaveAs
    case stepFieldProfile:
        // Profile uses left/right, not enter
    default:
        // Param field
        paramIndex := f.focusedField - stepFieldParamsStart
        if paramIndex >= 0 && paramIndex < len(f.Tool.Params) {
            param := f.Tool.Params[paramIndex]
            f.editing = true
            f.editBuffer = f.getParamValue(param.Name)
        }
    }
    return f, nil
}

func (f *StepForm) saveStep() (*StepForm, tea.Cmd) {
    // Validate required fields
    for _, param := range f.Tool.Params {
        if param.Required {
            val := f.getParamValue(param.Name)
            if val == "" {
                f.error = param.Name + " is required"
                return f, nil
            }
        }
    }

    // Profile required for integration tools
    if f.Tool.RequiresProfile && f.Step.Profile == "" {
        f.error = "Profile is required"
        return f, nil
    }

    // Return nil to signal save complete (parent handles)
    return nil, func() tea.Msg { return StepSavedMsg{Step: f.Step} }
}
```

## Validation

- [ ] Step form displays tool info and description
- [ ] Available variables shown at top
- [ ] Name field editable
- [ ] Profile dropdown shows for integration tools
- [ ] Profile loads async from API
- [ ] All tool params rendered dynamically
- [ ] Required params marked with asterisk
- [ ] Param descriptions show when focused
- [ ] String params editable as text
- [ ] Number params validate as numbers
- [ ] Boolean params toggle with enter
- [ ] Array/object params editable as JSON
- [ ] `save_as` field editable
- [ ] `[t]` triggers step test (placeholder for Phase 7)
- [ ] `[s]` validates required fields and saves
- [ ] `[Esc]` cancels without saving
- [ ] Missing required fields show error
