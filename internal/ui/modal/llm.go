package modal

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/client"
	"github.com/pxp/hub-tui/internal/ui/components"
	"github.com/pxp/hub-tui/internal/ui/theme"
)

// LLM modal view modes
type llmView int

const (
	llmViewList llmView = iota
	llmViewEdit // Phase 3
)

// LLMModal displays and manages LLM profiles.
type LLMModal struct {
	client   *client.Client
	profiles *client.LLMProfileList
	names    []string // sorted profile names for consistent ordering
	selected int
	loading  bool
	error    string

	// View state
	view llmView

	// Test state
	testing    bool
	testResult *client.LLMTestResult
	testName   string

	// Operation states
	deleting bool
	setting  bool
	confirm  *components.Confirmation

	// Edit mode
	editName         string               // original name (empty for create)
	editIsNew        bool                 // true if creating, false if editing
	form             *components.Form     // form component
	saving           bool                 // true while saving
	setDefaultOnSave bool                 // set as default after successful save
	integrations     []client.Integration // available integrations for select field
	loadingInt       bool                 // loading integrations
	loadingModels    bool                 // loading models for selected integration
	models           []client.ModelInfo   // available models for selected integration

	// Models pagination
	modelsPageSize   int      // models per page
	modelsCursors    []string // stack of cursors for previous pages (index 0 = page 1 start)
	modelsHasMore    bool     // has next page
	modelsNextCursor string   // cursor for next page
	modelsTotal      int      // total model count
	modelsPage       int      // current page number (1-based)
}

// NewLLMModal creates a new LLM profiles modal.
func NewLLMModal(c *client.Client) *LLMModal {
	return &LLMModal{
		client:  c,
		loading: true,
		view:    llmViewList,
		confirm: components.NewConfirmation(),
	}
}

// --- Message Types ---

// LLMProfilesLoadedMsg is sent when profiles are loaded.
type LLMProfilesLoadedMsg struct {
	Profiles *client.LLMProfileList
	Error    error
}

// LLMProfileTestedMsg is sent when a profile test completes.
type LLMProfileTestedMsg struct {
	Name   string
	Result *client.LLMTestResult
	Error  error
}

// LLMProfileDeletedMsg is sent when a profile is deleted.
type LLMProfileDeletedMsg struct {
	Name  string
	Error error
}

// LLMDefaultSetMsg is sent when the default profile is changed.
type LLMDefaultSetMsg struct {
	Name  string
	Error error
}

// LLMProfileSavedMsg is sent when a profile is created or updated.
type LLMProfileSavedMsg struct {
	Name  string
	IsNew bool
	Error error
}

// LLMIntegrationsLoadedMsg is sent when integrations are loaded for the edit form.
type LLMIntegrationsLoadedMsg struct {
	Integrations []client.Integration
	Error        error
}

// LLMOpenIntegrationsMsg signals the app to open the integrations modal for configuration.
type LLMOpenIntegrationsMsg struct {
	IntegrationName string // Which integration to configure
}

// LLMModelsLoadedMsg is sent when models are loaded for an integration.
type LLMModelsLoadedMsg struct {
	Integration string
	Models      []client.ModelInfo
	Pagination  client.ModelsPagination
	Error       error
}

// --- Commands ---

func (m *LLMModal) loadProfiles() tea.Cmd {
	return func() tea.Msg {
		profiles, err := m.client.ListLLMProfiles()
		return LLMProfilesLoadedMsg{Profiles: profiles, Error: err}
	}
}

func (m *LLMModal) testProfile(name string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.TestLLMProfile(name)
		return LLMProfileTestedMsg{Name: name, Result: result, Error: err}
	}
}

func (m *LLMModal) deleteProfile(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.DeleteLLMProfile(name)
		return LLMProfileDeletedMsg{Name: name, Error: err}
	}
}

func (m *LLMModal) setDefault(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.SetDefaultLLMProfile(name)
		return LLMDefaultSetMsg{Name: name, Error: err}
	}
}

func (m *LLMModal) loadIntegrations() tea.Cmd {
	return func() tea.Msg {
		integrations, err := m.client.ListIntegrations()
		return LLMIntegrationsLoadedMsg{Integrations: integrations, Error: err}
	}
}

func (m *LLMModal) loadModels(integration string, cursor string) tea.Cmd {
	limit := m.modelsPageSize
	if limit == 0 {
		limit = 10
	}
	return func() tea.Msg {
		result, err := m.client.ListIntegrationModels(integration, limit, cursor)
		if err != nil {
			return LLMModelsLoadedMsg{Integration: integration, Error: err}
		}
		return LLMModelsLoadedMsg{
			Integration: integration,
			Models:      result.Models,
			Pagination:  result.Pagination,
			Error:       nil,
		}
	}
}

func (m *LLMModal) saveProfile() tea.Cmd {
	values := m.form.Values()
	name := values["name"]
	isNew := m.editIsNew
	originalName := m.editName

	config := client.LLMProfileConfig{
		Integration: values["integration"],
		Profile:     values["profile"],
		Model:       values["model"],
	}

	// If editing and name changed, include new name for rename
	if !isNew && name != originalName {
		config.Name = name
	}

	return func() tea.Msg {
		var err error
		if isNew {
			err = m.client.CreateLLMProfile(name, config)
		} else {
			err = m.client.UpdateLLMProfile(originalName, config)
		}
		return LLMProfileSavedMsg{Name: name, IsNew: isNew, Error: err}
	}
}

// --- Modal Interface ---

// Init initializes the modal and triggers data fetch.
func (m *LLMModal) Init() tea.Cmd {
	return m.loadProfiles()
}

// Update handles input and messages.
func (m *LLMModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case LLMProfilesLoadedMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			m.profiles = msg.Profiles
			m.sortNames()
			m.error = ""
			// Reset selection if out of bounds
			if m.selected >= len(m.names) {
				m.selected = max(0, len(m.names)-1)
			}
		}
		return m, nil

	case LLMProfileTestedMsg:
		m.testing = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
			m.testResult = nil
		} else {
			m.testResult = msg.Result
			m.testName = msg.Name
			m.error = ""
		}
		return m, nil

	case LLMProfileDeletedMsg:
		m.deleting = false
		m.confirm.Clear()
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			m.error = ""
			// Refresh list
			m.loading = true
			return m, m.loadProfiles()
		}
		return m, nil

	case LLMDefaultSetMsg:
		m.setting = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			// If we were in edit view (came from save flow), return to list and refresh
			if m.view == llmViewEdit {
				m.view = llmViewList
				m.form = nil
				m.error = ""
				m.loading = true
				return m, m.loadProfiles()
			}
			// Otherwise just update local state (from list view [s] key)
			if m.profiles != nil {
				m.profiles.DefaultProfile = msg.Name
			}
			m.error = ""
		}
		return m, nil

	case LLMProfileSavedMsg:
		m.saving = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			// Success - check if we need to set as default
			if m.setDefaultOnSave {
				m.setDefaultOnSave = false
				m.setting = true
				return m, m.setDefault(msg.Name)
			}
			// Return to list and refresh
			m.view = llmViewList
			m.form = nil
			m.error = ""
			m.loading = true
			return m, m.loadProfiles()
		}
		return m, nil

	case LLMIntegrationsLoadedMsg:
		m.loadingInt = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			m.integrations = msg.Integrations
			m.populateIntegrationOptions()
			// Load models for current integration if configured
			integration := m.getSelectedIntegration()
			if integration != "" && !m.form.IsSelectedDisabled("integration") {
				m.loadingModels = true
				m.resetModelsPagination()
				return m, m.loadModels(integration, "")
			}
		}
		return m, nil

	case LLMModelsLoadedMsg:
		m.loadingModels = false
		// Only apply if this is for the currently selected integration
		if msg.Integration == m.getSelectedIntegration() {
			if msg.Error != nil {
				// Don't show error, just leave models empty
				m.models = nil
				m.modelsHasMore = false
				m.modelsTotal = 0
			} else {
				m.modelsHasMore = msg.Pagination.HasMore
				m.modelsNextCursor = msg.Pagination.NextCursor
				m.modelsTotal = msg.Pagination.Total
				m.models = msg.Models
			}
			m.populateModelOptions()
		}
		return m, nil

	case components.ConfirmationExpiredMsg:
		m.confirm.HandleExpired(msg)
		return m, nil

	case tea.KeyMsg:
		switch m.view {
		case llmViewList:
			return m.updateList(msg)
		case llmViewEdit:
			return m.updateEdit(msg)
		}
	}
	return m, nil
}

func (m *LLMModal) updateList(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.confirm.Clear()
		return nil, nil // Close modal

	case "up", "k":
		m.confirm.Clear()
		if m.selected > 0 {
			m.selected--
			m.clearTestResult()
		}

	case "down", "j":
		m.confirm.Clear()
		// +1 for the "+ New Profile" option
		if m.selected < len(m.names) {
			m.selected++
			m.clearTestResult()
		}

	case "t":
		m.confirm.Clear()
		// Test selected profile (not on "+ New Profile")
		if !m.loading && !m.testing && m.selected < len(m.names) {
			name := m.names[m.selected]
			m.testing = true
			m.testResult = nil
			m.error = ""
			return m, m.testProfile(name)
		}

	case "d":
		// Delete selected profile (not on "+ New Profile")
		if !m.loading && !m.deleting && m.selected < len(m.names) {
			name := m.names[m.selected]
			if execute, cmd := m.confirm.Check("delete", name); execute {
				m.deleting = true
				m.error = ""
				return m, m.deleteProfile(name)
			} else if cmd != nil {
				return m, cmd
			}
		}

	case "s":
		m.confirm.Clear()
		// Set as default (not on "+ New Profile")
		if !m.loading && !m.setting && m.selected < len(m.names) {
			name := m.names[m.selected]
			// Don't set if already default
			if m.profiles != nil && m.profiles.DefaultProfile != name {
				m.setting = true
				m.error = ""
				return m, m.setDefault(name)
			}
		}

	case "r":
		m.confirm.Clear()
		// Refresh
		m.loading = true
		m.error = ""
		m.clearTestResult()
		return m, m.loadProfiles()

	case "enter":
		m.confirm.Clear()
		if !m.loading {
			if m.selected < len(m.names) {
				// Edit existing profile
				return m, m.enterEditMode(m.names[m.selected])
			} else {
				// "+ New Profile" option selected
				return m, m.enterCreateMode()
			}
		}
	}
	return m, nil
}

func (m *LLMModal) updateEdit(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel and return to list
		m.view = llmViewList
		m.form = nil
		m.error = ""
		return m, nil

	case "ctrl+s":
		// Save the profile
		return m.doSave()

	case "c":
		// Open integrations modal to configure the selected integration
		// Only handle when integration field is focused and the selection is disabled
		if m.form != nil && m.form.IsFieldFocused("integration") && m.form.IsSelectedDisabled("integration") {
			integrationName := m.form.GetFieldValue("integration")
			return m, func() tea.Msg {
				return LLMOpenIntegrationsMsg{IntegrationName: integrationName}
			}
		}

	case "[", "p":
		// Previous page of models (only when model field is focused)
		if m.form != nil && m.form.IsFieldFocused("model") && m.modelsPage > 1 {
			return m, m.loadPrevModelsPage()
		}

	case "]", "n":
		// Next page of models (only when model field is focused)
		if m.form != nil && m.form.IsFieldFocused("model") && m.modelsHasMore {
			return m, m.loadNextModelsPage()
		}
	}

	// Clear error on navigation keys
	switch msg.String() {
	case "j", "k", "up", "down":
		m.error = ""
	}

	// Track integration selection before update
	prevIntegration := m.getSelectedIntegration()

	// Forward to form
	if m.form != nil {
		m.form.Update(msg)

		// Check if integration selection changed
		newIntegration := m.getSelectedIntegration()
		if prevIntegration != newIntegration {
			m.updateProfileOptions()
			// Clear models and load new ones if integration is configured
			m.models = nil
			m.resetModelsPagination()
			m.populateModelOptions()
			if newIntegration != "" && !m.form.IsSelectedDisabled("integration") {
				m.loadingModels = true
				return m, m.loadModels(newIntegration, "")
			}
		}
	}
	return m, nil
}

// doSave validates and saves the profile.
func (m *LLMModal) doSave() (Modal, tea.Cmd) {
	if m.saving || m.form == nil {
		return m, nil
	}

	// Block submit if integration is not configured
	if m.form.IsSelectedDisabled("integration") {
		m.error = "Please configure the integration first (press 'c')"
		return m, nil
	}

	// Validate required fields
	values := m.form.Values()
	if values["name"] == "" {
		m.error = "Name is required"
		return m, nil
	}
	if values["integration"] == "" {
		m.error = "Integration is required"
		return m, nil
	}
	if values["model"] == "" {
		m.error = "Model is required"
		return m, nil
	}
	m.saving = true
	m.setDefaultOnSave = m.form.GetFieldChecked("default")
	m.error = ""
	return m, m.saveProfile()
}

func (m *LLMModal) enterEditMode(profileName string) tea.Cmd {
	profile := m.profiles.Profiles[profileName]
	isDefault := m.profiles.DefaultProfile == profileName

	fields := []components.FormField{
		{Label: "Name", Key: "name", Value: profileName},
		{Label: "Integration", Key: "integration", Value: profile.Integration, Type: components.FieldSelect},
		{Label: "Integration Profile", Key: "profile", Value: profile.Profile, Type: components.FieldSelect},
		{Label: "Model", Key: "model", Value: profile.Model, Type: components.FieldSelect},
		{Label: "Set as default", Key: "default", Type: components.FieldCheckbox, Checked: isDefault},
	}

	m.form = components.NewForm("Edit Profile", fields)
	m.editName = profileName
	m.editIsNew = false
	m.view = llmViewEdit
	m.error = ""
	m.loadingInt = true
	m.models = nil
	m.resetModelsPagination()

	// If we already have integrations cached, populate immediately and load models
	if len(m.integrations) > 0 {
		m.loadingInt = false
		m.populateIntegrationOptions()
		// Load models for current integration if configured
		if profile.Integration != "" && !m.form.IsSelectedDisabled("integration") {
			m.loadingModels = true
			return m.loadModels(profile.Integration, "")
		}
		return nil
	}
	return m.loadIntegrations()
}

func (m *LLMModal) enterCreateMode() tea.Cmd {
	fields := []components.FormField{
		{Label: "Name", Key: "name", Value: ""},
		{Label: "Integration", Key: "integration", Value: "", Type: components.FieldSelect},
		{Label: "Integration Profile", Key: "profile", Value: "", Type: components.FieldSelect},
		{Label: "Model", Key: "model", Value: "", Type: components.FieldSelect},
		{Label: "Set as default", Key: "default", Type: components.FieldCheckbox, Checked: false},
	}

	m.form = components.NewForm("New Profile", fields)
	m.editName = ""
	m.editIsNew = true
	m.view = llmViewEdit
	m.error = ""
	m.loadingInt = true
	m.models = nil
	m.resetModelsPagination()

	// If we already have integrations cached, populate immediately
	if len(m.integrations) > 0 {
		m.loadingInt = false
		m.populateIntegrationOptions()
		// Load models for first configured integration if any
		integration := m.getSelectedIntegration()
		if integration != "" && !m.form.IsSelectedDisabled("integration") {
			m.loadingModels = true
			return m.loadModels(integration, "")
		}
		return nil
	}
	return m.loadIntegrations()
}

func (m *LLMModal) clearTestResult() {
	m.testResult = nil
	m.testName = ""
}

// resetModelsPagination resets pagination state for models.
func (m *LLMModal) resetModelsPagination() {
	m.modelsPageSize = 10
	m.modelsCursors = []string{""}  // First page cursor is empty
	m.modelsHasMore = false
	m.modelsNextCursor = ""
	m.modelsTotal = 0
	m.modelsPage = 1
}

// loadNextModelsPage loads the next page of models.
func (m *LLMModal) loadNextModelsPage() tea.Cmd {
	if !m.modelsHasMore || m.loadingModels {
		return nil
	}
	integration := m.getSelectedIntegration()
	if integration == "" {
		return nil
	}
	// Save current cursor for going back
	if m.modelsPage == len(m.modelsCursors) {
		m.modelsCursors = append(m.modelsCursors, m.modelsNextCursor)
	}
	m.modelsPage++
	m.loadingModels = true
	return m.loadModels(integration, m.modelsNextCursor)
}

// loadPrevModelsPage loads the previous page of models.
func (m *LLMModal) loadPrevModelsPage() tea.Cmd {
	if m.modelsPage <= 1 || m.loadingModels {
		return nil
	}
	integration := m.getSelectedIntegration()
	if integration == "" {
		return nil
	}
	m.modelsPage--
	cursor := m.modelsCursors[m.modelsPage-1]
	m.loadingModels = true
	return m.loadModels(integration, cursor)
}

// populateIntegrationOptions populates the integration select field with available integrations.
func (m *LLMModal) populateIntegrationOptions() {
	if m.form == nil {
		return
	}

	// Build list of all LLM integrations, marking unconfigured ones as disabled
	var integrationNames []string
	disabledOptions := make(map[string]bool)

	for _, integration := range m.integrations {
		if integration.Type == "llm" {
			integrationNames = append(integrationNames, integration.Name)
			if !integration.Configured {
				disabledOptions[integration.Name] = true
			}
		}
	}
	sort.Strings(integrationNames)

	// Get current value to preserve selection
	currentIntegration := m.form.GetFieldValue("integration")
	m.form.SetFieldOptions("integration", integrationNames, currentIntegration)
	m.form.SetFieldDisabledOptions("integration", disabledOptions)

	// Also update the profile options based on current integration
	m.updateProfileOptions()
}

// updateProfileOptions updates the profile select field based on the selected integration.
func (m *LLMModal) updateProfileOptions() {
	if m.form == nil {
		return
	}

	selectedIntegration := m.form.GetFieldValue("integration")
	var profileOptions []string

	// Find the selected integration and get its profiles
	for _, integration := range m.integrations {
		if integration.Name == selectedIntegration {
			// Always include "default" as first option
			profileOptions = append(profileOptions, "default")
			// Add configured profiles
			for _, p := range integration.Profiles {
				if p != "default" {
					profileOptions = append(profileOptions, p)
				}
			}
			break
		}
	}

	currentProfile := m.form.GetFieldValue("profile")
	m.form.SetFieldOptions("profile", profileOptions, currentProfile)
}

// getSelectedIntegration returns the currently selected integration name.
func (m *LLMModal) getSelectedIntegration() string {
	if m.form == nil {
		return ""
	}
	return m.form.GetFieldValue("integration")
}

// populateModelOptions updates the model select field with available models.
func (m *LLMModal) populateModelOptions() {
	if m.form == nil {
		return
	}

	// Extract model IDs for the form options
	modelIDs := make([]string, len(m.models))
	for i, model := range m.models {
		modelIDs[i] = model.ID
	}

	currentModel := m.form.GetFieldValue("model")
	m.form.SetFieldOptions("model", modelIDs, currentModel)
}

// getSelectedModelDescription returns the description of the currently selected model.
func (m *LLMModal) getSelectedModelDescription() string {
	if m.form == nil {
		return ""
	}
	selectedID := m.form.GetFieldValue("model")
	for _, model := range m.models {
		if model.ID == selectedID {
			return model.Description
		}
	}
	return ""
}

func (m *LLMModal) sortNames() {
	if m.profiles == nil || m.profiles.Profiles == nil {
		m.names = nil
		return
	}

	m.names = make([]string, 0, len(m.profiles.Profiles))
	for name := range m.profiles.Profiles {
		m.names = append(m.names, name)
	}
	sort.Strings(m.names)
}

// Title returns the modal title.
func (m *LLMModal) Title() string {
	switch m.view {
	case llmViewEdit:
		if m.editIsNew {
			return "New LLM Profile"
		}
		return "Edit LLM Profile"
	default:
		return "LLM Profiles"
	}
}

// View renders the modal content.
func (m *LLMModal) View() string {
	// Edit view
	if m.view == llmViewEdit {
		return m.viewEdit()
	}

	// List view
	if m.loading {
		return lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("Loading profiles...")
	}

	if m.error != "" && len(m.names) == 0 {
		// Show error with retry hint only if we have no data
		errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
		hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			errorStyle.Render("Error: "+m.error),
			"",
			hintStyle.Render("[r] Retry"),
		)
	}

	// Empty state is handled by showing only "+ New Profile" option below

	var lines []string

	// Styles
	defaultStyle := lipgloss.NewStyle().Foreground(theme.Warning)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	modelStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	providerStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	// Column widths for alignment
	maxNameLen := 0
	maxModelLen := 0
	for _, name := range m.names {
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}
		profile := m.profiles.Profiles[name]
		if len(profile.Model) > maxModelLen {
			maxModelLen = len(profile.Model)
		}
	}
	// Add space for star indicator
	maxNameLen += 2

	// Render each profile
	for i, name := range m.names {
		profile := m.profiles.Profiles[name]
		isDefault := m.profiles.DefaultProfile == name
		isSelected := i == m.selected

		// Name column with default indicator
		var nameStr string
		if isDefault {
			nameStr = "★ " + name
		} else {
			nameStr = "  " + name
		}

		// Pad name for alignment
		namePadded := nameStr + strings.Repeat(" ", maxNameLen-len(nameStr)+2)

		// Model column
		modelPadded := profile.Model + strings.Repeat(" ", maxModelLen-len(profile.Model)+2)

		// Provider column: integration (profile)
		providerStr := profile.Integration
		if profile.Profile != "" {
			providerStr += " (" + profile.Profile + ")"
		} else {
			providerStr += " (default)"
		}

		// Apply styles
		var line string
		if isSelected {
			if isDefault {
				line = "  " + defaultStyle.Render(namePadded) + modelStyle.Render(modelPadded) + providerStyle.Render(providerStr)
			} else {
				line = "  " + selectedStyle.Render(namePadded) + modelStyle.Render(modelPadded) + providerStyle.Render(providerStr)
			}
		} else {
			if isDefault {
				line = "  " + defaultStyle.Render(namePadded) + modelStyle.Render(modelPadded) + providerStyle.Render(providerStr)
			} else {
				line = "  " + normalStyle.Render(namePadded) + modelStyle.Render(modelPadded) + providerStyle.Render(providerStr)
			}
		}

		// Add selection indicator
		if isSelected {
			line = "›" + line[1:]
		}

		lines = append(lines, line)
	}

	// Add "+ New Profile" option
	newProfileStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	isNewSelected := m.selected == len(m.names)
	if isNewSelected {
		lines = append(lines, "› "+selectedStyle.Render("+ New Profile"))
	} else {
		lines = append(lines, "  "+newProfileStyle.Render("+ New Profile"))
	}

	// Show test result if any
	if m.testResult != nil {
		lines = append(lines, "")
		if m.testResult.Success {
			successStyle := lipgloss.NewStyle().Foreground(theme.Success)
			lines = append(lines, "  "+successStyle.Render(fmt.Sprintf("✓ Connected (%dms)", m.testResult.LatencyMs)))
		} else {
			errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
			errMsg := m.testResult.Error
			if errMsg == "" {
				errMsg = "Connection failed"
			}
			lines = append(lines, "  "+errorStyle.Render("✗ "+errMsg))
		}
	}

	// Show testing indicator
	if m.testing {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("  Testing..."))
	}

	// Show error inline if we have data but an operation failed
	if m.error != "" && len(m.names) > 0 {
		lines = append(lines, "")
		errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
		lines = append(lines, "  "+errorStyle.Render("Error: "+m.error))
	}

	// Hints
	lines = append(lines, "")
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	warningHintStyle := lipgloss.NewStyle().Foreground(theme.Warning)

	if m.confirm.IsPending("delete", "") {
		lines = append(lines, warningHintStyle.Render("  Press d again to delete"))
	} else {
		lines = append(lines, hintStyle.Render("  [t] Test  [s] Set default  [d] Delete  [r] Refresh"))
	}

	return strings.Join(lines, "\n")
}

// viewEdit renders the edit/create form.
func (m *LLMModal) viewEdit() string {
	var lines []string

	// Show loading indicator for integrations
	if m.loadingInt {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("  Loading integrations..."))
		return strings.Join(lines, "\n")
	}

	// Show form
	if m.form != nil {
		lines = append(lines, m.form.View())
	}

	// Check if selected integration is unconfigured AND the integration field is focused
	integrationFocused := m.form != nil && m.form.IsFieldFocused("integration")
	integrationDisabled := m.form != nil && m.form.IsSelectedDisabled("integration")
	showConfigureHint := integrationFocused && integrationDisabled

	// Show configure hint if integration not configured and field is focused
	if showConfigureHint {
		lines = append(lines, "")
		warnStyle := lipgloss.NewStyle().Foreground(theme.Warning)
		lines = append(lines, "  "+warnStyle.Render("This integration needs to be configured first."))
	}

	// Show loading models indicator
	if m.loadingModels {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("  Loading models..."))
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

	// Check if model field is focused for pagination hints
	modelFocused := m.form != nil && m.form.IsFieldFocused("model")

	// Show pagination info when model field is focused
	if modelFocused && m.modelsTotal > 0 && !m.loadingModels {
		lines = append(lines, "")
		pageInfo := fmt.Sprintf("  Page %d", m.modelsPage)
		if m.modelsTotal > 0 {
			totalPages := (m.modelsTotal + m.modelsPageSize - 1) / m.modelsPageSize
			pageInfo = fmt.Sprintf("  Page %d of %d (%d models)", m.modelsPage, totalPages, m.modelsTotal)
		}
		lines = append(lines, lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render(pageInfo))
	}

	// Show model description when model field is focused
	if modelFocused && !m.loadingModels {
		if desc := m.getSelectedModelDescription(); desc != "" {
			lines = append(lines, "")
			lines = append(lines, lipgloss.NewStyle().
				Foreground(theme.TextSecondary).
				Italic(true).
				Render("  "+desc))
		}
	}

	// Hints
	lines = append(lines, "")
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	if showConfigureHint {
		lines = append(lines, hintStyle.Render("  [c] Configure  [Ctrl+S] Save  [Esc] Cancel"))
	} else if modelFocused && (m.modelsHasMore || m.modelsPage > 1) {
		// Show pagination keys when on model field
		var pageHints []string
		if m.modelsPage > 1 {
			pageHints = append(pageHints, "[p] Prev")
		}
		if m.modelsHasMore {
			pageHints = append(pageHints, "[n] Next")
		}
		lines = append(lines, hintStyle.Render("  "+strings.Join(pageHints, "  ")+"  [Ctrl+S] Save  [Esc] Cancel"))
	} else {
		lines = append(lines, hintStyle.Render("  [Ctrl+S] Save  [Esc] Cancel"))
	}

	return strings.Join(lines, "\n")
}
