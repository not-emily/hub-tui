# Phase 4: LLM Provider Management

> **Depends on:** Phase 3 (LLM Config View)
> **Enables:** Phase 5.1 (LLM Profile Form)
>
> See: [Full Plan](../plan.md)

## Goal

Add and delete provider accounts from the LLM config view.

## Key Deliverables

- Provider form (viewLLMProviderForm)
- Add provider flow with provider dropdown, account name, API key fields
- Delete provider with double-press confirmation
- Provider header disappears when last account deleted

## Files to Modify

- `internal/ui/modal/integrations_llm.go` — Add provider form and delete logic

## Dependencies

**Internal:**
- Phase 3 complete (LLM config view exists)
- Client methods: `ListAvailableLLMProviders`, `AddLLMProvider`, `DeleteLLMProvider`

**External:** None

## Implementation Notes

### Provider Form State

Add fields to modal struct:

```go
// Provider form state
llmProviderForm     *components.Form
llmAvailableProviders []AvailableProvider
llmSavingProvider   bool
```

### Enter Provider Form

When user presses `[a]` in LLM config view:

```go
case "a":
    return m, m.enterLLMProviderForm()

func (m *IntegrationsModal) enterLLMProviderForm() tea.Cmd {
    return func() tea.Msg {
        // Load available providers
        available, err := m.client.ListAvailableLLMProviders(m.llmIntegration.Name)
        if err != nil {
            return llmErrorMsg{err: err}
        }
        return llmAvailableProvidersMsg{providers: available}
    }
}
```

On receiving available providers, build the form:

```go
case llmAvailableProvidersMsg:
    m.llmAvailableProviders = msg.providers
    m.view = viewLLMProviderForm

    // Build provider options
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
        {
            Label:    "API Key",
            Key:      "api_key",
            Type:     components.FieldText,
            Password: true,
        },
    })
    return m, nil
```

### Provider Form View

```go
func (m *IntegrationsModal) viewLLMProviderForm() string {
    if m.llmSavingProvider {
        return "Saving provider account..."
    }
    return m.llmProviderForm.View()
}
```

### Save Provider

On `Ctrl+S` in provider form:

```go
case "ctrl+s":
    if m.view == viewLLMProviderForm {
        return m, m.saveProvider()
    }

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

    m.llmSavingProvider = true
    return func() tea.Msg {
        err := m.client.AddLLMProvider(m.llmIntegration.Name, AddProviderRequest{
            Provider: providerName,
            Account:  values["account"],
            APIKey:   values["api_key"],
        })
        if err != nil {
            return llmProviderSavedMsg{err: err}
        }
        return llmProviderSavedMsg{}
    }
}
```

On success, return to config view and refresh:

```go
case llmProviderSavedMsg:
    m.llmSavingProvider = false
    if msg.err != nil {
        m.llmError = msg.err.Error()
        return m, nil
    }
    m.view = viewConfigLLM
    return m, m.loadLLMData()
```

### Delete Provider

Use the reusable Confirmation component (from `components/confirm.go`):

```go
// Add to modal struct
llmConfirm components.Confirmation
```

When user presses `[d]` on a provider account:

```go
case "d":
    if m.llmSelected < 0 || m.llmSelected >= len(m.llmItems) {
        return m, nil
    }
    item := m.llmItems[m.llmSelected]

    if item.Type == llmItemProviderAccount {
        key := fmt.Sprintf("provider:%s/%s", item.Provider, item.Account)
        if execute, cmd := m.llmConfirm.Check(key, item.Account); execute {
            return m, m.deleteProvider(item.Provider, item.Account)
        } else if cmd != nil {
            return m, cmd
        }
    }
    // Profile delete in Phase 5.2
```

Delete and refresh:

```go
func (m *IntegrationsModal) deleteProvider(provider, account string) tea.Cmd {
    return func() tea.Msg {
        err := m.client.DeleteLLMProvider(m.llmIntegration.Name, provider, account)
        if err != nil {
            return llmErrorMsg{err: err}
        }
        return llmProviderDeletedMsg{}
    }
}

case llmProviderDeletedMsg:
    return m, m.loadLLMData() // Refresh will remove empty provider headers
```

### Cancel Form

`Esc` in provider form returns to config view without saving:

```go
case "esc":
    if m.view == viewLLMProviderForm {
        m.view = viewConfigLLM
        m.llmProviderForm = nil
        return m, nil
    }
```

### Confirmation Hint

Show confirmation hint in status when pending:

```go
func (m *IntegrationsModal) viewLLM() string {
    // ... existing render ...

    if m.llmConfirm.IsPending() {
        b.WriteString(fmt.Sprintf("\nPress d again to delete %s", m.llmConfirm.PendingLabel()))
    }

    return b.String()
}
```

## Validation

- [ ] `go build` succeeds
- [ ] `[a]` opens provider form
- [ ] Provider dropdown shows available providers
- [ ] Can enter account name and API key
- [ ] `Ctrl+S` saves provider and returns to config view
- [ ] New provider account appears in list
- [ ] `Esc` cancels form without saving
- [ ] `[d]` on provider account shows confirmation hint
- [ ] Second `[d]` press deletes the account
- [ ] Provider header disappears when last account deleted
- [ ] Confirmation expires after 2 seconds
