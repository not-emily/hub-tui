package login

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/ui/theme"
)

// Field represents which input field is focused.
type Field int

const (
	FieldServerURL Field = iota
	FieldUsername
	FieldPassword
)

// State represents the login state.
type State int

const (
	StateInput State = iota
	StateConnecting
	StateError
)

// Model is the login form component.
type Model struct {
	width        int
	height       int
	focused      Field
	state        State
	error        string
	ctrlCPressed bool
	serverURL    textinput.Model
	username     textinput.Model
	password     textinput.Model

	// NeedsServerURL indicates if we need to prompt for server URL.
	NeedsServerURL bool
}

// New creates a new login model.
func New(needsServerURL bool, defaultServerURL string) Model {
	// Shared styles for inputs - no background to avoid gray boxes
	promptStyle := lipgloss.NewStyle().Foreground(theme.Accent)
	textStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	placeholderStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	serverURL := textinput.New()
	serverURL.Placeholder = "http://192.168.1.100:8787"
	serverURL.CharLimit = 256
	serverURL.Width = 35
	serverURL.PromptStyle = promptStyle
	serverURL.TextStyle = textStyle
	serverURL.PlaceholderStyle = placeholderStyle
	if defaultServerURL != "" {
		serverURL.SetValue(defaultServerURL)
	}

	username := textinput.New()
	username.Placeholder = "username"
	username.CharLimit = 64
	username.Width = 35
	username.PromptStyle = promptStyle
	username.TextStyle = textStyle
	username.PlaceholderStyle = placeholderStyle

	password := textinput.New()
	password.Placeholder = "password"
	password.CharLimit = 128
	password.Width = 35
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '*'
	password.PromptStyle = promptStyle
	password.TextStyle = textStyle
	password.PlaceholderStyle = placeholderStyle

	m := Model{
		NeedsServerURL: needsServerURL,
		serverURL:      serverURL,
		username:       username,
		password:       password,
		state:          StateInput,
	}

	// Focus the first relevant field
	if needsServerURL {
		m.focused = FieldServerURL
		m.serverURL.Focus()
	} else {
		m.focused = FieldUsername
		m.username.Focus()
	}

	return m
}

// SetSize sets the form dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetError sets an error message to display.
func (m *Model) SetError(err string) {
	m.error = err
	m.state = StateError
}

// SetConnecting sets the form to connecting state.
func (m *Model) SetConnecting() {
	m.state = StateConnecting
}

// Reset resets the form to input state.
func (m *Model) Reset() {
	m.state = StateInput
	m.error = ""
}

// SetCtrlCPressed sets the Ctrl+C pressed state for the quit hint.
func (m *Model) SetCtrlCPressed(pressed bool) {
	m.ctrlCPressed = pressed
}

// ServerURL returns the entered server URL.
func (m Model) ServerURL() string {
	return strings.TrimSpace(m.serverURL.Value())
}

// Username returns the entered username.
func (m Model) Username() string {
	return strings.TrimSpace(m.username.Value())
}

// Password returns the entered password.
func (m Model) Password() string {
	return m.password.Value()
}

// Update handles input events.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.state == StateConnecting {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			m.nextField()
			return m, nil
		case "shift+tab", "up":
			m.prevField()
			return m, nil
		case "enter":
			if m.focused == FieldPassword {
				// Submit form
				return m, nil
			}
			m.nextField()
			return m, nil
		}
	}

	// Update the focused input
	var cmd tea.Cmd
	switch m.focused {
	case FieldServerURL:
		m.serverURL, cmd = m.serverURL.Update(msg)
	case FieldUsername:
		m.username, cmd = m.username.Update(msg)
	case FieldPassword:
		m.password, cmd = m.password.Update(msg)
	}

	// Clear error on input
	if m.state == StateError {
		m.state = StateInput
		m.error = ""
	}

	return m, cmd
}

func (m *Model) nextField() {
	m.blurCurrent()

	if m.NeedsServerURL {
		switch m.focused {
		case FieldServerURL:
			m.focused = FieldUsername
		case FieldUsername:
			m.focused = FieldPassword
		case FieldPassword:
			m.focused = FieldServerURL
		}
	} else {
		switch m.focused {
		case FieldUsername:
			m.focused = FieldPassword
		case FieldPassword:
			m.focused = FieldUsername
		}
	}

	m.focusCurrent()
}

func (m *Model) prevField() {
	m.blurCurrent()

	if m.NeedsServerURL {
		switch m.focused {
		case FieldServerURL:
			m.focused = FieldPassword
		case FieldUsername:
			m.focused = FieldServerURL
		case FieldPassword:
			m.focused = FieldUsername
		}
	} else {
		switch m.focused {
		case FieldUsername:
			m.focused = FieldPassword
		case FieldPassword:
			m.focused = FieldUsername
		}
	}

	m.focusCurrent()
}

func (m *Model) blurCurrent() {
	switch m.focused {
	case FieldServerURL:
		m.serverURL.Blur()
	case FieldUsername:
		m.username.Blur()
	case FieldPassword:
		m.password.Blur()
	}
}

func (m *Model) focusCurrent() {
	switch m.focused {
	case FieldServerURL:
		m.serverURL.Focus()
	case FieldUsername:
		m.username.Focus()
	case FieldPassword:
		m.password.Focus()
	}
}

// View renders the login form.
func (m Model) View() string {
	var b strings.Builder

	// Title
	title := lipgloss.NewStyle().
		Foreground(theme.Accent).
		Bold(true).
		MarginBottom(1).
		Render("Welcome to hub-tui")

	b.WriteString(title)
	b.WriteString("\n\n")

	// Server URL field (if needed)
	if m.NeedsServerURL {
		b.WriteString(m.renderField("Server URL", m.serverURL.View(), m.focused == FieldServerURL))
		b.WriteString("\n")
	}

	// Username field
	b.WriteString(m.renderField("Username", m.username.View(), m.focused == FieldUsername))
	b.WriteString("\n")

	// Password field
	b.WriteString(m.renderField("Password", m.password.View(), m.focused == FieldPassword))
	b.WriteString("\n")

	// State message
	switch m.state {
	case StateConnecting:
		connecting := lipgloss.NewStyle().
			Foreground(theme.Warning).
			Render("Connecting...")
		b.WriteString("\n")
		b.WriteString(connecting)

	case StateError:
		errMsg := lipgloss.NewStyle().
			Foreground(theme.Error).
			Render(m.error)
		b.WriteString("\n")
		b.WriteString(errMsg)

	default:
		hint := lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Italic(true).
			Render("Press Enter to connect")
		b.WriteString("\n")
		b.WriteString(hint)
	}

	// Center the form
	formStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Surface).
		Padding(1, 2)

	form := formStyle.Render(b.String())

	// Ctrl+C hint below the form
	var quitHint string
	if m.ctrlCPressed {
		quitHint = lipgloss.NewStyle().
			Foreground(theme.Warning).
			Render("Ctrl+C again to quit")
	} else {
		quitHint = lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Italic(true).
			Render("Press Ctrl+C twice to quit")
	}

	// Join with newline instead of JoinVertical to avoid padding
	content := form + "\n\n" + quitHint

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

func (m Model) renderField(label, input string, focused bool) string {
	labelStyle := lipgloss.NewStyle().
		Foreground(theme.TextSecondary).
		Width(12)

	if focused {
		labelStyle = labelStyle.Foreground(theme.TextPrimary)
	}

	return labelStyle.Render(label+":") + " " + input
}

// IsSubmit checks if the Enter key was pressed on the password field.
func (m Model) IsSubmit(msg tea.KeyMsg) bool {
	return msg.String() == "enter" && m.focused == FieldPassword
}

// Validate checks if the form has valid input.
func (m Model) Validate() string {
	if m.NeedsServerURL && m.ServerURL() == "" {
		return "Server URL is required"
	}
	if m.Username() == "" {
		return "Username is required"
	}
	if m.Password() == "" {
		return "Password is required"
	}
	return ""
}
