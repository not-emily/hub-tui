package status

import (
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
	width        int
	state        State
	serverURL    string
	ctrlCPressed bool
	contextType  string // "hub", "assistant", etc.
	contextName  string // Name of assistant/workflow
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

	// Calculate padding between left and right
	padding := m.width - lipgloss.Width(leftContent) - lipgloss.Width(rightContent) - 2
	if padding < 1 {
		padding = 1
	}
	spacer := repeatSpace(padding)

	// Build the full status bar
	barStyle := lipgloss.NewStyle().
		Width(m.width).
		Padding(0, 1)

	return barStyle.Render(leftContent + spacer + rightContent)
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
