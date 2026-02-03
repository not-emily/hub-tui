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

func debugLog(msg string) {
	// Debug logging disabled - uncomment to enable
	// f, _ := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	// if f != nil {
	// 	f.WriteString(msg + "\n")
	// 	f.Close()
	// }
	_ = msg
}

// FieldSelectedMsg is sent when a field is selected.
type FieldSelectedMsg struct {
	Path string
}

// FieldCancelledMsg is sent when field selection is cancelled.
type FieldCancelledMsg struct{}

// PickerNeedsTestMsg is sent when the picker needs a variable's step to be tested.
type PickerNeedsTestMsg struct {
	VarName string
}

// PickerTestResultMsg is sent when a test requested by the picker completes.
type PickerTestResultMsg struct {
	VarName string
	Output  interface{}
	Error   error
}

// ExtractedField represents a field extracted from JSON.
type ExtractedField struct {
	Name       string      // friendly name (e.g., "Name", "Status")
	Path       string      // relative path (e.g., ".properties.Name.title[0].plain_text")
	Value      interface{} // sample value from first item
	ValueStr   string      // string representation for display
	Type       string      // "string", "number", "boolean", "array", "object", "array_item", "field_value"
	FromArray  bool        // true if extracted from array item (selecting shows value picker)
	ArrayIndex int         // for array_item/field_value type: which index this represents
}

// FieldPicker lets user select a field from step output.
type FieldPicker struct {
	StepName  string           // which step's output (for path prefix)
	RawOutput interface{}      // original output
	ItemsPath string           // path to items array (e.g., ".results")
	Fields    []ExtractedField // extracted fields
	Selected  int
	height    int

	// Navigation into nested structures
	navStack []navLevel // stack of navigation levels for drilling into arrays/objects

	// Raw mode fallback
	RawMode bool
	RawTree *TreePicker

	// Multi-step support
	multiStep       bool                   // true if showing multiple steps
	allSteps        map[string]interface{} // all previous step outputs
	varNames        []string               // ordered variable names (save_as values)
	varToStepName   map[string]string      // variable name -> step name mapping
	selectingStep   bool                   // true when picking which step
	selectedStepIdx int                    // selected step index

	// Test request state (when variable has no output yet)
	awaitingTest    bool   // true when waiting for test result
	awaitingTestVar string // variable name we're waiting on
	testError       string // error from test if any
}

// navLevel tracks a level in the navigation stack
type navLevel struct {
	path       string        // path to this level (e.g., ".recipes" or "[0]")
	isArray    bool          // true if this level is an array
	arrayData  []interface{} // the array data (for extracting field values)
	fieldPath  string        // if showing field values, the field path (e.g., ".name")
	selected   int           // selected index before drilling (for restoring)
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
// stepOutputs maps variable name (save_as) -> output value
// varToStepName maps variable name -> step name (for display)
func NewFieldPickerMulti(stepOutputs map[string]interface{}, varToStepName map[string]string) *FieldPicker {
	p := &FieldPicker{
		height:        10,
		multiStep:     true,
		allSteps:      stepOutputs,
		varToStepName: varToStepName,
		selectingStep: true, // Always show variable selection first
	}

	// Get sorted variable names for consistent ordering
	for name := range stepOutputs {
		p.varNames = append(p.varNames, name)
	}
	sort.Strings(p.varNames)

	return p
}

// SetHeight sets the visible height.
func (p *FieldPicker) SetHeight(height int) {
	p.height = height
}

func (p *FieldPicker) extractFields() {
	// Always extract from root object first - don't auto-drill into arrays
	// This gives users control over what level they want to select
	p.ItemsPath = ""
	p.Fields = p.extractTopLevelFields(p.RawOutput)

	// Sort fields by name
	sort.Slice(p.Fields, func(i, j int) bool {
		return p.Fields[i].Name < p.Fields[j].Name
	})

	// Add "(entire variable)" option at the top
	entireVar := ExtractedField{
		Name:      "(entire variable)",
		Path:      "", // empty path means just $stepName
		Value:     p.RawOutput,
		ValueStr:  p.describeRootOutput(),
		Type:      typeOf(p.RawOutput),
		FromArray: false,
	}
	p.Fields = append([]ExtractedField{entireVar}, p.Fields...)
}

// describeRootOutput returns a brief description of the actual root output structure.
func (p *FieldPicker) describeRootOutput() string {
	switch v := p.RawOutput.(type) {
	case map[string]interface{}:
		return fmt.Sprintf("{%d keys}", len(v))
	case []interface{}:
		return fmt.Sprintf("[%d items]", len(v))
	default:
		return truncateValue(formatValue(v), 20)
	}
}

// extractTopLevelFields extracts fields from the root level of the data.
// For objects, it shows each key. For arrays, it shows item fields with array notation.
func (p *FieldPicker) extractTopLevelFields(data interface{}) []ExtractedField {
	var fields []ExtractedField

	switch v := data.(type) {
	case map[string]interface{}:
		// Show each key in the object
		for key, val := range v {
			field := ExtractedField{
				Name:      key,
				Path:      "." + key,
				Value:     val,
				Type:      typeOf(val),
				FromArray: false,
			}
			// Format the value description
			switch tv := val.(type) {
			case []interface{}:
				field.ValueStr = fmt.Sprintf("[%d items]", len(tv))
			case map[string]interface{}:
				field.ValueStr = fmt.Sprintf("{%d keys}", len(tv))
			default:
				field.ValueStr = formatValue(val)
			}
			fields = append(fields, field)
		}

	case []interface{}:
		// Data is an array at root - show fields from first item
		if len(v) > 0 {
			// Push a synthetic nav level for the root array
			p.navStack = []navLevel{{
				path:      "",
				isArray:   true,
				arrayData: v,
			}}
			fields = p.extractFromItem(v[0], true)
			// Sort fields
			sort.Slice(fields, func(i, j int) bool {
				return fields[i].Name < fields[j].Name
			})
			// Add "(entire variable)" option
			entireVar := ExtractedField{
				Name:     "(entire variable)",
				Path:     "",
				Value:    v,
				ValueStr: fmt.Sprintf("[%d items]", len(v)),
				Type:     "array",
			}
			fields = append([]ExtractedField{entireVar}, fields...)
		}
	}

	return fields
}

const maxArrayItemsToShow = 20

// extractArrayItems creates fields for each item in an array (for index selection).
func (p *FieldPicker) extractArrayItems(arr []interface{}) []ExtractedField {
	var fields []ExtractedField

	// Add "(entire array)" option first
	entireArray := ExtractedField{
		Name:     "(entire array)",
		Path:     "", // empty means just the array itself
		Value:    arr,
		ValueStr: fmt.Sprintf("[%d items]", len(arr)),
		Type:     "array",
	}
	fields = append(fields, entireArray)

	// Add each array item (capped for large arrays)
	limit := len(arr)
	if limit > maxArrayItemsToShow {
		limit = maxArrayItemsToShow
	}

	for i := 0; i < limit; i++ {
		item := arr[i]
		// Create a preview of the item
		preview := p.itemPreview(item)
		fields = append(fields, ExtractedField{
			Name:       fmt.Sprintf("[%d]", i),
			Path:       fmt.Sprintf("[%d]", i),
			Value:      item,
			ValueStr:   preview,
			Type:       "array_item",
			ArrayIndex: i,
		})
	}

	// If array was truncated, add indicator
	if len(arr) > maxArrayItemsToShow {
		fields = append(fields, ExtractedField{
			Name:     fmt.Sprintf("... and %d more", len(arr)-maxArrayItemsToShow),
			Path:     "",
			ValueStr: "(use raw view for full list)",
			Type:     "truncated",
		})
	}

	return fields
}

// extractFieldValues extracts values of a specific field from all array items.
// Shows field values so user can pick which array index they want.
func (p *FieldPicker) extractFieldValues(arr []interface{}, fieldPath string) []ExtractedField {
	var fields []ExtractedField

	// Cap at maxArrayItemsToShow
	limit := len(arr)
	if limit > maxArrayItemsToShow {
		limit = maxArrayItemsToShow
	}

	for i := 0; i < limit; i++ {
		item := arr[i]
		// Extract the field value from this item
		val := p.getFieldValue(item, fieldPath)
		fields = append(fields, ExtractedField{
			Name:       fmt.Sprintf("[%d]", i),
			Path:       fmt.Sprintf("[%d]%s", i, fieldPath),
			Value:      val,
			ValueStr:   truncateValue(formatValue(val), 40),
			Type:       "field_value",
			ArrayIndex: i,
		})
	}

	// If array was truncated, add indicator
	if len(arr) > maxArrayItemsToShow {
		fields = append(fields, ExtractedField{
			Name:     fmt.Sprintf("... and %d more", len(arr)-maxArrayItemsToShow),
			Path:     "",
			ValueStr: "(use raw view for full list)",
			Type:     "truncated",
		})
	}

	return fields
}

// getFieldValue extracts a value from an object using a field path like ".name" or ".category".
func (p *FieldPicker) getFieldValue(item interface{}, fieldPath string) interface{} {
	if fieldPath == "" {
		return item
	}

	obj, ok := item.(map[string]interface{})
	if !ok {
		return nil
	}

	// Parse the field path - handle simple ".field" case
	path := strings.TrimPrefix(fieldPath, ".")
	parts := strings.SplitN(path, ".", 2)
	key := parts[0]

	val, ok := obj[key]
	if !ok {
		return nil
	}

	// If there's more path, recurse
	if len(parts) > 1 {
		return p.getFieldValue(val, "."+parts[1])
	}

	return val
}

// itemPreview returns a short preview string for an array item.
func (p *FieldPicker) itemPreview(item interface{}) string {
	switch v := item.(type) {
	case map[string]interface{}:
		// Try common name fields first
		for _, key := range []string{"name", "title", "id", "label", "text", "value", "description", "subject", "content"} {
			if val, ok := v[key]; ok {
				if str := extractStringValue(val); str != "" {
					return truncateValue(str, 35)
				}
			}
		}
		// Check for Notion-style properties.Name.title structure
		if props, ok := v["properties"].(map[string]interface{}); ok {
			for _, key := range []string{"Name", "Title", "name", "title"} {
				if prop, ok := props[key].(map[string]interface{}); ok {
					if str := extractNotionText(prop); str != "" {
						return truncateValue(str, 35)
					}
				}
			}
		}
		// Fall back to showing first few key-value pairs
		return p.objectPreview(v, 35)
	case []interface{}:
		return fmt.Sprintf("[%d items]", len(v))
	default:
		return truncateValue(formatValue(item), 35)
	}
}

// extractStringValue tries to get a string from various value types.
func extractStringValue(val interface{}) string {
	switch v := val.(type) {
	case string:
		return v
	case float64:
		if v == float64(int(v)) {
			return fmt.Sprintf("%d", int(v))
		}
		return fmt.Sprintf("%.2f", v)
	case bool:
		return fmt.Sprintf("%v", v)
	default:
		return ""
	}
}

// extractNotionText extracts text from Notion property structures.
func extractNotionText(prop map[string]interface{}) string {
	// title: [{plain_text: "..."}]
	if title, ok := prop["title"].([]interface{}); ok && len(title) > 0 {
		if first, ok := title[0].(map[string]interface{}); ok {
			if text, ok := first["plain_text"].(string); ok {
				return text
			}
		}
	}
	// rich_text: [{plain_text: "..."}]
	if richText, ok := prop["rich_text"].([]interface{}); ok && len(richText) > 0 {
		if first, ok := richText[0].(map[string]interface{}); ok {
			if text, ok := first["plain_text"].(string); ok {
				return text
			}
		}
	}
	return ""
}

// objectPreview returns a preview of an object's first few fields.
func (p *FieldPicker) objectPreview(obj map[string]interface{}, maxLen int) string {
	if len(obj) == 0 {
		return "{}"
	}
	// Get sorted keys for consistent ordering
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	totalLen := 2 // for "{ }"
	for _, k := range keys {
		v := obj[k]
		var valStr string
		switch tv := v.(type) {
		case string:
			valStr = fmt.Sprintf(`"%s"`, truncateValue(tv, 15))
		case float64, bool:
			valStr = fmt.Sprintf("%v", tv)
		case []interface{}:
			valStr = fmt.Sprintf("[%d]", len(tv))
		case map[string]interface{}:
			valStr = "{...}"
		default:
			valStr = "..."
		}
		part := fmt.Sprintf("%s: %s", k, valStr)
		if totalLen+len(part)+2 > maxLen { // +2 for ", "
			if len(parts) > 0 {
				parts = append(parts, "...")
			}
			break
		}
		parts = append(parts, part)
		totalLen += len(part) + 2
	}
	return "{" + strings.Join(parts, ", ") + "}"
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
	// Handle test result message
	if msg, ok := msg.(PickerTestResultMsg); ok {
		p.awaitingTest = false
		if msg.Error != nil {
			p.testError = msg.Error.Error()
		} else {
			// Update the output data and extract fields
			p.allSteps[msg.VarName] = msg.Output
			p.RawOutput = msg.Output
			p.testError = ""
			p.awaitingTestVar = "" // Clear awaiting state
			p.selectingStep = false // Transition to field selection
			p.Selected = 0
			p.extractFields()
		}
		return "", nil
	}

	// Handle step selection phase (or showing "needs test" prompt)
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
			if p.Selected < len(p.Fields)-1 {
				p.Selected++
			}

		case "k", "up":
			if p.Selected > 0 {
				p.Selected--
			}

		case "enter":
			if len(p.Fields) == 0 {
				return "", nil
			}

			field := p.Fields[p.Selected]

			// Ignore truncated indicator
			if field.Type == "truncated" {
				return "", nil
			}

			// Check if this is a field value from the value picker (e.g., selecting "Pasta Salad" from name values)
			if field.Type == "field_value" {
				// Return the path with the selected index
				return p.SelectedPath(), nil
			}

			// Check if this is a field from an array - show value picker
			if field.FromArray && field.Path != "" {
				// Get the array data from the current nav level
				if len(p.navStack) > 0 {
					currentLevel := p.navStack[len(p.navStack)-1]
					if currentLevel.isArray && currentLevel.arrayData != nil {
						// Show values of this field across all array items
						p.navStack = append(p.navStack, navLevel{
							path:      "", // no additional path yet
							isArray:   false,
							fieldPath: field.Path,
							selected:  p.Selected,
						})
						p.Fields = p.extractFieldValues(currentLevel.arrayData, field.Path)
						p.Selected = 0
						return "", nil
					}
				}
			}

			// Check if this field is an array we should drill into
			// (but not "(entire array)" which has empty path - that's for selecting the current array)
			if field.Type == "array" && !field.FromArray && field.Path != "" {
				// Drill into the array - show fields from first item
				arr, ok := field.Value.([]interface{})
				if ok && len(arr) > 0 {
					// Save current state to nav stack (including array data for later)
					p.navStack = append(p.navStack, navLevel{
						path:      field.Path,
						isArray:   true,
						arrayData: arr,
						selected:  p.Selected,
					})
					// Extract fields from first item with FromArray=true
					p.Fields = p.extractFromItem(arr[0], true)
					// Sort and add "(entire array)" option
					sort.Slice(p.Fields, func(i, j int) bool {
						return p.Fields[i].Name < p.Fields[j].Name
					})
					entireArray := ExtractedField{
						Name:     "(entire array)",
						Path:     "", // empty means just the array itself
						Value:    arr,
						ValueStr: fmt.Sprintf("[%d items]", len(arr)),
						Type:     "array",
					}
					p.Fields = append([]ExtractedField{entireArray}, p.Fields...)
					p.Selected = 0
					return "", nil
				}
			}

			// Return the selected path
			return p.SelectedPath(), nil

		case "tab":
			p.RawMode = true
			if p.RawTree == nil {
				p.RawTree = p.buildRawTree()
			}

		case "esc":
			if len(p.navStack) > 0 {
				// Pop navigation stack and go back up
				prev := p.navStack[len(p.navStack)-1]
				p.navStack = p.navStack[:len(p.navStack)-1]
				p.ItemsPath = p.currentNavPath()
				p.extractFields() // Re-extract at current level
				p.Selected = prev.selected
			} else if p.multiStep && len(p.varNames) > 1 {
				// Go back to step selection (only if there are multiple variables)
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
	// If awaiting test result, only handle esc
	if p.awaitingTest {
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
			p.awaitingTest = false
			p.awaitingTestVar = ""
			return "", func() tea.Msg { return FieldCancelledMsg{} }
		}
		return "", nil
	}

	// If we've selected a variable but it has no output, show test prompt
	if p.awaitingTestVar != "" {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			debugLog(fmt.Sprintf("FieldPicker: awaitingTestVar=%s, key=%s", p.awaitingTestVar, msg.String()))
			switch msg.String() {
			case "t":
				// Request test for this variable
				debugLog(fmt.Sprintf("FieldPicker: [t] pressed, returning PickerNeedsTestMsg for %s", p.awaitingTestVar))
				p.awaitingTest = true
				p.testError = ""
				return "", func() tea.Msg {
					debugLog("FieldPicker: PickerNeedsTestMsg command executing")
					return PickerNeedsTestMsg{VarName: p.awaitingTestVar}
				}
			case "enter":
				// Select entire variable without drilling into fields
				return "$" + p.awaitingTestVar, nil
			case "esc":
				// Go back to variable list
				p.awaitingTestVar = ""
				p.testError = ""
			}
		}
		return "", nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if p.selectedStepIdx < len(p.varNames)-1 {
				p.selectedStepIdx++
			}
		case "k", "up":
			if p.selectedStepIdx > 0 {
				p.selectedStepIdx--
			}
		case "enter":
			// Select this variable
			varName := p.varNames[p.selectedStepIdx]
			p.StepName = varName
			p.RawOutput = p.allSteps[varName]

			// Check if output is nil (not tested yet)
			if p.RawOutput == nil {
				p.awaitingTestVar = varName
				return "", nil
			}

			// Has output - extract fields and continue
			p.extractFields()
			p.selectingStep = false
			p.Selected = 0
		case "esc":
			return "", func() tea.Msg {
				return FieldCancelledMsg{}
			}
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

// currentNavPath returns the path built from the navigation stack.
func (p *FieldPicker) currentNavPath() string {
	var path string
	for _, level := range p.navStack {
		path += level.path
	}
	return path
}

// SelectedPath returns the full path for the selected field.
func (p *FieldPicker) SelectedPath() string {
	if p.Selected < 0 || p.Selected >= len(p.Fields) {
		return ""
	}
	return p.buildFullPath(p.Fields[p.Selected])
}

func (p *FieldPicker) buildFullPath(field ExtractedField) string {
	navPath := p.currentNavPath()
	// Path is always: $stepName + navPath + field.Path
	// navPath includes array indices from navigation (e.g., ".recipes[1]")
	// field.Path is the field within the current level (e.g., ".name")
	return "$" + p.StepName + navPath + field.Path
}

// View renders the field picker.
func (p *FieldPicker) View() string {
	if p.selectingStep {
		return p.renderStepSelection()
	}

	if p.RawMode && p.RawTree != nil {
		return p.renderRawMode()
	}

	return p.renderFieldList()
}

func (p *FieldPicker) renderStepSelection() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	errorStyle := lipgloss.NewStyle().Foreground(theme.Error)

	var lines []string

	// If awaiting test result, show loading
	if p.awaitingTest {
		lines = append(lines, headerStyle.Render("Variable: $"+p.awaitingTestVar))
		lines = append(lines, "")
		lines = append(lines, "Fetching sample output...")
		lines = append(lines, "")
		lines = append(lines, dimStyle.Render("[Esc] Cancel"))
		return strings.Join(lines, "\n")
	}

	// If showing "needs test" prompt for a selected variable
	if p.awaitingTestVar != "" {
		lines = append(lines, headerStyle.Render("Variable: $"+p.awaitingTestVar))
		lines = append(lines, "")

		if p.testError != "" {
			lines = append(lines, errorStyle.Render("Error: "+p.testError))
			lines = append(lines, "")
		}

		lines = append(lines, dimStyle.Render("No sample output available yet."))
		lines = append(lines, "")
		lines = append(lines, "Press [t] to fetch sample output and see available fields,")
		lines = append(lines, "or [Enter] to use the entire variable.")
		lines = append(lines, "")
		lines = append(lines, dimStyle.Render("[t] Fetch output  [Enter] Use $"+p.awaitingTestVar+"  [Esc] Back"))
		return strings.Join(lines, "\n")
	}

	lines = append(lines, headerStyle.Render("Select Variable"))
	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("Choose a previous step's output:"))
	lines = append(lines, "")

	for i, varName := range p.varNames {
		suffix := p.getStepNameSuffix(varName)

		// Show indicator if not tested
		if p.allSteps[varName] == nil {
			suffix += dimStyle.Render("  (not tested)")
		}

		if i == p.selectedStepIdx {
			lines = append(lines, selectedStyle.Render("> $"+varName)+suffix)
		} else {
			lines = append(lines, "  $"+varName+suffix)
		}
	}

	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("[j/k] Navigate  [Enter] Select  [Esc] Cancel"))

	return strings.Join(lines, "\n")
}

// getStepNameSuffix returns the step name suffix for display.
func (p *FieldPicker) getStepNameSuffix(varName string) string {
	if p.varToStepName != nil {
		if stepName, ok := p.varToStepName[varName]; ok && stepName != varName {
			return "  (from " + stepName + ")"
		}
	}
	return ""
}

func (p *FieldPicker) renderFieldList() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	dimStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
	selectedStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(theme.Success)

	var lines []string

	// Show current navigation path
	navPath := p.currentNavPath()
	if navPath != "" {
		lines = append(lines, headerStyle.Render("Select field from: $"+p.StepName+navPath))
	} else {
		lines = append(lines, headerStyle.Render("Select field from: $"+p.StepName))
	}
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
	if len(p.navStack) > 0 {
		lines = append(lines, dimStyle.Render("[j/k] Navigate  [Enter] Select  [Tab] Raw view  [Esc] Back"))
	} else {
		lines = append(lines, dimStyle.Render("[j/k] Navigate  [Enter] Select  [Tab] Raw view  [Esc] Cancel"))
	}

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
