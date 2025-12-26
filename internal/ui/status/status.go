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
	width     int
	state     State
	serverURL string
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

// View renders the status bar.
func (m Model) View() string {
	var text string
	var style lipgloss.Style

	switch m.state {
	case StateConnected:
		host := extractHost(m.serverURL)
		text = "Connected to hub (" + host + ")"
		style = lipgloss.NewStyle().
			Foreground(theme.Success).
			Background(theme.Surface)

	case StateConnecting:
		text = "Connecting..."
		style = lipgloss.NewStyle().
			Foreground(theme.Warning).
			Background(theme.Surface)

	default:
		text = "Disconnected"
		style = lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Background(theme.Surface)
	}

	// Build the full status bar
	barStyle := lipgloss.NewStyle().
		Width(m.width).
		Background(theme.Surface).
		Padding(0, 1)

	return barStyle.Render(style.Render(text))
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
