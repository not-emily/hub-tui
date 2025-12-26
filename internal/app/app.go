package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/client"
	"github.com/pxp/hub-tui/internal/config"
	"github.com/pxp/hub-tui/internal/ui/login"
	"github.com/pxp/hub-tui/internal/ui/status"
	"github.com/pxp/hub-tui/internal/ui/theme"
)

const quitHintDuration = 2 * time.Second

// AppState represents the current application state.
type AppState int

const (
	StateLogin AppState = iota
	StateMain
)

// Model is the root Bubble Tea model for hub-tui.
type Model struct {
	config       *config.Config
	client       *client.Client
	width        int
	height       int
	state        AppState
	quitting     bool
	ctrlCPressed bool

	// Sub-components
	login     login.Model
	statusBar status.Model
}

// New creates a new app model with the given config.
func New(cfg *config.Config) Model {
	needsServerURL := cfg.ServerURL == ""
	needsLogin := needsServerURL || cfg.Token == "" || client.IsTokenExpired(cfg.Token)

	m := Model{
		config:    cfg,
		statusBar: status.New(),
	}

	if needsLogin {
		m.state = StateLogin
		m.login = login.New(needsServerURL, cfg.ServerURL)
	} else {
		m.state = StateMain
		m.client = client.New(cfg.ServerURL)
		m.client.SetToken(cfg.Token)
	}

	return m
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	if m.state == StateMain {
		// Verify connection with health check
		return m.doHealthCheck()
	}
	return nil
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.login.SetSize(msg.Width, msg.Height)
		m.statusBar.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		// Global key handling
		if IsQuit(msg) {
			if m.ctrlCPressed {
				m.quitting = true
				return m, tea.Quit
			}
			m.ctrlCPressed = true
			m.login.SetCtrlCPressed(true)
			return m, tea.Tick(quitHintDuration, func(time.Time) tea.Msg {
				return QuitHintExpiredMsg{}
			})
		}

		if IsRedraw(msg) {
			return m, tea.ClearScreen
		}

		// Reset Ctrl+C state on any other key
		m.ctrlCPressed = false
		m.login.SetCtrlCPressed(false)

		// Route to current state handler
		switch m.state {
		case StateLogin:
			return m.updateLogin(msg)
		case StateMain:
			return m.updateMain(msg)
		}

	case QuitHintExpiredMsg:
		m.ctrlCPressed = false
		m.login.SetCtrlCPressed(false)
		return m, nil

	case LoginResultMsg:
		return m.handleLoginResult(msg)

	case HealthCheckMsg:
		return m.handleHealthCheck(msg)
	}

	return m, nil
}

func (m Model) updateLogin(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Check for form submission
	if m.login.IsSubmit(msg) {
		if err := m.login.Validate(); err != "" {
			m.login.SetError(err)
			return m, nil
		}

		m.login.SetConnecting()

		// Create client with the server URL
		serverURL := m.login.ServerURL()
		if serverURL == "" {
			serverURL = m.config.ServerURL
		}
		m.client = client.New(serverURL)

		return m, m.doLogin(m.login.Username(), m.login.Password())
	}

	// Update login form
	var cmd tea.Cmd
	m.login, cmd = m.login.Update(msg)
	return m, cmd
}

func (m Model) updateMain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Main view key handling will be expanded in later phases
	return m, nil
}

func (m Model) handleLoginResult(msg LoginResultMsg) (tea.Model, tea.Cmd) {
	if !msg.Success {
		m.login.SetError(msg.Error)
		return m, nil
	}

	// Store token and server URL in config
	m.config.ServerURL = m.client.BaseURL()
	m.config.Token = msg.Token
	m.config.TokenExp = msg.ExpiresAt
	if err := m.config.Save(); err != nil {
		m.login.SetError("Failed to save config: " + err.Error())
		return m, nil
	}

	// Set token on client
	m.client.SetToken(msg.Token)

	// Transition to main state
	m.state = StateMain
	m.statusBar.SetServerURL(m.client.BaseURL())
	m.statusBar.SetState(status.StateConnecting)

	return m, m.doHealthCheck()
}

func (m Model) handleHealthCheck(msg HealthCheckMsg) (tea.Model, tea.Cmd) {
	if msg.Success {
		m.statusBar.SetState(status.StateConnected)
	} else {
		m.statusBar.SetState(status.StateDisconnected)
		// If we were in login, show the error
		if m.state == StateLogin {
			m.login.SetError(msg.Error)
		}
	}
	return m, nil
}

func (m Model) doLogin(username, password string) tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.Login(username, password)
		if err != nil {
			return LoginResultMsg{Success: false, Error: err.Error()}
		}
		return LoginResultMsg{
			Success:   true,
			Token:     resp.Token,
			ExpiresAt: resp.ExpiresAt,
		}
	}
}

func (m Model) doHealthCheck() tea.Cmd {
	return func() tea.Msg {
		if err := m.client.Health(); err != nil {
			return HealthCheckMsg{Success: false, Error: err.Error()}
		}
		return HealthCheckMsg{Success: true}
	}
}

// View renders the UI.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var content string
	switch m.state {
	case StateLogin:
		content = m.login.View()
	case StateMain:
		content = m.renderMain()
	}

	// Build the full screen
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Background(theme.Background).
		Render(content)
}

func (m Model) renderMain() string {
	// Calculate heights
	statusHeight := 1
	mainHeight := m.height - statusHeight

	// Main content area (placeholder for now)
	mainContent := m.renderPlaceholder(mainHeight)

	// Status bar at bottom
	statusBar := m.statusBar.View()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		mainContent,
		statusBar,
	)
}

func (m Model) renderPlaceholder(height int) string {
	// Use explicit backgrounds to avoid gray bar artifacts
	title := theme.TitleStyle.Copy().
		Background(theme.Background).
		Render("hub-tui")
	subtitle := theme.SubtitleStyle.Copy().
		Background(theme.Background).
		Render("Terminal client for Hub")

	var hint string
	if m.ctrlCPressed {
		hint = theme.WarningStyle.Copy().
			Background(theme.Background).
			Render("Ctrl+C again to quit")
	} else {
		hint = theme.HintStyle.Copy().
			Background(theme.Background).
			Render("Press Ctrl+C twice to quit")
	}

	// Use string concatenation instead of JoinVertical to avoid padding issues
	content := title + "\n" + subtitle + "\n\n" + hint

	return lipgloss.Place(
		m.width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		content,
		lipgloss.WithWhitespaceBackground(theme.Background),
	)
}
