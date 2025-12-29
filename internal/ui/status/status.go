package status

import (
	"fmt"
	"net/url"

	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/ui/theme"
)

// State represents the connection state.
type State int

const (
	StateDisconnected State = iota
	StateConnecting
	StateConnected
)

// Model is the status bar component.
type Model struct {
	width              int
	state              State
	serverURL          string
	ctrlCPressed       bool
	contextType        string // "hub", "assistant", etc.
	contextName        string // Name of assistant/workflow
	runningCount       int    // Number of running tasks
	needsAttentionCount int   // Number of tasks needing attention
}

// New creates a new status bar model.
func New() Model {
	return Model{
		state: StateDisconnected,
	}
}

// SetWidth sets the status bar width.
func (m *Model) SetWidth(width int) {
	m.width = width
}

// SetState sets the connection state.
func (m *Model) SetState(state State) {
	m.state = state
}

// SetServerURL sets the server URL to display.
func (m *Model) SetServerURL(serverURL string) {
	m.serverURL = serverURL
}

// SetCtrlCPressed sets whether Ctrl+C was pressed.
func (m *Model) SetCtrlCPressed(pressed bool) {
	m.ctrlCPressed = pressed
}

// SetContext sets the current conversation context.
func (m *Model) SetContext(contextType, contextName string) {
	m.contextType = contextType
	m.contextName = contextName
}

// SetTaskCounts sets the running and needs-attention task counts.
func (m *Model) SetTaskCounts(running, needsAttention int) {
	m.runningCount = running
	m.needsAttentionCount = needsAttention
}

// View renders the status bar.
func (m Model) View() string {
	var statusText string
	var statusStyle lipgloss.Style

	switch m.state {
	case StateConnected:
		host := extractHost(m.serverURL)
		statusText = "Connected (" + host + ")"
		statusStyle = lipgloss.NewStyle().
			Foreground(theme.Success)

	case StateConnecting:
		statusText = "Connecting..."
		statusStyle = lipgloss.NewStyle().
			Foreground(theme.Warning)

	default:
		statusText = "Disconnected"
		statusStyle = lipgloss.NewStyle().
			Foreground(theme.TextSecondary)
	}

	leftContent := statusStyle.Render(statusText)

	// Add context indicator if in assistant mode
	if m.contextType == "assistant" && m.contextName != "" {
		contextStyle := lipgloss.NewStyle().
			Foreground(theme.Accent).
			Bold(true)
		leftContent += "  " + contextStyle.Render("@"+m.contextName)
	}

	// Build task indicator
	taskIndicator := m.taskIndicator()

	// Right side hint
	var rightContent string
	if m.ctrlCPressed {
		rightContent = lipgloss.NewStyle().
			Foreground(theme.Warning).
			Render("Press Ctrl+C again to quit")
	} else {
		rightContent = lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("Ctrl+C to quit")
	}

	// Calculate content widths
	leftWidth := lipgloss.Width(leftContent)
	taskWidth := lipgloss.Width(taskIndicator)
	rightWidth := lipgloss.Width(rightContent)

	// Calculate padding between left and task indicator
	totalContentWidth := leftWidth + taskWidth + rightWidth + 2 // +2 for padding
	if taskWidth > 0 {
		totalContentWidth += 2 // extra spacing around task indicator
	}
	padding := m.width - totalContentWidth
	if padding < 1 {
		padding = 1
	}

	// Build the full status bar
	barStyle := lipgloss.NewStyle().
		Width(m.width).
		Padding(0, 1)

	if taskWidth > 0 {
		// Split padding: some before task indicator, rest after
		leftPadding := padding / 2
		rightPadding := padding - leftPadding
		return barStyle.Render(leftContent + repeatSpace(leftPadding) + taskIndicator + repeatSpace(rightPadding) + rightContent)
	}

	return barStyle.Render(leftContent + repeatSpace(padding) + rightContent)
}

func repeatSpace(n int) string {
	if n <= 0 {
		return ""
	}
	s := make([]byte, n)
	for i := range s {
		s[i] = ' '
	}
	return string(s)
}

// taskIndicator returns the task count display string.
func (m Model) taskIndicator() string {
	if m.runningCount == 0 && m.needsAttentionCount == 0 {
		return ""
	}

	var parts []string

	if m.runningCount > 0 {
		runningStyle := lipgloss.NewStyle().
			Foreground(theme.Accent)
		parts = append(parts, runningStyle.Render(fmt.Sprintf("%d running", m.runningCount)))
	}

	if m.needsAttentionCount > 0 {
		attentionStyle := lipgloss.NewStyle().
			Foreground(theme.Warning).
			Bold(true)
		if m.needsAttentionCount == 1 {
			parts = append(parts, attentionStyle.Render("1 task needs attention"))
		} else {
			parts = append(parts, attentionStyle.Render(fmt.Sprintf("%d tasks need attention", m.needsAttentionCount)))
		}
	}

	if len(parts) == 1 {
		return parts[0]
	}

	separator := lipgloss.NewStyle().
		Foreground(theme.TextSecondary).
		Render(" · ")
	return parts[0] + separator + parts[1]
}

// IsConnected returns true if the status is connected.
func (m Model) IsConnected() bool {
	return m.state == StateConnected
}

// extractHost extracts the host:port from a URL.
func extractHost(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	return u.Host
}
