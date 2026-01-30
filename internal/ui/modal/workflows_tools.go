package modal

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/client"
	"github.com/pxp/hub-tui/internal/ui/components"
	"github.com/pxp/hub-tui/internal/ui/theme"
)

// BuilderToolsLoadedMsg is sent when tools are loaded from the API.
type BuilderToolsLoadedMsg struct {
	Tools *client.ToolsResponse
	Error error
}

// ToolSelectedMsg is sent when a tool is selected.
type ToolSelectedMsg struct {
	Tool client.Tool
	Type string // "module", "integration", "utility", "primitive"
}

// ToolBrowser wraps TreePicker for tool selection.
type ToolBrowser struct {
	picker  *components.TreePicker
	tools   *client.ToolsResponse
	loading bool
	err     error
}

// NewToolBrowser creates a new tool browser.
func NewToolBrowser() *ToolBrowser {
	return &ToolBrowser{
		loading: true,
	}
}

// Init starts loading tools from the API.
func (b *ToolBrowser) Init(c *client.Client) tea.Cmd {
	return func() tea.Msg {
		tools, err := c.GetBuilderTools()
		return BuilderToolsLoadedMsg{Tools: tools, Error: err}
	}
}

// SetTools populates the browser with tools from the API.
func (b *ToolBrowser) SetTools(tools *client.ToolsResponse) {
	b.tools = tools
	b.loading = false
	b.err = nil
	b.picker = components.NewTreePicker(b.buildTree())
	b.picker.SetHeight(12)
}

// SetError sets an error state.
func (b *ToolBrowser) SetError(err error) {
	b.err = err
	b.loading = false
}

// Update handles input and returns selected tool info.
func (b *ToolBrowser) Update(msg tea.Msg) (tool *client.Tool, toolType string, cmd tea.Cmd) {
	if b.loading || b.err != nil || b.picker == nil {
		// Handle escape to cancel while loading/error
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
			return nil, "", func() tea.Msg { return components.TreeCancelledMsg{} }
		}
		return nil, "", nil
	}

	selected, cmd := b.picker.Update(msg)
	if selected != nil {
		// Extract tool and type from node data
		if data, ok := selected.Data.(toolNodeData); ok {
			return &data.Tool, data.Type, nil
		}
	}
	return nil, "", cmd
}

// View renders the tool browser.
func (b *ToolBrowser) View() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	errorStyle := lipgloss.NewStyle().Foreground(theme.Error)

	var lines []string
	lines = append(lines, headerStyle.Render("Select Tool"))
	lines = append(lines, "")

	if b.loading {
		lines = append(lines, dimStyle.Render("Loading tools..."))
		lines = append(lines, "")
		lines = append(lines, dimStyle.Render("[Esc] Cancel"))
		return strings.Join(lines, "\n")
	}

	if b.err != nil {
		lines = append(lines, errorStyle.Render("Error: "+b.err.Error()))
		lines = append(lines, "")
		lines = append(lines, dimStyle.Render("[Esc] Cancel"))
		return strings.Join(lines, "\n")
	}

	if b.picker != nil {
		lines = append(lines, b.picker.View())
	}

	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("[Enter] Select  [←/Esc] Back  [j/k] Navigate"))

	return strings.Join(lines, "\n")
}

// toolNodeData holds tool info in tree node data.
type toolNodeData struct {
	Tool client.Tool
	Type string
}

// buildTree constructs the tree structure from tools response.
func (b *ToolBrowser) buildTree() []components.TreeNode {
	var nodes []components.TreeNode

	// Modules
	if len(b.tools.Tools.Modules) > 0 {
		moduleNodes := b.buildCategoryNodes(b.tools.Tools.Modules, "module")
		nodes = append(nodes, components.TreeNode{
			Key:         "modules",
			Label:       "Modules",
			Description: "Tools from installed modules",
			Children:    moduleNodes,
		})
	}

	// Integrations
	if len(b.tools.Tools.Integrations) > 0 {
		intNodes := b.buildCategoryNodes(b.tools.Tools.Integrations, "integration")
		nodes = append(nodes, components.TreeNode{
			Key:         "integrations",
			Label:       "Integrations",
			Description: "Tools from configured integrations",
			Children:    intNodes,
		})
	}

	// Utilities
	if len(b.tools.Tools.Utilities) > 0 {
		utilNodes := b.buildCategoryNodes(b.tools.Tools.Utilities, "utility")
		nodes = append(nodes, components.TreeNode{
			Key:         "utilities",
			Label:       "Utilities",
			Description: "Built-in utility tools",
			Children:    utilNodes,
		})
	}

	// Primitives
	if len(b.tools.Tools.Primitives) > 0 {
		primNodes := b.buildCategoryNodes(b.tools.Tools.Primitives, "primitive")
		nodes = append(nodes, components.TreeNode{
			Key:         "primitives",
			Label:       "Primitives",
			Description: "Low-level primitive operations",
			Children:    primNodes,
		})
	}

	return nodes
}

// buildCategoryNodes builds tree nodes for a category (modules, integrations, etc).
func (b *ToolBrowser) buildCategoryNodes(category map[string][]client.Tool, toolType string) []components.TreeNode {
	// Get sorted source names for consistent ordering
	sources := make([]string, 0, len(category))
	for source := range category {
		sources = append(sources, source)
	}
	sort.Strings(sources)

	var nodes []components.TreeNode
	for _, source := range sources {
		tools := category[source]
		var toolNodes []components.TreeNode
		for _, tool := range tools {
			toolNodes = append(toolNodes, components.TreeNode{
				Key:         tool.Target,
				Label:       tool.Name,
				Description: tool.Description,
				Data: toolNodeData{
					Tool: tool,
					Type: toolType,
				},
			})
		}

		// Sort tools by name
		sort.Slice(toolNodes, func(i, j int) bool {
			return toolNodes[i].Label < toolNodes[j].Label
		})

		nodes = append(nodes, components.TreeNode{
			Key:      source,
			Label:    source,
			Children: toolNodes,
		})
	}
	return nodes
}
