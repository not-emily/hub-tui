package modal

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/ui/theme"
)

// Modal defines the interface for modal overlays.
type Modal interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (Modal, tea.Cmd)
	View() string
	Title() string
}

// State tracks the currently active modal.
type State struct {
	Active Modal
	width  int
}

// NewState creates a new modal state.
func NewState() State {
	return State{}
}

// SetWidth updates the available width for modals.
func (s *State) SetWidth(width int) {
	s.width = width
}

// IsOpen returns true if a modal is currently open.
func (s *State) IsOpen() bool {
	return s.Active != nil
}

// Open opens a modal.
func (s *State) Open(m Modal) tea.Cmd {
	s.Active = m
	return m.Init()
}

// Close closes the current modal.
func (s *State) Close() {
	s.Active = nil
}

// Update handles input for the active modal.
// Returns true if the modal handled the message.
func (s *State) Update(msg tea.Msg) (bool, tea.Cmd) {
	if s.Active == nil {
		return false, nil
	}

	// Handle Esc to close
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "esc" {
			s.Active = nil
			return true, nil
		}
	}

	// Forward to modal
	var cmd tea.Cmd
	s.Active, cmd = s.Active.Update(msg)
	return true, cmd
}

// View renders the modal inline (not as overlay).
func (s *State) View() string {
	if s.Active == nil {
		return ""
	}

	// Title style
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Accent)

	// Hint style
	hintStyle := lipgloss.NewStyle().
		Foreground(theme.TextSecondary)

	// Build title bar: title on left, hint on right
	title := titleStyle.Render(s.Active.Title())
	hint := hintStyle.Render("Esc to close")

	// Calculate padding between title and hint
	// Border takes 2 chars (left + right), padding takes 2 chars (1 each side)
	innerWidth := s.width - 4
	titleWidth := lipgloss.Width(title)
	hintWidth := lipgloss.Width(hint)
	padding := innerWidth - titleWidth - hintWidth
	if padding < 1 {
		padding = 1
	}

	titleBar := title + repeatChar(' ', padding) + hint

	// Box style with border
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Accent).
		Padding(0, 1).
		Width(s.width - 2) // Account for border

	// Build modal content
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		titleBar,
		"",
		s.Active.View(),
	)

	return boxStyle.Render(content)
}

// repeatChar repeats a character n times.
func repeatChar(ch rune, n int) string {
	if n <= 0 {
		return ""
	}
	result := make([]rune, n)
	for i := range result {
		result[i] = ch
	}
	return string(result)
}
