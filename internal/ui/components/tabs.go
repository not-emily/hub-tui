package components

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/ui/theme"
)

// Tabs is a reusable tab component for modal views.
type Tabs struct {
	tabs        []string // Tab labels
	activeIndex int      // Currently selected tab (0-indexed)
	width       int      // Available width for rendering
}

// NewTabs creates a new tabs component with the given labels.
func NewTabs(labels []string) *Tabs {
	if len(labels) == 0 {
		panic("tabs: must provide at least one label")
	}
	return &Tabs{
		tabs:        labels,
		activeIndex: 0,
		width:       80, // Default width
	}
}

// SetActive sets the active tab by index.
// If index is out of bounds, it does nothing.
func (t *Tabs) SetActive(index int) {
	if index >= 0 && index < len(t.tabs) {
		t.activeIndex = index
	}
}

// ActiveIndex returns the current active tab index.
func (t *Tabs) ActiveIndex() int {
	return t.activeIndex
}

// Next moves to the next tab (wraps around to first tab).
func (t *Tabs) Next() {
	t.activeIndex = (t.activeIndex + 1) % len(t.tabs)
}

// Previous moves to the previous tab (wraps around to last tab).
func (t *Tabs) Previous() {
	t.activeIndex--
	if t.activeIndex < 0 {
		t.activeIndex = len(t.tabs) - 1
	}
}

// SetWidth sets the rendering width.
func (t *Tabs) SetWidth(width int) {
	t.width = width
}

// View renders the tab bar with active/inactive styling.
func (t *Tabs) View() string {
	var tabs []string
	for i, label := range t.tabs {
		var style lipgloss.Style
		if i == t.activeIndex {
			// Active tab: bold, highlighted
			style = lipgloss.NewStyle().
				Bold(true).
				Foreground(theme.TextPrimary).
				Background(theme.BorderActive).
				Padding(0, 2)
		} else {
			// Inactive tab: normal, dimmed
			style = lipgloss.NewStyle().
				Foreground(theme.TextSecondary).
				Background(theme.Background).
				Padding(0, 2)
		}
		tabs = append(tabs, style.Render(label))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}
