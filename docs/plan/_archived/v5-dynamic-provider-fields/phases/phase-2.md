# Phase 2: Dynamic Provider Form

> **Depends on:** Phase 1 (Client Layer)
> **Enables:** Complete feature
>
> See: [Full Plan](../plan.md)

## Goal

Update the provider form to fetch field requirements and dynamically render form fields.

## Key Deliverables

- `LLMProviderFieldsMsg` message type for async field loading
- Fetch fields when provider dropdown changes
- Rebuild form with dynamic fields based on API response
- Update `saveProvider()` to build `Fields` map from form values
- Client-side validation for required fields

## Files to Modify

- `internal/ui/modal/integrations_llm.go` — Dynamic form building, field fetching

## Implementation Notes

### New State Fields

Add to `IntegrationsModal` struct (in `integrations.go`):

```go
// LLM provider form state
llmProviderFields []client.ProviderFieldInfo // Field requirements for selected provider
llmLoadingFields  bool                        // Loading field requirements
```

### New Message Type

```go
// LLMProviderFieldsMsg is sent when provider field requirements are loaded.
type LLMProviderFieldsMsg struct {
    Provider string
    Fields   []client.ProviderFieldInfo
    Err      error
}
```

### Fetch Fields on Provider Change

When user changes provider dropdown in the form, fetch field requirements:

```go
// loadProviderFields fetches field requirements for the selected provider.
func (m *IntegrationsModal) loadProviderFields(providerName string) tea.Cmd {
    m.llmLoadingFields = true
    integration := m.llmIntegration.Name
    return func() tea.Msg {
        fields, err := m.client.GetLLMProviderFields(integration, providerName)
        if err != nil {
            return LLMProviderFieldsMsg{Provider: providerName, Err: err}
        }
        return LLMProviderFieldsMsg{Provider: providerName, Fields: fields}
    }
}
```

### Handle Fields Loaded

When fields arrive, rebuild the form:

```go
func (m *IntegrationsModal) handleLLMProviderFields(msg LLMProviderFieldsMsg) (Modal, tea.Cmd) {
    m.llmLoadingFields = false
    if msg.Err != nil {
        m.llmError = msg.Err.Error()
        return m, nil
    }

    m.llmProviderFields = msg.Fields
    m.rebuildProviderForm()
    return m, nil
}
```

### Rebuild Provider Form

Build form fields dynamically:

```go
func (m *IntegrationsModal) rebuildProviderForm() {
    // Get current values to preserve
    currentAccount := "default"
    currentProvider := ""
    if m.llmProviderForm != nil {
        currentAccount = m.llmProviderForm.GetFieldValue("account")
        currentProvider = m.llmProviderForm.GetFieldValue("provider")
    }

    // Build provider options
    providerOptions := make([]string, len(m.llmAvailableProviders))
    for i, p := range m.llmAvailableProviders {
        providerOptions[i] = p.DisplayName
    }

    // Start with static fields
    fields := []components.FormField{
        {
            Label:   "Provider",
            Key:     "provider",
            Type:    components.FieldSelect,
            Options: providerOptions,
            Value:   currentProvider,
        },
        {
            Label: "Account Name",
            Key:   "account",
            Type:  components.FieldText,
            Value: currentAccount,
        },
    }

    // Add dynamic fields from provider requirements
    for _, f := range m.llmProviderFields {
        field := components.FormField{
            Label:    f.Label,
            Key:      f.Key,
            Type:     components.FieldText,
            Value:    f.Default,
            Password: f.Secret,
            Required: f.Required,
        }
        fields = append(fields, field)
    }

    m.llmProviderForm = components.NewForm("Add Provider Account", fields)
}
```

### Update Provider Form Input Handler

Detect provider dropdown changes and trigger field fetch:

```go
func (m *IntegrationsModal) updateLLMProviderForm(msg tea.KeyMsg) (Modal, tea.Cmd) {
    // ... existing esc and ctrl+s handling ...

    // Track provider before form update
    prevProvider := ""
    if m.llmProviderForm != nil {
        prevProvider = m.llmProviderForm.GetFieldValue("provider")
    }

    // Forward to form
    if m.llmProviderForm != nil {
        m.llmProviderForm.Update(msg)
    }

    // Check if provider changed
    newProvider := m.llmProviderForm.GetFieldValue("provider")
    if newProvider != prevProvider && newProvider != "" {
        // Map display name to provider name
        providerName := ""
        for _, p := range m.llmAvailableProviders {
            if p.DisplayName == newProvider {
                providerName = p.Name
                break
            }
        }
        if providerName != "" {
            return m, m.loadProviderFields(providerName)
        }
    }

    return m, nil
}
```

### Update handleLLMAvailableProviders

After loading available providers, fetch fields for the first provider:

```go
func (m *IntegrationsModal) handleLLMAvailableProviders(msg LLMAvailableProvidersMsg) (Modal, tea.Cmd) {
    m.llmLoading = false
    if msg.Err != nil {
        m.llmError = msg.Err.Error()
        return m, nil
    }

    m.llmAvailableProviders = msg.Providers
    m.view = viewLLMProviderForm

    // Build initial form with just provider and account
    providerOptions := make([]string, len(m.llmAvailableProviders))
    for i, p := range m.llmAvailableProviders {
        providerOptions[i] = p.DisplayName
    }

    m.llmProviderForm = components.NewForm("Add Provider Account", []components.FormField{
        {
            Label:   "Provider",
            Key:     "provider",
            Type:    components.FieldSelect,
            Options: providerOptions,
        },
        {
            Label: "Account Name",
            Key:   "account",
            Type:  components.FieldText,
            Value: "default",
        },
    })

    // Fetch fields for first provider
    if len(m.llmAvailableProviders) > 0 {
        return m, m.loadProviderFields(m.llmAvailableProviders[0].Name)
    }

    return m, nil
}
```

### Update saveProvider

Build Fields map from form values:

```go
func (m *IntegrationsModal) saveProvider() tea.Cmd {
    values := m.llmProviderForm.Values()

    // Map display name back to provider name
    providerDisplayName := values["provider"]
    var providerName string
    for _, p := range m.llmAvailableProviders {
        if p.DisplayName == providerDisplayName {
            providerName = p.Name
            break
        }
    }

    // Build fields map from dynamic fields
    fields := make(map[string]string)
    for _, f := range m.llmProviderFields {
        if val, ok := values[f.Key]; ok {
            fields[f.Key] = val
        }
    }

    integration := m.llmIntegration.Name
    req := client.AddProviderRequest{
        Provider: providerName,
        Account:  values["account"],
        Fields:   fields,
    }

    return func() tea.Msg {
        err := m.client.AddLLMProvider(integration, req)
        if err != nil {
            return LLMProviderSavedMsg{Err: err}
        }
        return LLMProviderSavedMsg{}
    }
}
```

### Update View for Loading State

Show loading indicator while fetching fields:

```go
func (m *IntegrationsModal) viewLLMProviderForm() string {
    var lines []string

    // Show form
    if m.llmProviderForm != nil {
        lines = append(lines, m.llmProviderForm.View())
    }

    // Show loading indicator for fields
    if m.llmLoadingFields {
        lines = append(lines, "")
        lines = append(lines, lipgloss.NewStyle().
            Foreground(theme.TextSecondary).
            Render("  Loading fields..."))
    }

    // ... rest of view (error, saving, hints) ...
}
```

### Client-Side Validation

Before saving, validate required fields:

```go
func (m *IntegrationsModal) validateProviderForm() error {
    values := m.llmProviderForm.Values()

    // Check account name
    if strings.TrimSpace(values["account"]) == "" {
        return fmt.Errorf("account name is required")
    }

    // Check required dynamic fields
    for _, f := range m.llmProviderFields {
        if f.Required {
            val := strings.TrimSpace(values[f.Key])
            if val == "" {
                return fmt.Errorf("%s is required", f.Label)
            }
        }
    }

    return nil
}
```

Call validation in `updateLLMProviderForm` before save:

```go
case "ctrl+s":
    if !m.llmSavingProvider && m.llmProviderForm != nil {
        if err := m.validateProviderForm(); err != nil {
            m.llmError = err.Error()
            return m, nil
        }
        m.llmSavingProvider = true
        return m, m.saveProvider()
    }
```

## Validation

- [ ] Selecting a provider in the form fetches field requirements
- [ ] Form shows dynamic fields based on provider (api_key for openai, base_url for ollama)
- [ ] Fields respect `secret` property (masked input)
- [ ] Fields are pre-populated with `default` values
- [ ] Required field validation works before submit
- [ ] Changing provider preserves account name
- [ ] Submitting provider sends `{"fields": {...}}` format
- [ ] OpenAI provider can be added successfully
- [ ] Ollama provider can be added successfully
- [ ] Code compiles and runs without errors
