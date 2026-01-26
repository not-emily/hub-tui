package modal

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/client"
	"github.com/pxp/hub-tui/internal/ui/theme"
)

// View states for the assistants modal
const (
	assistantViewList = iota
	assistantViewDetail
	assistantViewCreate
	assistantViewEdit
	assistantViewConfirmDelete
	assistantViewConfirmClearHistory
	assistantViewMemory
	assistantViewMemoryAdd
	assistantViewMemoryConfirmDiscard
	assistantViewTemplateList
	assistantViewTemplatePreview
)

// Form field indices
const (
	formFieldName = iota
	formFieldDisplayName
	formFieldProfile
	formFieldPersona
	formFieldKeywords
	formFieldModules
	formFieldCount
)

// AssistantsModal displays and manages assistants.
type AssistantsModal struct {
	client     *client.Client
	assistants []client.Assistant
	selected   int
	loading    bool
	error      string

	// Detail view state
	view    int
	current *client.Assistant

	// Form state
	formFocused     int
	formName        string
	formDisplayName string
	formPersona     string
	formProfile     string
	formKeywords    string
	formModules     map[string]bool            // module name -> enabled
	formGather      map[string]map[string]bool // module -> tool -> enabled
	formError       string

	// Available options for form
	availableProfiles []client.LLMProfile
	availableModules  []client.Module

	// Module selection state (for navigating modules/tools)
	moduleSelected int
	inToolSelect   bool
	toolSelected   int

	// Memory state
	memoryEntries   []memoryEntry // Ordered list for navigation
	memorySelected  int           // Selected entry index
	memoryEditing   bool          // Currently editing a value
	memoryEditValue string        // Value being edited
	memoryNewKey    string        // When adding new entry
	memoryNewValue  string
	memoryDirty     bool   // Has unsaved changes
	memoryError     string // Memory-specific error
	memoryFocusKey  bool   // In add mode: true=key field, false=value field

	// Template state
	templates        []moduleTemplate // All templates from all modules
	templateSelected int              // Selected template index
	templateLoading  bool             // Loading templates
	templateError    string           // Template-specific error

	// Override fields for template creation
	templateOverrideName    string
	templateOverrideDisplay string
	templateFocusName       bool // true=name field, false=display name field
}

// memoryEntry represents a single memory key-value pair.
type memoryEntry struct {
	Key   string
	Value string
}

// moduleTemplate represents a template with its source module.
type moduleTemplate struct {
	Module   string
	Template client.AssistantTemplate
}

// NewAssistantsModal creates a new assistants modal.
func NewAssistantsModal(c *client.Client) *AssistantsModal {
	return &AssistantsModal{
		client:      c,
		loading:     true,
		view:        assistantViewList,
		formModules: make(map[string]bool),
		formGather:  make(map[string]map[string]bool),
	}
}

// Message types

// AssistantsLoadedMsg is sent when assistants are loaded.
type AssistantsLoadedMsg struct {
	Assistants []client.Assistant
	Error      error
}

// AssistantDetailMsg is sent when a single assistant is loaded.
type AssistantDetailMsg struct {
	Assistant *client.Assistant
	Error     error
}

// AssistantFormDataMsg is sent when form data (profiles + modules) is loaded.
type AssistantFormDataMsg struct {
	Profiles []client.LLMProfile
	Modules  []client.Module
	Error    error
}

// AssistantCreatedMsg is sent when an assistant is created.
type AssistantCreatedMsg struct {
	Assistant *client.Assistant
	Error     error
}

// AssistantUpdatedMsg is sent when an assistant is updated.
type AssistantUpdatedMsg struct {
	Assistant *client.Assistant
	Error     error
}

// AssistantDeletedMsg is sent when an assistant is deleted.
type AssistantDeletedMsg struct {
	Name  string
	Error error
}

// AssistantHistoryClearedMsg is sent when history is cleared.
type AssistantHistoryClearedMsg struct {
	Name  string
	Error error
}

// AssistantMemoryLoadedMsg is sent when memory is loaded.
type AssistantMemoryLoadedMsg struct {
	Memory *client.AssistantMemory
	Error  error
}

// AssistantMemorySavedMsg is sent when memory is saved.
type AssistantMemorySavedMsg struct {
	Error error
}

// AssistantTemplatesLoadedMsg is sent when templates are loaded.
type AssistantTemplatesLoadedMsg struct {
	Templates []moduleTemplate
	Error     error
}

// AssistantTemplateCreatedMsg is sent when an assistant is created from template.
type AssistantTemplateCreatedMsg struct {
	Assistant *client.Assistant
	Error     error
}

// Init initializes the modal and triggers data fetch.
func (m *AssistantsModal) Init() tea.Cmd {
	return m.loadAssistants()
}

func (m *AssistantsModal) loadAssistants() tea.Cmd {
	return func() tea.Msg {
		assistants, err := m.client.ListAssistants()
		return AssistantsLoadedMsg{Assistants: assistants, Error: err}
	}
}

func (m *AssistantsModal) loadAssistantDetail(name string) tea.Cmd {
	return func() tea.Msg {
		assistant, err := m.client.GetAssistant(name)
		return AssistantDetailMsg{Assistant: assistant, Error: err}
	}
}

func (m *AssistantsModal) loadFormData() tea.Cmd {
	return func() tea.Msg {
		profiles, err := m.client.ListLLMProfiles("llm")
		if err != nil {
			return AssistantFormDataMsg{Error: err}
		}

		modules, err := m.client.ListModules()
		if err != nil {
			return AssistantFormDataMsg{Error: err}
		}

		return AssistantFormDataMsg{
			Profiles: profiles.Profiles,
			Modules:  modules,
		}
	}
}

func (m *AssistantsModal) createAssistant() tea.Cmd {
	// Build request from form
	req := &client.CreateAssistantRequest{
		Name:        m.formName,
		DisplayName: m.formDisplayName,
		Persona:     m.formPersona,
		LLMProfile:  m.formProfile,
		Keywords:    parseKeywords(m.formKeywords),
		Modules:     m.getSelectedModules(),
		Gather:      m.getSelectedGather(),
	}

	return func() tea.Msg {
		assistant, err := m.client.CreateAssistant(req)
		return AssistantCreatedMsg{Assistant: assistant, Error: err}
	}
}

func (m *AssistantsModal) updateAssistant() tea.Cmd {
	if m.current == nil {
		return nil
	}

	req := &client.UpdateAssistantRequest{
		DisplayName: m.formDisplayName,
		Persona:     m.formPersona,
		LLMProfile:  m.formProfile,
		Keywords:    parseKeywords(m.formKeywords),
		Modules:     m.getSelectedModules(),
		Gather:      m.getSelectedGather(),
	}

	name := m.current.Name
	return func() tea.Msg {
		assistant, err := m.client.UpdateAssistant(name, req)
		return AssistantUpdatedMsg{Assistant: assistant, Error: err}
	}
}

func (m *AssistantsModal) deleteAssistant() tea.Cmd {
	if m.current == nil {
		return nil
	}

	name := m.current.Name
	return func() tea.Msg {
		err := m.client.DeleteAssistant(name)
		return AssistantDeletedMsg{Name: name, Error: err}
	}
}

func (m *AssistantsModal) clearHistory() tea.Cmd {
	if m.current == nil {
		return nil
	}

	name := m.current.Name
	return func() tea.Msg {
		err := m.client.ClearAssistantHistory(name)
		return AssistantHistoryClearedMsg{Name: name, Error: err}
	}
}

func (m *AssistantsModal) loadMemory() tea.Cmd {
	if m.current == nil {
		return nil
	}

	name := m.current.Name
	return func() tea.Msg {
		memory, err := m.client.GetAssistantMemory(name)
		return AssistantMemoryLoadedMsg{Memory: memory, Error: err}
	}
}

func (m *AssistantsModal) saveMemory() tea.Cmd {
	if m.current == nil {
		return nil
	}

	// Convert memoryEntries back to map
	entries := make(map[string]string)
	for _, e := range m.memoryEntries {
		entries[e.Key] = e.Value
	}

	name := m.current.Name
	memory := &client.AssistantMemory{Entries: entries}
	return func() tea.Msg {
		err := m.client.UpdateAssistantMemory(name, memory)
		return AssistantMemorySavedMsg{Error: err}
	}
}

func (m *AssistantsModal) loadAllTemplates() tea.Cmd {
	return func() tea.Msg {
		modules, err := m.client.ListModules()
		if err != nil {
			return AssistantTemplatesLoadedMsg{Error: err}
		}

		var templates []moduleTemplate
		for _, mod := range modules {
			if !mod.Enabled {
				continue
			}

			modTemplates, err := m.client.ListModuleAssistantTemplates(mod.Name)
			if err != nil {
				continue // Skip modules that fail
			}

			for _, t := range modTemplates {
				templates = append(templates, moduleTemplate{
					Module:   mod.Name,
					Template: t,
				})
			}
		}

		// Sort by module name, then by template name
		sort.Slice(templates, func(i, j int) bool {
			if templates[i].Module != templates[j].Module {
				return templates[i].Module < templates[j].Module
			}
			return templates[i].Template.Name < templates[j].Template.Name
		})

		return AssistantTemplatesLoadedMsg{Templates: templates}
	}
}

func (m *AssistantsModal) createFromTemplate() tea.Cmd {
	if m.templateSelected >= len(m.templates) {
		return nil
	}

	tmpl := m.templates[m.templateSelected]
	var overrides *client.TemplateOverrides
	if m.templateOverrideName != "" || m.templateOverrideDisplay != "" {
		overrides = &client.TemplateOverrides{
			Name:        m.templateOverrideName,
			DisplayName: m.templateOverrideDisplay,
		}
	}

	return func() tea.Msg {
		assistant, err := m.client.CreateAssistantFromTemplate(tmpl.Module, tmpl.Template.Name, overrides)
		return AssistantTemplateCreatedMsg{Assistant: assistant, Error: err}
	}
}

func (m *AssistantsModal) getSelectedModules() []string {
	var modules []string
	for name, enabled := range m.formModules {
		if enabled {
			modules = append(modules, name)
		}
	}
	return modules
}

func (m *AssistantsModal) getSelectedGather() map[string][]string {
	gather := make(map[string][]string)
	for mod, tools := range m.formGather {
		if !m.formModules[mod] {
			continue // Skip unselected modules
		}
		var selected []string
		for tool, enabled := range tools {
			if enabled {
				selected = append(selected, tool)
			}
		}
		if len(selected) > 0 {
			gather[mod] = selected
		}
	}
	return gather
}

func parseKeywords(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var keywords []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			keywords = append(keywords, p)
		}
	}
	return keywords
}

// Update handles input.
func (m *AssistantsModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case AssistantsLoadedMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			m.assistants = msg.Assistants
			m.error = ""
		}
		return m, nil

	case AssistantDetailMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
			m.view = assistantViewList
		} else {
			m.current = msg.Assistant
			m.error = ""
			m.view = assistantViewDetail
		}
		return m, nil

	case AssistantFormDataMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
			m.view = assistantViewList
		} else {
			m.availableProfiles = msg.Profiles
			m.availableModules = msg.Modules
			m.error = ""

			// Check if we have profiles
			if len(m.availableProfiles) == 0 {
				m.error = "No LLM profiles configured. Use /integrations to set up AI first."
				m.view = assistantViewList
				return m, nil
			}

			// Set default profile if not set
			if m.formProfile == "" {
				for _, p := range m.availableProfiles {
					if p.IsDefault {
						m.formProfile = p.Name
						break
					}
				}
				if m.formProfile == "" && len(m.availableProfiles) > 0 {
					m.formProfile = m.availableProfiles[0].Name
				}
			}
		}
		return m, nil

	case AssistantCreatedMsg:
		m.loading = false
		if msg.Error != nil {
			m.formError = msg.Error.Error()
		} else {
			m.view = assistantViewList
			m.formError = ""
			return m, m.loadAssistants()
		}
		return m, nil

	case AssistantUpdatedMsg:
		m.loading = false
		if msg.Error != nil {
			m.formError = msg.Error.Error()
		} else {
			m.current = msg.Assistant
			m.view = assistantViewDetail
			m.formError = ""
		}
		return m, nil

	case AssistantDeletedMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
			m.view = assistantViewDetail
		} else {
			m.current = nil
			m.view = assistantViewList
			return m, m.loadAssistants()
		}
		return m, nil

	case AssistantHistoryClearedMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			m.error = ""
		}
		m.view = assistantViewDetail
		return m, nil

	case AssistantMemoryLoadedMsg:
		m.loading = false
		if msg.Error != nil {
			m.memoryError = msg.Error.Error()
			m.view = assistantViewDetail
		} else {
			m.memoryError = ""
			m.view = assistantViewMemory
			// Convert map to ordered slice
			m.memoryEntries = nil
			if msg.Memory != nil {
				for k, v := range msg.Memory.Entries {
					m.memoryEntries = append(m.memoryEntries, memoryEntry{Key: k, Value: v})
				}
			}
			// Sort by key for consistent order
			for i := 0; i < len(m.memoryEntries); i++ {
				for j := i + 1; j < len(m.memoryEntries); j++ {
					if m.memoryEntries[i].Key > m.memoryEntries[j].Key {
						m.memoryEntries[i], m.memoryEntries[j] = m.memoryEntries[j], m.memoryEntries[i]
					}
				}
			}
			m.memorySelected = 0
			m.memoryDirty = false
			m.memoryEditing = false
		}
		return m, nil

	case AssistantMemorySavedMsg:
		m.loading = false
		if msg.Error != nil {
			m.memoryError = msg.Error.Error()
		} else {
			m.memoryError = ""
			m.memoryDirty = false
		}
		return m, nil

	case AssistantTemplatesLoadedMsg:
		m.templateLoading = false
		if msg.Error != nil {
			m.templateError = msg.Error.Error()
			m.view = assistantViewList
		} else {
			m.templateError = ""
			m.templates = msg.Templates
			m.templateSelected = 0
			m.view = assistantViewTemplateList
		}
		return m, nil

	case AssistantTemplateCreatedMsg:
		m.loading = false
		if msg.Error != nil {
			m.templateError = msg.Error.Error()
		} else {
			m.templateError = ""
			m.view = assistantViewList
			return m, m.loadAssistants()
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}
	return m, nil
}

func (m *AssistantsModal) handleKeyMsg(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch m.view {
	case assistantViewList:
		return m.handleListKeys(msg)
	case assistantViewDetail:
		return m.handleDetailKeys(msg)
	case assistantViewCreate, assistantViewEdit:
		return m.handleFormKeys(msg)
	case assistantViewConfirmDelete:
		return m.handleConfirmDeleteKeys(msg)
	case assistantViewConfirmClearHistory:
		return m.handleConfirmClearHistoryKeys(msg)
	case assistantViewMemory:
		return m.handleMemoryKeys(msg)
	case assistantViewMemoryAdd:
		return m.handleMemoryAddKeys(msg)
	case assistantViewTemplateList:
		return m.handleTemplateListKeys(msg)
	case assistantViewTemplatePreview:
		return m.handleTemplatePreviewKeys(msg)
	case assistantViewMemoryConfirmDiscard:
		return m.handleMemoryDiscardKeys(msg)
	}
	return m, nil
}

func (m *AssistantsModal) handleListKeys(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return nil, nil // Close modal
	case "up", "k":
		if m.selected > 0 {
			m.selected--
		}
	case "down", "j":
		if m.selected < len(m.assistants)-1 {
			m.selected++
		}
	case "enter":
		if !m.loading && len(m.assistants) > 0 {
			m.loading = true
			return m, m.loadAssistantDetail(m.assistants[m.selected].Name)
		}
	case "r":
		m.loading = true
		m.error = ""
		return m, m.loadAssistants()
	case "n":
		m.initCreateForm()
		m.loading = true
		return m, m.loadFormData()
	case "t":
		m.templateLoading = true
		m.templateError = ""
		m.templateSelected = 0
		m.templateOverrideName = ""
		m.templateOverrideDisplay = ""
		return m, m.loadAllTemplates()
	}
	return m, nil
}

func (m *AssistantsModal) handleDetailKeys(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.view = assistantViewList
		m.current = nil
		return m, nil
	case "e":
		m.initEditForm()
		m.loading = true
		return m, m.loadFormData()
	case "m":
		m.loading = true
		m.memoryError = ""
		return m, m.loadMemory()
	case "h":
		m.view = assistantViewConfirmClearHistory
		return m, nil
	case "d":
		m.view = assistantViewConfirmDelete
		return m, nil
	case "r":
		if m.current != nil {
			m.loading = true
			return m, m.loadAssistantDetail(m.current.Name)
		}
	}
	return m, nil
}

func (m *AssistantsModal) handleFormKeys(msg tea.KeyMsg) (Modal, tea.Cmd) {
	key := msg.String()

	// Handle module/tool selection specially
	if m.formFocused == formFieldModules {
		return m.handleModuleSelectionKeys(msg)
	}

	switch key {
	case "esc":
		if m.view == assistantViewCreate {
			m.view = assistantViewList
		} else {
			m.view = assistantViewDetail
		}
		return m, nil

	case "tab", "down":
		m.formFocused++
		if m.formFocused >= formFieldCount {
			m.formFocused = formFieldCount - 1
		}
		return m, nil

	case "shift+tab", "up":
		m.formFocused--
		if m.formFocused < 0 {
			m.formFocused = 0
		}
		// Skip name field in edit mode
		if m.view == assistantViewEdit && m.formFocused == formFieldName {
			m.formFocused = formFieldDisplayName
		}
		return m, nil

	case "enter", "ctrl+s":
		// Submit if valid
		if err := m.validateForm(); err != "" {
			m.formError = err
			return m, nil
		}
		m.loading = true
		if m.view == assistantViewCreate {
			return m, m.createAssistant()
		}
		return m, m.updateAssistant()

	case "left", "right":
		// Handle profile dropdown
		if m.formFocused == formFieldProfile {
			return m.handleProfileSelection(key)
		}
		return m, nil

	case "backspace":
		m.handleBackspace()
		return m, nil

	default:
		// Handle text input
		if len(key) == 1 {
			m.handleTextInput(key)
		}
		return m, nil
	}
}

func (m *AssistantsModal) handleProfileSelection(key string) (Modal, tea.Cmd) {
	if len(m.availableProfiles) == 0 {
		return m, nil
	}

	currentIdx := 0
	for i, p := range m.availableProfiles {
		if p.Name == m.formProfile {
			currentIdx = i
			break
		}
	}

	if key == "left" {
		currentIdx--
		if currentIdx < 0 {
			currentIdx = len(m.availableProfiles) - 1
		}
	} else {
		currentIdx++
		if currentIdx >= len(m.availableProfiles) {
			currentIdx = 0
		}
	}

	m.formProfile = m.availableProfiles[currentIdx].Name
	return m, nil
}

func (m *AssistantsModal) handleBackspace() {
	switch m.formFocused {
	case formFieldName:
		if len(m.formName) > 0 {
			m.formName = m.formName[:len(m.formName)-1]
		}
	case formFieldDisplayName:
		if len(m.formDisplayName) > 0 {
			m.formDisplayName = m.formDisplayName[:len(m.formDisplayName)-1]
		}
	case formFieldPersona:
		if len(m.formPersona) > 0 {
			m.formPersona = m.formPersona[:len(m.formPersona)-1]
		}
	case formFieldKeywords:
		if len(m.formKeywords) > 0 {
			m.formKeywords = m.formKeywords[:len(m.formKeywords)-1]
		}
	}
}

func (m *AssistantsModal) handleTextInput(key string) {
	switch m.formFocused {
	case formFieldName:
		// Only allow valid name characters
		if m.view == assistantViewCreate && isValidNameChar(key) {
			m.formName += strings.ToLower(key)
		}
	case formFieldDisplayName:
		m.formDisplayName += key
	case formFieldPersona:
		m.formPersona += key
	case formFieldKeywords:
		m.formKeywords += key
	}
}

func (m *AssistantsModal) handleModuleSelectionKeys(msg tea.KeyMsg) (Modal, tea.Cmd) {
	enabledModules := m.getEnabledModules()
	key := msg.String()

	switch key {
	case "esc":
		if m.view == assistantViewCreate {
			m.view = assistantViewList
		} else {
			m.view = assistantViewDetail
		}
		return m, nil

	case "tab":
		// Cycle back to first editable field
		m.inToolSelect = false
		m.toolSelected = 0
		m.moduleSelected = 0
		if m.view == assistantViewCreate {
			m.formFocused = formFieldName
		} else {
			m.formFocused = formFieldDisplayName
		}
		return m, nil

	case "shift+tab", "up", "k":
		if m.inToolSelect {
			// Move back to module row
			m.inToolSelect = false
			m.toolSelected = 0
		} else if m.moduleSelected > 0 {
			m.moduleSelected--
		} else {
			// Go back to previous field
			m.formFocused = formFieldKeywords
		}
		return m, nil

	case "down", "j":
		if m.inToolSelect {
			// Stay in tool select, or move to next module
			m.inToolSelect = false
			m.toolSelected = 0
			m.moduleSelected++
			if m.moduleSelected >= len(enabledModules) {
				m.moduleSelected = len(enabledModules) - 1
			}
		} else if m.moduleSelected < len(enabledModules)-1 {
			m.moduleSelected++
		}
		return m, nil

	case "right", "l":
		if m.moduleSelected < len(enabledModules) {
			mod := enabledModules[m.moduleSelected]
			if m.inToolSelect {
				// Navigate right within tools
				if m.toolSelected < len(mod.Tools)-1 {
					m.toolSelected++
				}
			} else if m.formModules[mod.Name] && len(mod.Tools) > 0 {
				// Enter tool selection
				m.inToolSelect = true
				m.toolSelected = 0
			}
		}
		return m, nil

	case "left", "h":
		if m.inToolSelect {
			if m.toolSelected > 0 {
				m.toolSelected--
			} else {
				m.inToolSelect = false
			}
		}
		return m, nil

	case " ", "enter":
		// Toggle (both Space and Enter toggle in module section)
		if m.moduleSelected < len(enabledModules) {
			mod := enabledModules[m.moduleSelected]
			if m.inToolSelect && len(mod.Tools) > 0 {
				// Toggle tool
				tool := mod.Tools[m.toolSelected]
				if m.formGather[mod.Name] == nil {
					m.formGather[mod.Name] = make(map[string]bool)
				}
				m.formGather[mod.Name][tool] = !m.formGather[mod.Name][tool]
			} else {
				// Toggle module
				m.formModules[mod.Name] = !m.formModules[mod.Name]
			}
		}
		return m, nil

	case "ctrl+s":
		// Submit form
		if err := m.validateForm(); err != "" {
			m.formError = err
			return m, nil
		}
		m.loading = true
		if m.view == assistantViewCreate {
			return m, m.createAssistant()
		}
		return m, m.updateAssistant()
	}

	return m, nil
}

func (m *AssistantsModal) getEnabledModules() []client.Module {
	var enabled []client.Module
	for _, mod := range m.availableModules {
		if mod.Enabled {
			enabled = append(enabled, mod)
		}
	}
	return enabled
}

func (m *AssistantsModal) handleConfirmDeleteKeys(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.view = assistantViewDetail
		return m, nil
	case "enter":
		m.loading = true
		return m, m.deleteAssistant()
	}
	return m, nil
}

func (m *AssistantsModal) handleConfirmClearHistoryKeys(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.view = assistantViewDetail
		return m, nil
	case "enter":
		m.loading = true
		return m, m.clearHistory()
	}
	return m, nil
}

func (m *AssistantsModal) handleMemoryKeys(msg tea.KeyMsg) (Modal, tea.Cmd) {
	key := msg.String()

	// If editing, handle edit keys
	if m.memoryEditing {
		switch key {
		case "esc":
			m.memoryEditing = false
			m.memoryEditValue = ""
			return m, nil
		case "enter":
			// Save edit
			if m.memorySelected < len(m.memoryEntries) {
				m.memoryEntries[m.memorySelected].Value = m.memoryEditValue
				m.memoryDirty = true
			}
			m.memoryEditing = false
			m.memoryEditValue = ""
			return m, nil
		case "backspace":
			if len(m.memoryEditValue) > 0 {
				m.memoryEditValue = m.memoryEditValue[:len(m.memoryEditValue)-1]
			}
			return m, nil
		default:
			if len(key) == 1 {
				m.memoryEditValue += key
			}
			return m, nil
		}
	}

	// Normal navigation
	switch key {
	case "esc":
		if m.memoryDirty {
			m.view = assistantViewMemoryConfirmDiscard
		} else {
			m.view = assistantViewDetail
		}
		return m, nil
	case "up", "k":
		if m.memorySelected > 0 {
			m.memorySelected--
		}
		return m, nil
	case "down", "j":
		if m.memorySelected < len(m.memoryEntries)-1 {
			m.memorySelected++
		}
		return m, nil
	case "enter":
		// Start editing
		if m.memorySelected < len(m.memoryEntries) {
			m.memoryEditing = true
			m.memoryEditValue = m.memoryEntries[m.memorySelected].Value
		}
		return m, nil
	case "a":
		// Add new entry
		m.view = assistantViewMemoryAdd
		m.memoryNewKey = ""
		m.memoryNewValue = ""
		m.memoryFocusKey = true
		return m, nil
	case "d":
		// Delete entry
		if m.memorySelected < len(m.memoryEntries) {
			m.memoryEntries = append(m.memoryEntries[:m.memorySelected], m.memoryEntries[m.memorySelected+1:]...)
			if m.memorySelected >= len(m.memoryEntries) && m.memorySelected > 0 {
				m.memorySelected--
			}
			m.memoryDirty = true
		}
		return m, nil
	case "s", "ctrl+s":
		// Save
		m.loading = true
		return m, m.saveMemory()
	}
	return m, nil
}

func (m *AssistantsModal) handleMemoryAddKeys(msg tea.KeyMsg) (Modal, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		m.view = assistantViewMemory
		return m, nil
	case "tab":
		m.memoryFocusKey = !m.memoryFocusKey
		return m, nil
	case "enter":
		// Add entry if key is not empty
		if m.memoryNewKey != "" {
			m.memoryEntries = append(m.memoryEntries, memoryEntry{
				Key:   m.memoryNewKey,
				Value: m.memoryNewValue,
			})
			m.memoryDirty = true
			m.memorySelected = len(m.memoryEntries) - 1
		}
		m.view = assistantViewMemory
		return m, nil
	case "backspace":
		if m.memoryFocusKey {
			if len(m.memoryNewKey) > 0 {
				m.memoryNewKey = m.memoryNewKey[:len(m.memoryNewKey)-1]
			}
		} else {
			if len(m.memoryNewValue) > 0 {
				m.memoryNewValue = m.memoryNewValue[:len(m.memoryNewValue)-1]
			}
		}
		return m, nil
	default:
		if len(key) == 1 {
			if m.memoryFocusKey {
				// Only allow valid key characters
				if isValidKeyChar(key) {
					m.memoryNewKey += key
				}
			} else {
				m.memoryNewValue += key
			}
		}
		return m, nil
	}
}

func (m *AssistantsModal) handleMemoryDiscardKeys(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.view = assistantViewMemory
		return m, nil
	case "s":
		// Save and exit
		m.loading = true
		m.view = assistantViewDetail
		return m, m.saveMemory()
	case "enter":
		// Discard and exit
		m.memoryDirty = false
		m.view = assistantViewDetail
		return m, nil
	}
	return m, nil
}

func (m *AssistantsModal) handleTemplateListKeys(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.view = assistantViewList
		return m, nil
	case "up", "k":
		if m.templateSelected > 0 {
			m.templateSelected--
		}
		return m, nil
	case "down", "j":
		if m.templateSelected < len(m.templates)-1 {
			m.templateSelected++
		}
		return m, nil
	case "enter":
		if len(m.templates) > 0 && m.templateSelected < len(m.templates) {
			// Initialize override fields with template defaults
			tmpl := m.templates[m.templateSelected].Template
			m.templateOverrideName = tmpl.Name
			m.templateOverrideDisplay = tmpl.DisplayName
			m.templateFocusName = true
			m.view = assistantViewTemplatePreview
		}
		return m, nil
	}
	return m, nil
}

func (m *AssistantsModal) handleTemplatePreviewKeys(msg tea.KeyMsg) (Modal, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		m.view = assistantViewTemplateList
		return m, nil
	case "tab":
		m.templateFocusName = !m.templateFocusName
		return m, nil
	case "enter":
		// Create assistant from template
		m.loading = true
		return m, m.createFromTemplate()
	case "backspace":
		if m.templateFocusName {
			if len(m.templateOverrideName) > 0 {
				m.templateOverrideName = m.templateOverrideName[:len(m.templateOverrideName)-1]
			}
		} else {
			if len(m.templateOverrideDisplay) > 0 {
				m.templateOverrideDisplay = m.templateOverrideDisplay[:len(m.templateOverrideDisplay)-1]
			}
		}
		return m, nil
	default:
		if len(key) == 1 {
			if m.templateFocusName {
				// Only allow valid name characters
				if isValidNameChar(key) {
					m.templateOverrideName += strings.ToLower(key)
				}
			} else {
				m.templateOverrideDisplay += key
			}
		}
		return m, nil
	}
}

func isValidKeyChar(s string) bool {
	if len(s) != 1 {
		return false
	}
	c := s[0]
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-'
}

func (m *AssistantsModal) initCreateForm() {
	m.view = assistantViewCreate
	m.formFocused = formFieldName
	m.formName = ""
	m.formDisplayName = ""
	m.formPersona = ""
	m.formProfile = ""
	m.formKeywords = ""
	m.formModules = make(map[string]bool)
	m.formGather = make(map[string]map[string]bool)
	m.formError = ""
	m.moduleSelected = 0
	m.inToolSelect = false
	m.toolSelected = 0
}

func (m *AssistantsModal) initEditForm() {
	if m.current == nil {
		return
	}

	m.view = assistantViewEdit
	m.formFocused = formFieldDisplayName // Skip name in edit mode
	m.formName = m.current.Name
	m.formDisplayName = m.current.DisplayName
	m.formPersona = m.current.Persona
	m.formProfile = m.current.LLMProfile
	m.formKeywords = strings.Join(m.current.Keywords, ", ")

	// Initialize modules
	m.formModules = make(map[string]bool)
	for _, mod := range m.current.Modules {
		m.formModules[mod] = true
	}

	// Initialize gather
	m.formGather = make(map[string]map[string]bool)
	for mod, tools := range m.current.Gather {
		m.formGather[mod] = make(map[string]bool)
		for _, tool := range tools {
			m.formGather[mod][tool] = true
		}
	}

	m.formError = ""
	m.moduleSelected = 0
	m.inToolSelect = false
	m.toolSelected = 0
}

func (m *AssistantsModal) validateForm() string {
	// Name validation (create only)
	if m.view == assistantViewCreate {
		if m.formName == "" {
			return "Name is required"
		}
		if !isValidName(m.formName) {
			return "Name must be lowercase alphanumeric with hyphens only"
		}
	}

	// Display name
	if m.formDisplayName == "" {
		return "Display name is required"
	}

	// Profile
	if m.formProfile == "" {
		return "LLM profile is required"
	}

	// Persona
	if len(m.formPersona) < 10 {
		return "Persona must be at least 10 characters"
	}

	return ""
}

var validNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)

func isValidName(s string) bool {
	return validNameRegex.MatchString(s)
}

func isValidNameChar(s string) bool {
	if len(s) != 1 {
		return false
	}
	c := s[0]
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-'
}

// Title returns the modal title.
func (m *AssistantsModal) Title() string {
	switch m.view {
	case assistantViewDetail:
		if m.current != nil {
			return "Assistant: " + m.current.Name
		}
	case assistantViewCreate:
		return "Create Assistant"
	case assistantViewEdit:
		if m.current != nil {
			return "Edit: " + m.current.Name
		}
	case assistantViewConfirmDelete:
		return "Delete Assistant"
	case assistantViewConfirmClearHistory:
		return "Clear History"
	case assistantViewMemory, assistantViewMemoryAdd, assistantViewMemoryConfirmDiscard:
		if m.current != nil {
			title := m.current.Name + " — Core Memory"
			if m.memoryDirty {
				title += " *"
			}
			return title
		}
		return "Core Memory"
	case assistantViewTemplateList:
		return "Create from Template"
	case assistantViewTemplatePreview:
		if m.templateSelected < len(m.templates) {
			return "Create from Template — " + m.templates[m.templateSelected].Template.Name
		}
		return "Create from Template"
	}
	return "Assistants"
}

// View renders the modal content.
func (m *AssistantsModal) View() string {
	if m.loading || m.templateLoading {
		return lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("Loading...")
	}

	if m.error != "" && m.view != assistantViewCreate && m.view != assistantViewEdit {
		return m.viewError()
	}

	switch m.view {
	case assistantViewDetail:
		return m.viewDetail()
	case assistantViewCreate, assistantViewEdit:
		return m.viewForm()
	case assistantViewConfirmDelete:
		return m.viewConfirmDelete()
	case assistantViewConfirmClearHistory:
		return m.viewConfirmClearHistory()
	case assistantViewMemory:
		return m.viewMemory()
	case assistantViewMemoryAdd:
		return m.viewMemoryAdd()
	case assistantViewMemoryConfirmDiscard:
		return m.viewMemoryConfirmDiscard()
	case assistantViewTemplateList:
		return m.viewTemplateList()
	case assistantViewTemplatePreview:
		return m.viewTemplatePreview()
	default:
		return m.viewList()
	}
}

func (m *AssistantsModal) viewError() string {
	errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		errorStyle.Render("Error: "+m.error),
		"",
		hintStyle.Render("[r] Retry  [Esc] Back"),
	)
}

func (m *AssistantsModal) viewList() string {
	if len(m.assistants) == 0 {
		hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			hintStyle.Render("No assistants found."),
			"",
			hintStyle.Render("[n] New assistant  [t] From template"),
		)
	}

	var lines []string

	enabledStyle := lipgloss.NewStyle().Foreground(theme.Success)
	disabledStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	// Calculate max name length for alignment
	maxNameLen := 0
	for _, a := range m.assistants {
		if len(a.Name) > maxNameLen {
			maxNameLen = len(a.Name)
		}
	}
	if maxNameLen < 15 {
		maxNameLen = 15
	}

	for i, a := range m.assistants {
		// Status indicator
		var indicator string
		if a.Enabled {
			indicator = enabledStyle.Render("●")
		} else {
			indicator = disabledStyle.Render("○")
		}

		// Name with selection highlight
		var name string
		if i == m.selected {
			name = selectedStyle.Render(a.Name)
		} else {
			name = normalStyle.Render(a.Name)
		}

		// Pad name for alignment
		namePadding := maxNameLen - len(a.Name) + 2
		if namePadding < 2 {
			namePadding = 2
		}

		// Profile and display name
		var info string
		if a.LLMProfile != "" {
			info = a.LLMProfile
		}
		if a.DisplayName != "" && a.DisplayName != a.Name {
			if info != "" {
				info += "  "
			}
			info += a.DisplayName
		}

		line := fmt.Sprintf("  %s %s%s%s",
			indicator,
			name,
			strings.Repeat(" ", namePadding),
			dimStyle.Render(info),
		)

		lines = append(lines, line)
	}

	// Add legend and hints
	lines = append(lines, "")
	legendStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	lines = append(lines, legendStyle.Render("  ● enabled  ○ disabled"))
	lines = append(lines, "")
	lines = append(lines, legendStyle.Render("  [Enter] View  [n] New  [t] From template  [r] Refresh"))

	return strings.Join(lines, "\n")
}

func (m *AssistantsModal) viewDetail() string {
	if m.current == nil {
		return "No assistant selected"
	}

	a := m.current
	var lines []string

	labelStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	valueStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	accentStyle := lipgloss.NewStyle().Foreground(theme.Accent)

	// Display name
	if a.DisplayName != "" {
		lines = append(lines, labelStyle.Render("Display Name:  ")+valueStyle.Render(a.DisplayName))
	}

	// LLM Profile
	lines = append(lines, labelStyle.Render("LLM Profile:   ")+accentStyle.Render(a.LLMProfile))

	// Keywords
	if len(a.Keywords) > 0 {
		lines = append(lines, labelStyle.Render("Keywords:      ")+valueStyle.Render(strings.Join(a.Keywords, ", ")))
	}

	// Modules
	if len(a.Modules) > 0 {
		lines = append(lines, labelStyle.Render("Modules:       ")+valueStyle.Render(strings.Join(a.Modules, ", ")))
	}

	// Gather tools
	if len(a.Gather) > 0 {
		lines = append(lines, labelStyle.Render("Gather:"))
		for mod, tools := range a.Gather {
			lines = append(lines, "  "+accentStyle.Render(mod)+" → "+valueStyle.Render(strings.Join(tools, ", ")))
		}
	}

	// Persona (in a box)
	if a.Persona != "" {
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("Persona:"))

		// Create a bordered box for persona
		personaStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border).
			Padding(0, 1).
			Width(60)

		// Truncate if too long
		persona := a.Persona
		if len(persona) > 500 {
			persona = persona[:497] + "..."
		}

		lines = append(lines, personaStyle.Render(persona))
	}

	// Hints
	lines = append(lines, "")
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	lines = append(lines, hintStyle.Render("[e] Edit  [m] Memory  [h] Clear history  [d] Delete  [Esc] Back"))

	return strings.Join(lines, "\n")
}

func (m *AssistantsModal) viewForm() string {
	var lines []string

	labelStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	valueStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	focusedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	errorStyle := lipgloss.NewStyle().Foreground(theme.Error)

	// Helper to render field
	renderField := func(label, value string, focused bool, readonly bool) string {
		l := labelStyle.Render(label)
		var v string
		if focused && !readonly {
			v = focusedStyle.Render("[" + value + "_]")
		} else if readonly {
			v = lipgloss.NewStyle().Foreground(theme.TextSecondary).Render(value)
		} else {
			v = valueStyle.Render("[" + value + "]")
		}
		return l + v
	}

	// Name field (create only)
	if m.view == assistantViewCreate {
		lines = append(lines, renderField("Name:          ", m.formName, m.formFocused == formFieldName, false))
	} else {
		lines = append(lines, renderField("Name:          ", m.formName, false, true))
	}

	// Display name
	lines = append(lines, renderField("Display Name:  ", m.formDisplayName, m.formFocused == formFieldDisplayName, false))

	// LLM Profile (dropdown)
	profileLabel := labelStyle.Render("LLM Profile:   ")
	if m.formFocused == formFieldProfile {
		profileValue := focusedStyle.Render("◀ " + m.formProfile + " ▶")
		lines = append(lines, profileLabel+profileValue)
	} else {
		lines = append(lines, profileLabel+valueStyle.Render(m.formProfile))
	}

	// Persona
	lines = append(lines, "")
	if m.formFocused == formFieldPersona {
		lines = append(lines, focusedStyle.Render("Persona:"))
	} else {
		lines = append(lines, labelStyle.Render("Persona:"))
	}

	personaStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border).
		Padding(0, 1).
		Width(55)

	if m.formFocused == formFieldPersona {
		personaStyle = personaStyle.BorderForeground(theme.Accent)
	}

	persona := m.formPersona
	if len(persona) > 300 {
		persona = persona[:297] + "..."
	}
	if persona == "" {
		persona = " "
	}
	lines = append(lines, personaStyle.Render(persona))

	// Keywords
	lines = append(lines, "")
	lines = append(lines, renderField("Keywords:      ", m.formKeywords, m.formFocused == formFieldKeywords, false))
	lines = append(lines, labelStyle.Render("               (comma-separated, optional)"))

	// Modules section
	lines = append(lines, "")
	if m.formFocused == formFieldModules {
		lines = append(lines, focusedStyle.Render("Modules & Gather Tools:"))
	} else {
		lines = append(lines, labelStyle.Render("Modules & Gather Tools:"))
	}

	enabledModules := m.getEnabledModules()
	if len(enabledModules) == 0 {
		lines = append(lines, labelStyle.Render("  No modules enabled"))
	} else {
		for i, mod := range enabledModules {
			isModuleSelected := m.formFocused == formFieldModules && m.moduleSelected == i && !m.inToolSelect

			// Module checkbox
			checkbox := "[ ]"
			if m.formModules[mod.Name] {
				checkbox = "[x]"
			}

			// Add cursor for selected module
			cursor := "  "
			if isModuleSelected {
				cursor = "> "
			}

			modLine := cursor + checkbox + " " + mod.Name
			if isModuleSelected {
				modLine = focusedStyle.Render(modLine)
			} else {
				modLine = valueStyle.Render(modLine)
			}
			lines = append(lines, modLine)

			// Show tools if module is selected
			if m.formModules[mod.Name] && len(mod.Tools) > 0 {
				var toolParts []string
				for j, tool := range mod.Tools {
					isToolSelected := m.formFocused == formFieldModules && m.moduleSelected == i && m.inToolSelect && m.toolSelected == j

					toolCheck := "[ ]"
					if m.formGather[mod.Name] != nil && m.formGather[mod.Name][tool] {
						toolCheck = "[x]"
					}

					toolStr := toolCheck + " " + tool
					if isToolSelected {
						toolStr = focusedStyle.Render(toolStr)
					}
					toolParts = append(toolParts, toolStr)
				}
				lines = append(lines, "      "+strings.Join(toolParts, "  "))
			}
		}
	}

	// Error message
	if m.formError != "" {
		lines = append(lines, "")
		lines = append(lines, errorStyle.Render("Error: "+m.formError))
	}

	// Hints
	lines = append(lines, "")
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	if m.formFocused == formFieldModules {
		lines = append(lines, hintStyle.Render("[Space/Enter] Toggle  [j/k] Navigate  [l/→] Tools  [Ctrl+S] Save  [Esc] Cancel"))
	} else if m.formFocused == formFieldProfile {
		lines = append(lines, hintStyle.Render("[←→] Change profile  [Tab] Next  [Ctrl+S] Save  [Esc] Cancel"))
	} else {
		lines = append(lines, hintStyle.Render("[Tab] Next field  [Ctrl+S] Save  [Esc] Cancel"))
	}

	return strings.Join(lines, "\n")
}

func (m *AssistantsModal) viewConfirmDelete() string {
	if m.current == nil {
		return ""
	}

	var lines []string

	warningStyle := lipgloss.NewStyle().Foreground(theme.Warning)
	textStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	lines = append(lines, warningStyle.Render(fmt.Sprintf("Delete \"%s\"?", m.current.Name)))
	lines = append(lines, "")
	lines = append(lines, textStyle.Render("This will permanently delete this assistant"))
	lines = append(lines, textStyle.Render("and all its conversation history."))
	lines = append(lines, "")
	lines = append(lines, hintStyle.Render("[Enter] Delete  [Esc] Cancel"))

	return strings.Join(lines, "\n")
}

func (m *AssistantsModal) viewConfirmClearHistory() string {
	if m.current == nil {
		return ""
	}

	var lines []string

	warningStyle := lipgloss.NewStyle().Foreground(theme.Warning)
	textStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	lines = append(lines, warningStyle.Render(fmt.Sprintf("Clear history for \"%s\"?", m.current.Name)))
	lines = append(lines, "")
	lines = append(lines, textStyle.Render("This will delete all conversation history."))
	lines = append(lines, textStyle.Render("Core memory will be preserved."))
	lines = append(lines, "")
	lines = append(lines, hintStyle.Render("[Enter] Clear  [Esc] Cancel"))

	return strings.Join(lines, "\n")
}

func (m *AssistantsModal) viewMemory() string {
	var lines []string

	labelStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	keyStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent)
	errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	if m.memoryError != "" {
		lines = append(lines, errorStyle.Render("Error: "+m.memoryError))
		lines = append(lines, "")
	}

	if len(m.memoryEntries) == 0 {
		lines = append(lines, labelStyle.Render("No memory entries."))
		lines = append(lines, "")
		lines = append(lines, hintStyle.Render("[a] Add entry  [Esc] Back"))
		return strings.Join(lines, "\n")
	}

	// Render entries
	for i, entry := range m.memoryEntries {
		isSelected := i == m.memorySelected

		// Key line
		cursor := "  "
		if isSelected {
			cursor = "> "
		}

		var keyLine string
		if isSelected {
			keyLine = selectedStyle.Render(cursor + entry.Key)
		} else {
			keyLine = keyStyle.Render(cursor + entry.Key)
		}
		lines = append(lines, keyLine)

		// Value line (indent and possibly editing)
		if m.memoryEditing && isSelected {
			// Show edit input
			editStyle := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(theme.Accent).
				Padding(0, 1).
				Width(50)
			lines = append(lines, "    "+editStyle.Render(m.memoryEditValue+"_"))
		} else {
			// Truncate long values
			value := entry.Value
			if len(value) > 60 {
				value = value[:57] + "..."
			}
			lines = append(lines, "    "+valueStyle.Render(value))
		}

		lines = append(lines, "")
	}

	// Hints
	if m.memoryEditing {
		lines = append(lines, hintStyle.Render("[Enter] Save edit  [Esc] Cancel"))
	} else {
		hint := "[Enter] Edit  [a] Add  [d] Delete  [s] Save"
		if m.memoryDirty {
			hint += "  [Esc] Back (unsaved)"
		} else {
			hint += "  [Esc] Back"
		}
		lines = append(lines, hintStyle.Render(hint))
	}

	return strings.Join(lines, "\n")
}

func (m *AssistantsModal) viewMemoryAdd() string {
	var lines []string

	labelStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	focusedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	lines = append(lines, labelStyle.Render("Add Memory Entry"))
	lines = append(lines, "")

	// Key field
	keyLabel := "Key:   "
	if m.memoryFocusKey {
		keyLabel = focusedStyle.Render(keyLabel)
		lines = append(lines, keyLabel+"["+m.memoryNewKey+"_]")
	} else {
		lines = append(lines, labelStyle.Render(keyLabel)+"["+valueStyle.Render(m.memoryNewKey)+"]")
	}

	// Value field
	valueLabel := "Value: "
	if !m.memoryFocusKey {
		valueLabel = focusedStyle.Render(valueLabel)
		lines = append(lines, valueLabel+"["+m.memoryNewValue+"_]")
	} else {
		lines = append(lines, labelStyle.Render(valueLabel)+"["+valueStyle.Render(m.memoryNewValue)+"]")
	}

	lines = append(lines, "")
	lines = append(lines, hintStyle.Render("[Tab] Switch field  [Enter] Add  [Esc] Cancel"))

	return strings.Join(lines, "\n")
}

func (m *AssistantsModal) viewMemoryConfirmDiscard() string {
	var lines []string

	warningStyle := lipgloss.NewStyle().Foreground(theme.Warning)
	textStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	lines = append(lines, warningStyle.Render("Unsaved Changes"))
	lines = append(lines, "")
	lines = append(lines, textStyle.Render("You have unsaved memory changes."))
	lines = append(lines, "")
	lines = append(lines, hintStyle.Render("[s] Save & exit  [Enter] Discard & exit  [Esc] Cancel"))

	return strings.Join(lines, "\n")
}

func (m *AssistantsModal) viewTemplateList() string {
	var lines []string

	labelStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	moduleStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent)
	normalStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	if m.templateError != "" {
		lines = append(lines, errorStyle.Render("Error: "+m.templateError))
		lines = append(lines, "")
		lines = append(lines, hintStyle.Render("[Esc] Back"))
		return strings.Join(lines, "\n")
	}

	if len(m.templates) == 0 {
		lines = append(lines, labelStyle.Render("No assistant templates available."))
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("Templates are provided by modules. Enable modules"))
		lines = append(lines, labelStyle.Render("with assistant templates to see them here."))
		lines = append(lines, "")
		lines = append(lines, hintStyle.Render("[Esc] Back"))
		return strings.Join(lines, "\n")
	}

	// Group templates by module
	currentModule := ""
	for i, tmpl := range m.templates {
		// Add module header when module changes
		if tmpl.Module != currentModule {
			if currentModule != "" {
				lines = append(lines, "") // Add spacing between modules
			}
			lines = append(lines, moduleStyle.Render(tmpl.Module))
			currentModule = tmpl.Module
		}

		// Template entry
		isSelected := i == m.templateSelected
		indicator := "○"
		if isSelected {
			indicator = "●"
		}

		name := tmpl.Template.Name
		displayName := tmpl.Template.DisplayName

		var line string
		if isSelected {
			line = fmt.Sprintf("  %s %s", selectedStyle.Render(indicator), selectedStyle.Render(name))
		} else {
			line = fmt.Sprintf("  %s %s", dimStyle.Render(indicator), normalStyle.Render(name))
		}

		// Add display name if different
		if displayName != "" && displayName != name {
			padding := 20 - len(name)
			if padding < 2 {
				padding = 2
			}
			line += strings.Repeat(" ", padding) + dimStyle.Render(displayName)
		}

		lines = append(lines, line)
	}

	lines = append(lines, "")
	lines = append(lines, hintStyle.Render("[Enter] Select  [Esc] Cancel"))

	return strings.Join(lines, "\n")
}

func (m *AssistantsModal) viewTemplatePreview() string {
	if m.templateSelected >= len(m.templates) {
		return "No template selected"
	}

	tmpl := m.templates[m.templateSelected]
	var lines []string

	labelStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	valueStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	accentStyle := lipgloss.NewStyle().Foreground(theme.Accent)
	focusedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	if m.templateError != "" {
		lines = append(lines, errorStyle.Render("Error: "+m.templateError))
		lines = append(lines, "")
	}

	// Module
	lines = append(lines, labelStyle.Render("From module:   ")+accentStyle.Render(tmpl.Module))
	lines = append(lines, "")

	// Editable override fields
	nameLabel := "Name:          "
	if m.templateFocusName {
		nameLabel = focusedStyle.Render(nameLabel)
		lines = append(lines, nameLabel+"["+m.templateOverrideName+"_]")
	} else {
		lines = append(lines, labelStyle.Render(nameLabel)+"["+valueStyle.Render(m.templateOverrideName)+"]")
	}

	displayLabel := "Display Name:  "
	if !m.templateFocusName {
		displayLabel = focusedStyle.Render(displayLabel)
		lines = append(lines, displayLabel+"["+m.templateOverrideDisplay+"_]")
	} else {
		lines = append(lines, labelStyle.Render(displayLabel)+"["+valueStyle.Render(m.templateOverrideDisplay)+"]")
	}

	// Divider
	lines = append(lines, "")
	lines = append(lines, labelStyle.Render(strings.Repeat("─", 55)))
	lines = append(lines, "")

	// Template details (read-only preview)
	lines = append(lines, labelStyle.Render("Persona:"))

	personaStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border).
		Padding(0, 1).
		Width(55)

	persona := tmpl.Template.Persona
	if len(persona) > 300 {
		persona = persona[:297] + "..."
	}
	if persona == "" {
		persona = "(none)"
	}
	lines = append(lines, personaStyle.Render(persona))

	lines = append(lines, "")

	// Modules
	if len(tmpl.Template.Modules) > 0 {
		lines = append(lines, labelStyle.Render("Modules:       ")+valueStyle.Render(strings.Join(tmpl.Template.Modules, ", ")))
	}

	// Profile
	if tmpl.Template.LLMProfile != "" {
		lines = append(lines, labelStyle.Render("Profile:       ")+accentStyle.Render(tmpl.Template.LLMProfile))
	}

	// Keywords
	if len(tmpl.Template.Keywords) > 0 {
		lines = append(lines, labelStyle.Render("Keywords:      ")+valueStyle.Render(strings.Join(tmpl.Template.Keywords, ", ")))
	}

	lines = append(lines, "")
	lines = append(lines, hintStyle.Render("[Tab] Switch field  [Enter] Create  [Esc] Cancel"))

	return strings.Join(lines, "\n")
}
