# Phase 5.1: LLM Profile Form

> **Depends on:** Phase 4 (LLM Provider Management)
> **Enables:** Phase 5.2 (LLM Profile Operations)
>
> See: [Full Plan](../plan.md)

## Goal

Create and edit LLM profiles with cascading dropdowns and model pagination.

## Key Deliverables

- Profile form (viewLLMProfileForm)
- Cascading dropdowns: Provider → Account → Model
- Model pagination with [p]/[n] keys
- Create new profile
- Edit existing profile
- "Set as default" checkbox

## Files to Modify

- `internal/ui/modal/integrations_llm.go` — Add profile form logic

## Dependencies

**Internal:**
- Phase 4 complete (provider management works)
- Client methods: `CreateLLMProfile`, `UpdateLLMProfile`, `ListIntegrationModels`

**External:** None

## Implementation Notes

### Profile Form State

Add fields to modal struct:

```go
// Profile form state
llmProfileForm     *components.Form
llmEditingProfile  *LLMProfile // nil if creating new
llmSavingProfile   bool

// Model pagination
llmModels          []ModelInfo
llmModelsCursors   []string // stack for back navigation
llmModelsHasMore   bool
llmModelsNextCursor string
llmModelsPage      int
llmLoadingModels   bool
```

### Enter Profile Form

**New profile** (`[n]` key):

```go
case "n":
    m.llmEditingProfile = nil
    return m, m.enterLLMProfileForm()
```

**Edit existing** (`Enter` on profile):

```go
case "enter":
    item := m.llmItems[m.llmSelected]
    if item.Type == llmItemProfile {
        m.llmEditingProfile = item.Profile
        return m, m.enterLLMProfileForm()
    }
```

### Build Profile Form

```go
func (m *IntegrationsModal) enterLLMProfileForm() tea.Cmd {
    m.view = viewLLMProfileForm

    // Build provider options from configured providers
    providerOptions := make([]string, 0)
    for _, p := range m.llmProviders {
        if len(p.Accounts) > 0 {
            providerOptions = append(providerOptions, p.DisplayName)
        }
    }

    // Initial values
    nameVal := ""
    providerVal := ""
    accountVal := ""
    modelVal := ""
    isDefault := false

    if m.llmEditingProfile != nil {
        nameVal = m.llmEditingProfile.Name
        providerVal = m.getProviderDisplayName(m.llmEditingProfile.Provider)
        accountVal = m.llmEditingProfile.Account
        modelVal = m.llmEditingProfile.Model
        isDefault = m.llmEditingProfile.IsDefault
    } else if len(providerOptions) > 0 {
        providerVal = providerOptions[0]
    }

    m.llmProfileForm = components.NewForm("LLM Profile", []components.FormField{
        {
            Label: "Name",
            Key:   "name",
            Type:  components.FieldText,
            Value: nameVal,
        },
        {
            Label:   "Provider",
            Key:     "provider",
            Type:    components.FieldSelect,
            Options: providerOptions,
            Value:   providerVal,
        },
        {
            Label:   "Account",
            Key:     "account",
            Type:    components.FieldSelect,
            Options: []string{}, // populated by cascade
            Value:   accountVal,
        },
        {
            Label:   "Model",
            Key:     "model",
            Type:    components.FieldSelect,
            Options: []string{}, // populated by cascade
            Value:   modelVal,
        },
        {
            Label:   "Set as default",
            Key:     "is_default",
            Type:    components.FieldCheckbox,
            Checked: isDefault,
        },
    })

    // Trigger initial cascade
    return m.cascadeFromProvider()
}
```

### Cascading Logic

When provider changes, reload accounts:

```go
func (m *IntegrationsModal) cascadeFromProvider() tea.Cmd {
    providerDisplayName := m.llmProfileForm.GetFieldValue("provider")
    providerName := m.getProviderName(providerDisplayName)

    // Find accounts for this provider
    var accounts []string
    for _, p := range m.llmProviders {
        if p.Provider == providerName {
            accounts = p.Accounts
            break
        }
    }

    m.llmProfileForm.SetFieldOptions("account", accounts, "")

    // Reset models and trigger model load
    m.llmModels = nil
    m.llmModelsCursors = nil
    m.llmModelsPage = 1
    return m.loadModels("")
}
```

When account changes, reload models:

```go
func (m *IntegrationsModal) cascadeFromAccount() tea.Cmd {
    m.llmModels = nil
    m.llmModelsCursors = nil
    m.llmModelsPage = 1
    return m.loadModels("")
}
```

### Model Loading

```go
const modelsPageSize = 10

func (m *IntegrationsModal) loadModels(cursor string) tea.Cmd {
    m.llmLoadingModels = true
    providerDisplayName := m.llmProfileForm.GetFieldValue("provider")
    providerName := m.getProviderName(providerDisplayName)

    return func() tea.Msg {
        result, err := m.client.ListIntegrationModels(providerName, modelsPageSize, cursor)
        if err != nil {
            return llmModelsLoadedMsg{err: err}
        }
        return llmModelsLoadedMsg{
            models:     result.Models,
            hasMore:    result.Pagination.HasMore,
            nextCursor: result.Pagination.NextCursor,
        }
    }
}

case llmModelsLoadedMsg:
    m.llmLoadingModels = false
    if msg.err != nil {
        m.llmError = msg.err.Error()
        return m, nil
    }
    m.llmModels = msg.models
    m.llmModelsHasMore = msg.hasMore
    m.llmModelsNextCursor = msg.nextCursor

    // Update model options
    modelOptions := make([]string, len(m.llmModels))
    for i, model := range m.llmModels {
        modelOptions[i] = model.ID
    }
    m.llmProfileForm.SetFieldOptions("model", modelOptions, "")
    return m, nil
```

### Model Pagination

Handle `[p]`/`[n]` when model field is focused:

```go
case "p": // Previous page
    if m.llmProfileForm.IsFieldFocused("model") && len(m.llmModelsCursors) > 0 {
        // Pop cursor stack
        prevCursor := ""
        if len(m.llmModelsCursors) > 1 {
            prevCursor = m.llmModelsCursors[len(m.llmModelsCursors)-2]
            m.llmModelsCursors = m.llmModelsCursors[:len(m.llmModelsCursors)-1]
        } else {
            m.llmModelsCursors = nil
        }
        m.llmModelsPage--
        return m, m.loadModels(prevCursor)
    }

case "n": // Next page
    if m.llmProfileForm.IsFieldFocused("model") && m.llmModelsHasMore {
        m.llmModelsCursors = append(m.llmModelsCursors, m.llmModelsNextCursor)
        m.llmModelsPage++
        return m, m.loadModels(m.llmModelsNextCursor)
    }
```

### Model Description

Show model description when model field is focused:

```go
func (m *IntegrationsModal) viewLLMProfileForm() string {
    var b strings.Builder
    b.WriteString(m.llmProfileForm.View())

    // Show model description if model field focused
    if m.llmProfileForm.IsFieldFocused("model") {
        modelID := m.llmProfileForm.GetFieldValue("model")
        for _, model := range m.llmModels {
            if model.ID == modelID && model.Description != "" {
                b.WriteString(fmt.Sprintf("\n\n%s", model.Description))
                break
            }
        }

        // Pagination info
        if m.llmModelsHasMore || m.llmModelsPage > 1 {
            b.WriteString(fmt.Sprintf("\n\nPage %d  [p] prev  [n] next", m.llmModelsPage))
        }
    }

    return b.String()
}
```

### Detect Field Changes

Need to detect when provider or account field changes to trigger cascade:

```go
func (m *IntegrationsModal) updateLLMProfileForm(msg tea.KeyMsg) (Modal, tea.Cmd) {
    prevProvider := m.llmProfileForm.GetFieldValue("provider")
    prevAccount := m.llmProfileForm.GetFieldValue("account")

    // Let form handle the key
    m.llmProfileForm.Update(msg)

    // Check for cascades
    newProvider := m.llmProfileForm.GetFieldValue("provider")
    newAccount := m.llmProfileForm.GetFieldValue("account")

    if newProvider != prevProvider {
        return m, m.cascadeFromProvider()
    }
    if newAccount != prevAccount {
        return m, m.cascadeFromAccount()
    }

    return m, nil
}
```

### Save Profile

```go
case "ctrl+s":
    if m.view == viewLLMProfileForm {
        return m, m.saveProfile()
    }

func (m *IntegrationsModal) saveProfile() tea.Cmd {
    values := m.llmProfileForm.Values()
    providerName := m.getProviderName(values["provider"])
    isDefault := m.llmProfileForm.GetFieldChecked("is_default")

    m.llmSavingProfile = true

    return func() tea.Msg {
        req := CreateProfileRequest{
            Name:     values["name"],
            Provider: providerName,
            Account:  values["account"],
            Model:    values["model"],
        }

        var err error
        if m.llmEditingProfile != nil {
            // Update existing
            err = m.client.UpdateLLMProfile(m.llmIntegration.Name, m.llmEditingProfile.Name, UpdateProfileRequest{
                Name:     values["name"], // may be renamed
                Provider: providerName,
                Account:  values["account"],
                Model:    values["model"],
            })
        } else {
            // Create new
            err = m.client.CreateLLMProfile(m.llmIntegration.Name, req)
        }

        if err != nil {
            return llmProfileSavedMsg{err: err}
        }

        // Set default if requested
        if isDefault {
            _ = m.client.SetDefaultLLMProfile(m.llmIntegration.Name, values["name"])
        }

        return llmProfileSavedMsg{}
    }
}
```

## Validation

- [ ] `go build` succeeds
- [ ] `[n]` opens new profile form
- [ ] `Enter` on profile opens edit form with values populated
- [ ] Provider dropdown shows configured providers
- [ ] Account dropdown updates when provider changes
- [ ] Model dropdown updates when account changes
- [ ] `[p]`/`[n]` paginate models when model field focused
- [ ] Model description shows when model selected
- [ ] `Ctrl+S` saves profile
- [ ] New profile appears in list after save
- [ ] Edited profile reflects changes
- [ ] "Set as default" checkbox works
- [ ] `Esc` cancels without saving
