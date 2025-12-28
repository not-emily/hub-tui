package modal

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/client"
	"github.com/pxp/hub-tui/internal/ui/components"
	"github.com/pxp/hub-tui/internal/ui/theme"
)

// Modal view modes
type integrationsView int

const (
	viewList integrationsView = iota
	viewProfiles
	viewConfigure
)

// IntegrationsModal displays and configures integrations.
type IntegrationsModal struct {
	client       *client.Client
	integrations []client.Integration
	selected     int
	loading      bool
	error        string

	// Current view
	view integrationsView

	// Profile selection
	profileSelected int
	profileOptions  []string // existing profiles + "New profile"
	newProfileName  string
	enteringName    bool

	// Configure mode
	configName    string
	configProfile string
	form          *components.Form
	saving        bool
	testing       bool
	testResult    string
}

// NewIntegrationsModal creates a new integrations modal.
func NewIntegrationsModal(c *client.Client) *IntegrationsModal {
	return &IntegrationsModal{
		client:  c,
		loading: true,
		view:    viewList,
	}
}

// IntegrationsLoadedMsg is sent when integrations are loaded.
type IntegrationsLoadedMsg struct {
	Integrations []client.Integration
	Error        error
}

// IntegrationConfiguredMsg is sent when an integration is configured.
type IntegrationConfiguredMsg struct {
	Name  string
	Error error
}

// IntegrationTestedMsg is sent when an integration is tested.
type IntegrationTestedMsg struct {
	Name  string
	Error error
}

// Init initializes the modal and triggers data fetch.
func (m *IntegrationsModal) Init() tea.Cmd {
	return m.loadIntegrations()
}

func (m *IntegrationsModal) loadIntegrations() tea.Cmd {
	return func() tea.Msg {
		integrations, err := m.client.ListIntegrations()
		return IntegrationsLoadedMsg{Integrations: integrations, Error: err}
	}
}

func (m *IntegrationsModal) configureIntegration() tea.Cmd {
	config := m.form.Values()
	name := m.configName
	profile := m.configProfile
	return func() tea.Msg {
		err := m.client.ConfigureIntegration(name, profile, config)
		return IntegrationConfiguredMsg{Name: name, Error: err}
	}
}

func (m *IntegrationsModal) testIntegration() tea.Cmd {
	name := m.integrations[m.selected].Name
	return func() tea.Msg {
		err := m.client.TestIntegration(name)
		return IntegrationTestedMsg{Name: name, Error: err}
	}
}

// Update handles input.
func (m *IntegrationsModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case IntegrationsLoadedMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			m.integrations = msg.Integrations
			m.error = ""
		}
		return m, nil

	case IntegrationConfiguredMsg:
		m.saving = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			// Success - go back to list and refresh
			m.view = viewList
			m.form = nil
			m.loading = true
			return m, m.loadIntegrations()
		}
		return m, nil

	case IntegrationTestedMsg:
		m.testing = false
		if msg.Error != nil {
			m.testResult = "✗ " + msg.Error.Error()
		} else {
			m.testResult = "✓ Connection successful"
		}
		return m, nil

	case tea.KeyMsg:
		switch m.view {
		case viewList:
			return m.updateList(msg)
		case viewProfiles:
			return m.updateProfiles(msg)
		case viewConfigure:
			return m.updateConfigure(msg)
		}
	}
	return m, nil
}

func (m *IntegrationsModal) updateList(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return nil, nil // Close modal
	case "up", "k":
		if m.selected > 0 {
			m.selected--
			m.testResult = ""
		}
	case "down", "j":
		if m.selected < len(m.integrations)-1 {
			m.selected++
			m.testResult = ""
		}
	case "enter":
		if !m.loading && len(m.integrations) > 0 {
			m.enterProfilesView()
		}
	case "t":
		if !m.loading && !m.testing && len(m.integrations) > 0 {
			m.testing = true
			m.testResult = ""
			return m, m.testIntegration()
		}
	case "r":
		m.loading = true
		m.error = ""
		m.testResult = ""
		return m, m.loadIntegrations()
	}
	return m, nil
}

func (m *IntegrationsModal) updateProfiles(msg tea.KeyMsg) (Modal, tea.Cmd) {
	// Handle new profile name entry
	if m.enteringName {
		switch msg.String() {
		case "esc":
			m.enteringName = false
			m.newProfileName = ""
			return m, nil
		case "enter":
			if m.newProfileName != "" {
				m.configProfile = m.newProfileName
				m.enteringName = false
				m.enterConfigureMode()
			}
			return m, nil
		case "backspace":
			if len(m.newProfileName) > 0 {
				m.newProfileName = m.newProfileName[:len(m.newProfileName)-1]
			}
			return m, nil
		default:
			char := msg.String()
			// Allow alphanumeric and underscore/hyphen
			if len(char) == 1 && (char[0] >= 'a' && char[0] <= 'z' ||
				char[0] >= 'A' && char[0] <= 'Z' ||
				char[0] >= '0' && char[0] <= '9' ||
				char[0] == '_' || char[0] == '-') {
				m.newProfileName += char
			}
			return m, nil
		}
	}

	switch msg.String() {
	case "esc":
		m.view = viewList
		m.error = ""
		return m, nil
	case "up", "k":
		if m.profileSelected > 0 {
			m.profileSelected--
		}
	case "down", "j":
		if m.profileSelected < len(m.profileOptions)-1 {
			m.profileSelected++
		}
	case "enter":
		option := m.profileOptions[m.profileSelected]
		if option == "+ New profile" {
			m.enteringName = true
			m.newProfileName = ""
		} else {
			m.configProfile = option
			m.enterConfigureMode()
		}
	}
	return m, nil
}

func (m *IntegrationsModal) updateConfigure(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.view = viewProfiles
		m.form = nil
		m.error = ""
		return m, nil
	}

	// Forward to form
	if m.form != nil {
		submit := m.form.Update(msg)
		if submit && !m.saving {
			m.saving = true
			return m, m.configureIntegration()
		}
	}
	return m, nil
}

func (m *IntegrationsModal) enterProfilesView() {
	integration := m.integrations[m.selected]
	m.configName = integration.Name
	m.view = viewProfiles
	m.profileSelected = 0
	m.error = ""

	// Build profile options: existing profiles + new profile option
	m.profileOptions = make([]string, 0, len(integration.Profiles)+1)
	for _, p := range integration.Profiles {
		m.profileOptions = append(m.profileOptions, p)
	}
	m.profileOptions = append(m.profileOptions, "+ New profile")

	// Default to "default" if no profiles exist
	if len(integration.Profiles) == 0 {
		m.profileSelected = 0 // Will be "+ New profile"
	}
}

func (m *IntegrationsModal) enterConfigureMode() {
	integration := m.integrations[m.selected]
	m.view = viewConfigure
	m.error = ""

	// Build form fields from integration's required fields
	var fields []components.FormField
	for _, fieldName := range integration.Fields {
		fields = append(fields, components.FormField{
			Label:    fieldName,
			Key:      fieldName,
			Password: strings.Contains(strings.ToLower(fieldName), "key") ||
				strings.Contains(strings.ToLower(fieldName), "secret") ||
				strings.Contains(strings.ToLower(fieldName), "password") ||
				strings.Contains(strings.ToLower(fieldName), "token"),
		})
	}

	// If no fields defined, add a generic API key field
	if len(fields) == 0 {
		fields = append(fields, components.FormField{
			Label:    "API Key",
			Key:      "api_key",
			Password: true,
		})
	}

	m.form = components.NewForm("Configure "+integration.Name, fields)
}

// Title returns the modal title.
func (m *IntegrationsModal) Title() string {
	switch m.view {
	case viewProfiles:
		return m.configName + ": Select Profile"
	case viewConfigure:
		return fmt.Sprintf("Configure: %s (%s)", m.configName, m.configProfile)
	default:
		return "Integrations"
	}
}

// View renders the modal content.
func (m *IntegrationsModal) View() string {
	switch m.view {
	case viewProfiles:
		return m.viewProfilesContent()
	case viewConfigure:
		return m.viewConfigureContent()
	default:
		return m.viewListContent()
	}
}

func (m *IntegrationsModal) viewListContent() string {
	if m.loading {
		return lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("Loading integrations...")
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

	if len(m.integrations) == 0 {
		return lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("No integrations found.")
	}

	var lines []string

	configuredStyle := lipgloss.NewStyle().Foreground(theme.Success)
	notConfiguredStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	descStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	for i, integration := range m.integrations {
		// Status indicator
		var indicator string
		if integration.Configured {
			indicator = configuredStyle.Render("✓")
		} else {
			indicator = notConfiguredStyle.Render("✗")
		}

		// Name with selection highlight
		var name string
		if i == m.selected {
			name = selectedStyle.Render(integration.Name)
		} else {
			name = normalStyle.Render(integration.Name)
		}

		// Build line with profiles info
		line := fmt.Sprintf("  %s %s", indicator, name)

		// Pad name for alignment
		padding := 16 - len(integration.Name)
		if padding < 2 {
			padding = 2
		}

		// Show profiles or status
		if len(integration.Profiles) > 0 {
			profilesStr := strings.Join(integration.Profiles, ", ")
			line += strings.Repeat(" ", padding) + descStyle.Render(profilesStr)
		} else {
			line += strings.Repeat(" ", padding) + descStyle.Render("Not configured")
		}

		lines = append(lines, line)
	}

	// Add test result if present
	if m.testResult != "" {
		lines = append(lines, "")
		var resultStyle lipgloss.Style
		if strings.HasPrefix(m.testResult, "✓") {
			resultStyle = lipgloss.NewStyle().Foreground(theme.Success)
		} else {
			resultStyle = lipgloss.NewStyle().Foreground(theme.Error)
		}
		lines = append(lines, "  "+resultStyle.Render(m.testResult))
	}

	// Add testing indicator
	if m.testing {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("  Testing..."))
	}

	// Add hints
	lines = append(lines, "")
	legendStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	lines = append(lines, legendStyle.Render("  [Enter] Configure  [t] Test  [r] Refresh"))

	return strings.Join(lines, "\n")
}

func (m *IntegrationsModal) viewProfilesContent() string {
	var lines []string

	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	newStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	// Show entering name mode
	if m.enteringName {
		lines = append(lines, "  Enter profile name:")
		lines = append(lines, "")
		cursorStyle := lipgloss.NewStyle().Foreground(theme.Accent).Underline(true)
		nameDisplay := selectedStyle.Render(m.newProfileName) + cursorStyle.Render(" ")
		lines = append(lines, "  "+nameDisplay)
		lines = append(lines, "")
		legendStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
		lines = append(lines, legendStyle.Render("  [Enter] Confirm  [Esc] Cancel"))
		return strings.Join(lines, "\n")
	}

	// Show profile options
	for i, option := range m.profileOptions {
		var line string
		if option == "+ New profile" {
			if i == m.profileSelected {
				line = "  " + selectedStyle.Render(option)
			} else {
				line = "  " + newStyle.Render(option)
			}
		} else {
			if i == m.profileSelected {
				line = "  " + selectedStyle.Render("● "+option)
			} else {
				line = "  " + normalStyle.Render("○ "+option)
			}
		}
		lines = append(lines, line)
	}

	// Add hints
	lines = append(lines, "")
	legendStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	lines = append(lines, legendStyle.Render("  [Enter] Select  [Esc] Back"))

	return strings.Join(lines, "\n")
}

func (m *IntegrationsModal) viewConfigureContent() string {
	var lines []string

	// Show form
	if m.form != nil {
		lines = append(lines, m.form.View())
	}

	// Show error if any
	if m.error != "" {
		lines = append(lines, "")
		errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
		lines = append(lines, "  "+errorStyle.Render("Error: "+m.error))
	}

	// Show saving indicator
	if m.saving {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("  Saving..."))
	}

	// Add hints
	lines = append(lines, "")
	legendStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	lines = append(lines, legendStyle.Render("  [Enter] Save  [Esc] Back"))

	return strings.Join(lines, "\n")
}
