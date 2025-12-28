package modal

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/client"
	"github.com/pxp/hub-tui/internal/ui/theme"
)

// TaskRun represents a task run for the modal.
type TaskRun struct {
	ID        string
	Workflow  string
	Status    string
	StartedAt time.Time
	EndedAt   time.Time
	Error     string
	Result    *client.RunResult
}

// isRunSuccess returns true if the run completed successfully.
func isRunSuccess(r client.Run) bool {
	if r.Status == "failed" {
		return false
	}
	if r.Result != nil && !r.Result.Success {
		return false
	}
	return true
}

// formatRunOutput extracts a readable output string from the run result.
func formatRunOutput(result *client.RunResult) string {
	if result == nil {
		return ""
	}

	var outputs []string
	for _, step := range result.Steps {
		if step.Error != "" {
			outputs = append(outputs, fmt.Sprintf("[%s] Error: %s", step.StepName, step.Error))
		} else if step.Output != nil {
			// Try to format the output nicely
			switch v := step.Output.(type) {
			case string:
				outputs = append(outputs, fmt.Sprintf("[%s] %s", step.StepName, v))
			case map[string]interface{}:
				if msg, ok := v["message"].(string); ok {
					outputs = append(outputs, fmt.Sprintf("[%s] %s", step.StepName, msg))
				} else {
					// JSON encode it
					b, _ := json.MarshalIndent(v, "", "  ")
					outputs = append(outputs, fmt.Sprintf("[%s]\n%s", step.StepName, string(b)))
				}
			default:
				b, _ := json.MarshalIndent(v, "", "  ")
				outputs = append(outputs, fmt.Sprintf("[%s]\n%s", step.StepName, string(b)))
			}
		}
	}

	return strings.Join(outputs, "\n")
}

// TasksModal displays running, completed, and failed tasks.
type TasksModal struct {
	client    *client.Client
	running   []TaskRun
	completed []TaskRun
	failed    []TaskRun
	allRuns   []TaskRun // Combined list for navigation
	selected  int
	loading   bool
	error     string
	view      tasksView
	detailRun *TaskRun // Run being viewed in detail
}

type tasksView int

const (
	viewTasksList tasksView = iota
	viewTaskDetail
)

// NewTasksModal creates a new tasks modal with pre-loaded task state.
func NewTasksModal(c *client.Client) *TasksModal {
	return &TasksModal{
		client:  c,
		loading: true,
		view:    viewTasksList,
	}
}

// NewTasksModalWithState creates a new tasks modal with pre-loaded task state.
func NewTasksModalWithState(c *client.Client, running, completed, failed []TaskRun) *TasksModal {
	m := &TasksModal{
		client:    c,
		running:   running,
		completed: completed,
		failed:    failed,
		loading:   false,
		view:      viewTasksList,
	}
	m.buildAllRuns()
	return m
}

func (m *TasksModal) buildAllRuns() {
	m.allRuns = nil
	m.allRuns = append(m.allRuns, m.running...)
	m.allRuns = append(m.allRuns, m.completed...)
	m.allRuns = append(m.allRuns, m.failed...)
}

// TasksLoadedMsg is sent when tasks are loaded.
type TasksLoadedMsg struct {
	Running   []TaskRun
	Completed []TaskRun
	Failed    []TaskRun
	Error     error
}

// TaskCancelRequestMsg is sent when a cancel is requested.
type TaskCancelRequestMsg struct {
	RunID string
}

// Init initializes the modal.
func (m *TasksModal) Init() tea.Cmd {
	// If we already have state, no need to load
	if !m.loading {
		return nil
	}
	return m.loadTasks()
}

func (m *TasksModal) loadTasks() tea.Cmd {
	return func() tea.Msg {
		runs, err := m.client.ListRuns()
		if err != nil {
			return TasksLoadedMsg{Error: err}
		}

		var running, completed, failed []TaskRun
		for _, r := range runs {
			tr := TaskRun{
				ID:        r.ID,
				Workflow:  r.Workflow,
				Status:    r.Status,
				StartedAt: r.StartedAt,
				EndedAt:   r.EndedAt,
				Error:     r.Error,
				Result:    r.Result,
			}
			if r.Status == "running" {
				running = append(running, tr)
			} else if isRunSuccess(r) {
				completed = append(completed, tr)
			} else {
				failed = append(failed, tr)
			}
		}

		return TasksLoadedMsg{Running: running, Completed: completed, Failed: failed}
	}
}

// Update handles input.
func (m *TasksModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case TasksLoadedMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			m.running = msg.Running
			m.completed = msg.Completed
			m.failed = msg.Failed
			m.buildAllRuns()
			m.error = ""
		}
		return m, nil

	case tea.KeyMsg:
		if m.view == viewTaskDetail {
			return m.updateDetail(msg)
		}
		return m.updateList(msg)
	}
	return m, nil
}

func (m *TasksModal) updateList(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return nil, nil // Close modal
	case "up", "k":
		if m.selected > 0 {
			m.selected--
		}
	case "down", "j":
		if m.selected < len(m.allRuns)-1 {
			m.selected++
		}
	case "enter":
		if len(m.allRuns) > 0 && m.selected < len(m.allRuns) {
			run := m.allRuns[m.selected]
			m.detailRun = &run
			m.view = viewTaskDetail
		}
	case "c":
		// Cancel selected running task
		if len(m.allRuns) > 0 && m.selected < len(m.allRuns) {
			run := m.allRuns[m.selected]
			if run.Status == "running" {
				return m, m.cancelTask(run.ID)
			}
		}
	}
	return m, nil
}

func (m *TasksModal) updateDetail(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.view = viewTasksList
		m.detailRun = nil
	case "c":
		// Cancel if running
		if m.detailRun != nil && m.detailRun.Status == "running" {
			return m, m.cancelTask(m.detailRun.ID)
		}
	}
	return m, nil
}

// Title returns the modal title.
func (m *TasksModal) Title() string {
	if m.view == viewTaskDetail && m.detailRun != nil {
		return "Task: " + m.detailRun.Workflow
	}
	return "Tasks"
}

// View renders the modal content.
func (m *TasksModal) View() string {
	if m.view == viewTaskDetail {
		return m.viewDetail()
	}
	return m.viewList()
}

func (m *TasksModal) viewList() string {
	if m.loading {
		return lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("Loading tasks...")
	}

	if m.error != "" {
		errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
		hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			errorStyle.Render("Error: "+m.error),
			"",
			hintStyle.Render("[r] Retry"),
		)
	}

	if len(m.allRuns) == 0 {
		return lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("No tasks.")
	}

	var lines []string
	runIndex := 0 // Track index across all sections for selection

	headerStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	timeStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	runningIndicator := lipgloss.NewStyle().Foreground(theme.Warning).Render("●")
	completedIndicator := lipgloss.NewStyle().Foreground(theme.Success).Render("✓")
	failedIndicator := lipgloss.NewStyle().Foreground(theme.Error).Render("✗")

	// Running section
	if len(m.running) > 0 {
		lines = append(lines, headerStyle.Render("Running:"))
		for _, r := range m.running {
			name := normalStyle.Render(r.Workflow)
			if runIndex == m.selected {
				name = selectedStyle.Render(r.Workflow)
			}
			elapsed := formatElapsed(r.StartedAt)
			line := fmt.Sprintf("  %s %s    %s", runningIndicator, name, timeStyle.Render("Started "+elapsed))
			lines = append(lines, line)
			runIndex++
		}
		lines = append(lines, "")
	}

	// Completed section
	if len(m.completed) > 0 {
		lines = append(lines, headerStyle.Render("Completed:"))
		for _, r := range m.completed {
			name := normalStyle.Render(r.Workflow)
			if runIndex == m.selected {
				name = selectedStyle.Render(r.Workflow)
			}
			elapsed := formatElapsed(r.EndedAt)
			line := fmt.Sprintf("  %s %s    %s", completedIndicator, name, timeStyle.Render("Completed "+elapsed))
			lines = append(lines, line)
			runIndex++
		}
		lines = append(lines, "")
	}

	// Failed section
	if len(m.failed) > 0 {
		lines = append(lines, headerStyle.Render("Failed:"))
		for _, r := range m.failed {
			name := normalStyle.Render(r.Workflow)
			if runIndex == m.selected {
				name = selectedStyle.Render(r.Workflow)
			}
			elapsed := formatElapsed(r.EndedAt)
			errText := ""
			if r.Error != "" {
				errText = "\n      " + lipgloss.NewStyle().Foreground(theme.Error).Render(r.Error)
			}
			line := fmt.Sprintf("  %s %s    %s%s", failedIndicator, name, timeStyle.Render("Failed "+elapsed), errText)
			lines = append(lines, line)
			runIndex++
		}
		lines = append(lines, "")
	}

	// Hints
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	hints := "[Enter] Details"
	if len(m.running) > 0 {
		hints += "  [c] Cancel"
	}
	lines = append(lines, hintStyle.Render(hints))

	return strings.Join(lines, "\n")
}

func (m *TasksModal) viewDetail() string {
	if m.detailRun == nil {
		return "No task selected"
	}

	r := m.detailRun
	labelStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	valueStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)

	var statusStyle lipgloss.Style
	switch r.Status {
	case "running":
		statusStyle = lipgloss.NewStyle().Foreground(theme.Warning)
	case "completed":
		statusStyle = lipgloss.NewStyle().Foreground(theme.Success)
	default:
		statusStyle = lipgloss.NewStyle().Foreground(theme.Error)
	}

	var lines []string

	lines = append(lines, labelStyle.Render("Status:    ")+statusStyle.Render(r.Status))
	lines = append(lines, labelStyle.Render("Started:   ")+valueStyle.Render(formatTime(r.StartedAt)))

	if !r.EndedAt.IsZero() {
		lines = append(lines, labelStyle.Render("Ended:     ")+valueStyle.Render(formatTime(r.EndedAt)))
		duration := r.EndedAt.Sub(r.StartedAt)
		lines = append(lines, labelStyle.Render("Duration:  ")+valueStyle.Render(formatDuration(duration)))
	}

	if r.Error != "" {
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("Error:"))
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.Error).Render("  "+r.Error))
	}

	output := formatRunOutput(r.Result)
	if output != "" {
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("Output:"))
		// Indent output lines
		for _, line := range strings.Split(output, "\n") {
			lines = append(lines, "  "+valueStyle.Render(line))
		}
	}

	lines = append(lines, "")
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	hints := "[Esc] Back"
	if r.Status == "running" {
		hints += "  [c] Cancel"
	}
	lines = append(lines, hintStyle.Render(hints))

	return strings.Join(lines, "\n")
}

// cancelTask returns a command to reload tasks after cancelling.
func (m *TasksModal) cancelTask(runID string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.CancelRun(runID)
		if err != nil {
			return TasksLoadedMsg{Error: err}
		}
		// Reload tasks after cancel
		runs, err := m.client.ListRuns()
		if err != nil {
			return TasksLoadedMsg{Error: err}
		}

		var running, completed, failed []TaskRun
		for _, r := range runs {
			tr := TaskRun{
				ID:        r.ID,
				Workflow:  r.Workflow,
				Status:    r.Status,
				StartedAt: r.StartedAt,
				EndedAt:   r.EndedAt,
				Error:     r.Error,
				Result:    r.Result,
			}
			if r.Status == "running" {
				running = append(running, tr)
			} else if isRunSuccess(r) {
				completed = append(completed, tr)
			} else {
				failed = append(failed, tr)
			}
		}

		return TasksLoadedMsg{Running: running, Completed: completed, Failed: failed}
	}
}

// formatElapsed returns a human-readable elapsed time.
func formatElapsed(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	elapsed := time.Since(t)
	if elapsed < time.Minute {
		return "just now"
	} else if elapsed < time.Hour {
		mins := int(elapsed.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d min ago", mins)
	} else if elapsed < 24*time.Hour {
		hours := int(elapsed.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	days := int(elapsed.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}

// formatTime formats a time for display.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05")
}

// formatDuration formats a duration for display.
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return "< 1s"
	} else if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}
