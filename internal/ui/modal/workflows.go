package modal

import (
	"fmt"
	"strings"

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
	descStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

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

		// Build line with description
		line := fmt.Sprintf("  %s %s", indicator, name)
		if wf.Description != "" {
			// Pad name to align descriptions
			padding := 20 - len(wf.Name)
			if padding < 2 {
				padding = 2
			}
			line += strings.Repeat(" ", padding) + descStyle.Render(wf.Description)
		}

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
