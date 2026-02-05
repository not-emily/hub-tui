package modal

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/client"
	"github.com/pxp/hub-tui/internal/ui/components"
	"github.com/pxp/hub-tui/internal/ui/theme"
)

// workflowsView represents the current view state.
type workflowsView int

const (
	wfViewList workflowsView = iota
	wfViewBuilder
)

// WorkflowsModal displays and manages workflows.
type WorkflowsModal struct {
	client    *client.Client
	workflows []client.Workflow
	selected  int
	loading   bool
	error     string

	// View state
	view    workflowsView
	builder *WorkflowBuilder
	confirm *components.Confirmation
}

// NewWorkflowsModal creates a new workflows modal.
func NewWorkflowsModal(c *client.Client) *WorkflowsModal {
	return &WorkflowsModal{
		client:  c,
		loading: true,
		view:    wfViewList,
		confirm: components.NewConfirmation(),
	}
}

// WorkflowsLoadedMsg is sent when workflows are loaded.
type WorkflowsLoadedMsg struct {
	Workflows []client.Workflow
	Error     error
}

func (m WorkflowsLoadedMsg) IsAsyncModalMessage() {}
func (m WorkflowsLoadedMsg) AuthError() error     { return m.Error }

// WorkflowLoadedMsg is sent when a single workflow is loaded for editing.
type WorkflowLoadedMsg struct {
	Workflow *client.Workflow
	Error    error
}

func (m WorkflowLoadedMsg) IsAsyncModalMessage() {}
func (m WorkflowLoadedMsg) AuthError() error     { return m.Error }

// WorkflowDeletedMsg is sent when a workflow is deleted.
type WorkflowDeletedMsg struct {
	Name  string
	Error error
}

func (m WorkflowDeletedMsg) IsAsyncModalMessage() {}
func (m WorkflowDeletedMsg) AuthError() error     { return m.Error }

// WorkflowSavedMsg is sent when a workflow is saved.
type WorkflowSavedMsg struct {
	Name  string
	IsNew bool
	Error error
}

func (m WorkflowSavedMsg) IsAsyncModalMessage() {}
func (m WorkflowSavedMsg) AuthError() error     { return m.Error }

// WorkflowRunMsg is sent when a workflow run is initiated.
type WorkflowRunMsg struct {
	Name  string
	Error error
}

func (m WorkflowRunMsg) IsAsyncModalMessage() {}
func (m WorkflowRunMsg) AuthError() error     { return m.Error }

// WorkflowValidatedMsg is sent when workflow validation completes.
type WorkflowValidatedMsg struct {
	Valid  bool
	Errors []client.ValidationError
	Error  error // network/API error
}

func (m WorkflowValidatedMsg) IsAsyncModalMessage() {}
func (m WorkflowValidatedMsg) AuthError() error     { return m.Error }

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

func (m *WorkflowsModal) loadWorkflowForEdit(name string) tea.Cmd {
	return func() tea.Msg {
		wf, err := m.client.GetWorkflow(name)
		return WorkflowLoadedMsg{Workflow: wf, Error: err}
	}
}

func (m *WorkflowsModal) deleteWorkflow(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.DeleteWorkflow(name)
		return WorkflowDeletedMsg{Name: name, Error: err}
	}
}

// Update handles input.
func (m *WorkflowsModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	debugLog(fmt.Sprintf("WorkflowsModal.Update: msg=%T, view=%v", msg, m.view))

	// Handle messages regardless of view
	switch msg := msg.(type) {
	case components.ConfirmationExpiredMsg:
		m.confirm.HandleExpired(msg)
		return m, nil

	case WorkflowsLoadedMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			m.workflows = msg.Workflows
			m.error = ""
			// Adjust selection if out of bounds
			if m.selected >= len(m.workflows) && m.selected > 0 {
				m.selected = len(m.workflows) - 1
			}
		}
		return m, nil

	case WorkflowLoadedMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			m.builder = NewWorkflowBuilder(m.client, false)
			m.builder.LoadWorkflow(msg.Workflow)
			m.view = wfViewBuilder
			m.error = ""
			return m, m.builder.Init()
		}
		return m, nil

	case WorkflowDeletedMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			m.error = ""
			// Refresh list
			m.loading = true
			return m, m.loadWorkflows()
		}
		return m, nil

	case WorkflowSavedMsg:
		m.loading = false
		if msg.Error != nil {
			// Pass error to builder so it can display it
			if m.builder != nil {
				m.builder.Error = msg.Error.Error()
				m.builder.Loading = false
				m.builder.viewState = builderViewList // Go back to list view to show error
			} else {
				m.error = msg.Error.Error()
			}
		} else {
			m.error = ""
			m.builder = nil
			m.view = wfViewList
			// Refresh list
			m.loading = true
			return m, m.loadWorkflows()
		}
		return m, nil
	}

	// Route to appropriate view handler
	switch m.view {
	case wfViewBuilder:
		return m.updateBuilder(msg)
	default:
		return m.updateList(msg)
	}
}

func (m *WorkflowsModal) updateList(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return nil, nil // Close modal

		case "up", "k":
			m.confirm.Clear()
			if m.selected > 0 {
				m.selected--
			}

		case "down", "j":
			m.confirm.Clear()
			if m.selected < len(m.workflows)-1 {
				m.selected++
			}

		case "r":
			m.confirm.Clear()
			m.loading = true
			m.error = ""
			return m, m.loadWorkflows()

		case "n":
			m.confirm.Clear()
			// Create new workflow
			m.builder = NewWorkflowBuilder(m.client, true)
			m.view = wfViewBuilder
			return m, m.builder.Init()

		case "e", "enter":
			m.confirm.Clear()
			// Edit selected workflow
			if len(m.workflows) > 0 {
				m.loading = true
				m.error = ""
				return m, m.loadWorkflowForEdit(m.workflows[m.selected].Name)
			}

		case "d":
			// Delete selected workflow (double-press confirmation)
			if len(m.workflows) > 0 {
				wf := m.workflows[m.selected]
				if execute, cmd := m.confirm.Check("delete", wf.Name); execute {
					m.loading = true
					return m, m.deleteWorkflow(wf.Name)
				} else if cmd != nil {
					return m, cmd
				}
			}
		}
	}
	return m, nil
}

func (m *WorkflowsModal) updateBuilder(msg tea.Msg) (Modal, tea.Cmd) {
	debugLog(fmt.Sprintf("WorkflowsModal.updateBuilder: msg=%T, builder=%v", msg, m.builder != nil))
	if m.builder == nil {
		m.view = wfViewList
		return m, nil
	}

	builder, cmd := m.builder.Update(msg)
	debugLog(fmt.Sprintf("WorkflowsModal.updateBuilder: after builder.Update, builder=%v, cmd=%v", builder != nil, cmd != nil))
	if builder == nil {
		// Builder closed
		m.builder = nil
		m.view = wfViewList
		return m, nil
	}
	m.builder = builder
	return m, cmd
}

// Title returns the modal title.
func (m *WorkflowsModal) Title() string {
	switch m.view {
	case wfViewBuilder:
		if m.builder != nil && !m.builder.IsNew {
			return "Edit Workflow"
		}
		return "Create Workflow"
	default:
		return "Workflows"
	}
}

// View renders the modal content.
func (m *WorkflowsModal) View() string {
	switch m.view {
	case wfViewBuilder:
		if m.builder != nil {
			return m.builder.View()
		}
		return "Loading..."
	default:
		return m.renderList()
	}
}

func (m *WorkflowsModal) renderList() string {
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
			hintStyle.Render("[r] Retry  [n] New workflow"),
		)
	}

	if len(m.workflows) == 0 {
		dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			dimStyle.Render("No workflows found."),
			"",
			dimStyle.Render("[n] Create new workflow"),
		)
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

	// Show delete confirmation warning or normal hints
	wf := m.workflows[m.selected]
	if m.confirm.IsPending("delete", wf.Name) {
		warningStyle := lipgloss.NewStyle().Foreground(theme.Warning)
		lines = append(lines, warningStyle.Render("  Press d again to delete"))
	} else {
		lines = append(lines, legendStyle.Render("  [n]ew  [e]dit  [d]elete  [r]efresh"))
	}

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
