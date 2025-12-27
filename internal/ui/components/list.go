package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/ui/theme"
)

// ListItem represents a single item in a list.
type ListItem struct {
	Label       string
	Description string
}

// List is a reusable navigable list component.
type List struct {
	items    []ListItem
	selected int
	height   int
	offset   int // For scrolling
}

// NewList creates a new list with the given items.
func NewList(items []ListItem) List {
	return List{
		items:    items,
		selected: 0,
		height:   10,
		offset:   0,
	}
}

// NewSimpleList creates a list from string labels.
func NewSimpleList(labels []string) List {
	items := make([]ListItem, len(labels))
	for i, label := range labels {
		items[i] = ListItem{Label: label}
	}
	return NewList(items)
}

// SetHeight sets the visible height of the list.
func (l *List) SetHeight(height int) {
	l.height = height
}

// Items returns the list items.
func (l *List) Items() []ListItem {
	return l.items
}

// Selected returns the currently selected index.
func (l *List) Selected() int {
	return l.selected
}

// SelectedItem returns the currently selected item, or nil if empty.
func (l *List) SelectedItem() *ListItem {
	if len(l.items) == 0 {
		return nil
	}
	return &l.items[l.selected]
}

// Update handles keyboard navigation.
func (l *List) Update(msg tea.Msg) {
	if len(l.items) == 0 {
		return
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			l.selected--
			if l.selected < 0 {
				l.selected = 0
			}
			l.ensureVisible()
		case "down", "j":
			l.selected++
			if l.selected >= len(l.items) {
				l.selected = len(l.items) - 1
			}
			l.ensureVisible()
		case "home", "g":
			l.selected = 0
			l.offset = 0
		case "end", "G":
			l.selected = len(l.items) - 1
			l.ensureVisible()
		}
	}
}

// ensureVisible adjusts offset to keep selected item visible.
func (l *List) ensureVisible() {
	if l.selected < l.offset {
		l.offset = l.selected
	}
	if l.selected >= l.offset+l.height {
		l.offset = l.selected - l.height + 1
	}
}

// View renders the list.
func (l *List) View() string {
	if len(l.items) == 0 {
		return lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Render("(empty)")
	}

	selectedStyle := lipgloss.NewStyle().
		Foreground(theme.Accent).
		Bold(true)

	normalStyle := lipgloss.NewStyle().
		Foreground(theme.TextPrimary)

	descStyle := lipgloss.NewStyle().
		Foreground(theme.TextSecondary)

	var lines []string

	// Calculate visible range
	end := l.offset + l.height
	if end > len(l.items) {
		end = len(l.items)
	}

	for i := l.offset; i < end; i++ {
		item := l.items[i]
		var line string

		if i == l.selected {
			line = selectedStyle.Render("› " + item.Label)
		} else {
			line = normalStyle.Render("  " + item.Label)
		}

		if item.Description != "" {
			line += " " + descStyle.Render(item.Description)
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}
