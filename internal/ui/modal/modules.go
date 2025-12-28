package modal

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/client"
	"github.com/pxp/hub-tui/internal/ui/theme"
)

// ModulesModal displays and manages modules.
type ModulesModal struct {
	client   *client.Client
	modules  []client.Module
	selected int
	loading  bool
	error    string
}

// NewModulesModal creates a new modules modal.
func NewModulesModal(c *client.Client) *ModulesModal {
	return &ModulesModal{
		client:  c,
		loading: true,
	}
}

// ModulesLoadedMsg is sent when modules are loaded.
type ModulesLoadedMsg struct {
	Modules []client.Module
	Error   error
}

// ModuleToggledMsg is sent when a module is toggled.
type ModuleToggledMsg struct {
	Name    string
	Enabled bool
	Error   error
}

// Init initializes the modal and triggers data fetch.
func (m *ModulesModal) Init() tea.Cmd {
	return m.loadModules()
}

func (m *ModulesModal) loadModules() tea.Cmd {
	return func() tea.Msg {
		modules, err := m.client.ListModules()
		return ModulesLoadedMsg{Modules: modules, Error: err}
	}
}

func (m *ModulesModal) toggleModule() tea.Cmd {
	if len(m.modules) == 0 {
		return nil
	}
	mod := m.modules[m.selected]
	return func() tea.Msg {
		var err error
		if mod.Enabled {
			err = m.client.DisableModule(mod.Name)
		} else {
			err = m.client.EnableModule(mod.Name)
		}
		return ModuleToggledMsg{Name: mod.Name, Enabled: !mod.Enabled, Error: err}
	}
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
					break
				}
			}
			m.error = ""
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return nil, nil // Close modal
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.modules)-1 {
				m.selected++
			}
		case "enter":
			if !m.loading && len(m.modules) > 0 {
				return m, m.toggleModule()
			}
		case "r":
			m.loading = true
			m.error = ""
			return m, m.loadModules()
		}
	}
	return m, nil
}

// Title returns the modal title.
func (m *ModulesModal) Title() string {
	return "Modules"
}

// View renders the modal content.
func (m *ModulesModal) View() string {
	if m.loading {
		return lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("Loading modules...")
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

	if len(m.modules) == 0 {
		return lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("No modules found.")
	}

	var lines []string

	enabledStyle := lipgloss.NewStyle().Foreground(theme.Success)
	disabledStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
	descStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

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

		// Build line with description
		line := fmt.Sprintf("  %s %s", indicator, name)
		if mod.Description != "" {
			// Pad name to align descriptions
			padding := 20 - len(mod.Name)
			if padding < 2 {
				padding = 2
			}
			line += strings.Repeat(" ", padding) + descStyle.Render(mod.Description)
		}

		lines = append(lines, line)
	}

	// Add legend and hints
	lines = append(lines, "")
	legendStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	lines = append(lines, legendStyle.Render("  ● enabled  ○ disabled"))
	lines = append(lines, "")
	lines = append(lines, legendStyle.Render("  [Enter] Toggle  [r] Refresh"))

	return strings.Join(lines, "\n")
}
