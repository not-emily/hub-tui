package chat

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/ui/theme"
)

// Autocomplete manages completion suggestions.
type Autocomplete struct {
	visible     bool
	suggestions []string
	selected    int
	prefix      InputPrefix
	partial     string // The partial text being completed
	width       int
}

// NewAutocomplete creates a new autocomplete component.
func NewAutocomplete() Autocomplete {
	return Autocomplete{}
}

// SetWidth sets the display width.
func (a *Autocomplete) SetWidth(width int) {
	a.width = width
}

// Show displays the autocomplete with the given suggestions.
func (a *Autocomplete) Show(prefix InputPrefix, partial string, suggestions []string) {
	a.visible = true
	a.suggestions = suggestions
	a.selected = 0
	a.prefix = prefix
	a.partial = partial
}

// Hide hides the autocomplete.
func (a *Autocomplete) Hide() {
	a.visible = false
	a.suggestions = nil
	a.selected = 0
}

// IsVisible returns true if autocomplete is showing.
func (a Autocomplete) IsVisible() bool {
	return a.visible
}

// MoveUp moves selection up.
func (a *Autocomplete) MoveUp() {
	if a.selected > 0 {
		a.selected--
	} else {
		a.selected = len(a.suggestions) - 1
	}
}

// MoveDown moves selection down.
func (a *Autocomplete) MoveDown() {
	if a.selected < len(a.suggestions)-1 {
		a.selected++
	} else {
		a.selected = 0
	}
}

// Selected returns the currently selected suggestion.
func (a Autocomplete) Selected() string {
	if a.selected >= 0 && a.selected < len(a.suggestions) {
		return a.suggestions[a.selected]
	}
	return ""
}

// Prefix returns the prefix character for the current completion.
func (a Autocomplete) Prefix() InputPrefix {
	return a.prefix
}

// View renders the autocomplete menu.
func (a Autocomplete) View() string {
	if !a.visible || len(a.suggestions) == 0 {
		return ""
	}

	menuStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.Surface).
		Padding(0, 1).
		Width(a.width - 4)

	var items []string
	for i, s := range a.suggestions {
		style := lipgloss.NewStyle().Foreground(theme.TextPrimary)
		if i == a.selected {
			style = style.Background(theme.Surface).Bold(true)
		}
		items = append(items, style.Render(s))
	}

	return menuStyle.Render(strings.Join(items, "\n"))
}

// FilterSuggestions filters a list of items by partial match.
func FilterSuggestions(items []string, partial string) []string {
	if partial == "" {
		return items
	}

	partial = strings.ToLower(partial)
	var matches []string
	for _, item := range items {
		if strings.Contains(strings.ToLower(item), partial) {
			matches = append(matches, item)
		}
	}
	return matches
}
