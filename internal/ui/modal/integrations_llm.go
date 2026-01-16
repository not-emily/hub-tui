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

// LLM list item types for navigation
type llmItemType int

const (
	llmItemProviderAccount llmItemType = iota
	llmItemProfile
	llmItemNewProvider
	llmItemNewProfile
)

// llmListItem represents a selectable item in the LLM config view.
type llmListItem struct {
	Type            llmItemType
	Provider        string            // for provider accounts
	ProviderDisplay string            // display name for provider
	Account         string            // for provider accounts
	Profile         *client.LLMProfile // for profiles
}

// LLMDataLoadedMsg is sent when LLM providers and profiles are loaded.
type LLMDataLoadedMsg struct {
	Providers []client.ProviderAccount
	Profiles  []client.LLMProfile
	Error     error
}

// LLMAvailableProvidersMsg is sent when available providers are loaded for the form.
type LLMAvailableProvidersMsg struct {
	Providers []client.AvailableProvider
	Err       error
}

// LLMProviderFieldsMsg is sent when provider field requirements are loaded.
type LLMProviderFieldsMsg struct {
	Provider string
	Fields   []client.ProviderFieldInfo
	Err      error
}

// LLMProviderSavedMsg is sent when a provider is added.
type LLMProviderSavedMsg struct {
	Err error
}

// LLMProviderDeletedMsg is sent when a provider is deleted.
type LLMProviderDeletedMsg struct {
	Err error
}

// LLMErrorMsg is sent when an LLM operation fails.
type LLMErrorMsg struct {
	Err error
}

// LLMModelsLoadedMsg is sent when models are loaded for the profile form.
type LLMModelsLoadedMsg struct {
	Models     []client.ModelInfo
	HasMore    bool
	NextCursor string
	Err        error
}

// LLMProfileSavedMsg is sent when a profile is saved.
type LLMProfileSavedMsg struct {
	Err error
}

// LLMProfileDeletedMsg is sent when a profile is deleted.
type LLMProfileDeletedMsg struct {
	Err error
}

// LLMProfileTestedMsg is sent when a profile connectivity test completes.
type LLMProfileTestedMsg struct {
	Result *client.LLMTestResult
	Err    error
}

// LLMProfileDefaultSetMsg is sent when a profile is set as default.
type LLMProfileDefaultSetMsg struct {
	Err error
}

// enterLLMConfig enters the LLM configuration view for the given integration.
func (m *IntegrationsModal) enterLLMConfig(integration client.Integration) (Modal, tea.Cmd) {
	m.view = viewConfigLLM
	m.llmIntegration = integration
	m.llmLoading = true
	m.llmError = ""
	m.llmSelected = 0
	return m, m.loadLLMData()
}

// loadLLMData loads providers and profiles for the current LLM integration.
func (m *IntegrationsModal) loadLLMData() tea.Cmd {
	integration := m.llmIntegration.Name
	return func() tea.Msg {
		providers, err := m.client.ListLLMProviders(integration)
		if err != nil {
			return LLMDataLoadedMsg{Error: err}
		}

		profileList, err := m.client.ListLLMProfiles(integration)
		if err != nil {
			return LLMDataLoadedMsg{Error: err}
		}

		return LLMDataLoadedMsg{
			Providers: providers,
			Profiles:  profileList.Profiles,
		}
	}
}

// handleLLMDataLoaded processes the loaded LLM data.
func (m *IntegrationsModal) handleLLMDataLoaded(msg LLMDataLoadedMsg) (Modal, tea.Cmd) {
	m.llmLoading = false
	if msg.Error != nil {
		m.llmError = msg.Error.Error()
		return m, nil
	}

	m.llmProviders = msg.Providers
	m.llmProfiles = msg.Profiles
	m.llmError = ""
	m.buildLLMItems()

	// Reset selection if out of bounds
	if m.llmSelected >= len(m.llmItems) {
		m.llmSelected = max(0, len(m.llmItems)-1)
	}

	return m, nil
}

// buildLLMItems creates a flattened list for navigation from providers and profiles.
// Profiles are listed first (more frequently modified), then providers.
func (m *IntegrationsModal) buildLLMItems() {
	m.llmItems = nil

	// Add profiles first (more commonly modified)
	for i := range m.llmProfiles {
		m.llmItems = append(m.llmItems, llmListItem{
			Type:    llmItemProfile,
			Profile: &m.llmProfiles[i],
		})
	}

	// Add "+ New Profile" option
	m.llmItems = append(m.llmItems, llmListItem{
		Type: llmItemNewProfile,
	})

	// Add provider accounts
	for _, p := range m.llmProviders {
		for _, acct := range p.Accounts {
			m.llmItems = append(m.llmItems, llmListItem{
				Type:            llmItemProviderAccount,
				Provider:        p.Provider,
				ProviderDisplay: p.DisplayName,
				Account:         acct,
			})
		}
	}

	// Add "+ New Provider" option
	m.llmItems = append(m.llmItems, llmListItem{
		Type: llmItemNewProvider,
	})
}

// updateLLM handles input for LLM config views.
func (m *IntegrationsModal) updateLLM(msg tea.KeyMsg) (Modal, tea.Cmd) {
	// Route to sub-view handlers
	if m.view == viewLLMProviderForm {
		return m.updateLLMProviderForm(msg)
	}
	if m.view == viewLLMProfileForm {
		return m.updateLLMProfileForm(msg)
	}

	// Clear error on any key
	if m.llmError != "" {
		m.llmError = ""
	}

	// Clear confirmation and test result on navigation
	if msg.String() == "j" || msg.String() == "k" || msg.String() == "up" || msg.String() == "down" {
		m.llmConfirm.Clear()
		m.llmTestResult = nil
	}

	switch msg.String() {
	case "esc":
		m.view = viewList
		m.llmError = ""
		m.llmConfirm.Clear()
		return m, nil

	case "j", "down":
		if m.llmSelected < len(m.llmItems)-1 {
			m.llmSelected++
		}

	case "k", "up":
		if m.llmSelected > 0 {
			m.llmSelected--
		}

	case "r":
		m.llmLoading = true
		m.llmError = ""
		m.llmConfirm.Clear()
		return m, m.loadLLMData()

	case "enter":
		if m.llmSelected >= 0 && m.llmSelected < len(m.llmItems) {
			item := m.llmItems[m.llmSelected]
			switch item.Type {
			case llmItemNewProvider:
				m.llmLoading = true
				return m, m.loadAvailableProviders()
			case llmItemNewProfile:
				m.llmEditingProfile = nil
				return m.enterLLMProfileForm()
			case llmItemProfile:
				m.llmEditingProfile = item.Profile
				return m.enterLLMProfileForm()
			}
		}

	case "d":
		if m.llmSelected >= 0 && m.llmSelected < len(m.llmItems) {
			item := m.llmItems[m.llmSelected]
			if item.Type == llmItemProviderAccount {
				key := "provider:" + item.Provider + "/" + item.Account
				if execute, cmd := m.llmConfirm.Check(key, item.Account); execute {
					return m, m.deleteProvider(item.Provider, item.Account)
				} else if cmd != nil {
					return m, cmd
				}
			} else if item.Type == llmItemProfile {
				key := "profile:" + item.Profile.Name
				if execute, cmd := m.llmConfirm.Check(key, item.Profile.Name); execute {
					return m, m.deleteProfile(item.Profile.Name)
				} else if cmd != nil {
					return m, cmd
				}
			}
		}

	case "t":
		// Test profile connectivity
		if m.llmSelected >= 0 && m.llmSelected < len(m.llmItems) {
			item := m.llmItems[m.llmSelected]
			if item.Type == llmItemProfile {
				m.llmTesting = true
				m.llmTestResult = nil
				return m, m.testProfile(item.Profile.Name)
			}
		}

	case "s":
		// Set as default profile
		if m.llmSelected >= 0 && m.llmSelected < len(m.llmItems) {
			item := m.llmItems[m.llmSelected]
			if item.Type == llmItemProfile && !item.Profile.IsDefault {
				return m, m.setDefaultProfile(item.Profile.Name)
			}
		}
	}

	return m, nil
}

// updateLLMProviderForm handles input for the provider form.
func (m *IntegrationsModal) updateLLMProviderForm(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.view = viewConfigLLM
		m.llmProviderForm = nil
		m.llmProviderFields = nil
		m.llmError = ""
		return m, nil

	case "ctrl+s":
		if !m.llmSavingProvider && m.llmProviderForm != nil {
			// Validate before saving
			if err := m.validateProviderForm(); err != nil {
				m.llmError = err.Error()
				return m, nil
			}
			m.llmSavingProvider = true
			return m, m.saveProvider()
		}
		return m, nil
	}

	// Track provider before form update
	prevProvider := ""
	if m.llmProviderForm != nil {
		prevProvider = m.llmProviderForm.GetFieldValue("provider")
	}

	// Forward to form
	if m.llmProviderForm != nil {
		m.llmProviderForm.Update(msg)
	}

	// Check if provider changed
	newProvider := m.llmProviderForm.GetFieldValue("provider")
	if newProvider != prevProvider && newProvider != "" {
		// Map display name to provider name
		providerName := ""
		for _, p := range m.llmAvailableProviders {
			if p.DisplayName == newProvider {
				providerName = p.Name
				break
			}
		}
		if providerName != "" {
			m.llmError = "" // Clear any previous error
			return m, m.loadProviderFields(providerName)
		}
	}

	return m, nil
}

// loadAvailableProviders fetches the list of available providers for the form.
func (m *IntegrationsModal) loadAvailableProviders() tea.Cmd {
	integration := m.llmIntegration.Name
	return func() tea.Msg {
		providers, err := m.client.ListAvailableLLMProviders(integration)
		if err != nil {
			return LLMAvailableProvidersMsg{Err: err}
		}
		return LLMAvailableProvidersMsg{Providers: providers}
	}
}

// handleLLMAvailableProviders builds the provider form when available providers are loaded.
func (m *IntegrationsModal) handleLLMAvailableProviders(msg LLMAvailableProvidersMsg) (Modal, tea.Cmd) {
	m.llmLoading = false
	if msg.Err != nil {
		m.llmError = msg.Err.Error()
		return m, nil
	}

	m.llmAvailableProviders = msg.Providers
	m.view = viewLLMProviderForm

	// Build provider options from available providers
	providerOptions := make([]string, len(m.llmAvailableProviders))
	for i, p := range m.llmAvailableProviders {
		providerOptions[i] = p.DisplayName
	}

	// Build initial form with just provider and account (fields loaded after provider selection)
	m.llmProviderForm = components.NewForm("Add Provider Account", []components.FormField{
		{
			Label:   "Provider",
			Key:     "provider",
			Type:    components.FieldSelect,
			Options: providerOptions,
		},
		{
			Label: "Account Name",
			Key:   "account",
			Type:  components.FieldText,
			Value: "default",
		},
	})

	// Clear any previous field requirements
	m.llmProviderFields = nil

	// Fetch fields for first provider
	if len(m.llmAvailableProviders) > 0 {
		return m, m.loadProviderFields(m.llmAvailableProviders[0].Name)
	}

	return m, nil
}

// loadProviderFields fetches field requirements for the selected provider.
func (m *IntegrationsModal) loadProviderFields(providerName string) tea.Cmd {
	m.llmLoadingFields = true
	integration := m.llmIntegration.Name
	return func() tea.Msg {
		fields, err := m.client.GetLLMProviderFields(integration, providerName)
		if err != nil {
			return LLMProviderFieldsMsg{Provider: providerName, Err: err}
		}
		return LLMProviderFieldsMsg{Provider: providerName, Fields: fields}
	}
}

// handleLLMProviderFields processes the loaded provider field requirements.
func (m *IntegrationsModal) handleLLMProviderFields(msg LLMProviderFieldsMsg) (Modal, tea.Cmd) {
	m.llmLoadingFields = false
	if msg.Err != nil {
		m.llmError = msg.Err.Error()
		return m, nil
	}

	m.llmProviderFields = msg.Fields
	m.rebuildProviderForm()
	return m, nil
}

// rebuildProviderForm rebuilds the provider form with dynamic fields.
func (m *IntegrationsModal) rebuildProviderForm() {
	// Get current values to preserve
	currentAccount := "default"
	currentProvider := ""
	if m.llmProviderForm != nil {
		currentAccount = m.llmProviderForm.GetFieldValue("account")
		currentProvider = m.llmProviderForm.GetFieldValue("provider")
	}

	// Build provider options
	providerOptions := make([]string, len(m.llmAvailableProviders))
	for i, p := range m.llmAvailableProviders {
		providerOptions[i] = p.DisplayName
	}

	// Start with static fields
	fields := []components.FormField{
		{
			Label:   "Provider",
			Key:     "provider",
			Type:    components.FieldSelect,
			Options: providerOptions,
			Value:   currentProvider,
		},
		{
			Label: "Account Name",
			Key:   "account",
			Type:  components.FieldText,
			Value: currentAccount,
		},
	}

	// Add dynamic fields from provider requirements
	for _, f := range m.llmProviderFields {
		field := components.FormField{
			Label:    f.Label,
			Key:      f.Key,
			Type:     components.FieldText,
			Value:    f.Default,
			Password: f.Secret,
			Required: f.Required,
		}
		fields = append(fields, field)
	}

	m.llmProviderForm = components.NewForm("Add Provider Account", fields)
}

// validateProviderForm validates the provider form before saving.
func (m *IntegrationsModal) validateProviderForm() error {
	values := m.llmProviderForm.Values()

	// Check account name
	if strings.TrimSpace(values["account"]) == "" {
		return fmt.Errorf("account name is required")
	}

	// Check required dynamic fields
	for _, f := range m.llmProviderFields {
		if f.Required {
			val := strings.TrimSpace(values[f.Key])
			if val == "" {
				return fmt.Errorf("%s is required", f.Label)
			}
		}
	}

	return nil
}

// saveProvider saves the provider from the form.
func (m *IntegrationsModal) saveProvider() tea.Cmd {
	values := m.llmProviderForm.Values()

	// Map display name back to provider name
	providerDisplayName := values["provider"]
	var providerName string
	for _, p := range m.llmAvailableProviders {
		if p.DisplayName == providerDisplayName {
			providerName = p.Name
			break
		}
	}

	// Build fields map from dynamic fields (only include non-empty values)
	fields := make(map[string]string)
	for _, f := range m.llmProviderFields {
		if val, ok := values[f.Key]; ok && val != "" {
			fields[f.Key] = val
		}
	}

	integration := m.llmIntegration.Name
	req := client.AddProviderRequest{
		Provider: providerName,
		Account:  values["account"],
		Fields:   fields,
	}

	return func() tea.Msg {
		err := m.client.AddLLMProvider(integration, req)
		if err != nil {
			return LLMProviderSavedMsg{Err: err}
		}
		return LLMProviderSavedMsg{}
	}
}

// handleLLMProviderSaved processes the result of saving a provider.
func (m *IntegrationsModal) handleLLMProviderSaved(msg LLMProviderSavedMsg) (Modal, tea.Cmd) {
	m.llmSavingProvider = false
	if msg.Err != nil {
		m.llmError = msg.Err.Error()
		return m, nil
	}

	// Success - return to config view and refresh
	m.view = viewConfigLLM
	m.llmProviderForm = nil
	m.llmLoading = true
	return m, m.loadLLMData()
}

// deleteProvider deletes a provider account.
func (m *IntegrationsModal) deleteProvider(provider, account string) tea.Cmd {
	integration := m.llmIntegration.Name
	return func() tea.Msg {
		err := m.client.DeleteLLMProvider(integration, provider, account)
		if err != nil {
			return LLMProviderDeletedMsg{Err: err}
		}
		return LLMProviderDeletedMsg{}
	}
}

// handleLLMProviderDeleted processes the result of deleting a provider.
func (m *IntegrationsModal) handleLLMProviderDeleted(msg LLMProviderDeletedMsg) (Modal, tea.Cmd) {
	if msg.Err != nil {
		m.llmError = msg.Err.Error()
		return m, nil
	}

	// Success - refresh will remove empty provider headers
	return m, m.loadLLMData()
}

// --- Profile Form ---

const modelsPageSize = 15

// enterLLMProfileForm sets up and enters the profile form.
func (m *IntegrationsModal) enterLLMProfileForm() (Modal, tea.Cmd) {
	m.view = viewLLMProfileForm
	m.llmError = ""

	// Build provider options from configured providers (only those with accounts)
	var providerOptions []string
	for _, p := range m.llmProviders {
		if len(p.Accounts) > 0 {
			providerOptions = append(providerOptions, p.DisplayName)
		}
	}

	// Initial values
	nameVal := ""
	providerVal := ""
	accountVal := ""
	modelVal := ""
	isDefault := false

	if m.llmEditingProfile != nil {
		nameVal = m.llmEditingProfile.Name
		providerVal = m.getProviderDisplayName(m.llmEditingProfile.Provider)
		accountVal = m.llmEditingProfile.Account
		modelVal = m.llmEditingProfile.Model
		isDefault = m.llmEditingProfile.IsDefault
	} else if len(providerOptions) > 0 {
		providerVal = providerOptions[0]
	}

	m.llmProfileForm = components.NewForm("LLM Profile", []components.FormField{
		{
			Label: "Name",
			Key:   "name",
			Type:  components.FieldText,
			Value: nameVal,
		},
		{
			Label:   "Provider",
			Key:     "provider",
			Type:    components.FieldSelect,
			Options: providerOptions,
			Value:   providerVal,
		},
		{
			Label:   "Account",
			Key:     "account",
			Type:    components.FieldSelect,
			Options: []string{}, // populated by cascade
			Value:   accountVal,
		},
		{
			Label:   "Model",
			Key:     "model",
			Type:    components.FieldSelect,
			Options: []string{}, // populated by cascade
			Value:   modelVal,
		},
		{
			Label:   "Set as default",
			Key:     "is_default",
			Type:    components.FieldCheckbox,
			Checked: isDefault,
		},
	})

	// Reset model pagination state
	m.llmModels = nil
	m.llmModelsCursor = ""
	m.llmModelsCursorStack = nil
	m.llmModelsHasMore = false
	m.llmModelsPage = 1

	// Trigger initial cascade to populate account and model options
	return m, m.cascadeFromProvider()
}

// getProviderDisplayName returns the display name for a provider name.
func (m *IntegrationsModal) getProviderDisplayName(providerName string) string {
	for _, p := range m.llmProviders {
		if p.Provider == providerName {
			return p.DisplayName
		}
	}
	return providerName
}

// getProviderName returns the provider name for a display name.
func (m *IntegrationsModal) getProviderName(displayName string) string {
	for _, p := range m.llmProviders {
		if p.DisplayName == displayName {
			return p.Provider
		}
	}
	return displayName
}

// cascadeFromProvider updates account options when provider changes.
func (m *IntegrationsModal) cascadeFromProvider() tea.Cmd {
	providerDisplayName := m.llmProfileForm.GetFieldValue("provider")
	providerName := m.getProviderName(providerDisplayName)

	// Find accounts for this provider
	var accounts []string
	for _, p := range m.llmProviders {
		if p.Provider == providerName {
			accounts = p.Accounts
			break
		}
	}

	// Update account dropdown
	currentAccount := m.llmProfileForm.GetFieldValue("account")
	m.llmProfileForm.SetFieldOptions("account", accounts, currentAccount)

	// Reset models and trigger model load
	m.llmModels = nil
	m.llmModelsCursor = ""
	m.llmModelsCursorStack = nil
	m.llmModelsPage = 1
	return m.loadModels("")
}

// cascadeFromAccount reloads models when account changes.
func (m *IntegrationsModal) cascadeFromAccount() tea.Cmd {
	m.llmModels = nil
	m.llmModelsCursor = ""
	m.llmModelsCursorStack = nil
	m.llmModelsPage = 1
	return m.loadModels("")
}

// loadModels fetches models for the current provider with pagination.
func (m *IntegrationsModal) loadModels(cursor string) tea.Cmd {
	m.llmLoadingModels = true
	providerDisplayName := m.llmProfileForm.GetFieldValue("provider")
	providerName := m.getProviderName(providerDisplayName)
	integration := m.llmIntegration.Name

	return func() tea.Msg {
		result, err := m.client.ListLLMModels(integration, providerName, modelsPageSize, cursor)
		if err != nil {
			return LLMModelsLoadedMsg{Err: err}
		}
		return LLMModelsLoadedMsg{
			Models:     result.Models,
			HasMore:    result.Pagination.HasMore,
			NextCursor: result.Pagination.NextCursor,
		}
	}
}

// handleLLMModelsLoaded processes the loaded models.
func (m *IntegrationsModal) handleLLMModelsLoaded(msg LLMModelsLoadedMsg) (Modal, tea.Cmd) {
	m.llmLoadingModels = false
	if msg.Err != nil {
		m.llmError = msg.Err.Error()
		return m, nil
	}

	m.llmModels = msg.Models
	m.llmModelsHasMore = msg.HasMore
	m.llmModelsCursor = msg.NextCursor

	// Update model options
	modelOptions := make([]string, len(m.llmModels))
	for i, model := range m.llmModels {
		modelOptions[i] = model.ID
	}

	// Try to preserve current selection, or use editing profile's model
	currentModel := m.llmProfileForm.GetFieldValue("model")
	if currentModel == "" && m.llmEditingProfile != nil {
		currentModel = m.llmEditingProfile.Model
	}
	m.llmProfileForm.SetFieldOptions("model", modelOptions, currentModel)

	return m, nil
}

// updateLLMProfileForm handles input for the profile form.
func (m *IntegrationsModal) updateLLMProfileForm(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.view = viewConfigLLM
		m.llmProfileForm = nil
		m.llmEditingProfile = nil
		m.llmError = ""
		return m, nil

	case "ctrl+s":
		if !m.llmSavingProfile && m.llmProfileForm != nil {
			m.llmSavingProfile = true
			return m, m.saveProfile()
		}
		return m, nil

	case "p":
		// Previous page of models (only when model field is focused)
		if m.llmProfileForm.IsFieldFocused("model") && m.llmModelsPage > 1 {
			// Pop from cursor stack
			if len(m.llmModelsCursorStack) > 0 {
				prevCursor := ""
				if len(m.llmModelsCursorStack) > 1 {
					prevCursor = m.llmModelsCursorStack[len(m.llmModelsCursorStack)-2]
				}
				m.llmModelsCursorStack = m.llmModelsCursorStack[:len(m.llmModelsCursorStack)-1]
				m.llmModelsPage--
				return m, m.loadModels(prevCursor)
			}
		}

	case "n":
		// Next page of models (only when model field is focused)
		if m.llmProfileForm.IsFieldFocused("model") && m.llmModelsHasMore {
			m.llmModelsCursorStack = append(m.llmModelsCursorStack, m.llmModelsCursor)
			m.llmModelsPage++
			return m, m.loadModels(m.llmModelsCursor)
		}
	}

	// Track values before form update for cascade detection
	prevProvider := m.llmProfileForm.GetFieldValue("provider")
	prevAccount := m.llmProfileForm.GetFieldValue("account")

	// Let form handle the key
	if m.llmProfileForm != nil {
		m.llmProfileForm.Update(msg)
	}

	// Check for cascades
	newProvider := m.llmProfileForm.GetFieldValue("provider")
	newAccount := m.llmProfileForm.GetFieldValue("account")

	if newProvider != prevProvider {
		return m, m.cascadeFromProvider()
	}
	if newAccount != prevAccount {
		return m, m.cascadeFromAccount()
	}

	return m, nil
}

// saveProfile saves the profile from the form.
func (m *IntegrationsModal) saveProfile() tea.Cmd {
	values := m.llmProfileForm.Values()
	providerName := m.getProviderName(values["provider"])
	isDefault := m.llmProfileForm.GetFieldChecked("is_default")
	integration := m.llmIntegration.Name
	editingProfile := m.llmEditingProfile

	return func() tea.Msg {
		var err error
		profileName := values["name"]

		if editingProfile != nil {
			// For now, delete and recreate (hub-core doesn't have update endpoint)
			// Delete old profile first if name changed
			if editingProfile.Name != profileName {
				_ = m.client.DeleteLLMProfile(integration, editingProfile.Name)
			} else {
				_ = m.client.DeleteLLMProfile(integration, profileName)
			}
		}

		// Create the profile
		err = m.client.CreateLLMProfile(integration, client.CreateProfileRequest{
			Name:     profileName,
			Provider: providerName,
			Account:  values["account"],
			Model:    values["model"],
		})
		if err != nil {
			return LLMProfileSavedMsg{Err: err}
		}

		// Set default if requested
		if isDefault {
			_ = m.client.SetDefaultLLMProfile(integration, profileName)
		}

		return LLMProfileSavedMsg{}
	}
}

// handleLLMProfileSaved processes the result of saving a profile.
func (m *IntegrationsModal) handleLLMProfileSaved(msg LLMProfileSavedMsg) (Modal, tea.Cmd) {
	m.llmSavingProfile = false
	if msg.Err != nil {
		m.llmError = msg.Err.Error()
		return m, nil
	}

	// Success - return to config view and refresh
	m.view = viewConfigLLM
	m.llmProfileForm = nil
	m.llmEditingProfile = nil
	m.llmLoading = true
	return m, m.loadLLMData()
}

// deleteProfile deletes an LLM profile.
func (m *IntegrationsModal) deleteProfile(profileName string) tea.Cmd {
	integration := m.llmIntegration.Name
	return func() tea.Msg {
		err := m.client.DeleteLLMProfile(integration, profileName)
		if err != nil {
			return LLMProfileDeletedMsg{Err: err}
		}
		return LLMProfileDeletedMsg{}
	}
}

// handleLLMProfileDeleted processes the result of deleting a profile.
func (m *IntegrationsModal) handleLLMProfileDeleted(msg LLMProfileDeletedMsg) (Modal, tea.Cmd) {
	if msg.Err != nil {
		m.llmError = msg.Err.Error()
		return m, nil
	}

	// Success - refresh
	return m, m.loadLLMData()
}

// testProfile tests an LLM profile's connectivity.
func (m *IntegrationsModal) testProfile(profileName string) tea.Cmd {
	integration := m.llmIntegration.Name
	return func() tea.Msg {
		result, err := m.client.TestLLMProfile(integration, profileName)
		if err != nil {
			return LLMProfileTestedMsg{Err: err}
		}
		return LLMProfileTestedMsg{Result: result}
	}
}

// handleLLMProfileTested processes the result of testing a profile.
func (m *IntegrationsModal) handleLLMProfileTested(msg LLMProfileTestedMsg) (Modal, tea.Cmd) {
	m.llmTesting = false
	if msg.Err != nil {
		m.llmError = msg.Err.Error()
		m.llmTestResult = nil
		return m, nil
	}

	m.llmTestResult = msg.Result
	return m, nil
}

// setDefaultProfile sets a profile as the default.
func (m *IntegrationsModal) setDefaultProfile(profileName string) tea.Cmd {
	integration := m.llmIntegration.Name
	return func() tea.Msg {
		err := m.client.SetDefaultLLMProfile(integration, profileName)
		if err != nil {
			return LLMProfileDefaultSetMsg{Err: err}
		}
		return LLMProfileDefaultSetMsg{}
	}
}

// handleLLMProfileDefaultSet processes the result of setting a default profile.
func (m *IntegrationsModal) handleLLMProfileDefaultSet(msg LLMProfileDefaultSetMsg) (Modal, tea.Cmd) {
	if msg.Err != nil {
		m.llmError = msg.Err.Error()
		return m, nil
	}

	// Success - refresh to update the default indicator
	return m, m.loadLLMData()
}

// viewLLMProfileForm renders the profile form.
func (m *IntegrationsModal) viewLLMProfileForm() string {
	var lines []string

	// Show form
	if m.llmProfileForm != nil {
		lines = append(lines, m.llmProfileForm.View())
	}

	// Show model description when model field is focused
	if m.llmProfileForm != nil && m.llmProfileForm.IsFieldFocused("model") {
		modelID := m.llmProfileForm.GetFieldValue("model")
		for _, model := range m.llmModels {
			if model.ID == modelID && model.Description != "" {
				lines = append(lines, "")
				descStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary).Italic(true)
				// Truncate long descriptions
				desc := model.Description
				if len(desc) > 80 {
					desc = desc[:77] + "..."
				}
				lines = append(lines, "  "+descStyle.Render(desc))
				break
			}
		}

		// Pagination info
		if m.llmModelsHasMore || m.llmModelsPage > 1 {
			lines = append(lines, "")
			pageStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
			pageInfo := fmt.Sprintf("  Page %d", m.llmModelsPage)
			if m.llmModelsPage > 1 {
				pageInfo += "  [p] prev"
			}
			if m.llmModelsHasMore {
				pageInfo += "  [n] next"
			}
			lines = append(lines, pageStyle.Render(pageInfo))
		}
	}

	// Show loading indicator for models
	if m.llmLoadingModels {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("  Loading models..."))
	}

	// Show error if any
	if m.llmError != "" {
		lines = append(lines, "")
		errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
		lines = append(lines, "  "+errorStyle.Render("Error: "+m.llmError))
	}

	// Show saving indicator
	if m.llmSavingProfile {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("  Saving..."))
	}

	// Hints
	lines = append(lines, "")
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	lines = append(lines, hintStyle.Render("  [Ctrl+S] Save  [Esc] Cancel"))

	return strings.Join(lines, "\n")
}

// viewLLM renders the LLM configuration view.
func (m *IntegrationsModal) viewLLM() string {
	// Handle sub-views
	if m.view == viewLLMProviderForm {
		return m.viewLLMProviderForm()
	}
	if m.view == viewLLMProfileForm {
		return m.viewLLMProfileForm()
	}

	if m.llmLoading {
		return lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("  Loading...")
	}

	if m.llmError != "" && len(m.llmItems) == 0 {
		errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
		hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			errorStyle.Render("  Error: "+m.llmError),
			"",
			hintStyle.Render("  [r] Retry  [Esc] Back"),
		)
	}

	var lines []string

	// Styles
	headerStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary).Bold(true)
	providerStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	defaultStyle := lipgloss.NewStyle().Foreground(theme.Warning)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	newItemStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	// --- Profiles Section (first - more frequently modified) ---
	lines = append(lines, headerStyle.Render("  Profiles"))

	for i, item := range m.llmItems {
		if item.Type == llmItemProfile {
			profile := item.Profile

			cursor := "  "
			if i == m.llmSelected {
				cursor = "> "
			}

			// Default indicator
			defaultMark := "  "
			if profile.IsDefault {
				defaultMark = "★ "
			}

			// Profile info: name    provider/account · model
			name := profile.Name
			info := profile.Provider + "/" + profile.Account + " · " + profile.Model

			// Pad name for alignment
			namePadded := name + strings.Repeat(" ", max(0, 12-len(name)))

			var profileLine string
			if profile.IsDefault {
				profileLine = cursor + defaultStyle.Render(defaultMark+namePadded) + dimStyle.Render(info)
			} else if i == m.llmSelected {
				profileLine = cursor + selectedStyle.Render(defaultMark+namePadded) + dimStyle.Render(info)
			} else {
				profileLine = cursor + normalStyle.Render(defaultMark+namePadded) + dimStyle.Render(info)
			}

			lines = append(lines, profileLine)
		} else if item.Type == llmItemNewProfile {
			// Add spacing before "+ New Profile" to separate from list
			lines = append(lines, "")
			cursor := "  "
			if i == m.llmSelected {
				cursor = "> "
				lines = append(lines, selectedStyle.Render(cursor+"  + New Profile"))
			} else {
				lines = append(lines, newItemStyle.Render(cursor+"  + New Profile"))
			}
		}
	}

	// Separator
	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("  ─────────────────────────────────"))
	lines = append(lines, "")

	// --- Providers Section ---
	lines = append(lines, headerStyle.Render("  Providers"))

	currentProvider := ""
	for i, item := range m.llmItems {
		if item.Type == llmItemProviderAccount {
			// Insert provider header if changed
			if item.Provider != currentProvider {
				currentProvider = item.Provider
				displayName := item.ProviderDisplay
				if displayName == "" {
					displayName = item.Provider
				}
				lines = append(lines, providerStyle.Render("    "+displayName))
			}

			// Render account
			cursor := "  "
			if i == m.llmSelected {
				cursor = "> "
			}

			accountLine := cursor + "  • " + item.Account
			if i == m.llmSelected {
				lines = append(lines, selectedStyle.Render(accountLine))
			} else {
				lines = append(lines, normalStyle.Render(accountLine))
			}
		} else if item.Type == llmItemNewProvider {
			// Add spacing before "+ New Provider" to separate from list
			lines = append(lines, "")
			cursor := "  "
			if i == m.llmSelected {
				cursor = "> "
				lines = append(lines, selectedStyle.Render(cursor+"  + New Provider"))
			} else {
				lines = append(lines, newItemStyle.Render(cursor+"  + New Provider"))
			}
		}
	}

	// Error message if present (inline)
	if m.llmError != "" {
		lines = append(lines, "")
		errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
		lines = append(lines, errorStyle.Render("  Error: "+m.llmError))
	}

	// Test result
	if m.llmTesting {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("  Testing..."))
	} else if m.llmTestResult != nil {
		lines = append(lines, "")
		if m.llmTestResult.Success {
			successStyle := lipgloss.NewStyle().Foreground(theme.Success)
			lines = append(lines, successStyle.Render(fmt.Sprintf("  ✓ Test passed (%dms)", m.llmTestResult.LatencyMs)))
		} else {
			errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
			errMsg := m.llmTestResult.Error
			if errMsg == "" {
				errMsg = "Unknown error"
			}
			lines = append(lines, errorStyle.Render("  ✗ Test failed: "+errMsg))
		}
	}

	// Confirmation hint if pending
	if m.llmConfirm.IsPendingAny() {
		lines = append(lines, "")
		warnStyle := lipgloss.NewStyle().Foreground(theme.Warning)
		lines = append(lines, warnStyle.Render("  Press d again to delete "+m.llmConfirm.PendingID()))
	}

	// Hints
	lines = append(lines, "")
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	// Show context-appropriate hints based on selected item
	var hints string
	if m.llmSelected >= 0 && m.llmSelected < len(m.llmItems) {
		item := m.llmItems[m.llmSelected]
		switch item.Type {
		case llmItemProfile:
			if item.Profile.IsDefault {
				hints = "  [Enter] Edit  [t] Test  [d] Delete  [r] Refresh  [Esc] Back"
			} else {
				hints = "  [Enter] Edit  [t] Test  [s] Set Default  [d] Delete  [r] Refresh  [Esc] Back"
			}
		case llmItemProviderAccount:
			hints = "  [d] Delete  [r] Refresh  [Esc] Back"
		case llmItemNewProfile, llmItemNewProvider:
			hints = "  [Enter] Create  [r] Refresh  [Esc] Back"
		default:
			hints = "  [r] Refresh  [Esc] Back"
		}
	} else {
		hints = "  [r] Refresh  [Esc] Back"
	}
	lines = append(lines, hintStyle.Render(hints))

	return strings.Join(lines, "\n")
}

// viewLLMProviderForm renders the provider form.
func (m *IntegrationsModal) viewLLMProviderForm() string {
	var lines []string

	// Show form
	if m.llmProviderForm != nil {
		lines = append(lines, m.llmProviderForm.View())
	}

	// Show loading indicator for fields
	if m.llmLoadingFields {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("  Loading fields..."))
	}

	// Show error if any
	if m.llmError != "" {
		lines = append(lines, "")
		errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
		lines = append(lines, "  "+errorStyle.Render("Error: "+m.llmError))
	}

	// Show saving indicator
	if m.llmSavingProvider {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("  Saving..."))
	}

	// Hints
	lines = append(lines, "")
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	lines = append(lines, hintStyle.Render("  [Ctrl+S] Save  [Esc] Cancel"))

	return strings.Join(lines, "\n")
}
