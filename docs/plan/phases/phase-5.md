# Phase 5: Tool Picker

> **Depends on:** Phase 3.2 (Builder Editing)
> **Enables:** Phase 6 (Step Detail Form)
>
> See: [Full Plan](../plan.md)

## Goal

Create the TreePicker component and tool browser for selecting tools when adding/editing steps.

## Key Deliverables

- `TreePicker` reusable component in `ui/components/`
- Tool browser using TreePicker (type → source → tool hierarchy)
- Fetch and cache tools from builder API
- Return selected tool to step editor

## Files to Create

- `internal/ui/components/treepicker.go` — Reusable hierarchical picker
- `internal/ui/modal/workflows_tools.go` — Tool browser wrapper

## Implementation Notes

### TreePicker Component

```go
package components

import (
    "strings"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

// TreeNode represents a node in the tree
type TreeNode struct {
    Key         string        // unique identifier
    Label       string        // display text
    Description string        // optional help text
    Children    []TreeNode    // nil or empty = leaf node
    Data        interface{}   // arbitrary payload
}

// TreePicker provides hierarchical navigation
type TreePicker struct {
    Root        []TreeNode
    path        []int       // navigation path (indices at each level)
    selected    int         // selected index at current level
    styles      TreeStyles
}

type TreeStyles struct {
    Normal      lipgloss.Style
    Selected    lipgloss.Style
    Breadcrumb  lipgloss.Style
    Description lipgloss.Style
}

func DefaultTreeStyles() TreeStyles {
    return TreeStyles{
        Normal:      lipgloss.NewStyle(),
        Selected:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),
        Breadcrumb:  lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
        Description: lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
    }
}

func NewTreePicker(nodes []TreeNode) *TreePicker {
    return &TreePicker{
        Root:     nodes,
        path:     []int{},
        selected: 0,
        styles:   DefaultTreeStyles(),
    }
}

func (t *TreePicker) SetStyles(styles TreeStyles) {
    t.styles = styles
}
```

### Navigation Methods

```go
// CurrentLevel returns nodes at the current depth
func (t *TreePicker) CurrentLevel() []TreeNode {
    nodes := t.Root
    for _, idx := range t.path {
        if idx < len(nodes) && len(nodes[idx].Children) > 0 {
            nodes = nodes[idx].Children
        }
    }
    return nodes
}

// SelectedNode returns the currently highlighted node
func (t *TreePicker) SelectedNode() *TreeNode {
    nodes := t.CurrentLevel()
    if t.selected >= 0 && t.selected < len(nodes) {
        return &nodes[t.selected]
    }
    return nil
}

// CanGoDeeper returns true if selected node has children
func (t *TreePicker) CanGoDeeper() bool {
    node := t.SelectedNode()
    return node != nil && len(node.Children) > 0
}

// CanGoBack returns true if not at root level
func (t *TreePicker) CanGoBack() bool {
    return len(t.path) > 0
}

// Breadcrumb returns the current path as labels
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
```

### Update Method

```go
// SelectedMsg is sent when a leaf node is selected
type TreeSelectedMsg struct {
    Node TreeNode
}

// Update handles input and returns selected node if leaf selected
func (t *TreePicker) Update(msg tea.Msg) (selected *TreeNode, cmd tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "j", "down":
            nodes := t.CurrentLevel()
            if t.selected < len(nodes)-1 {
                t.selected++
            }

        case "k", "up":
            if t.selected > 0 {
                t.selected--
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
            } else {
                // Leaf node - return selection
                return node, nil
            }

        case "h", "left", "backspace":
            if len(t.path) > 0 {
                // Go back
                t.selected = t.path[len(t.path)-1]
                t.path = t.path[:len(t.path)-1]
            }

        case "esc":
            if len(t.path) > 0 {
                // Go back to root first
                t.path = []int{}
                t.selected = 0
            } else {
                // At root, signal cancel
                return nil, func() tea.Msg { return TreeCancelledMsg{} }
            }
        }
    }
    return nil, nil
}

type TreeCancelledMsg struct{}
```

### View Method

```go
func (t *TreePicker) View() string {
    var lines []string

    // Breadcrumb
    crumbs := t.Breadcrumb()
    if len(crumbs) > 0 {
        breadcrumb := strings.Join(crumbs, " > ")
        lines = append(lines, t.styles.Breadcrumb.Render(breadcrumb))
        lines = append(lines, "")
    }

    // Current level items
    nodes := t.CurrentLevel()
    for i, node := range nodes {
        prefix := "  "
        suffix := ""

        if len(node.Children) > 0 {
            suffix = " >"  // Indicate has children
        }

        label := prefix + node.Label + suffix

        if i == t.selected {
            lines = append(lines, t.styles.Selected.Render("> "+node.Label+suffix))
            // Show description for selected item
            if node.Description != "" {
                lines = append(lines, t.styles.Description.Render("  "+node.Description))
            }
        } else {
            lines = append(lines, t.styles.Normal.Render(label))
        }
    }

    return strings.Join(lines, "\n")
}
```

### Tool Browser Wrapper

```go
// internal/ui/modal/workflows_tools.go

package modal

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/pxp/hub-tui/internal/client"
    "github.com/pxp/hub-tui/internal/ui/components"
)

// ToolBrowser wraps TreePicker for tool selection
type ToolBrowser struct {
    picker  *components.TreePicker
    tools   *client.ToolsResponse
    loading bool
    error   string
}

func NewToolBrowser() *ToolBrowser {
    return &ToolBrowser{
        loading: true,
    }
}

func (b *ToolBrowser) Init(c *client.Client) tea.Cmd {
    return func() tea.Msg {
        tools, err := c.GetBuilderTools()
        return BuilderToolsLoadedMsg{Tools: tools, Error: err}
    }
}

func (b *ToolBrowser) SetTools(tools *client.ToolsResponse) {
    b.tools = tools
    b.loading = false
    b.picker = components.NewTreePicker(b.buildTree())
}

func (b *ToolBrowser) buildTree() []components.TreeNode {
    var nodes []components.TreeNode

    // Modules
    if len(b.tools.Tools.Modules) > 0 {
        moduleNodes := b.buildCategoryNodes(b.tools.Tools.Modules)
        nodes = append(nodes, components.TreeNode{
            Key:      "modules",
            Label:    "Modules",
            Children: moduleNodes,
        })
    }

    // Integrations
    if len(b.tools.Tools.Integrations) > 0 {
        intNodes := b.buildCategoryNodes(b.tools.Tools.Integrations)
        nodes = append(nodes, components.TreeNode{
            Key:      "integrations",
            Label:    "Integrations",
            Children: intNodes,
        })
    }

    // Utilities
    if len(b.tools.Tools.Utilities) > 0 {
        utilNodes := b.buildCategoryNodes(b.tools.Tools.Utilities)
        nodes = append(nodes, components.TreeNode{
            Key:      "utilities",
            Label:    "Utilities",
            Children: utilNodes,
        })
    }

    // Primitives
    if len(b.tools.Tools.Primitives) > 0 {
        primNodes := b.buildCategoryNodes(b.tools.Tools.Primitives)
        nodes = append(nodes, components.TreeNode{
            Key:      "primitives",
            Label:    "Primitives",
            Children: primNodes,
        })
    }

    return nodes
}

func (b *ToolBrowser) buildCategoryNodes(category map[string][]client.Tool) []components.TreeNode {
    var nodes []components.TreeNode
    for source, tools := range category {
        var toolNodes []components.TreeNode
        for _, tool := range tools {
            toolNodes = append(toolNodes, components.TreeNode{
                Key:         tool.Target,
                Label:       tool.Name,
                Description: tool.Description,
                Data:        tool,  // Store full tool for later
            })
        }
        nodes = append(nodes, components.TreeNode{
            Key:      source,
            Label:    source,
            Children: toolNodes,
        })
    }
    return nodes
}
```

### Integration with Builder

```go
// In workflows_builder.go

case ViewToolPicker:
    if b.toolBrowser == nil {
        b.toolBrowser = NewToolBrowser()
        return b, b.toolBrowser.Init(b.client)
    }
    return b.toolBrowser.View()

// In Update:
case BuilderToolsLoadedMsg:
    if b.toolBrowser != nil {
        b.toolBrowser.SetTools(msg.Tools)
        // Also cache for later use
        b.Tools = msg.Tools
    }
    return b, nil

case ViewToolPicker:
    if b.toolBrowser != nil {
        selected, cmd := b.toolBrowser.picker.Update(msg)
        if selected != nil {
            // Tool selected - configure step
            tool := selected.Data.(client.Tool)
            b.configureStepWithTool(tool)
            b.View = ViewStepDetail
            b.toolBrowser = nil
        }
        return b, cmd
    }
```

## Validation

- [ ] `TreePicker` component compiles independently
- [ ] j/k navigates within current level
- [ ] Enter/right on category goes deeper
- [ ] Backspace/left/Esc goes back up
- [ ] Breadcrumb shows current path
- [ ] Leaf nodes (tools) return selection on Enter
- [ ] Tool browser fetches tools on open
- [ ] Tools are organized by type > source > tool
- [ ] Selected tool description shows below selection
- [ ] Selected tool data is available to builder
- [ ] Esc at root level cancels tool selection
