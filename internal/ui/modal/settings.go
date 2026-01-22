package modal

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/config"
	"github.com/pxp/hub-tui/internal/ui/components"
	"github.com/pxp/hub-tui/internal/ui/theme"
)

// SettingsSavedMsg is sent when settings are saved.
type SettingsSavedMsg struct {
	Config *config.Config
	Error  error
}

// RefreshConnectionMsg is sent when the user requests a connection refresh.
type RefreshConnectionMsg struct{}

// LogoutMsg is sent when the user requests to logout.
type LogoutMsg struct{}

// SettingsModal displays and edits configuration.
type SettingsModal struct {
	config     *config.Config
	connected  bool
	refreshing bool
	editing    bool
	form       *components.Form
	error      string
	confirm    *components.Confirmation
}

// NewSettingsModal creates a new settings modal.
func NewSettingsModal(cfg *config.Config, connected bool) *SettingsModal {
	return &SettingsModal{
		config:    cfg,
		connected: connected,
		confirm:   components.NewConfirmation(),
	}
}

// SetConnected updates the connection status.
func (m *SettingsModal) SetConnected(connected bool) {
	m.connected = connected
	m.refreshing = false
}

// Init initializes the modal.
func (m *SettingsModal) Init() tea.Cmd {
	return nil
}

// Update handles input.
func (m *SettingsModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case SettingsSavedMsg:
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			// Update local config and exit edit mode
			m.config = msg.Config
			m.editing = false
			m.form = nil
			m.error = ""
		}
		return m, nil

	case components.ConfirmationExpiredMsg:
		m.confirm.HandleExpired(msg)
		return m, nil

	case tea.KeyMsg:
		if m.editing {
			return m.updateEditing(msg)
		}
		return m.updateViewing(msg)
	}
	return m, nil
}

// updateViewing handles input in view mode.
func (m *SettingsModal) updateViewing(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.confirm.Clear()
		return nil, nil // Close modal
	case "e":
		m.confirm.Clear()
		// Enter edit mode
		m.editing = true
		m.error = ""
		m.form = components.NewForm("Edit Settings", []components.FormField{
			{
				Label: "Server URL",
				Key:   "server_url",
				Value: m.config.ServerURL,
				Type:  components.FieldText,
			},
		})
	case "r":
		m.confirm.Clear()
		// Refresh connection
		m.refreshing = true
		return m, func() tea.Msg { return RefreshConnectionMsg{} }
	case "l":
		// Logout with confirmation
		if execute, cmd := m.confirm.Check("logout", ""); execute {
			return m, func() tea.Msg { return LogoutMsg{} }
		} else if cmd != nil {
			return m, cmd
		}
	}
	return m, nil
}

// updateEditing handles input in edit mode.
func (m *SettingsModal) updateEditing(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel edit mode
		m.editing = false
		m.form = nil
		m.error = ""
		return m, nil
	case "ctrl+s":
		// Save settings
		return m, m.saveSettings()
	}

	// Pass to form
	m.form.Update(msg)
	return m, nil
}

// saveSettings saves the settings to the config file.
func (m *SettingsModal) saveSettings() tea.Cmd {
	serverURL := m.form.GetFieldValue("server_url")
	return func() tea.Msg {
		// Create updated config (preserve token info)
		newConfig := &config.Config{
			ServerURL: strings.TrimSpace(serverURL),
			Token:     m.config.Token,
			TokenExp:  m.config.TokenExp,
		}

		// Save to disk
		if err := newConfig.Save(); err != nil {
			return SettingsSavedMsg{Error: err}
		}

		return SettingsSavedMsg{Config: newConfig}
	}
}

// Title returns the modal title.
func (m *SettingsModal) Title() string {
	return "Settings"
}

// View renders the settings content.
func (m *SettingsModal) View() string {
	if m.editing {
		return m.viewEditing()
	}
	return m.viewDisplay()
}

// viewDisplay renders the read-only settings view.
func (m *SettingsModal) viewDisplay() string {
	labelStyle := lipgloss.NewStyle().
		Foreground(theme.TextSecondary).
		Width(14)

	valueStyle := lipgloss.NewStyle().
		Foreground(theme.TextPrimary)

	successStyle := lipgloss.NewStyle().
		Foreground(theme.Success)

	errorStyle := lipgloss.NewStyle().
		Foreground(theme.Error)

	hintStyle := lipgloss.NewStyle().
		Foreground(theme.TextSecondary).
		Italic(true)

	var lines []string

	// Server URL
	serverURL := m.config.ServerURL
	if serverURL == "" {
		serverURL = "(not set)"
	}
	lines = append(lines,
		labelStyle.Render("Server URL:")+valueStyle.Render(serverURL),
	)

	// Connection status
	var connStatus string
	if m.refreshing {
		connStatus = hintStyle.Render("Checking...")
	} else if m.connected {
		connStatus = successStyle.Render("Connected")
	} else {
		connStatus = errorStyle.Render("Disconnected")
	}
	lines = append(lines,
		labelStyle.Render("Status:")+connStatus,
	)

	lines = append(lines, "")

	// Token expiry
	tokenExp := m.formatTokenExpiry()
	lines = append(lines,
		labelStyle.Render("Token expires:")+valueStyle.Render(tokenExp),
	)

	lines = append(lines, "")
	lines = append(lines, "")

	// Config file location
	configPath, _ := config.DefaultPath()
	lines = append(lines,
		hintStyle.Render("Config: "+configPath),
	)

	lines = append(lines, "")

	// Show confirmation hint or regular hints
	if m.confirm.IsPending("logout", "") {
		warningStyle := lipgloss.NewStyle().Foreground(theme.Warning)
		lines = append(lines, warningStyle.Render("Press l again to logout"))
	} else {
		lines = append(lines, hintStyle.Render("[e] Edit  [r] Refresh  [l] Logout"))
	}

	return strings.Join(lines, "\n")
}

// viewEditing renders the edit form.
func (m *SettingsModal) viewEditing() string {
	var lines []string

	// Form
	lines = append(lines, m.form.View())

	// Error message
	if m.error != "" {
		errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
		lines = append(lines, "")
		lines = append(lines, errorStyle.Render("Error: "+m.error))
	}

	// Hints
	lines = append(lines, "")
	hintStyle := lipgloss.NewStyle().
		Foreground(theme.TextSecondary).
		Italic(true)
	lines = append(lines, hintStyle.Render("[Ctrl+S] Save  [Esc] Cancel"))

	return strings.Join(lines, "\n")
}

// formatTokenExpiry formats the token expiry date.
func (m *SettingsModal) formatTokenExpiry() string {
	if m.config.TokenExp == "" {
		return "(unknown)"
	}

	// Try to parse the expiry time
	t, err := time.Parse(time.RFC3339, m.config.TokenExp)
	if err != nil {
		return m.config.TokenExp
	}

	// Format nicely
	if time.Now().After(t) {
		return "Expired"
	}

	return t.Format("2006-01-02 15:04")
}
