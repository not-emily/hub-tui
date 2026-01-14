package modal

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/client"
	"github.com/pxp/hub-tui/internal/ui/theme"
)

// WorkflowsModal displays and manages workflows.
type WorkflowsModal struct {
	client    *client.Client
	workflows []client.Workflow
	selected  int
	loading   bool
	error     string
}

// NewWorkflowsModal creates a new workflows modal.
func NewWorkflowsModal(c *client.Client) *WorkflowsModal {
	return &WorkflowsModal{
		client:  c,
		loading: true,
	}
}

// WorkflowsLoadedMsg is sent when workflows are loaded.
type WorkflowsLoadedMsg struct {
	Workflows []client.Workflow
	Error     error
}

// WorkflowRunMsg is sent when a workflow run is initiated.
type WorkflowRunMsg struct {
	Name  string
	Error error
}

// Init initializes the modal and triggers data fetch.
func (m *WorkflowsModal) Init() tea.Cmd {
	return m.loadWorkflows()
}

func (m *WorkflowsModal) loadWorkflows() tea.Cmd {
	return func() tea.Msg {
		workflows, err := m.client.ListWorkflows()
		return WorkflowsLoadedMsg{Workflows: workflows, Error: err}
	}
}

// Update handles input.
func (m *WorkflowsModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case WorkflowsLoadedMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			m.workflows = msg.Workflows
			m.error = ""
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return nil, nil // Close modal
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.workflows)-1 {
				m.selected++
			}
		case "r":
			m.loading = true
			m.error = ""
			return m, m.loadWorkflows()
		}
	}
	return m, nil
}

// Title returns the modal title.
func (m *WorkflowsModal) Title() string {
	return "Workflows"
}

// View renders the modal content.
func (m *WorkflowsModal) View() string {
	if m.loading {
		return lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("Loading workflows...")
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

	if len(m.workflows) == 0 {
		return lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("No workflows found.")
	}

	var lines []string

	enabledStyle := lipgloss.NewStyle().Foreground(theme.Success)
	disabledStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	// Calculate max name length for alignment
	maxNameLen := 0
	for _, wf := range m.workflows {
		if len(wf.Name) > maxNameLen {
			maxNameLen = len(wf.Name)
		}
	}
	if maxNameLen < 15 {
		maxNameLen = 15
	}

	for i, wf := range m.workflows {
		// Status indicator
		var indicator string
		if wf.Enabled {
			indicator = enabledStyle.Render("●")
		} else {
			indicator = disabledStyle.Render("○")
		}

		// Name with selection highlight
		var name string
		if i == m.selected {
			name = selectedStyle.Render(wf.Name)
		} else {
			name = normalStyle.Render(wf.Name)
		}

		// Pad name for alignment
		namePadding := maxNameLen - len(wf.Name) + 2
		if namePadding < 2 {
			namePadding = 2
		}

		// Trigger info column
		var triggerInfo string
		switch wf.Trigger.Type {
		case "schedule":
			if wf.Frequency != "" {
				triggerInfo = wf.Frequency
			} else {
				triggerInfo = "scheduled"
			}
		case "manual":
			triggerInfo = "manual"
		case "webhook":
			triggerInfo = "webhook"
		case "condition":
			triggerInfo = "condition"
		default:
			triggerInfo = "manual" // default fallback
		}

		// Next run for scheduled workflows
		var nextRunInfo string
		if wf.Trigger.Type == "schedule" && wf.NextRun != nil {
			nextRunInfo = "  Next: " + formatRelativeTime(*wf.NextRun)
		}

		line := fmt.Sprintf("  %s %s%s%s%s",
			indicator,
			name,
			strings.Repeat(" ", namePadding),
			dimStyle.Render(triggerInfo),
			dimStyle.Render(nextRunInfo),
		)

		lines = append(lines, line)
	}

	// Add legend and hints
	lines = append(lines, "")
	legendStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	lines = append(lines, legendStyle.Render("  ● enabled  ○ disabled"))
	lines = append(lines, "")
	lines = append(lines, legendStyle.Render("  Use #workflow to run  [r] Refresh"))

	return strings.Join(lines, "\n")
}

// formatRelativeTime formats a time as a human-readable relative duration.
func formatRelativeTime(t time.Time) string {
	now := time.Now()
	if t.Before(now) {
		return "overdue"
	}

	d := t.Sub(now)

	if d < time.Minute {
		return "< 1m"
	} else if d < time.Hour {
		mins := int(d.Minutes())
		return fmt.Sprintf("%dm", mins)
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		mins := int(d.Minutes()) % 60
		if mins > 0 {
			return fmt.Sprintf("%dh %dm", hours, mins)
		}
		return fmt.Sprintf("%dh", hours)
	}

	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	if hours > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	return fmt.Sprintf("%dd", days)
}
