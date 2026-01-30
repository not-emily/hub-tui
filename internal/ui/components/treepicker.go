package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/ui/theme"
)

// TreeNode represents a node in the tree.
type TreeNode struct {
	Key         string      // unique identifier
	Label       string      // display text
	Description string      // optional help text
	Children    []TreeNode  // nil or empty = leaf node
	Data        interface{} // arbitrary payload
}

// TreePicker provides hierarchical navigation.
type TreePicker struct {
	Root     []TreeNode
	path     []int // navigation path (indices at each level)
	selected int   // selected index at current level
	height   int   // visible height for scrolling
	offset   int   // scroll offset
}

// NewTreePicker creates a new tree picker with the given root nodes.
func NewTreePicker(nodes []TreeNode) *TreePicker {
	return &TreePicker{
		Root:     nodes,
		path:     []int{},
		selected: 0,
		height:   10,
		offset:   0,
	}
}

// SetHeight sets the visible height for scrolling.
func (t *TreePicker) SetHeight(height int) {
	t.height = height
}

// CurrentLevel returns nodes at the current depth.
func (t *TreePicker) CurrentLevel() []TreeNode {
	nodes := t.Root
	for _, idx := range t.path {
		if idx < len(nodes) && len(nodes[idx].Children) > 0 {
			nodes = nodes[idx].Children
		}
	}
	return nodes
}

// SelectedNode returns the currently highlighted node.
func (t *TreePicker) SelectedNode() *TreeNode {
	nodes := t.CurrentLevel()
	if t.selected >= 0 && t.selected < len(nodes) {
		return &nodes[t.selected]
	}
	return nil
}

// CanGoDeeper returns true if selected node has children.
func (t *TreePicker) CanGoDeeper() bool {
	node := t.SelectedNode()
	return node != nil && len(node.Children) > 0
}

// CanGoBack returns true if not at root level.
func (t *TreePicker) CanGoBack() bool {
	return len(t.path) > 0
}

// Breadcrumb returns the current path as labels.
func (t *TreePicker) Breadcrumb() []string {
	var crumbs []string
	nodes := t.Root
	for _, idx := range t.path {
		if idx < len(nodes) {
			crumbs = append(crumbs, nodes[idx].Label)
			nodes = nodes[idx].Children
		}
	}
	return crumbs
}

// TreeSelectedMsg is sent when a leaf node is selected.
type TreeSelectedMsg struct {
	Node TreeNode
}

// TreeCancelledMsg is sent when selection is cancelled.
type TreeCancelledMsg struct{}

// Update handles input and returns selected node if leaf selected.
func (t *TreePicker) Update(msg tea.Msg) (selected *TreeNode, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			nodes := t.CurrentLevel()
			if t.selected < len(nodes)-1 {
				t.selected++
				t.ensureVisible()
			}

		case "k", "up":
			if t.selected > 0 {
				t.selected--
				t.ensureVisible()
			}

		case "l", "right", "enter":
			node := t.SelectedNode()
			if node == nil {
				return nil, nil
			}

			if len(node.Children) > 0 {
				// Go deeper
				t.path = append(t.path, t.selected)
				t.selected = 0
				t.offset = 0
			} else {
				// Leaf node - return selection
				return node, nil
			}

		case "h", "left", "backspace":
			if len(t.path) > 0 {
				// Go back
				t.selected = t.path[len(t.path)-1]
				t.path = t.path[:len(t.path)-1]
				t.ensureVisible()
			}

		case "esc":
			if len(t.path) > 0 {
				// Go back one level (same as h/left/backspace)
				t.selected = t.path[len(t.path)-1]
				t.path = t.path[:len(t.path)-1]
				t.ensureVisible()
			} else {
				// At root, signal cancel
				return nil, func() tea.Msg { return TreeCancelledMsg{} }
			}
		}
	}
	return nil, nil
}

// ensureVisible adjusts offset to keep selected item visible.
func (t *TreePicker) ensureVisible() {
	if t.selected < t.offset {
		t.offset = t.selected
	}
	if t.selected >= t.offset+t.height {
		t.offset = t.selected - t.height + 1
	}
}

// View renders the tree picker.
func (t *TreePicker) View() string {
	var lines []string

	selectedStyle := lipgloss.NewStyle().
		Foreground(theme.Accent).
		Bold(true)

	normalStyle := lipgloss.NewStyle().
		Foreground(theme.TextPrimary)

	dimStyle := lipgloss.NewStyle().
		Foreground(theme.TextSecondary)

	// Breadcrumb
	crumbs := t.Breadcrumb()
	if len(crumbs) > 0 {
		breadcrumb := strings.Join(crumbs, " › ")
		lines = append(lines, dimStyle.Render(breadcrumb))
		lines = append(lines, "")
	}

	// Current level items
	nodes := t.CurrentLevel()
	if len(nodes) == 0 {
		lines = append(lines, dimStyle.Render("(empty)"))
		return strings.Join(lines, "\n")
	}

	// Calculate visible range
	end := t.offset + t.height
	if end > len(nodes) {
		end = len(nodes)
	}

	for i := t.offset; i < end; i++ {
		node := nodes[i]
		suffix := ""

		if len(node.Children) > 0 {
			suffix = " ›" // Indicate has children
		}

		if i == t.selected {
			line := selectedStyle.Render("> "+node.Label+suffix)
			// Show description to the right for selected item
			if node.Description != "" {
				line += "  " + dimStyle.Render(node.Description)
			}
			lines = append(lines, line)
		} else {
			lines = append(lines, normalStyle.Render("  "+node.Label+suffix))
		}
	}

	return strings.Join(lines, "\n")
}
