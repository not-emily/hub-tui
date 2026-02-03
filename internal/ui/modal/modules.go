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

// View states for the modules modal.
const (
	moduleViewList = iota
	moduleViewDetail
	moduleViewAvailable
	moduleViewConfirmUninstall
)

// ModulesModal displays and manages modules.
type ModulesModal struct {
	client   *client.Client
	isAdmin  bool
	modules  []client.Module
	selected int
	loading  bool
	error    string

	// View state
	view         int
	detailSource int            // Which view we came from (list or available)
	current      *client.Module // Selected installed module for detail view

	// Available modules (admin)
	availableModules  []client.AvailableModule
	availableSelected int
	availableLoading  bool
	currentAvailable  *client.AvailableModule // Selected available module for detail view

	// Confirmation
	confirm *components.Confirmation

	// Uninstall state (admin)
	uninstallResult *client.UninstallResult
	uninstallTarget string

	// Update tracking (admin)
	availableByName map[string]*client.AvailableModule
}

// NewModulesModal creates a new modules modal.
func NewModulesModal(c *client.Client, isAdmin bool) *ModulesModal {
	return &ModulesModal{
		client:  c,
		isAdmin: isAdmin,
		loading: true,
		view:    moduleViewList,
		confirm: components.NewConfirmation(),
	}
}

// ModulesLoadedMsg is sent when modules are loaded.
type ModulesLoadedMsg struct {
	Modules []client.Module
	Error   error
}

func (m ModulesLoadedMsg) IsAsyncModalMessage() {}
func (m ModulesLoadedMsg) AuthError() error     { return m.Error }

// ModuleToggledMsg is sent when a module is toggled.
type ModuleToggledMsg struct {
	Name    string
	Enabled bool
	Error   error
}

func (m ModuleToggledMsg) IsAsyncModalMessage() {}
func (m ModuleToggledMsg) AuthError() error     { return m.Error }

// AvailableModulesLoadedMsg is sent when available modules are loaded.
type AvailableModulesLoadedMsg struct {
	Modules []client.AvailableModule
	Error   error
}

func (m AvailableModulesLoadedMsg) IsAsyncModalMessage() {}
func (m AvailableModulesLoadedMsg) AuthError() error     { return m.Error }

// ModuleInstalledMsg is sent when a module is installed.
type ModuleInstalledMsg struct {
	Name  string
	Error error
}

func (m ModuleInstalledMsg) IsAsyncModalMessage() {}
func (m ModuleInstalledMsg) AuthError() error     { return m.Error }

// ModuleUninstallResultMsg is sent after initial uninstall attempt.
type ModuleUninstallResultMsg struct {
	Result *client.UninstallResult
	Error  error
}

func (m ModuleUninstallResultMsg) IsAsyncModalMessage() {}
func (m ModuleUninstallResultMsg) AuthError() error     { return m.Error }

// ModuleUninstalledMsg is sent when a module is force uninstalled.
type ModuleUninstalledMsg struct {
	Name  string
	Error error
}

func (m ModuleUninstalledMsg) IsAsyncModalMessage() {}
func (m ModuleUninstalledMsg) AuthError() error     { return m.Error }

// ModuleUpdatedMsg is sent when a module is updated.
type ModuleUpdatedMsg struct {
	Result *client.UpdateResult
	Error  error
}

func (m ModuleUpdatedMsg) IsAsyncModalMessage() {}
func (m ModuleUpdatedMsg) AuthError() error     { return m.Error }

// Init initializes the modal and triggers data fetch.
func (m *ModulesModal) Init() tea.Cmd {
	if m.isAdmin {
		// Load both lists for admins (to show update indicators)
		return tea.Batch(m.loadModules(), m.loadAvailableModules())
	}
	return m.loadModules()
}

func (m *ModulesModal) loadModules() tea.Cmd {
	return func() tea.Msg {
		modules, err := m.client.ListModules()
		return ModulesLoadedMsg{Modules: modules, Error: err}
	}
}

func (m *ModulesModal) loadAvailableModules() tea.Cmd {
	return func() tea.Msg {
		modules, err := m.client.ListAvailableModules()
		return AvailableModulesLoadedMsg{Modules: modules, Error: err}
	}
}

func (m *ModulesModal) installModule(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.InstallModule(name)
		return ModuleInstalledMsg{Name: name, Error: err}
	}
}

func (m *ModulesModal) uninstallModule(name string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.UninstallModule(name)
		return ModuleUninstallResultMsg{Result: result, Error: err}
	}
}

func (m *ModulesModal) uninstallModuleForce(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.UninstallModuleForce(name)
		return ModuleUninstalledMsg{Name: name, Error: err}
	}
}

func (m *ModulesModal) updateModule(name string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.UpdateModule(name)
		return ModuleUpdatedMsg{Result: result, Error: err}
	}
}

// hasUpdate returns true if the module has an update available.
func (m *ModulesModal) hasUpdate(moduleName string) bool {
	if avail, ok := m.availableByName[moduleName]; ok {
		return avail.UpdateAvailable
	}
	return false
}

// getAvailableVersion returns the latest available version for a module.
func (m *ModulesModal) getAvailableVersion(moduleName string) string {
	if avail, ok := m.availableByName[moduleName]; ok {
		return avail.Version
	}
	return ""
}


// Update handles input.
func (m *ModulesModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case ModulesLoadedMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			m.modules = msg.Modules
			m.error = ""
		}
		return m, nil

	case ModuleToggledMsg:
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			// Update local state
			for i, mod := range m.modules {
				if mod.Name == msg.Name {
					m.modules[i].Enabled = msg.Enabled
					// Update current if in detail view
					if m.current != nil && m.current.Name == msg.Name {
						m.current.Enabled = msg.Enabled
					}
					break
				}
			}
			m.error = ""
		}
		return m, nil

	case AvailableModulesLoadedMsg:
		m.availableLoading = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			m.availableModules = msg.Modules
			m.error = ""

			// Build lookup map for update tracking
			m.availableByName = make(map[string]*client.AvailableModule)
			for i := range m.availableModules {
				m.availableByName[m.availableModules[i].Name] = &m.availableModules[i]
			}

			// Update currentAvailable if we're viewing a module detail
			if m.currentAvailable != nil {
				if avail, ok := m.availableByName[m.currentAvailable.Name]; ok {
					m.currentAvailable = avail
				}
			}
		}
		return m, nil

	case ModuleInstalledMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			m.error = ""
			// Refresh both lists after install
			return m, tea.Batch(m.loadModules(), m.loadAvailableModules())
		}
		return m, nil

	case ModuleUninstallResultMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
			return m, nil
		}

		if msg.Result.ConfirmRequired {
			// Show confirmation with affected users
			m.uninstallResult = msg.Result
			m.uninstallTarget = msg.Result.Module
			m.view = moduleViewConfirmUninstall
		} else {
			// Uninstall succeeded without needing confirmation
			m.view = moduleViewAvailable
			m.currentAvailable = nil
			return m, tea.Batch(m.loadModules(), m.loadAvailableModules())
		}
		return m, nil

	case ModuleUninstalledMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
			m.view = moduleViewDetail
			return m, nil
		}

		// Clear uninstall state
		m.uninstallResult = nil
		m.uninstallTarget = ""
		m.currentAvailable = nil

		// Return to available view and refresh
		m.view = moduleViewAvailable
		return m, tea.Batch(m.loadModules(), m.loadAvailableModules())

	case ModuleUpdatedMsg:
		m.loading = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
			return m, nil
		}
		// Refresh both lists to get new version info
		return m, tea.Batch(m.loadModules(), m.loadAvailableModules())

	case components.ConfirmationExpiredMsg:
		m.confirm.HandleExpired(msg)
		return m, nil

	case tea.KeyMsg:
		switch m.view {
		case moduleViewList:
			return m.updateListView(msg)
		case moduleViewDetail:
			return m.updateDetailView(msg)
		case moduleViewAvailable:
			return m.updateAvailableView(msg)
		case moduleViewConfirmUninstall:
			return m.updateConfirmUninstallView(msg)
		}
	}
	return m, nil
}

func (m *ModulesModal) updateListView(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return nil, nil // Close modal
	case "up", "k":
		if m.selected > 0 {
			m.selected--
			m.confirm.Clear() // Clear confirmation on navigation
		}
	case "down", "j":
		if m.selected < len(m.modules)-1 {
			m.selected++
			m.confirm.Clear() // Clear confirmation on navigation
		}
	case "space", " ":
		if !m.loading && len(m.modules) > 0 {
			mod := m.modules[m.selected]
			if mod.Enabled {
				// Disable requires double-press confirmation
				if execute, cmd := m.confirm.Check("disable", mod.Name); execute {
					return m, m.disableModule()
				} else if cmd != nil {
					return m, cmd
				}
			} else {
				// Enable is immediate
				return m, m.enableModule()
			}
		}
	case "enter":
		if !m.loading && len(m.modules) > 0 {
			m.current = &m.modules[m.selected]
			m.detailSource = moduleViewList
			m.view = moduleViewDetail
			m.confirm.Clear()
		}
	case "b":
		if m.isAdmin {
			m.availableLoading = true
			m.view = moduleViewAvailable
			m.availableSelected = 0
			m.confirm.Clear()
			return m, m.loadAvailableModules()
		}
	case "u":
		if m.isAdmin && len(m.modules) > 0 {
			mod := m.modules[m.selected]
			if m.hasUpdate(mod.Name) {
				m.loading = true
				m.confirm.Clear()
				return m, m.updateModule(mod.Name)
			}
		}
	case "r":
		m.loading = true
		m.error = ""
		m.confirm.Clear()
		return m, m.loadModules()
	}
	return m, nil
}

func (m *ModulesModal) updateDetailView(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Return to the view we came from
		if m.detailSource == moduleViewAvailable {
			m.view = moduleViewAvailable
			m.currentAvailable = nil
		} else {
			m.view = moduleViewList
			m.current = nil
		}
		m.confirm.Clear()
	case "space", " ":
		// Only for installed modules
		if m.detailSource == moduleViewList && m.current != nil {
			if m.current.Enabled {
				// Disable requires double-press confirmation
				if execute, cmd := m.confirm.Check("disable", m.current.Name); execute {
					return m, m.disableCurrentModule()
				} else if cmd != nil {
					return m, cmd
				}
			} else {
				// Enable is immediate
				return m, m.enableCurrentModule()
			}
		}
	case "i":
		// Install available module (admin only)
		if m.isAdmin && m.detailSource == moduleViewAvailable && m.currentAvailable != nil {
			if !m.currentAvailable.Installed {
				m.loading = true
				return m, m.installModule(m.currentAvailable.Name)
			}
		}
	case "x":
		// Uninstall module (admin only, from available detail view, only if installed)
		if m.isAdmin && m.detailSource == moduleViewAvailable && m.currentAvailable != nil {
			if m.currentAvailable.Installed {
				m.loading = true
				return m, m.uninstallModule(m.currentAvailable.Name)
			}
		}
	case "u":
		// Update module (admin only)
		if m.isAdmin {
			var moduleName string
			if m.detailSource == moduleViewAvailable && m.currentAvailable != nil && m.currentAvailable.Installed {
				moduleName = m.currentAvailable.Name
			} else if m.detailSource == moduleViewList && m.current != nil {
				moduleName = m.current.Name
			}
			if moduleName != "" && m.hasUpdate(moduleName) {
				m.loading = true
				return m, m.updateModule(moduleName)
			}
		}
	}
	return m, nil
}

func (m *ModulesModal) updateConfirmUninstallView(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Force uninstall
		m.loading = true
		return m, m.uninstallModuleForce(m.uninstallTarget)
	case "esc":
		// Cancel - return to detail view
		m.view = moduleViewDetail
		m.uninstallResult = nil
		m.uninstallTarget = ""
	}
	return m, nil
}

func (m *ModulesModal) updateAvailableView(msg tea.KeyMsg) (Modal, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.view = moduleViewList
		m.confirm.Clear()
	case "up", "k":
		if m.availableSelected > 0 {
			m.availableSelected--
		}
	case "down", "j":
		if m.availableSelected < len(m.availableModules)-1 {
			m.availableSelected++
		}
	case "enter":
		if !m.availableLoading && len(m.availableModules) > 0 {
			m.currentAvailable = &m.availableModules[m.availableSelected]
			m.detailSource = moduleViewAvailable
			m.view = moduleViewDetail
		}
	case "r":
		m.availableLoading = true
		m.error = ""
		return m, m.loadAvailableModules()
	}
	return m, nil
}

func (m *ModulesModal) enableModule() tea.Cmd {
	if len(m.modules) == 0 {
		return nil
	}
	mod := m.modules[m.selected]
	return func() tea.Msg {
		err := m.client.EnableModule(mod.Name)
		return ModuleToggledMsg{Name: mod.Name, Enabled: true, Error: err}
	}
}

func (m *ModulesModal) disableModule() tea.Cmd {
	if len(m.modules) == 0 {
		return nil
	}
	mod := m.modules[m.selected]
	return func() tea.Msg {
		err := m.client.DisableModule(mod.Name)
		return ModuleToggledMsg{Name: mod.Name, Enabled: false, Error: err}
	}
}

func (m *ModulesModal) enableCurrentModule() tea.Cmd {
	if m.current == nil {
		return nil
	}
	name := m.current.Name
	return func() tea.Msg {
		err := m.client.EnableModule(name)
		return ModuleToggledMsg{Name: name, Enabled: true, Error: err}
	}
}

func (m *ModulesModal) disableCurrentModule() tea.Cmd {
	if m.current == nil {
		return nil
	}
	name := m.current.Name
	return func() tea.Msg {
		err := m.client.DisableModule(name)
		return ModuleToggledMsg{Name: name, Enabled: false, Error: err}
	}
}

// Title returns the modal title.
func (m *ModulesModal) Title() string {
	switch m.view {
	case moduleViewDetail:
		if m.detailSource == moduleViewAvailable && m.currentAvailable != nil {
			return "Module: " + m.currentAvailable.Name
		}
		if m.current != nil {
			return "Module: " + m.current.Name
		}
		return "Module"
	case moduleViewAvailable:
		return "Browse Available Modules"
	case moduleViewConfirmUninstall:
		return "Confirm Uninstall"
	default:
		return "Modules"
	}
}

// View renders the modal content.
func (m *ModulesModal) View() string {
	if m.loading || m.availableLoading {
		return lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("Loading...")
	}

	if m.error != "" {
		errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
		hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			errorStyle.Render("Error: "+m.error),
			"",
			hintStyle.Render("[r] Retry  [Esc] Back"),
		)
	}

	switch m.view {
	case moduleViewDetail:
		return m.viewDetail()
	case moduleViewAvailable:
		return m.viewAvailable()
	case moduleViewConfirmUninstall:
		return m.viewConfirmUninstall()
	default:
		return m.viewList()
	}
}

func (m *ModulesModal) viewList() string {
	if len(m.modules) == 0 {
		hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
		hints := "[r] Refresh"
		if m.isAdmin {
			hints += "  [b] Browse"
		}
		return lipgloss.JoinVertical(
			lipgloss.Left,
			hintStyle.Render("No modules installed."),
			"",
			hintStyle.Render(hints),
		)
	}

	var lines []string

	enabledStyle := lipgloss.NewStyle().Foreground(theme.Success)
	disabledStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	descStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	updateStyle := lipgloss.NewStyle().Foreground(theme.Warning)

	hasAnyUpdates := false
	for i, mod := range m.modules {
		// Status indicator
		var indicator string
		if mod.Enabled {
			indicator = enabledStyle.Render("●")
		} else {
			indicator = disabledStyle.Render("○")
		}

		// Name with selection highlight
		var name string
		if i == m.selected {
			name = selectedStyle.Render(mod.Name)
		} else {
			name = normalStyle.Render(mod.Name)
		}

		// Build line
		line := fmt.Sprintf("  %s %s", indicator, name)

		// Pad name
		padding := 20 - len(mod.Name)
		if padding < 2 {
			padding = 2
		}
		line += strings.Repeat(" ", padding)

		// Show version with update indicator if available
		hasUpdate := m.hasUpdate(mod.Name)
		if hasUpdate {
			hasAnyUpdates = true
			availVersion := m.getAvailableVersion(mod.Name)
			line += updateStyle.Render(mod.Version+" → "+availVersion+" ⬆")
		} else if mod.Version != "" {
			line += descStyle.Render(mod.Version)
		}

		// Add description if there's room
		if mod.Description != "" {
			line += "  " + descStyle.Render(mod.Description)
		}

		lines = append(lines, line)
	}

	// Add legend and hints
	lines = append(lines, "")
	legendStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	legend := "  ● enabled  ○ disabled"
	if hasAnyUpdates {
		legend += "  " + updateStyle.Render("⬆ update available")
	}
	lines = append(lines, legend)
	lines = append(lines, "")

	// Build hints based on state
	if len(m.modules) > 0 {
		mod := m.modules[m.selected]
		if mod.Enabled && m.confirm.IsPending("disable", mod.Name) {
			warningStyle := lipgloss.NewStyle().Foreground(theme.Warning)
			lines = append(lines, warningStyle.Render("  Press Space again to disable"))
		} else {
			toggleAction := "Enable"
			if mod.Enabled {
				toggleAction = "Disable"
			}
			hints := "  [Space] " + toggleAction + "  [Enter] Detail"
			if m.isAdmin && m.hasUpdate(mod.Name) {
				hints += "  [u] Update"
			}
			hints += "  [r] Refresh"
			if m.isAdmin {
				hints += "  [b] Browse"
			}
			lines = append(lines, legendStyle.Render(hints))
		}
	} else {
		hints := "  [r] Refresh"
		if m.isAdmin {
			hints += "  [b] Browse"
		}
		lines = append(lines, legendStyle.Render(hints))
	}

	return strings.Join(lines, "\n")
}

func (m *ModulesModal) viewDetail() string {
	// Route to appropriate detail view based on source
	if m.detailSource == moduleViewAvailable && m.currentAvailable != nil {
		return m.viewAvailableDetail()
	}
	return m.viewInstalledDetail()
}

func (m *ModulesModal) viewInstalledDetail() string {
	if m.current == nil {
		return ""
	}

	mod := m.current
	var lines []string

	titleStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	valueStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	enabledStyle := lipgloss.NewStyle().Foreground(theme.Success)
	disabledStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	// Title
	lines = append(lines, titleStyle.Render(mod.Name))
	lines = append(lines, "")

	// Description
	if mod.Description != "" {
		lines = append(lines, valueStyle.Render(mod.Description))
		lines = append(lines, "")
	}

	// Version with update indicator
	hasUpdate := m.hasUpdate(mod.Name)
	if mod.Version != "" {
		versionLine := labelStyle.Render("Version: ") + valueStyle.Render(mod.Version)
		if hasUpdate {
			updateStyle := lipgloss.NewStyle().Foreground(theme.Warning)
			availVersion := m.getAvailableVersion(mod.Name)
			versionLine += "  " + updateStyle.Render("⬆ "+availVersion+" available")
		}
		lines = append(lines, versionLine)
	}

	// Status
	var statusText string
	if mod.Enabled {
		statusText = enabledStyle.Render("Enabled")
	} else {
		statusText = disabledStyle.Render("Disabled")
	}
	lines = append(lines, labelStyle.Render("Status: ")+statusText)

	// Tools
	if len(mod.Tools) > 0 {
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("Tools:"))
		for _, tool := range mod.Tools {
			lines = append(lines, "  • "+valueStyle.Render(tool))
		}
	}

	// Hints
	lines = append(lines, "")
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	warningStyle := lipgloss.NewStyle().Foreground(theme.Warning)

	if mod.Enabled && m.confirm.IsPending("disable", mod.Name) {
		lines = append(lines, warningStyle.Render("Press Space again to disable")+"  "+hintStyle.Render("[Esc] Back"))
	} else if mod.Enabled {
		hints := "[Space] Disable"
		if m.isAdmin && hasUpdate {
			hints += "  [u] Update"
		}
		hints += "  [Esc] Back"
		lines = append(lines, hintStyle.Render(hints))
	} else {
		hints := "[Space] Enable"
		if m.isAdmin && hasUpdate {
			hints += "  [u] Update"
		}
		hints += "  [Esc] Back"
		lines = append(lines, hintStyle.Render(hints))
	}

	return strings.Join(lines, "\n")
}

func (m *ModulesModal) viewAvailableDetail() string {
	if m.currentAvailable == nil {
		return ""
	}

	mod := m.currentAvailable
	var lines []string

	titleStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	valueStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	installedStyle := lipgloss.NewStyle().Foreground(theme.Success)

	// Title
	lines = append(lines, titleStyle.Render(mod.Name))
	lines = append(lines, "")

	// Description
	if mod.Description != "" {
		lines = append(lines, valueStyle.Render(mod.Description))
		lines = append(lines, "")
	}

	// Version
	lines = append(lines, labelStyle.Render("Version: ")+valueStyle.Render(mod.Version))

	// Install status
	if mod.Installed {
		lines = append(lines, labelStyle.Render("Status: ")+installedStyle.Render("Installed")+" ("+mod.InstalledVersion+")")
		if mod.UpdateAvailable {
			updateStyle := lipgloss.NewStyle().Foreground(theme.Warning)
			lines = append(lines, updateStyle.Render("  ⬆ Update available: "+mod.Version))
		}
	} else {
		lines = append(lines, labelStyle.Render("Status: ")+valueStyle.Render("Not installed"))
	}

	// Keywords
	if len(mod.Keywords) > 0 {
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("Keywords: ")+valueStyle.Render(strings.Join(mod.Keywords, ", ")))
	}

	// Hints
	lines = append(lines, "")
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	if mod.Installed {
		if m.isAdmin {
			hints := ""
			if mod.UpdateAvailable {
				hints += "[u] Update  "
			}
			hints += "[x] Uninstall  [Esc] Back"
			lines = append(lines, hintStyle.Render(hints))
		} else {
			lines = append(lines, hintStyle.Render("[Esc] Back"))
		}
	} else {
		lines = append(lines, hintStyle.Render("[i] Install  [Esc] Back"))
	}

	return strings.Join(lines, "\n")
}

func (m *ModulesModal) viewConfirmUninstall() string {
	if m.uninstallResult == nil {
		return ""
	}

	var lines []string

	warningStyle := lipgloss.NewStyle().Foreground(theme.Warning).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	valueStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	// Warning header
	lines = append(lines, warningStyle.Render("Uninstall \""+m.uninstallTarget+"\"?"))
	lines = append(lines, "")

	// Affected users
	if len(m.uninstallResult.AffectedUsers) > 0 {
		lines = append(lines, labelStyle.Render(fmt.Sprintf("This module is enabled by %d user(s):", len(m.uninstallResult.AffectedUsers))))
		for _, user := range m.uninstallResult.AffectedUsers {
			lines = append(lines, "  • "+valueStyle.Render(user))
		}
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("Uninstalling will disable it for these users."))
	}

	// Hints
	lines = append(lines, "")
	lines = append(lines, hintStyle.Render("[Enter] Uninstall  [Esc] Cancel"))

	return strings.Join(lines, "\n")
}

func (m *ModulesModal) viewAvailable() string {
	if len(m.availableModules) == 0 {
		hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			hintStyle.Render("No modules available in registry."),
			"",
			hintStyle.Render("[Esc] Back  [r] Refresh"),
		)
	}

	var lines []string

	installedStyle := lipgloss.NewStyle().Foreground(theme.Success)
	notInstalledStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	descStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	versionStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	for i, mod := range m.availableModules {
		// Status indicator
		var indicator string
		if mod.Installed {
			indicator = installedStyle.Render("●")
		} else {
			indicator = notInstalledStyle.Render("○")
		}

		// Name with selection highlight
		var name string
		if i == m.availableSelected {
			name = selectedStyle.Render(mod.Name)
		} else {
			name = normalStyle.Render(mod.Name)
		}

		// Build line with version and description
		line := fmt.Sprintf("  %s %s", indicator, name)

		// Pad name to align versions
		padding := 20 - len(mod.Name)
		if padding < 2 {
			padding = 2
		}
		line += strings.Repeat(" ", padding) + versionStyle.Render(mod.Version)

		if mod.Description != "" {
			line += "  " + descStyle.Render(mod.Description)
		}

		lines = append(lines, line)
	}

	// Add legend and hints
	lines = append(lines, "")
	legendStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	lines = append(lines, legendStyle.Render("  ● installed  ○ not installed"))
	lines = append(lines, "")
	lines = append(lines, legendStyle.Render("  [Enter] Detail  [Esc] Back  [r] Refresh"))

	return strings.Join(lines, "\n")
}
