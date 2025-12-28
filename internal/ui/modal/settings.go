package modal

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/config"
	"github.com/pxp/hub-tui/internal/ui/theme"
)

// SettingsModal displays current configuration.
type SettingsModal struct {
	config    *config.Config
	connected bool
}

// NewSettingsModal creates a new settings modal.
func NewSettingsModal(cfg *config.Config, connected bool) *SettingsModal {
	return &SettingsModal{
		config:    cfg,
		connected: connected,
	}
}

// Init initializes the modal.
func (m *SettingsModal) Init() tea.Cmd {
	return nil
}

// Update handles input.
func (m *SettingsModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			return nil, nil // Close modal
		}
	}
	return m, nil
}

// Title returns the modal title.
func (m *SettingsModal) Title() string {
	return "Settings"
}

// View renders the settings content.
func (m *SettingsModal) View() string {
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
	if m.connected {
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
