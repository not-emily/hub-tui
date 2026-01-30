# Phase 4: Trigger Form

> **Depends on:** Phase 3.2 (Builder Editing)
> **Enables:** Phase 5 (Tool Picker)
>
> See: [Full Plan](../plan.md)

## Goal

Implement the trigger configuration form with type selection and schedule options.

## Key Deliverables

- Trigger type radio (Manual / Scheduled)
- Schedule configuration (frequency, time, days)
- Schedule preview display (cron, description, next runs)
- Integration with builder state

## Files to Create

- `internal/ui/modal/workflows_schedule.go` — Trigger form component

## Implementation Notes

### ScheduleForm Struct

```go
package modal

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/pxp/hub-tui/internal/client"
)

type ScheduleForm struct {
    client *client.Client

    // Trigger type
    TriggerType string  // "manual" or "schedule"

    // Schedule fields (only used when TriggerType == "schedule")
    Frequency string   // "daily", "weekly", "monthly"
    Time      string   // "HH:MM" format
    Days      []string // ["MON", "TUE", ...] for weekly

    // Preview
    Preview *client.SchedulePreview

    // UI state
    focusedField int
    editing      bool   // true when editing time input
    timeInput    string // current time input value
    loading      bool
    error        string
}

const (
    fieldTriggerType = iota
    fieldFrequency
    fieldTime
    fieldDays
)

var (
    frequencyOptions = []string{"daily", "weekly", "monthly"}
    dayOptions       = []string{"MON", "TUE", "WED", "THU", "FRI", "SAT", "SUN"}
)
```

### Constructor

```go
func NewScheduleForm(c *client.Client, trigger client.TriggerConfig) *ScheduleForm {
    f := &ScheduleForm{
        client:      c,
        TriggerType: trigger.Type,
        Frequency:   trigger.Frequency,
        Time:        trigger.Time,
        Days:        trigger.Days,
    }

    // Defaults
    if f.TriggerType == "" {
        f.TriggerType = "manual"
    }
    if f.Frequency == "" {
        f.Frequency = "daily"
    }
    if f.Time == "" {
        f.Time = "08:00"
    }
    if f.Days == nil {
        f.Days = []string{}
    }

    return f
}
```

### View Rendering

```go
func (f *ScheduleForm) View() string {
    var lines []string

    lines = append(lines, headerStyle.Render("Trigger Configuration"))
    lines = append(lines, "")

    // Trigger type radio
    lines = append(lines, f.renderTriggerType())
    lines = append(lines, "")

    // Schedule options (only if scheduled)
    if f.TriggerType == "schedule" {
        lines = append(lines, f.renderFrequency())
        lines = append(lines, f.renderTime())

        if f.Frequency == "weekly" {
            lines = append(lines, f.renderDays())
        }

        lines = append(lines, "")

        // Preview
        if f.Preview != nil {
            lines = append(lines, f.renderPreview())
        } else if f.loading {
            lines = append(lines, dimStyle.Render("Loading preview..."))
        }
    }

    lines = append(lines, "")
    lines = append(lines, f.renderHints())

    if f.error != "" {
        lines = append(lines, errorStyle.Render("Error: "+f.error))
    }

    return strings.Join(lines, "\n")
}

func (f *ScheduleForm) renderTriggerType() string {
    manual := "( ) Manual"
    scheduled := "( ) Scheduled"

    if f.TriggerType == "manual" {
        manual = "(•) Manual"
    } else {
        scheduled = "(•) Scheduled"
    }

    line := "Type: " + manual + "  " + scheduled

    if f.focusedField == fieldTriggerType {
        return selectedStyle.Render(line)
    }
    return line
}

func (f *ScheduleForm) renderFrequency() string {
    line := "Frequency: "
    for i, opt := range frequencyOptions {
        display := opt
        if opt == f.Frequency {
            display = "[" + opt + "]"
        }
        if i > 0 {
            line += "  "
        }
        line += display
    }

    if f.focusedField == fieldFrequency {
        return selectedStyle.Render(line)
    }
    return line
}

func (f *ScheduleForm) renderTime() string {
    line := "Time: "
    if f.editing && f.focusedField == fieldTime {
        line += "[" + f.timeInput + "_]"
    } else {
        line += f.Time
    }

    if f.focusedField == fieldTime {
        return selectedStyle.Render(line)
    }
    return line
}

func (f *ScheduleForm) renderDays() string {
    line := "Days: "
    for i, day := range dayOptions {
        checked := contains(f.Days, day)
        if checked {
            line += "[x] " + day
        } else {
            line += "[ ] " + day
        }
        if i < len(dayOptions)-1 {
            line += "  "
        }
    }

    if f.focusedField == fieldDays {
        return selectedStyle.Render(line)
    }
    return line
}

func (f *ScheduleForm) renderPreview() string {
    var lines []string
    lines = append(lines, dimStyle.Render("Cron: "+f.Preview.Cron))
    lines = append(lines, dimStyle.Render(f.Preview.Description))
    lines = append(lines, "")
    lines = append(lines, "Next runs:")
    for _, run := range f.Preview.NextRuns {
        lines = append(lines, "  "+formatNextRun(run))
    }
    return strings.Join(lines, "\n")
}

func (f *ScheduleForm) renderHints() string {
    if f.editing {
        return dimStyle.Render("[Enter] Confirm  [Esc] Cancel")
    }
    return dimStyle.Render("[j/k] Navigate  [Enter] Edit/Toggle  [s] Save  [Esc] Cancel")
}
```

### Update Handling

```go
func (f *ScheduleForm) Update(msg tea.Msg) (*ScheduleForm, tea.Cmd) {
    switch msg := msg.(type) {
    case SchedulePreviewedMsg:
        f.loading = false
        if msg.Error != nil {
            f.error = msg.Error.Error()
        } else {
            f.Preview = msg.Preview
            f.error = ""
        }
        return f, nil

    case tea.KeyMsg:
        return f.handleKeyPress(msg)
    }
    return f, nil
}

func (f *ScheduleForm) handleKeyPress(msg tea.KeyMsg) (*ScheduleForm, tea.Cmd) {
    // Handle time editing
    if f.editing {
        return f.handleTimeInput(msg)
    }

    switch msg.String() {
    case "j", "down":
        f.focusedField = f.nextField()
    case "k", "up":
        f.focusedField = f.prevField()

    case "enter", " ":
        return f.handleSelect()

    case "h", "left":
        return f.handleLeft()
    case "l", "right":
        return f.handleRight()

    case "s":
        // Save and return to builder
        return f, nil  // Parent handles save

    case "esc":
        return nil, nil  // Cancel
    }

    return f, nil
}

func (f *ScheduleForm) handleSelect() (*ScheduleForm, tea.Cmd) {
    switch f.focusedField {
    case fieldTriggerType:
        // Toggle between manual and schedule
        if f.TriggerType == "manual" {
            f.TriggerType = "schedule"
            return f, f.fetchPreview()
        } else {
            f.TriggerType = "manual"
            f.Preview = nil
        }

    case fieldTime:
        f.editing = true
        f.timeInput = f.Time

    case fieldDays:
        // Toggle currently selected day
        // Need to track which day is "focused" within the days field
        // Simplified: toggle first unselected day
    }

    return f, nil
}

func (f *ScheduleForm) fetchPreview() tea.Cmd {
    f.loading = true
    return func() tea.Msg {
        preview, err := f.client.PreviewSchedule(&client.ScheduleRequest{
            Frequency: f.Frequency,
            Time:      f.Time,
            Days:      f.Days,
        })
        return SchedulePreviewedMsg{Preview: preview, Error: err}
    }
}
```

### ToTriggerConfig

```go
func (f *ScheduleForm) ToTriggerConfig() client.TriggerConfig {
    if f.TriggerType == "manual" {
        return client.TriggerConfig{Type: "manual"}
    }

    return client.TriggerConfig{
        Type:      "schedule",
        Cron:      f.Preview.Cron,  // Use generated cron
        Frequency: f.Frequency,
        Time:      f.Time,
        Days:      f.Days,
    }
}
```

### Integration with Builder

In `workflows_builder.go`, add handling for trigger form:

```go
case ViewTriggerForm:
    if b.scheduleForm == nil {
        b.scheduleForm = NewScheduleForm(b.client, b.Trigger)
    }
    return b.scheduleForm.View()

// In Update:
case ViewTriggerForm:
    form, cmd := b.scheduleForm.Update(msg)
    if form == nil {
        // Form cancelled or saved
        b.View = ViewList
        if b.scheduleForm != nil {
            // Save was requested
            b.Trigger = b.scheduleForm.ToTriggerConfig()
            b.Dirty = true
        }
        b.scheduleForm = nil
    }
    return b, cmd
```

## Validation

- [ ] Trigger type toggles between Manual and Scheduled
- [ ] Manual trigger shows no additional fields
- [ ] Scheduled trigger shows frequency, time fields
- [ ] Weekly frequency shows day checkboxes
- [ ] Daily/monthly frequencies hide day checkboxes
- [ ] Can edit time field (Enter to edit, Enter to confirm)
- [ ] Time validates format (HH:MM)
- [ ] Preview fetches on schedule change
- [ ] Preview shows cron, description, and next 3 runs
- [ ] `[s]` saves trigger config and returns to builder
- [ ] `[Esc]` cancels and returns to builder without saving
- [ ] Builder shows updated trigger summary after save
