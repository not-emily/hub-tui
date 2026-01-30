package modal

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/client"
	"github.com/pxp/hub-tui/internal/ui/theme"
)

// Schedule form field constants
const (
	schedFieldTriggerType = iota
	schedFieldFrequency
	schedFieldTime
	schedFieldDays
)

var (
	frequencyOptions = []string{"daily", "weekly", "monthly"}
	dayOptions       = []string{"MON", "TUE", "WED", "THU", "FRI", "SAT", "SUN"}
)

// SchedulePreviewedMsg is sent when a schedule preview is loaded.
type SchedulePreviewedMsg struct {
	Preview *client.SchedulePreview
	Error   error
}

// ScheduleForm handles trigger configuration.
type ScheduleForm struct {
	client *client.Client

	// Trigger type
	TriggerType string // "manual" or "schedule"

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
	dayIndex     int    // which day is focused for toggle (0-6)
	loading      bool
	error        string
	saved        bool // true if user pressed save (vs cancel)
}

// NewScheduleForm creates a new schedule form.
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

// Init initializes the form.
func (f *ScheduleForm) Init() tea.Cmd {
	if f.TriggerType == "schedule" {
		return f.fetchPreview()
	}
	return nil
}

// Update handles messages.
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

	case "enter":
		return f.handleSelect()

	case "h", "left":
		return f.handleLeft()
	case "l", "right":
		return f.handleRight()

	case " ":
		// Space toggles for days
		if f.focusedField == schedFieldDays {
			return f.toggleDay()
		}
		return f.handleSelect()

	case "ctrl+s":
		// Save and return to builder
		f.saved = true
		return nil, nil

	case "esc", "q":
		// Cancel
		f.saved = false
		return nil, nil
	}

	return f, nil
}

func (f *ScheduleForm) handleTimeInput(msg tea.KeyMsg) (*ScheduleForm, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Validate and confirm
		if f.isValidTime(f.timeInput) {
			f.Time = f.timeInput
			f.editing = false
			return f, f.fetchPreview()
		}
		f.error = "Invalid time format. Use HH:MM (e.g., 08:00, 14:30)"
	case "esc":
		f.editing = false
		f.timeInput = ""
	case "backspace":
		if len(f.timeInput) > 0 {
			f.timeInput = f.timeInput[:len(f.timeInput)-1]
		}
	default:
		// Only allow digits and colon
		ch := msg.String()
		if len(ch) == 1 && (ch[0] >= '0' && ch[0] <= '9' || ch[0] == ':') {
			if len(f.timeInput) < 5 {
				f.timeInput += ch
			}
		}
	}
	return f, nil
}

func (f *ScheduleForm) isValidTime(t string) bool {
	if len(t) != 5 || t[2] != ':' {
		return false
	}
	hour := t[0:2]
	min := t[3:5]
	h := (hour[0]-'0')*10 + (hour[1] - '0')
	m := (min[0]-'0')*10 + (min[1] - '0')
	return h < 24 && m < 60
}

func (f *ScheduleForm) handleSelect() (*ScheduleForm, tea.Cmd) {
	switch f.focusedField {
	case schedFieldTriggerType:
		// Toggle between manual and schedule
		if f.TriggerType == "manual" {
			f.TriggerType = "schedule"
			return f, f.fetchPreview()
		} else {
			f.TriggerType = "manual"
			f.Preview = nil
		}

	case schedFieldFrequency:
		// Cycle through frequency options
		for i, opt := range frequencyOptions {
			if opt == f.Frequency {
				f.Frequency = frequencyOptions[(i+1)%len(frequencyOptions)]
				return f, f.fetchPreview()
			}
		}
		f.Frequency = frequencyOptions[0]
		return f, f.fetchPreview()

	case schedFieldTime:
		f.editing = true
		f.timeInput = f.Time
		f.error = ""

	case schedFieldDays:
		return f.toggleDay()
	}

	return f, nil
}

func (f *ScheduleForm) handleLeft() (*ScheduleForm, tea.Cmd) {
	switch f.focusedField {
	case schedFieldFrequency:
		for i, opt := range frequencyOptions {
			if opt == f.Frequency {
				if i > 0 {
					f.Frequency = frequencyOptions[i-1]
					return f, f.fetchPreview()
				}
			}
		}
	case schedFieldDays:
		if f.dayIndex > 0 {
			f.dayIndex--
		}
	}
	return f, nil
}

func (f *ScheduleForm) handleRight() (*ScheduleForm, tea.Cmd) {
	switch f.focusedField {
	case schedFieldFrequency:
		for i, opt := range frequencyOptions {
			if opt == f.Frequency {
				if i < len(frequencyOptions)-1 {
					f.Frequency = frequencyOptions[i+1]
					return f, f.fetchPreview()
				}
			}
		}
	case schedFieldDays:
		if f.dayIndex < len(dayOptions)-1 {
			f.dayIndex++
		}
	}
	return f, nil
}

func (f *ScheduleForm) toggleDay() (*ScheduleForm, tea.Cmd) {
	day := dayOptions[f.dayIndex]
	if f.containsDay(day) {
		// Remove day
		var newDays []string
		for _, d := range f.Days {
			if d != day {
				newDays = append(newDays, d)
			}
		}
		f.Days = newDays
	} else {
		// Add day
		f.Days = append(f.Days, day)
	}
	return f, f.fetchPreview()
}

func (f *ScheduleForm) containsDay(day string) bool {
	for _, d := range f.Days {
		if d == day {
			return true
		}
	}
	return false
}

func (f *ScheduleForm) nextField() int {
	if f.TriggerType == "manual" {
		return schedFieldTriggerType
	}

	max := schedFieldTime
	if f.Frequency == "weekly" {
		max = schedFieldDays
	}

	if f.focusedField < max {
		return f.focusedField + 1
	}
	return f.focusedField
}

func (f *ScheduleForm) prevField() int {
	if f.focusedField > schedFieldTriggerType {
		return f.focusedField - 1
	}
	return f.focusedField
}

func (f *ScheduleForm) fetchPreview() tea.Cmd {
	if f.TriggerType != "schedule" {
		return nil
	}
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

// ToTriggerConfig converts the form state to a TriggerConfig.
func (f *ScheduleForm) ToTriggerConfig() client.TriggerConfig {
	if f.TriggerType == "manual" {
		return client.TriggerConfig{Type: "manual"}
	}

	config := client.TriggerConfig{
		Type:      "schedule",
		Frequency: f.Frequency,
		Time:      f.Time,
		Days:      f.Days,
	}

	// Use generated cron if available
	if f.Preview != nil {
		config.Cron = f.Preview.Cron
	}

	return config
}

// Saved returns true if the user saved (vs cancelled).
func (f *ScheduleForm) Saved() bool {
	return f.saved
}

// View renders the form.
func (f *ScheduleForm) View() string {
	var lines []string

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	errorStyle := lipgloss.NewStyle().Foreground(theme.Error)

	lines = append(lines, headerStyle.Render("Trigger Configuration"))
	lines = append(lines, "")

	// Error display
	if f.error != "" {
		lines = append(lines, errorStyle.Render("Error: "+f.error))
		lines = append(lines, "")
	}

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
		if f.loading {
			lines = append(lines, dimStyle.Render("Loading preview..."))
		} else if f.Preview != nil {
			lines = append(lines, f.renderPreview()...)
		}
	}

	lines = append(lines, "")
	lines = append(lines, f.renderHints())

	return strings.Join(lines, "\n")
}

func (f *ScheduleForm) renderTriggerType() string {
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	manual := "( ) Manual"
	scheduled := "( ) Scheduled"

	if f.TriggerType == "manual" {
		manual = "(*) Manual"
	} else {
		scheduled = "(*) Scheduled"
	}

	line := labelStyle.Render("Type:      ") + manual + "   " + scheduled

	if f.focusedField == schedFieldTriggerType {
		return selectedStyle.Render(line)
	}
	return line
}

func (f *ScheduleForm) renderFrequency() string {
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	var opts []string
	for _, opt := range frequencyOptions {
		if opt == f.Frequency {
			opts = append(opts, "["+opt+"]")
		} else {
			opts = append(opts, " "+opt+" ")
		}
	}

	line := labelStyle.Render("Frequency: ") + strings.Join(opts, "  ")

	if f.focusedField == schedFieldFrequency {
		return selectedStyle.Render(line)
	}
	return line
}

func (f *ScheduleForm) renderTime() string {
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	var line string
	if f.editing && f.focusedField == schedFieldTime {
		line = labelStyle.Render("Time:      ") + selectedStyle.Render(f.timeInput+"_") + " " + dimStyle.Render("[Enter to confirm]")
	} else {
		timeDisplay := f.Time
		if f.focusedField == schedFieldTime {
			line = labelStyle.Render("Time:      ") + selectedStyle.Render(timeDisplay) + " " + dimStyle.Render("[Enter to edit]")
		} else {
			line = labelStyle.Render("Time:      ") + timeDisplay
		}
	}

	if f.focusedField == schedFieldTime && !f.editing {
		return selectedStyle.Render(line)
	}
	return line
}

func (f *ScheduleForm) renderDays() string {
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	var opts []string
	for i, day := range dayOptions {
		checked := f.containsDay(day)
		var display string
		if checked {
			display = "[x]" + day
		} else {
			display = "[ ]" + day
		}
		// Highlight the focused day when in days field
		if f.focusedField == schedFieldDays && i == f.dayIndex {
			display = selectedStyle.Render(display)
		}
		opts = append(opts, display)
	}

	line := labelStyle.Render("Days:      ") + strings.Join(opts, " ")

	if f.focusedField == schedFieldDays {
		return line // Individual day highlighting handled above
	}
	return line
}

func (f *ScheduleForm) renderPreview() []string {
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	labelStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	var lines []string
	lines = append(lines, labelStyle.Render("Preview:"))
	lines = append(lines, dimStyle.Render("  Cron: "+f.Preview.Cron))
	lines = append(lines, dimStyle.Render("  "+f.Preview.Description))

	if len(f.Preview.NextRuns) > 0 {
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("  Next runs:"))
		for i, run := range f.Preview.NextRuns {
			if i >= 3 {
				break // Show max 3
			}
			lines = append(lines, dimStyle.Render("    "+run))
		}
	}

	return lines
}

func (f *ScheduleForm) renderHints() string {
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	if f.editing {
		return dimStyle.Render("[Enter] Confirm  [Esc] Cancel")
	}

	hints := "[j/k] Navigate  [Enter] Toggle"
	if f.focusedField == schedFieldFrequency {
		hints = "[h/l] Change  [j/k] Navigate"
	} else if f.focusedField == schedFieldDays {
		hints = "[h/l] Select day  [Space] Toggle  [j/k] Navigate"
	}
	hints += "  [Ctrl+s] Save  [Esc] Cancel"

	return dimStyle.Render(hints)
}
