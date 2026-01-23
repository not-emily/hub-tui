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

// Modal view modes
type integrationsView int

const (
	viewList integrationsView = iota
	viewProfiles
	viewConfigure
	viewCheckingDependencies
	viewDependencyBlocked
	viewHubUpdating
)

// LLM config type views (offset to avoid collision, defined in integrations_llm.go)
const (
	viewConfigLLM integrationsView = iota + 100
	viewLLMProviderForm
	viewLLMProfileForm
)

// IntegrationsModal displays and configures integrations.
type IntegrationsModal struct {
	client       *client.Client
	integrations []client.Integration
	selected     int
	loading      bool
	error        string
	isAdmin      bool // Whether the current user has admin permissions

	// Tab state
	tabs *components.Tabs

	// Dependencies state
	dependencies     []client.Dependency
	depLoading       bool
	depError         string
	depInstalling    string // Name of dependency being installed (empty if none)
	selectedDepIndex int    // Selected dependency in list

	// Dependency check state (for blocking config until deps satisfied)
	pendingIntegration client.Integration  // Integration waiting for dependency check
	unsatisfiedDeps    []client.Dependency // Dependencies that need to be installed

	// Hub update state
	hubUpdateInfo    *client.HubUpdateInfo
	hubUpdateLoading bool
	hubUpdateError   string
	hubUpdateConfirm bool // Confirmation dialog state
	hubUpdating      bool // Update in progress

	// Current view
	view integrationsView

	// Profile selection (api_key config type)
	profileSelected int
	profileOptions  []string // existing profiles + "New profile"
	newProfileName  string
	enteringName    bool

	// Configure mode (api_key config type)
	configName    string
	configProfile string
	form          *components.Form
	saving        bool
	testing       bool
	testResult    string

	// LLM config type state (implemented in integrations_llm.go)
	llmIntegration client.Integration        // current integration being configured
	llmProviders   []client.ProviderAccount  // loaded providers
	llmProfiles    []client.LLMProfile       // loaded profiles
	llmItems       []llmListItem             // flattened list for navigation
	llmSelected    int                       // current selection index
	llmLoading     bool
	llmError       string

	// LLM provider form state
	llmProviderForm       *components.Form
	llmAvailableProviders []client.AvailableProvider
	llmProviderFields     []client.ProviderFieldInfo // Field requirements for selected provider
	llmLoadingFields      bool                       // Loading field requirements
	llmSavingProvider     bool

	// LLM profile form state
	llmProfileForm    *components.Form
	llmEditingProfile *client.LLMProfile // nil if creating new
	llmSavingProfile  bool

	// Model pagination state
	llmModels            []client.ModelInfo
	llmModelsCursor      string   // current cursor (empty = first page)
	llmModelsCursorStack []string // stack of previous cursors for back navigation
	llmModelsHasMore     bool
	llmModelsPage        int
	llmLoadingModels     bool

	// LLM profile testing state
	llmTesting    bool
	llmTestResult *client.LLMTestResult

	// LLM confirmation state
	llmConfirm components.Confirmation
}

// NewIntegrationsModal creates a new integrations modal.
func NewIntegrationsModal(c *client.Client, isAdmin bool) *IntegrationsModal {
	return &IntegrationsModal{
		client:  c,
		isAdmin: isAdmin,
		loading: true,
		view:    viewList,
		tabs:    components.NewTabs([]string{"Integrations", "Dependencies"}),
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

// DependenciesLoadedMsg is sent when dependencies are loaded.
type DependenciesLoadedMsg struct {
	Dependencies []client.Dependency
	Err          error
}

// DependencyInstalledMsg is sent when a dependency is installed.
type DependencyInstalledMsg struct {
	Name   string
	Status *client.Dependency
	Err    error
}

// DependencyCheckMsg is sent after checking dependencies for an integration.
type DependencyCheckMsg struct {
	Integration  string
	Dependencies []client.Dependency
	Err          error
}

// HubUpdatesLoadedMsg is sent when hub update info is loaded.
type HubUpdatesLoadedMsg struct {
	UpdateInfo *client.HubUpdateInfo
	Err        error
}

// HubUpdateAppliedMsg is sent when hub update is applied.
type HubUpdateAppliedMsg struct {
	Success bool
	Message string
	Err     error
}

// HubUpdateConfirmExpiredMsg is sent when the update confirmation times out.
type HubUpdateConfirmExpiredMsg struct{}

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

func (m *IntegrationsModal) fetchDependencies() tea.Cmd {
	return func() tea.Msg {
		deps, err := m.client.GetDependencies("") // Empty = all dependencies
		return DependenciesLoadedMsg{Dependencies: deps, Err: err}
	}
}

func (m *IntegrationsModal) installDependency(name, version string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.InstallDependency(name, version)
		if err != nil {
			return DependencyInstalledMsg{Name: name, Err: err}
		}
		return DependencyInstalledMsg{Name: name, Status: &result.Status, Err: nil}
	}
}

func (m *IntegrationsModal) checkHubUpdates() tea.Cmd {
	return func() tea.Msg {
		info, err := m.client.GetHubUpdates()
		return HubUpdatesLoadedMsg{UpdateInfo: info, Err: err}
	}
}

func (m *IntegrationsModal) applyHubUpdate() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.ApplyHubUpdate()
		if err != nil {
			return HubUpdateAppliedMsg{Success: false, Err: err}
		}
		return HubUpdateAppliedMsg{Success: result.Success, Message: result.Message, Err: nil}
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

	case LLMDataLoadedMsg:
		return m.handleLLMDataLoaded(msg)

	case LLMAvailableProvidersMsg:
		return m.handleLLMAvailableProviders(msg)

	case LLMProviderFieldsMsg:
		return m.handleLLMProviderFields(msg)

	case LLMProviderSavedMsg:
		return m.handleLLMProviderSaved(msg)

	case LLMProviderDeletedMsg:
		return m.handleLLMProviderDeleted(msg)

	case LLMErrorMsg:
		m.llmLoading = false
		m.llmLoadingFields = false
		m.llmSavingProvider = false
		m.llmSavingProfile = false
		m.llmLoadingModels = false
		m.llmError = msg.Err.Error()
		return m, nil

	case LLMModelsLoadedMsg:
		return m.handleLLMModelsLoaded(msg)

	case LLMProfileSavedMsg:
		return m.handleLLMProfileSaved(msg)

	case LLMProfileDeletedMsg:
		return m.handleLLMProfileDeleted(msg)

	case LLMProfileTestedMsg:
		return m.handleLLMProfileTested(msg)

	case LLMProfileDefaultSetMsg:
		return m.handleLLMProfileDefaultSet(msg)

	case components.ConfirmationExpiredMsg:
		m.llmConfirm.HandleExpired(msg)
		return m, nil

	case DependenciesLoadedMsg:
		m.depLoading = false
		if msg.Err != nil {
			m.depError = msg.Err.Error()
		} else {
			m.dependencies = msg.Dependencies
			m.depError = ""
		}
		return m, nil

	case DependencyCheckMsg:
		if msg.Err != nil {
			// Error checking dependencies - show error but allow proceeding
			// (better to let them try than block completely)
			m.depError = msg.Err.Error()
			return m.proceedToLLMConfig()
		}

		// Check if any dependencies are unsatisfied
		var unsatisfied []client.Dependency
		for _, dep := range msg.Dependencies {
			if !dep.Installed || dep.NeedsUpdate {
				unsatisfied = append(unsatisfied, dep)
			}
		}

		if len(unsatisfied) == 0 {
			// All satisfied, proceed to config
			return m.proceedToLLMConfig()
		}

		// Block config until dependencies are installed
		m.unsatisfiedDeps = unsatisfied
		m.view = viewDependencyBlocked
		return m, nil

	case DependencyInstalledMsg:
		m.depInstalling = ""
		if msg.Err != nil {
			m.depError = fmt.Sprintf("Failed to install %s: %s", msg.Name, msg.Err.Error())
			return m, nil
		}

		// Update dependency in main list (Dependencies tab)
		for i, dep := range m.dependencies {
			if dep.Name == msg.Name {
				m.dependencies[i] = *msg.Status
				break
			}
		}

		// If we're in blocked view, also update unsatisfied deps list
		if m.view == viewDependencyBlocked {
			for i, dep := range m.unsatisfiedDeps {
				if dep.Name == msg.Name {
					m.unsatisfiedDeps[i] = *msg.Status
					break
				}
			}

			// Check if all deps are now satisfied
			allSatisfied := true
			for _, dep := range m.unsatisfiedDeps {
				if !dep.Installed || dep.NeedsUpdate {
					allSatisfied = false
					break
				}
			}

			if allSatisfied {
				// All satisfied! Proceed to config
				return m.proceedToLLMConfig()
			}
		}

		m.depError = ""
		return m, nil

	case HubUpdatesLoadedMsg:
		m.hubUpdateLoading = false
		if msg.Err != nil {
			// Check if 404 (repo is private)
			if apiErr, ok := msg.Err.(*client.APIError); ok && apiErr.StatusCode == 404 {
				m.hubUpdateError = "Updates not available (repository is private)"
			} else {
				m.hubUpdateError = msg.Err.Error()
			}
			return m, nil
		}
		m.hubUpdateInfo = msg.UpdateInfo
		m.hubUpdateError = ""
		return m, nil

	case HubUpdateAppliedMsg:
		if msg.Err != nil {
			m.hubUpdateError = fmt.Sprintf("Update failed: %s", msg.Err.Error())
			m.hubUpdating = false
			return m, nil
		}
		// Update initiated, server will restart
		m.hubUpdateError = ""
		m.hubUpdating = true
		m.view = viewHubUpdating
		return m, nil

	case HubUpdateConfirmExpiredMsg:
		m.hubUpdateConfirm = false
		return m, nil

	case tea.KeyMsg:
		switch m.view {
		case viewList:
			return m.updateList(msg)
		case viewProfiles:
			return m.updateProfiles(msg)
		case viewConfigure:
			return m.updateConfigure(msg)
		case viewConfigLLM, viewLLMProviderForm, viewLLMProfileForm:
			return m.updateLLM(msg)
		case viewCheckingDependencies:
			// Allow Esc to cancel while checking
			if msg.String() == "esc" {
				m.view = viewList
				return m, nil
			}
		case viewDependencyBlocked:
			return m.updateDependencyBlocked(msg)
		case viewHubUpdating:
			// Allow Esc to close (update continues in background)
			if msg.String() == "esc" {
				m.view = viewList
				m.hubUpdating = false
				return m, nil
			}
		}
	}
	return m, nil
}

func (m *IntegrationsModal) updateList(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return nil, nil // Close modal
	case "tab":
		m.tabs.Next()
		// If switching to Dependencies tab, load data as needed
		if m.tabs.ActiveIndex() == 1 {
			var cmds []tea.Cmd
			if m.dependencies == nil {
				m.depLoading = true
				cmds = append(cmds, m.fetchDependencies())
			}
			if m.hubUpdateInfo == nil && m.isAdmin && !m.hubUpdateLoading {
				m.hubUpdateLoading = true
				cmds = append(cmds, m.checkHubUpdates())
			}
			if len(cmds) > 0 {
				return m, tea.Batch(cmds...)
			}
		}
		return m, nil
	case "shift+tab":
		m.tabs.Previous()
		return m, nil
	case "up", "k":
		if m.tabs.ActiveIndex() == 0 {
			// Integrations tab
			if m.selected > 0 {
				m.selected--
				m.testResult = ""
			}
		} else {
			// Dependencies tab
			if m.selectedDepIndex > 0 {
				m.selectedDepIndex--
			}
		}
	case "down", "j":
		if m.tabs.ActiveIndex() == 0 {
			// Integrations tab
			if m.selected < len(m.integrations)-1 {
				m.selected++
				m.testResult = ""
			}
		} else {
			// Dependencies tab
			if m.selectedDepIndex < len(m.dependencies)-1 {
				m.selectedDepIndex++
			}
		}
	case "enter":
		if m.tabs.ActiveIndex() == 0 {
			// Integrations tab
			if !m.loading && len(m.integrations) > 0 {
				integration := m.integrations[m.selected]
				switch integration.ConfigType {
				case "llm":
					return m.enterLLMConfig(integration)
				case "api_key", "":
					// api_key is the default for backwards compatibility
					m.enterProfilesView()
				default:
					m.error = fmt.Sprintf("Unknown config type: %s", integration.ConfigType)
				}
			}
		} else {
			// Dependencies tab - install/update
			if m.isAdmin && len(m.dependencies) > 0 && m.depInstalling == "" {
				dep := m.dependencies[m.selectedDepIndex]
				if !dep.Installed || dep.NeedsUpdate {
					m.depInstalling = dep.Name
					return m, m.installDependency(dep.Name, dep.RequiredVersion)
				}
			}
		}
	case "t":
		if m.tabs.ActiveIndex() == 0 && !m.loading && !m.testing && len(m.integrations) > 0 {
			m.testing = true
			m.testResult = ""
			return m, m.testIntegration()
		}
	case "r":
		if m.tabs.ActiveIndex() == 0 {
			// Refresh integrations
			m.loading = true
			m.error = ""
			m.testResult = ""
		} else {
			// Refresh dependencies and hub updates
			m.depLoading = true
			m.depError = ""
			m.hubUpdateInfo = nil
			m.hubUpdateError = ""
			var cmds []tea.Cmd
			cmds = append(cmds, m.fetchDependencies())
			if m.isAdmin {
				m.hubUpdateLoading = true
				cmds = append(cmds, m.checkHubUpdates())
			}
			return m, tea.Batch(cmds...)
		}
		return m, m.loadIntegrations()
	case "c":
		// Check for hub updates (Dependencies tab only)
		if m.tabs.ActiveIndex() == 1 && m.isAdmin && !m.hubUpdateLoading {
			m.hubUpdateLoading = true
			m.hubUpdateError = ""
			return m, m.checkHubUpdates()
		}
	case "u":
		// Apply hub update (Dependencies tab only, with confirmation)
		if m.tabs.ActiveIndex() == 1 && m.isAdmin && m.hubUpdateInfo != nil && m.hubUpdateInfo.UpdateAvailable {
			if m.hubUpdateConfirm {
				// Confirmed, apply update
				m.hubUpdateConfirm = false
				m.hubUpdating = true
				return m, m.applyHubUpdate()
			} else {
				// First press, show confirmation
				m.hubUpdateConfirm = true
				// Clear confirmation after 3 seconds
				return m, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
					return HubUpdateConfirmExpiredMsg{}
				})
			}
		}
	}
	return m, nil
}

func (m *IntegrationsModal) updateDependencyBlocked(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.view = viewList
		m.unsatisfiedDeps = nil
		return m, nil
	case "enter":
		if m.isAdmin && m.depInstalling == "" {
			// Install first unsatisfied dependency
			for _, dep := range m.unsatisfiedDeps {
				if !dep.Installed || dep.NeedsUpdate {
					m.depInstalling = dep.Name
					return m, m.installDependency(dep.Name, dep.RequiredVersion)
				}
			}
		}
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
	case "ctrl+s":
		if !m.saving && m.form != nil {
			m.saving = true
			return m, m.configureIntegration()
		}
		return m, nil
	}

	// Forward to form
	if m.form != nil {
		m.form.Update(msg)
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
	case viewConfigLLM:
		return m.llmIntegration.DisplayName + " Configuration"
	case viewLLMProviderForm:
		return m.llmIntegration.DisplayName + ": Add Provider"
	case viewLLMProfileForm:
		return m.llmIntegration.DisplayName + ": Profile"
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
	case viewConfigLLM, viewLLMProviderForm, viewLLMProfileForm:
		return m.viewLLM()
	case viewCheckingDependencies:
		return m.viewCheckingDependencies()
	case viewDependencyBlocked:
		return m.viewDependencyBlocked()
	case viewHubUpdating:
		return m.viewHubUpdating()
	default:
		return m.viewListContent()
	}
}

func (m *IntegrationsModal) viewCheckingDependencies() string {
	return lipgloss.NewStyle().
		Foreground(theme.TextSecondary).
		Render("Checking dependencies...")
}

func (m *IntegrationsModal) viewDependencyBlocked() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Warning)
	b.WriteString(titleStyle.Render("Dependencies Required"))
	b.WriteString("\n\n")

	textStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	b.WriteString(textStyle.Render(
		"The following dependencies must be installed before configuring this integration:"))
	b.WriteString("\n\n")

	// List unsatisfied dependencies
	for _, dep := range m.unsatisfiedDeps {
		var status string
		if !dep.Installed {
			status = "Not installed"
		} else if dep.NeedsUpdate {
			status = fmt.Sprintf("Outdated (installed: %s, required: %s)",
				dep.InstalledVersion, dep.RequiredVersion)
		}

		depStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
		b.WriteString(depStyle.Render(fmt.Sprintf("  • %s: %s", dep.Name, status)))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Show error if any
	if m.depError != "" {
		errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
		b.WriteString(errorStyle.Render("Error: " + m.depError))
		b.WriteString("\n\n")
	}

	// Show installing status if in progress
	if m.depInstalling != "" {
		loadingStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary).Italic(true)
		b.WriteString(loadingStyle.Render(fmt.Sprintf("Installing %s...", m.depInstalling)))
		b.WriteString("\n\n")
	}

	// Show appropriate hints based on admin status
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	if m.isAdmin {
		b.WriteString(hintStyle.Render("[Enter] Install dependencies  [Esc] Cancel"))
	} else {
		messageStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary).Italic(true)
		b.WriteString(messageStyle.Render(
			"Please contact your administrator to install these dependencies."))
		b.WriteString("\n\n")
		b.WriteString(hintStyle.Render("[Esc] Back"))
	}

	return b.String()
}

func (m *IntegrationsModal) viewHubUpdating() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.TextPrimary)
	b.WriteString(titleStyle.Render("Hub Update in Progress"))
	b.WriteString("\n\n")

	textStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	b.WriteString(textStyle.Render("Hub-core is updating and will restart momentarily."))
	b.WriteString("\n")
	b.WriteString(textStyle.Render("You will be disconnected briefly. The TUI will reconnect automatically."))
	b.WriteString("\n\n")

	loadingStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary).Italic(true)
	b.WriteString(loadingStyle.Render("Waiting for server to restart..."))
	b.WriteString("\n\n")

	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	b.WriteString(hintStyle.Render("[Esc] Close (update will continue in background)"))

	return b.String()
}

func (m *IntegrationsModal) viewListContent() string {
	var b strings.Builder

	// Render tabs at top
	b.WriteString(m.tabs.View())
	b.WriteString("\n")

	// Add separator line
	separatorStyle := lipgloss.NewStyle().Foreground(theme.Border)
	b.WriteString(separatorStyle.Render(strings.Repeat("─", 120)))
	b.WriteString("\n\n")

	// Render content based on active tab
	if m.tabs.ActiveIndex() == 0 {
		b.WriteString(m.viewIntegrationsList())
	} else {
		b.WriteString(m.viewDependenciesList())
	}

	return b.String()
}

func (m *IntegrationsModal) viewIntegrationsList() string {
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

		// Name with selection highlight - prefer DisplayName if available
		displayName := integration.DisplayName
		if displayName == "" {
			displayName = integration.Name
		}
		var name string
		if i == m.selected {
			name = selectedStyle.Render(displayName)
		} else {
			name = normalStyle.Render(displayName)
		}

		// Build line with status info
		line := fmt.Sprintf("  %s %s", indicator, name)

		// Pad name for alignment
		padding := 16 - len(displayName)
		if padding < 2 {
			padding = 2
		}

		// Show status - profiles for api_key type, simple status for others
		var statusStr string
		if integration.Configured && len(integration.Profiles) > 0 {
			statusStr = strings.Join(integration.Profiles, ", ")
		} else if !integration.Configured {
			statusStr = "Not configured"
		}
		if statusStr != "" {
			line += strings.Repeat(" ", padding) + descStyle.Render(statusStr)
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

func (m *IntegrationsModal) viewDependenciesList() string {
	if m.depLoading {
		return lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("Loading dependencies...")
	}

	if m.depError != "" {
		errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
		hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			errorStyle.Render("Error: "+m.depError),
			"",
			hintStyle.Render("[r] Retry"),
		)
	}

	if len(m.dependencies) == 0 {
		return lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("No dependencies configured.")
	}

	var lines []string

	// Table header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.TextSecondary)
	lines = append(lines, headerStyle.Render(
		fmt.Sprintf("  %-12s %-18s %-12s %-12s %s",
			"Tool", "Status", "Installed", "Required", "Actions")))

	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)

	// Table rows
	for i, dep := range m.dependencies {
		status := m.dependencyStatusString(dep)
		actions := m.dependencyActionsString(dep)

		installedVer := dep.InstalledVersion
		if installedVer == "" {
			installedVer = "-"
		}

		row := fmt.Sprintf("  %-12s %-18s %-12s %-12s %s",
			dep.Name,
			status,
			installedVer,
			dep.RequiredVersion,
			actions)

		if i == m.selectedDepIndex {
			lines = append(lines, selectedStyle.Render(row))
		} else {
			lines = append(lines, normalStyle.Render(row))
		}
	}

	// Show installing status if in progress
	if m.depInstalling != "" {
		lines = append(lines, "")
		loadingStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary).Italic(true)
		lines = append(lines, loadingStyle.Render(fmt.Sprintf("  Installing %s...", m.depInstalling)))
	}

	// Hub Version section
	lines = append(lines, "")
	separatorStyle := lipgloss.NewStyle().Foreground(theme.Border)
	lines = append(lines, separatorStyle.Render(strings.Repeat("─", 60)))
	lines = append(lines, "")

	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.TextPrimary)
	lines = append(lines, sectionStyle.Render("Hub Version"))
	lines = append(lines, "")

	if m.hubUpdateLoading {
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.TextSecondary).Render("  Checking for updates..."))
	} else if m.hubUpdateError != "" {
		errorStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary).Italic(true)
		lines = append(lines, errorStyle.Render("  "+m.hubUpdateError))
	} else if m.hubUpdateInfo != nil {
		currentStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
		lines = append(lines, currentStyle.Render(fmt.Sprintf("  Current: %s", m.hubUpdateInfo.CurrentVersion)))

		if m.hubUpdateInfo.UpdateAvailable {
			updateStyle := lipgloss.NewStyle().Foreground(theme.Success)
			lines = append(lines, updateStyle.Render(fmt.Sprintf("  Latest:  %s (update available)", m.hubUpdateInfo.LatestVersion)))
			lines = append(lines, "")

			if m.hubUpdateInfo.ReleaseURL != "" {
				linkStyle := lipgloss.NewStyle().Foreground(theme.Link)
				lines = append(lines, "  Release: "+linkStyle.Render(m.hubUpdateInfo.ReleaseURL))
			}

			if m.isAdmin {
				lines = append(lines, "")
				if m.hubUpdating {
					loadingStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary).Italic(true)
					lines = append(lines, loadingStyle.Render("  Applying update..."))
				} else if m.hubUpdateConfirm {
					warningStyle := lipgloss.NewStyle().Foreground(theme.Warning)
					lines = append(lines, warningStyle.Render("  ⚠️  Server will restart. Press [u] again to confirm."))
				}
			}
		} else {
			upToDateStyle := lipgloss.NewStyle().Foreground(theme.Success)
			lines = append(lines, upToDateStyle.Render("  ✓ Up to date"))
		}
	} else {
		// Not loaded yet
		hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
		if m.isAdmin {
			lines = append(lines, hintStyle.Render("  Press [c] to check for updates"))
		} else {
			lines = append(lines, hintStyle.Render("  Version check available to administrators only"))
		}
	}

	// Hints
	lines = append(lines, "")
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	// Check if any dependency needs action
	hasActionable := false
	for _, dep := range m.dependencies {
		if !dep.Installed || dep.NeedsUpdate {
			hasActionable = true
			break
		}
	}

	// Build hints based on state
	var hints []string
	if m.isAdmin && hasActionable {
		hints = append(hints, "[Enter] Install/Update")
	}
	if m.isAdmin && m.hubUpdateInfo != nil && m.hubUpdateInfo.UpdateAvailable && !m.hubUpdating {
		hints = append(hints, "[u] Update Hub")
	}
	if m.isAdmin && m.hubUpdateInfo == nil && !m.hubUpdateLoading {
		hints = append(hints, "[c] Check Updates")
	}
	hints = append(hints, "[r] Refresh")

	lines = append(lines, hintStyle.Render("  "+strings.Join(hints, "  ")))

	return strings.Join(lines, "\n")
}

func (m *IntegrationsModal) dependencyStatusString(dep client.Dependency) string {
	if !dep.Installed {
		return "✗ Not installed"
	}
	if dep.NeedsUpdate {
		return "⚠️ Outdated"
	}
	return "✓ Up to date"
}

func (m *IntegrationsModal) dependencyActionsString(dep client.Dependency) string {
	if !m.isAdmin {
		if !dep.Installed {
			return "Contact admin"
		}
		return ""
	}

	if !dep.Installed {
		return "[Enter] Install"
	}
	if dep.NeedsUpdate {
		return "[Enter] Update"
	}
	return ""
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
	lines = append(lines, legendStyle.Render("  [Ctrl+S] Save  [Esc] Back"))

	return strings.Join(lines, "\n")
}
