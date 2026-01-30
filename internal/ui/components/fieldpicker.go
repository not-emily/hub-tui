package components

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/ui/theme"
)

// FieldSelectedMsg is sent when a field is selected.
type FieldSelectedMsg struct {
	Path string
}

// FieldCancelledMsg is sent when field selection is cancelled.
type FieldCancelledMsg struct{}

// ExtractedField represents a field extracted from JSON.
type ExtractedField struct {
	Name      string      // friendly name (e.g., "Name", "Status")
	Path      string      // relative path (e.g., ".properties.Name.title[0].plain_text")
	Value     interface{} // sample value from first item
	ValueStr  string      // string representation for display
	Type      string      // "string", "number", "boolean", "array", "object"
	FromArray bool        // true if extracted from array item
}

// FieldPicker lets user select a field from step output.
type FieldPicker struct {
	StepName  string           // which step's output (for path prefix)
	RawOutput interface{}      // original output
	ItemsPath string           // path to items array (e.g., ".results")
	Fields    []ExtractedField // extracted fields
	Selected  int
	height    int

	// Array handling
	showArrayChoice bool
	arrayChoice     int // 0 = first item, 1 = all items

	// Raw mode fallback
	RawMode bool
	RawTree *TreePicker

	// Multi-step support
	multiStep       bool                   // true if showing multiple steps
	allSteps        map[string]interface{} // all previous step outputs
	stepNames       []string               // ordered step names
	selectingStep   bool                   // true when picking which step
	selectedStepIdx int                    // selected step index
}

// NewFieldPicker creates a new field picker for a single step's output.
func NewFieldPicker(stepName string, output interface{}) *FieldPicker {
	p := &FieldPicker{
		StepName:  stepName,
		RawOutput: output,
		height:    10,
	}

	p.extractFields()
	return p
}

// NewFieldPickerMulti creates a field picker for multiple previous steps.
func NewFieldPickerMulti(stepOutputs map[string]interface{}) *FieldPicker {
	p := &FieldPicker{
		height:    10,
		multiStep: true,
		allSteps:  stepOutputs,
	}

	// Get sorted step names for consistent ordering
	for name := range stepOutputs {
		p.stepNames = append(p.stepNames, name)
	}
	sort.Strings(p.stepNames)

	// If only one step, go directly to its fields
	if len(p.stepNames) == 1 {
		p.multiStep = false
		p.StepName = p.stepNames[0]
		p.RawOutput = stepOutputs[p.stepNames[0]]
		p.extractFields()
	} else {
		p.selectingStep = true
	}

	return p
}

// SetHeight sets the visible height.
func (p *FieldPicker) SetHeight(height int) {
	p.height = height
}

func (p *FieldPicker) extractFields() {
	// Find the items array
	itemsPath, items := p.findItemsArray(p.RawOutput)
	p.ItemsPath = itemsPath

	if items != nil && len(items) > 0 {
		// Extract from first item
		p.Fields = p.extractFromItem(items[0], true)
	} else {
		// Not an array, extract from root
		p.Fields = p.extractFromItem(p.RawOutput, false)
	}

	// Sort fields by name
	sort.Slice(p.Fields, func(i, j int) bool {
		return p.Fields[i].Name < p.Fields[j].Name
	})
}

func (p *FieldPicker) findItemsArray(data interface{}) (string, []interface{}) {
	obj, ok := data.(map[string]interface{})
	if !ok {
		// Data itself might be an array
		if arr, ok := data.([]interface{}); ok {
			return "", arr
		}
		return "", nil
	}

	// Look for common array field names
	arrayNames := []string{"results", "data", "items", "records", "entries", "rows", "list"}
	for _, name := range arrayNames {
		if val, ok := obj[name]; ok {
			if arr, ok := val.([]interface{}); ok && len(arr) > 0 {
				return "." + name, arr
			}
		}
	}

	// Look for first array field with items
	for key, val := range obj {
		if arr, ok := val.([]interface{}); ok && len(arr) > 0 {
			return "." + key, arr
		}
	}

	return "", nil
}

func (p *FieldPicker) extractFromItem(item interface{}, fromArray bool) []ExtractedField {
	var fields []ExtractedField

	obj, ok := item.(map[string]interface{})
	if !ok {
		// Primitive value
		fields = append(fields, ExtractedField{
			Name:      "value",
			Path:      "",
			Value:     item,
			ValueStr:  formatValue(item),
			Type:      typeOf(item),
			FromArray: fromArray,
		})
		return fields
	}

	// Check for Notion-style properties
	if props, ok := obj["properties"].(map[string]interface{}); ok {
		for name, prop := range props {
			if extracted := p.extractNotionProperty(name, prop); extracted != nil {
				extracted.FromArray = fromArray
				fields = append(fields, *extracted)
			}
		}
		// Also add the id if present
		if id, ok := obj["id"].(string); ok {
			fields = append(fields, ExtractedField{
				Name:      "id",
				Path:      ".id",
				Value:     id,
				ValueStr:  formatValue(id),
				Type:      "string",
				FromArray: fromArray,
			})
		}
		return fields
	}

	// Generic extraction - find leaf values
	p.extractLeafValues(obj, "", &fields, fromArray, 0)

	return fields
}

func (p *FieldPicker) extractNotionProperty(name string, prop interface{}) *ExtractedField {
	obj, ok := prop.(map[string]interface{})
	if !ok {
		return nil
	}

	// Notion title: {title: [{plain_text: "..."}]}
	if title, ok := obj["title"].([]interface{}); ok && len(title) > 0 {
		if first, ok := title[0].(map[string]interface{}); ok {
			if text, ok := first["plain_text"].(string); ok {
				return &ExtractedField{
					Name:     name,
					Path:     fmt.Sprintf(".properties.%s.title[0].plain_text", name),
					Value:    text,
					ValueStr: formatValue(text),
					Type:     "string",
				}
			}
		}
	}

	// Notion rich_text: {rich_text: [{plain_text: "..."}]}
	if richText, ok := obj["rich_text"].([]interface{}); ok && len(richText) > 0 {
		if first, ok := richText[0].(map[string]interface{}); ok {
			if text, ok := first["plain_text"].(string); ok {
				return &ExtractedField{
					Name:     name,
					Path:     fmt.Sprintf(".properties.%s.rich_text[0].plain_text", name),
					Value:    text,
					ValueStr: formatValue(text),
					Type:     "string",
				}
			}
		}
	}

	// Notion status: {status: {name: "..."}}
	if status, ok := obj["status"].(map[string]interface{}); ok {
		if statusName, ok := status["name"].(string); ok {
			return &ExtractedField{
				Name:     name,
				Path:     fmt.Sprintf(".properties.%s.status.name", name),
				Value:    statusName,
				ValueStr: formatValue(statusName),
				Type:     "string",
			}
		}
	}

	// Notion select: {select: {name: "..."}}
	if sel, ok := obj["select"].(map[string]interface{}); ok {
		if selName, ok := sel["name"].(string); ok {
			return &ExtractedField{
				Name:     name,
				Path:     fmt.Sprintf(".properties.%s.select.name", name),
				Value:    selName,
				ValueStr: formatValue(selName),
				Type:     "string",
			}
		}
	}

	// Notion date: {date: {start: "..."}}
	if date, ok := obj["date"].(map[string]interface{}); ok {
		if start, ok := date["start"].(string); ok {
			return &ExtractedField{
				Name:     name,
				Path:     fmt.Sprintf(".properties.%s.date.start", name),
				Value:    start,
				ValueStr: formatValue(start),
				Type:     "string",
			}
		}
	}

	// Notion number: {number: 123}
	if num, ok := obj["number"]; ok && num != nil {
		return &ExtractedField{
			Name:     name,
			Path:     fmt.Sprintf(".properties.%s.number", name),
			Value:    num,
			ValueStr: formatValue(num),
			Type:     "number",
		}
	}

	// Notion checkbox: {checkbox: true}
	if cb, ok := obj["checkbox"]; ok {
		return &ExtractedField{
			Name:     name,
			Path:     fmt.Sprintf(".properties.%s.checkbox", name),
			Value:    cb,
			ValueStr: formatValue(cb),
			Type:     "boolean",
		}
	}

	// Notion url: {url: "..."}
	if url, ok := obj["url"].(string); ok {
		return &ExtractedField{
			Name:     name,
			Path:     fmt.Sprintf(".properties.%s.url", name),
			Value:    url,
			ValueStr: formatValue(url),
			Type:     "string",
		}
	}

	// Notion email: {email: "..."}
	if email, ok := obj["email"].(string); ok {
		return &ExtractedField{
			Name:     name,
			Path:     fmt.Sprintf(".properties.%s.email", name),
			Value:    email,
			ValueStr: formatValue(email),
			Type:     "string",
		}
	}

	// Notion phone_number: {phone_number: "..."}
	if phone, ok := obj["phone_number"].(string); ok {
		return &ExtractedField{
			Name:     name,
			Path:     fmt.Sprintf(".properties.%s.phone_number", name),
			Value:    phone,
			ValueStr: formatValue(phone),
			Type:     "string",
		}
	}

	return nil
}

func (p *FieldPicker) extractLeafValues(obj map[string]interface{}, prefix string, fields *[]ExtractedField, fromArray bool, depth int) {
	if depth > 5 {
		return // Prevent infinite recursion
	}

	for key, val := range obj {
		path := prefix + "." + key

		switch v := val.(type) {
		case string:
			*fields = append(*fields, ExtractedField{
				Name:      key,
				Path:      path,
				Value:     v,
				ValueStr:  formatValue(v),
				Type:      "string",
				FromArray: fromArray,
			})
		case float64:
			*fields = append(*fields, ExtractedField{
				Name:      key,
				Path:      path,
				Value:     v,
				ValueStr:  formatValue(v),
				Type:      "number",
				FromArray: fromArray,
			})
		case bool:
			*fields = append(*fields, ExtractedField{
				Name:      key,
				Path:      path,
				Value:     v,
				ValueStr:  formatValue(v),
				Type:      "boolean",
				FromArray: fromArray,
			})
		case map[string]interface{}:
			// Check if it's a simple wrapper with one useful value
			if len(v) == 1 {
				for innerKey, innerVal := range v {
					switch iv := innerVal.(type) {
					case string, float64, bool:
						*fields = append(*fields, ExtractedField{
							Name:      key,
							Path:      path + "." + innerKey,
							Value:     iv,
							ValueStr:  formatValue(iv),
							Type:      typeOf(iv),
							FromArray: fromArray,
						})
						continue
					}
				}
			}
			// Recurse into nested objects
			p.extractLeafValues(v, path, fields, fromArray, depth+1)
		}
	}
}

// Update handles input and returns selected path if a field is selected.
func (p *FieldPicker) Update(msg tea.Msg) (selectedPath string, cmd tea.Cmd) {
	// Handle step selection phase
	if p.selectingStep {
		return p.handleStepSelection(msg)
	}

	if p.RawMode {
		return p.handleRawMode(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if p.showArrayChoice {
				if p.arrayChoice < 1 {
					p.arrayChoice++
				}
			} else if p.Selected < len(p.Fields)-1 {
				p.Selected++
			}

		case "k", "up":
			if p.showArrayChoice {
				if p.arrayChoice > 0 {
					p.arrayChoice--
				}
			} else if p.Selected > 0 {
				p.Selected--
			}

		case "enter":
			if len(p.Fields) == 0 {
				return "", nil
			}

			if p.showArrayChoice {
				// Array choice made, return path
				return p.SelectedPath(), nil
			}

			field := p.Fields[p.Selected]
			if field.FromArray {
				// Show array choice
				p.showArrayChoice = true
				p.arrayChoice = 1 // Default to "all items"
			} else {
				// Return directly
				return p.SelectedPath(), nil
			}

		case "tab":
			p.RawMode = true
			if p.RawTree == nil {
				p.RawTree = p.buildRawTree()
			}

		case "esc":
			if p.showArrayChoice {
				p.showArrayChoice = false
			} else if p.multiStep {
				// Go back to step selection
				p.selectingStep = true
				p.Selected = 0
				p.Fields = nil
			} else {
				return "", func() tea.Msg { return FieldCancelledMsg{} }
			}
		}
	}

	return "", nil
}

func (p *FieldPicker) handleStepSelection(msg tea.Msg) (string, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if p.selectedStepIdx < len(p.stepNames)-1 {
				p.selectedStepIdx++
			}
		case "k", "up":
			if p.selectedStepIdx > 0 {
				p.selectedStepIdx--
			}
		case "enter":
			// Select this step and show its fields
			stepName := p.stepNames[p.selectedStepIdx]
			p.StepName = stepName
			p.RawOutput = p.allSteps[stepName]
			p.extractFields()
			p.selectingStep = false
			p.Selected = 0
		case "esc":
			return "", func() tea.Msg { return FieldCancelledMsg{} }
		}
	}
	return "", nil
}

func (p *FieldPicker) handleRawMode(msg tea.Msg) (string, tea.Cmd) {
	if p.RawTree == nil {
		return "", nil
	}

	// Check for tab to exit raw mode
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "tab" {
		p.RawMode = false
		return "", nil
	}

	node, cmd := p.RawTree.Update(msg)
	if node != nil {
		// Build path from raw tree selection
		return p.buildRawPath(node), nil
	}

	// Check for TreeCancelledMsg
	if cmd != nil {
		// The tree might have sent a cancel - we need to check the actual message
		// For now, just return the command
	}

	return "", cmd
}

func (p *FieldPicker) buildRawTree() *TreePicker {
	nodes := p.buildTreeNodes(p.RawOutput, "")
	tree := NewTreePicker(nodes)
	tree.SetHeight(p.height)
	return tree
}

func (p *FieldPicker) buildTreeNodes(data interface{}, path string) []TreeNode {
	var nodes []TreeNode

	switch v := data.(type) {
	case map[string]interface{}:
		// Sort keys for consistent ordering
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, key := range keys {
			val := v[key]
			childPath := path + "." + key
			node := TreeNode{
				Key:   key,
				Label: key,
				Data:  childPath,
			}

			// Check if has children
			switch child := val.(type) {
			case map[string]interface{}:
				node.Children = p.buildTreeNodes(child, childPath)
				node.Description = fmt.Sprintf("{%d keys}", len(child))
			case []interface{}:
				if len(child) > 0 {
					// Show first item's structure
					node.Children = p.buildTreeNodes(child[0], childPath+"[0]")
					node.Description = fmt.Sprintf("[%d items]", len(child))
				} else {
					node.Description = "[]"
				}
			default:
				node.Description = truncateValue(formatValue(val), 30)
			}

			nodes = append(nodes, node)
		}

	case []interface{}:
		if len(v) > 0 {
			// Build nodes from first item
			return p.buildTreeNodes(v[0], path+"[0]")
		}
	}

	return nodes
}

func (p *FieldPicker) buildRawPath(node *TreeNode) string {
	if node.Data == nil {
		return ""
	}
	path, ok := node.Data.(string)
	if !ok {
		return ""
	}
	return "$" + p.StepName + path
}

// SelectedPath returns the full path for the selected field.
func (p *FieldPicker) SelectedPath() string {
	if p.Selected < 0 || p.Selected >= len(p.Fields) {
		return ""
	}
	return p.buildFullPath(p.Fields[p.Selected])
}

func (p *FieldPicker) buildFullPath(field ExtractedField) string {
	var path string

	if field.FromArray {
		// $step.results[].field or $step.results[0].field
		if p.arrayChoice == 0 {
			path = "$" + p.StepName + p.ItemsPath + "[0]" + field.Path
		} else {
			path = "$" + p.StepName + p.ItemsPath + "[]" + field.Path
		}
	} else {
		path = "$" + p.StepName + field.Path
	}

	return path
}

// View renders the field picker.
func (p *FieldPicker) View() string {
	if p.selectingStep {
		return p.renderStepSelection()
	}

	if p.RawMode && p.RawTree != nil {
		return p.renderRawMode()
	}

	if p.showArrayChoice {
		return p.renderArrayChoice()
	}

	return p.renderFieldList()
}

func (p *FieldPicker) renderStepSelection() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)

	var lines []string

	lines = append(lines, headerStyle.Render("Select Variable"))
	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("Choose a previous step's output:"))
	lines = append(lines, "")

	for i, name := range p.stepNames {
		display := "$" + name
		if i == p.selectedStepIdx {
			lines = append(lines, selectedStyle.Render("> "+display))
		} else {
			lines = append(lines, "  "+display)
		}
	}

	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("[j/k] Navigate  [Enter] Select  [Esc] Cancel"))

	return strings.Join(lines, "\n")
}

func (p *FieldPicker) renderFieldList() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(theme.Success)

	var lines []string

	lines = append(lines, headerStyle.Render("Select field from: $"+p.StepName))
	lines = append(lines, "")

	if len(p.Fields) == 0 {
		lines = append(lines, dimStyle.Render("No fields extracted. Press [Tab] for raw view."))
		lines = append(lines, "")
		lines = append(lines, dimStyle.Render("[Tab] Raw view  [Esc] Cancel"))
		return strings.Join(lines, "\n")
	}

	// Show extracted fields
	for i, field := range p.Fields {
		name := fmt.Sprintf("%-20s", field.Name)
		value := truncateValue(field.ValueStr, 35)

		if i == p.Selected {
			lines = append(lines, selectedStyle.Render("> "+name)+valueStyle.Render(value))
		} else {
			lines = append(lines, "  "+name+dimStyle.Render(value))
		}
	}

	lines = append(lines, "")

	// Show full path for selected
	if p.Selected < len(p.Fields) {
		fullPath := p.buildFullPath(p.Fields[p.Selected])
		lines = append(lines, dimStyle.Render("Path: "+fullPath))
	}

	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("[j/k] Navigate  [Enter] Select  [Tab] Raw view  [Esc] Cancel"))

	return strings.Join(lines, "\n")
}

func (p *FieldPicker) renderArrayChoice() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)

	var lines []string

	lines = append(lines, headerStyle.Render("This field is from an array"))
	lines = append(lines, "")
	lines = append(lines, "Apply to:")
	lines = append(lines, "")

	choices := []string{"First item only", "All items (map)"}
	for i, choice := range choices {
		if i == p.arrayChoice {
			lines = append(lines, selectedStyle.Render("> "+choice))
		} else {
			lines = append(lines, "  "+choice)
		}
	}

	lines = append(lines, "")

	// Show resulting path
	field := p.Fields[p.Selected]
	var previewPath string
	if p.arrayChoice == 0 {
		previewPath = "$" + p.StepName + p.ItemsPath + "[0]" + field.Path
	} else {
		previewPath = "$" + p.StepName + p.ItemsPath + "[]" + field.Path
	}
	lines = append(lines, dimStyle.Render("Path: "+previewPath))

	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("[j/k] Navigate  [Enter] Select  [Esc] Back"))

	return strings.Join(lines, "\n")
}

func (p *FieldPicker) renderRawMode() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)

	var lines []string

	lines = append(lines, headerStyle.Render("Raw JSON Browser"))
	lines = append(lines, "")

	if p.RawTree != nil {
		lines = append(lines, p.RawTree.View())
	}

	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("[j/k] Navigate  [Enter] Select  [Tab] Back to fields  [Esc] Cancel"))

	return strings.Join(lines, "\n")
}

// Helper functions

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return `"` + val + `"`
	case float64:
		if val == float64(int(val)) {
			return fmt.Sprintf("%d", int(val))
		}
		return fmt.Sprintf("%.2f", val)
	case bool:
		return fmt.Sprintf("%v", val)
	case nil:
		return "null"
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func typeOf(v interface{}) string {
	switch v.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "unknown"
	}
}

func truncateValue(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
