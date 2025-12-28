# Phase 6.2: Resource Modals

> **Depends on:** Phase 6.1 (Modal Framework)
> **Enables:** Phase 7 (Background Tasks)
>
> See: [Full Plan](../plan.md)

## Goal

Build interactive modals for managing hub resources: modules, integrations, and workflows.

## Key Deliverables

- Modules modal (list, enable/disable)
- Integrations modal (list, configure, test)
- Workflows modal (browse list)
- Each modal refreshes its cache on open
- Form component for configuration input

## Files to Create

- `internal/ui/modal/modules.go` — Modules management modal
- `internal/ui/modal/integrations.go` — Integrations config modal
- `internal/ui/modal/workflows.go` — Workflows browse modal
- `internal/ui/components/form.go` — Reusable form component
- `internal/client/integrations.go` — Integration endpoints

## Files to Modify

- `internal/app/app.go` — Add modal commands
- `internal/client/modules.go` — Add enable/disable endpoints

## Dependencies

**Internal:** Modal framework from Phase 6.1

**External:** None

## Implementation Notes

### Modules Modal

```
Modules                                    [Esc to close]

  ● recipes           Manage recipes and meal plans
  ○ workouts          Track workouts and fitness
  ● notes             Quick notes and snippets

  ● = enabled, ○ = disabled

  [Enter] Toggle  [r] Refresh  [Esc] Close
```

Features:
- List all modules with enabled/disabled indicator
- Enter toggles enable/disable
- API call on toggle: `POST /modules/{name}/enable` or `/disable`
- Refresh list on open

```go
type ModulesModal struct {
    modules  []Module
    selected int
    loading  bool
    error    string
}

func (m *ModulesModal) toggle() tea.Cmd {
    mod := m.modules[m.selected]
    if mod.Enabled {
        return disableModuleCmd(mod.Name)
    }
    return enableModuleCmd(mod.Name)
}
```

### Integrations Modal

Two-level modal:
1. List of integrations
2. Configure form for selected integration

**List View:**
```
Integrations                               [Esc to close]

  ✓ claude          Configured
  ✓ openai          Configured
  ✗ notion          Not configured
  ✓ ollama          Configured

  [Enter] Configure  [t] Test  [Esc] Close
```

**Configure View:**
```
Configure: notion                          [Esc to cancel]

  API Key: ****************************

  [Enter] Save  [Esc] Cancel
```

Features:
- List integrations with configured status
- Enter opens configure form
- `t` tests the integration (`POST /integrations/{name}/test`)
- Form saves via `POST /integrations/{name}/configure`

### Workflows Modal

Read-only browse:

```
Workflows                                  [Esc to close]

  morning_briefing    Daily morning summary
  weekly_report       End of week report
  backup_notes        Backup notes to cloud

  Trigger: #morning_briefing

  [Esc] Close
```

Features:
- List all workflows
- Show description
- Remind user they can trigger with `#name`
- No actions (triggering is via `#workflow` in chat)

### Form Component

Reusable for configuration:

```go
type Form struct {
    fields   []FormField
    focused  int
    values   map[string]string
}

type FormField struct {
    Label    string
    Key      string
    Value    string
    Password bool  // Mask input
}

func (f *Form) Update(msg tea.Msg) tea.Cmd {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "tab", "down":
            f.focused = (f.focused + 1) % len(f.fields)
        case "shift+tab", "up":
            f.focused = (f.focused - 1 + len(f.fields)) % len(f.fields)
        case "enter":
            return f.submit()
        default:
            // Type into focused field
        }
    }
    return nil
}
```

### Cache Refresh

Each modal refreshes its data source on open:

```go
func NewModulesModal(client *Client) *ModulesModal {
    m := &ModulesModal{loading: true}
    // Trigger async fetch
    return m
}

// In Update:
case ModulesLoadedMsg:
    m.modules = msg.Modules
    m.loading = false
```

### Error Handling

- Show errors inline in modal
- Failed API calls show error message
- Allow retry with `r` to refresh

```
Modules                                    [Esc to close]

  Error: Failed to load modules

  [r] Retry  [Esc] Close
```

## Validation

How do we know this phase is complete?

- [ ] `/modules` opens modules modal
- [ ] Modules modal shows all modules with enabled/disabled status
- [ ] Enter toggles module enable/disable
- [ ] Toggle persists (API call succeeds)
- [ ] `/integrations` opens integrations modal
- [ ] Integrations show configured status
- [ ] Enter opens configuration form
- [ ] Form input works (typing, tab between fields)
- [ ] Password fields are masked
- [ ] Save button calls configure API
- [ ] `t` tests integration and shows result
- [ ] `/workflows` opens workflows modal
- [ ] Workflows listed with descriptions
- [ ] All modals refresh data on open
- [ ] Errors display gracefully
