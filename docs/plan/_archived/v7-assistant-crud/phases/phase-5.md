# Phase 5: Module Templates

> **Depends on:** Phase 3 (Create & Edit)
> **Enables:** Completes assistant management feature
>
> See: [Full Plan](../plan.md)

## Goal

Allow users to create assistants from module-provided templates, enabling quick setup of specialized assistants.

## Key Deliverables

- Template selection view (list templates from all enabled modules)
- Template preview showing what will be created
- Optional overrides (name, display_name) before creation
- One-click creation from template

## Files to Modify

- `internal/ui/modal/assistants.go` — Add template views and handlers

## Dependencies

**Internal:**
- Phase 1 client methods (ListModuleAssistantTemplates, CreateAssistantFromTemplate)
- Phase 2 modal list view (for [t] trigger)

**External:** None

## Implementation Notes

### Template Data Structure

From the API, templates look like:
```json
{
  "name": "chef",
  "display_name": "Chef Assistant",
  "persona": "You are a helpful cooking assistant...",
  "modules": ["recipes"],
  "llm_profile": "default",
  "keywords": ["recipe", "cook", "meal"]
}
```

### New View States

```go
const (
    // ... existing ...
    viewTemplateList    // NEW - Select a template
    viewTemplatePreview // NEW - Preview and optionally customize
)
```

### Template View State

```go
type AssistantsModal struct {
    // ... existing fields ...

    // Template state
    templates        []moduleTemplate  // All templates from all modules
    templateSelected int
    templateLoading  bool

    // Override fields for template creation
    templateOverrideName    string
    templateOverrideDisplay string
}

type moduleTemplate struct {
    Module   string
    Template client.AssistantTemplate
}
```

### Template List View

When user presses [t] from the list view, show templates grouped by module:

```
  Create from Template

  recipes
    ● chef              Chef Assistant
    ○ meal_planner      Meal Planning Assistant

  fitness
    ● trainer           Fitness Trainer

  [Enter] Select  [Esc] Cancel
```

If no templates available:
```
  Create from Template

  No assistant templates available.

  Templates are provided by modules. Enable modules
  with assistant templates to see them here.

  [Esc] Back
```

### Template Preview View

After selecting a template:

```
  Create from Template — chef

  From module:   recipes

  Name:          [chef__________]  (optional override)
  Display Name:  [Chef Assistant]  (optional override)

  ─────────────────────────────────────────────────────

  Persona:
  You are a helpful cooking assistant who specializes
  in recipe recommendations, meal planning, and cooking
  tips. You have access to the user's recipe collection.

  Modules:    recipes
  Profile:    default
  Keywords:   recipe, cook, meal

  [Enter] Create  [Esc] Cancel
```

### Workflow

1. User presses [t] from assistant list
2. Load templates from all enabled modules
3. Show template list grouped by module
4. User selects template
5. Show preview with optional name/display_name overrides
6. User presses Enter to create
7. API creates assistant from template
8. Return to list with new assistant selected

### Loading Templates

Templates need to be loaded from each enabled module:

```go
func (m *AssistantsModal) loadAllTemplates() tea.Cmd {
    return func() tea.Msg {
        modules, err := m.client.ListModules()
        if err != nil {
            return TemplatesLoadedMsg{Error: err}
        }

        var templates []moduleTemplate
        for _, mod := range modules {
            if !mod.Enabled {
                continue
            }

            modTemplates, err := m.client.ListModuleAssistantTemplates(mod.Name)
            if err != nil {
                continue // Skip modules that fail
            }

            for _, t := range modTemplates {
                templates = append(templates, moduleTemplate{
                    Module:   mod.Name,
                    Template: t,
                })
            }
        }

        return TemplatesLoadedMsg{Templates: templates}
    }
}
```

### Message Types

```go
type TemplatesLoadedMsg struct {
    Templates []moduleTemplate
    Error     error
}

type TemplateCreatedMsg struct {
    Assistant *client.Assistant
    Error     error
}
```

### Key Bindings

**Template list:**
| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Navigate templates |
| `Enter` | Select template (go to preview) |
| `Esc` | Back to assistant list |

**Template preview:**
| Key | Action |
|-----|--------|
| `Tab` | Move between override fields |
| `Enter` | Create assistant from template |
| `Esc` | Back to template list |

### Edge Cases

- **Template name conflict**: If an assistant with the template's default name already exists, require the user to provide an override name
- **Invalid LLM profile**: If template references a profile that doesn't exist, show warning and require user to select a valid profile
- **Module disabled after template load**: Re-validate before creation

## Validation

How do we know this phase is complete?

- [ ] [t] from list opens template selection view
- [ ] Templates are loaded from all enabled modules
- [ ] Templates are grouped by module in the list
- [ ] Can navigate and select a template
- [ ] Preview shows full template details
- [ ] Can override name and display_name before creation
- [ ] Empty override fields use template defaults
- [ ] Enter creates assistant from template
- [ ] Success returns to list with new assistant
- [ ] Handles name conflicts (prompts for override)
- [ ] Shows "no templates" message when none available
- [ ] Handles API errors gracefully
- [ ] Esc at any point cancels and goes back
