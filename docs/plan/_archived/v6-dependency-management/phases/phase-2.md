# Phase 2: Tab Component

> **Depends on:** None
> **Enables:** Phase 4 (Dependencies tab uses tabs component)
>
> See: [Full Plan](../plan.md)

## Goal

Create a reusable tab component for modal views that can be used in integrations modal and future modals.

## Key Deliverables

- `Tabs` component with state management and rendering
- Tab switching methods (Next, Previous, SetActive)
- Styled tab bar (active tab highlighted, inactive tabs dimmed)
- Width-aware rendering

## Files to Create

- `internal/ui/components/tabs.go` — Tab component implementation

## Dependencies

**Internal:** None (standalone component)

**External:**
- `github.com/charmbracelet/lipgloss` — Styling

## Implementation Notes

### Tabs Structure

```go
type Tabs struct {
    tabs        []string  // Tab labels
    activeIndex int       // Currently selected tab (0-indexed)
    width       int       // Available width for rendering
}
```

### Constructor

```go
func NewTabs(labels []string) *Tabs {
    return &Tabs{
        tabs:        labels,
        activeIndex: 0,
        width:       80, // Default width
    }
}
```

### Methods

```go
// SetActive sets the active tab by index
func (t *Tabs) SetActive(index int) {
    if index >= 0 && index < len(t.tabs) {
        t.activeIndex = index
    }
}

// ActiveIndex returns the current active tab index
func (t *Tabs) ActiveIndex() int {
    return t.activeIndex
}

// Next moves to the next tab (wraps around)
func (t *Tabs) Next() {
    t.activeIndex = (t.activeIndex + 1) % len(t.tabs)
}

// Previous moves to the previous tab (wraps around)
func (t *Tabs) Previous() {
    t.activeIndex--
    if t.activeIndex < 0 {
        t.activeIndex = len(t.tabs) - 1
    }
}

// SetWidth sets the rendering width
func (t *Tabs) SetWidth(width int) {
    t.width = width
}
```

### Rendering

```go
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

    tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

    // Add bottom border
    border := lipgloss.NewStyle().
        Foreground(theme.Border).
        Width(t.width).
        Render(strings.Repeat("─", t.width))

    return lipgloss.JoinVertical(lipgloss.Left, tabBar, border)
}
```

### Styling Notes

Use existing theme colors:
- Active tab: `theme.TextPrimary` text, `theme.BorderActive` background
- Inactive tab: `theme.TextSecondary` text, `theme.Background` background
- Border: `theme.Border` color

Match the visual style of other modals (clean, minimal, keyboard-first).

### Usage Pattern

Parent modal owns the tabs component:
```go
type SomeModal struct {
    tabs *components.Tabs
    // ... other fields
}

func NewSomeModal() *SomeModal {
    return &SomeModal{
        tabs: components.NewTabs([]string{"Tab 1", "Tab 2"}),
    }
}

func (m *SomeModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "tab":
            m.tabs.Next()
            return m, nil
        case "shift+tab":
            m.tabs.Previous()
            return m, nil
        }
    }
    // ... other update logic
}

func (m *SomeModal) View() string {
    var b strings.Builder

    // Render tabs at top
    b.WriteString(m.tabs.View())
    b.WriteString("\n\n")

    // Render content based on active tab
    switch m.tabs.ActiveIndex() {
    case 0:
        b.WriteString("Content for Tab 1")
    case 1:
        b.WriteString("Content for Tab 2")
    }

    return b.String()
}
```

### Edge Cases

- Single tab: Still render tab bar (consistency), but tab switching does nothing
- Empty tabs: Constructor should panic or return error (invalid usage)
- Width < total tab width: Tabs may overflow (acceptable for v1, can truncate later)

## Validation

How do we know this phase is complete?

- [ ] `Tabs` type defined in `components/tabs.go`
- [ ] All methods implemented: `NewTabs`, `SetActive`, `ActiveIndex`, `Next`, `Previous`, `SetWidth`, `View`
- [ ] `View()` renders styled tab bar with active/inactive styling
- [ ] Tab wrapping works (Next from last tab goes to first, Previous from first goes to last)
- [ ] Code compiles
- [ ] Manual test: Create a simple test modal with tabs, verify rendering and switching
- [ ] Visually matches existing modal styling (colors, spacing)
