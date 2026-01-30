# Phase 7.2: Field Picker

> **Depends on:** Phase 7.1 (Step Testing)
> **Enables:** Phase 8 (Transform Forms)
>
> See: [Full Plan](../plan.md)

## Goal

Create the FieldPicker component for selecting fields from step output with smart extraction.

## Key Deliverables

- `FieldPicker` component in `ui/components/`
- Smart extraction heuristics (Notion patterns, nested values)
- "Apply to: [First item] [All items]" for array fields
- Raw tree fallback with `[Tab]`
- Path generation for selected fields

## Files to Create

- `internal/ui/components/fieldpicker.go` — Field extraction and selection component

## Implementation Notes

### FieldPicker Struct

```go
package components

import (
    "encoding/json"
    "fmt"
    "strings"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

// ExtractedField represents a field extracted from JSON
type ExtractedField struct {
    Name      string      // friendly name (e.g., "Name", "Status")
    Path      string      // relative path (e.g., ".properties.Name.title[0].plain_text")
    Value     interface{} // sample value from first item
    ValueStr  string      // string representation for display
    Type      string      // "string", "number", "boolean", "array", "object"
    FromArray bool        // true if extracted from array item
}

// FieldPicker lets user select a field from step output
type FieldPicker struct {
    StepName    string           // which step's output (for path prefix)
    RawOutput   interface{}      // original output
    ItemsPath   string           // path to items array (e.g., ".results")
    Fields      []ExtractedField // extracted fields
    Selected    int

    // Array handling
    showArrayChoice bool
    arrayChoice     int  // 0 = first item, 1 = all items

    // Raw mode fallback
    RawMode   bool
    RawTree   *TreePicker

    styles FieldStyles
}

type FieldStyles struct {
    Normal      lipgloss.Style
    Selected    lipgloss.Style
    Value       lipgloss.Style
    Path        lipgloss.Style
    Header      lipgloss.Style
}

func DefaultFieldStyles() FieldStyles {
    return FieldStyles{
        Normal:   lipgloss.NewStyle(),
        Selected: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),
        Value:    lipgloss.NewStyle().Foreground(lipgloss.Color("2")),  // green
        Path:     lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
        Header:   lipgloss.NewStyle().Bold(true),
    }
}
```

### Constructor with Smart Extraction

```go
func NewFieldPicker(stepName string, output interface{}) *FieldPicker {
    p := &FieldPicker{
        StepName:  stepName,
        RawOutput: output,
        styles:    DefaultFieldStyles(),
    }

    p.extractFields()
    return p
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
    arrayNames := []string{"results", "data", "items", "records", "entries"}
    for _, name := range arrayNames {
        if val, ok := obj[name]; ok {
            if arr, ok := val.([]interface{}); ok {
                return "." + name, arr
            }
        }
    }

    // Look for first array field
    for key, val := range obj {
        if arr, ok := val.([]interface{}); ok && len(arr) > 0 {
            return "." + key, arr
        }
    }

    return "", nil
}
```

### Smart Extraction Heuristics

```go
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
    if num, ok := obj["number"]; ok {
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

    return nil
}

func (p *FieldPicker) extractLeafValues(obj map[string]interface{}, prefix string, fields *[]ExtractedField, fromArray bool, depth int) {
    if depth > 5 {
        return // Prevent infinite recursion
    }

    for key, val := range obj {
        path := prefix + "." + key

        switch v := val.(type) {
        case string, float64, bool:
            *fields = append(*fields, ExtractedField{
                Name:      key,
                Path:      path,
                Value:     v,
                ValueStr:  formatValue(v),
                Type:      typeOf(v),
                FromArray: fromArray,
            })
        case map[string]interface{}:
            // Check if it's a simple wrapper with one useful value
            if len(v) == 1 {
                for innerKey, innerVal := range v {
                    if _, ok := innerVal.(string); ok {
                        *fields = append(*fields, ExtractedField{
                            Name:      key,
                            Path:      path + "." + innerKey,
                            Value:     innerVal,
                            ValueStr:  formatValue(innerVal),
                            Type:      typeOf(innerVal),
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
```

### View Rendering

```go
func (p *FieldPicker) View() string {
    if p.RawMode {
        return p.RawTree.View()
    }

    if p.showArrayChoice {
        return p.renderArrayChoice()
    }

    var lines []string

    lines = append(lines, p.styles.Header.Render("Select field from: $"+p.StepName))
    lines = append(lines, "")

    if len(p.Fields) == 0 {
        lines = append(lines, "No fields extracted. Press [Tab] for raw view.")
        return strings.Join(lines, "\n")
    }

    // Show extracted fields
    for i, field := range p.Fields {
        line := p.formatFieldLine(field, i == p.Selected)
        lines = append(lines, line)
    }

    lines = append(lines, "")

    // Show full path for selected
    if p.Selected < len(p.Fields) {
        fullPath := p.buildFullPath(p.Fields[p.Selected])
        lines = append(lines, p.styles.Path.Render("Path: "+fullPath))
    }

    lines = append(lines, "")
    lines = append(lines, "[j/k] Navigate  [Enter] Select  [Tab] Raw view  [Esc] Cancel")

    return strings.Join(lines, "\n")
}

func (p *FieldPicker) formatFieldLine(field ExtractedField, selected bool) string {
    // Format: Name        "value"
    name := fmt.Sprintf("%-15s", field.Name)
    value := p.styles.Value.Render(truncate(field.ValueStr, 40))

    line := "  " + name + value

    if selected {
        return p.styles.Selected.Render("> " + field.Name + strings.Repeat(" ", 15-len(field.Name)) + truncate(field.ValueStr, 40))
    }
    return line
}

func (p *FieldPicker) renderArrayChoice() string {
    var lines []string

    lines = append(lines, p.styles.Header.Render("This field is from an array"))
    lines = append(lines, "")
    lines = append(lines, "Apply to:")
    lines = append(lines, "")

    choices := []string{"First item only", "All items"}
    for i, choice := range choices {
        if i == p.arrayChoice {
            lines = append(lines, p.styles.Selected.Render("> "+choice))
        } else {
            lines = append(lines, "  "+choice)
        }
    }

    lines = append(lines, "")
    lines = append(lines, "[j/k] Navigate  [Enter] Select  [Esc] Back")

    return strings.Join(lines, "\n")
}
```

### Path Building

```go
func (p *FieldPicker) buildFullPath(field ExtractedField) string {
    var path string

    if field.FromArray {
        // $step.results[].field or $step.results[0].field
        if p.showArrayChoice && p.arrayChoice == 0 {
            path = "$" + p.StepName + p.ItemsPath + "[0]" + field.Path
        } else {
            path = "$" + p.StepName + p.ItemsPath + "[]" + field.Path
        }
    } else {
        path = "$" + p.StepName + field.Path
    }

    return path
}

func (p *FieldPicker) SelectedPath() string {
    if p.Selected >= 0 && p.Selected < len(p.Fields) {
        return p.buildFullPath(p.Fields[p.Selected])
    }
    return ""
}
```

### Update Handling

```go
type FieldSelectedMsg struct {
    Path string
}

type FieldCancelledMsg struct{}

func (p *FieldPicker) Update(msg tea.Msg) (selectedPath string, cmd tea.Cmd) {
    if p.RawMode {
        // Delegate to raw tree
        node, cmd := p.RawTree.Update(msg)
        if node != nil {
            return p.buildRawPath(node), nil
        }
        // Check for cancel
        if _, ok := msg.(tea.KeyMsg); ok {
            if msg.(tea.KeyMsg).String() == "tab" {
                p.RawMode = false
            }
        }
        return "", cmd
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
            if p.showArrayChoice {
                // Array choice made, return path
                return p.SelectedPath(), nil
            }

            field := p.Fields[p.Selected]
            if field.FromArray {
                // Show array choice
                p.showArrayChoice = true
                p.arrayChoice = 1  // Default to "all items"
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
            } else {
                return "", func() tea.Msg { return FieldCancelledMsg{} }
            }
        }
    }

    return "", nil
}
```

### Helper Functions

```go
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

func truncate(s string, max int) string {
    if len(s) <= max {
        return s
    }
    return s[:max-3] + "..."
}
```

## Validation

- [ ] `FieldPicker` component compiles independently
- [ ] Extracts fields from flat objects
- [ ] Finds items array (results, data, items, etc.)
- [ ] Extracts Notion title properties correctly
- [ ] Extracts Notion status properties correctly
- [ ] Extracts Notion date properties correctly
- [ ] Shows field name and sample value
- [ ] j/k navigates field list
- [ ] Enter on array field shows "Apply to" choice
- [ ] "First item" generates `[0]` path
- [ ] "All items" generates `[]` path
- [ ] Full path shown at bottom
- [ ] Tab toggles raw tree view
- [ ] Raw tree allows deep navigation
- [ ] Esc cancels picker
- [ ] Path returned to parent on selection
