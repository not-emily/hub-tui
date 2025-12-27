package chat

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/ui/theme"
)

// Input is the chat input component.
type Input struct {
	textarea textarea.Model
	width    int
}

// NewInput creates a new chat input.
func NewInput() Input {
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.ShowLineNumbers = false
	ta.CharLimit = 4096
	ta.SetHeight(1)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(theme.TextSecondary)
	ta.FocusedStyle.Text = lipgloss.NewStyle().Foreground(theme.TextPrimary)
	ta.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(theme.Accent)
	ta.BlurredStyle = ta.FocusedStyle
	ta.Prompt = "> "
	ta.Focus()

	return Input{
		textarea: ta,
	}
}

// SetWidth sets the input width.
func (i *Input) SetWidth(width int) {
	i.width = width
	i.textarea.SetWidth(width - 2) // Account for border/padding
}

// Focus focuses the input.
func (i *Input) Focus() {
	i.textarea.Focus()
}

// Blur unfocuses the input.
func (i *Input) Blur() {
	i.textarea.Blur()
}

// Value returns the current input text.
func (i Input) Value() string {
	return strings.TrimSpace(i.textarea.Value())
}

// SetValue sets the input text.
func (i *Input) SetValue(s string) {
	i.textarea.SetValue(s)
}

// Reset clears the input.
func (i *Input) Reset() {
	i.textarea.Reset()
	i.textarea.SetHeight(1)
}

// IsEmpty returns true if the input is empty.
func (i Input) IsEmpty() bool {
	return i.Value() == ""
}

// Update handles input events.
func (i Input) Update(msg tea.Msg) (Input, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// Don't handle enter here - let parent handle submission
			return i, nil
		case "ctrl+j", "alt+enter":
			// Ctrl+J inserts newline (standard terminal newline)
			// Alt+Enter also works in some terminals
			i.textarea.InsertString("\n")
			// Grow input if needed (up to 5 lines)
			lines := strings.Count(i.textarea.Value(), "\n") + 1
			if lines > 1 && lines <= 5 {
				i.textarea.SetHeight(lines)
			}
			return i, nil
		}
	}

	i.textarea, cmd = i.textarea.Update(msg)
	return i, cmd
}

// View renders the input.
func (i Input) View() string {
	inputStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(theme.Surface).
		Width(i.width).
		MarginBottom(1)

	return inputStyle.Render(i.textarea.View())
}
