package modal

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/ui/theme"
)

// HelpModal displays command and keyboard reference.
type HelpModal struct {
	scroll int
	height int
}

// NewHelpModal creates a new help modal.
func NewHelpModal() *HelpModal {
	return &HelpModal{
		scroll: 0,
		height: 14, // Visible lines
	}
}

// Init initializes the modal.
func (m *HelpModal) Init() tea.Cmd {
	return nil
}

// Update handles input.
func (m *HelpModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return nil, nil // Close modal
		case "up", "k":
			if m.scroll > 0 {
				m.scroll--
			}
		case "down", "j":
			maxScroll := m.contentLen() - m.height
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.scroll < maxScroll {
				m.scroll++
			}
		}
	}
	return m, nil
}

// contentLen returns the number of lines in the help content.
func (m *HelpModal) contentLen() int {
	return 27 // Update this if content changes
}

// Title returns the modal title.
func (m *HelpModal) Title() string {
	return "Help"
}

// View renders the help content.
func (m *HelpModal) View() string {
	headerStyle := lipgloss.NewStyle().
		Foreground(theme.Accent).
		Bold(true)

	cmdStyle := lipgloss.NewStyle().
		Foreground(theme.TextPrimary)

	descStyle := lipgloss.NewStyle().
		Foreground(theme.TextSecondary)

	content := []string{
		headerStyle.Render("Commands"),
		"",
		cmdStyle.Render("  @{assistant}") + descStyle.Render("  Switch to assistant"),
		cmdStyle.Render("  #{workflow} ") + descStyle.Render("  Run workflow"),
		"",
		cmdStyle.Render("  /hub        ") + descStyle.Render("  Return to hub context"),
		cmdStyle.Render("  /modules    ") + descStyle.Render("  Manage modules"),
		cmdStyle.Render("  /integrations") + descStyle.Render(" Configure integrations"),
		cmdStyle.Render("  /workflows  ") + descStyle.Render("  Browse workflows"),
		cmdStyle.Render("  /tasks      ") + descStyle.Render("  View tasks"),
		cmdStyle.Render("  /settings   ") + descStyle.Render("  Settings"),
		cmdStyle.Render("  /help       ") + descStyle.Render("  This help"),
		cmdStyle.Render("  /clear      ") + descStyle.Render("  Clear chat"),
		cmdStyle.Render("  /refresh    ") + descStyle.Render("  Refresh cache"),
		cmdStyle.Render("  /exit       ") + descStyle.Render("  Exit"),
		"",
		headerStyle.Render("Keyboard"),
		"",
		cmdStyle.Render("  Enter    ") + descStyle.Render("  Send / Select"),
		cmdStyle.Render("  Ctrl+J   ") + descStyle.Render("  New line"),
		cmdStyle.Render("  Tab      ") + descStyle.Render("  Autocomplete"),
		cmdStyle.Render("  Ctrl+C   ") + descStyle.Render("  Exit (×2)"),
		cmdStyle.Render("  Esc      ") + descStyle.Render("  Back / Cancel"),
		cmdStyle.Render("  q        ") + descStyle.Render("  Close modal"),
		cmdStyle.Render("  j/k      ") + descStyle.Render("  Navigate lists"),
		cmdStyle.Render("  ↑/↓      ") + descStyle.Render("  Scroll chat"),
	}

	// Apply scrolling
	start := m.scroll
	if start >= len(content) {
		start = len(content) - 1
	}
	if start < 0 {
		start = 0
	}

	end := start + m.height
	if end > len(content) {
		end = len(content)
	}

	visible := content[start:end]
	return strings.Join(visible, "\n")
}
