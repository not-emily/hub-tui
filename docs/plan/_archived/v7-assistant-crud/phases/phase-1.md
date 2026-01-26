# Phase 1: Client Layer

> **Depends on:** None
> **Enables:** Phase 2 (List & Detail Views)
>
> See: [Full Plan](../plan.md)

## Goal

Add all assistant-related HTTP client methods needed for CRUD operations, memory management, and template creation.

## Key Deliverables

- Expanded Assistant type with all fields (modules, gather, etc.)
- CRUD methods: Get, Create, Update, Delete
- Memory methods: GetMemory, UpdateMemory
- History method: ClearHistory
- Template methods: ListTemplates, CreateFromTemplate
- Update Module type to include Tools field

## Files to Modify

- `internal/client/assistants.go` — Add new methods and expand types
- `internal/client/modules.go` — Add Tools field to Module struct

## Dependencies

**Internal:** None

**External:** None (using existing net/http patterns)

## Implementation Notes

### Assistant Types

```go
// Full assistant returned from API
type Assistant struct {
    Name        string              `json:"name"`
    DisplayName string              `json:"display_name"`
    Persona     string              `json:"persona"`
    Modules     []string            `json:"modules,omitempty"`
    Gather      map[string][]string `json:"gather,omitempty"`
    LLMProfile  string              `json:"llm_profile"`
    Keywords    []string            `json:"keywords,omitempty"`
    Enabled     bool                `json:"enabled"`
}

// Request types
type CreateAssistantRequest struct {
    Name        string              `json:"name"`
    DisplayName string              `json:"display_name"`
    Persona     string              `json:"persona"`
    Modules     []string            `json:"modules,omitempty"`
    Gather      map[string][]string `json:"gather,omitempty"`
    LLMProfile  string              `json:"llm_profile"`
    Keywords    []string            `json:"keywords,omitempty"`
}

type UpdateAssistantRequest struct {
    DisplayName string              `json:"display_name,omitempty"`
    Persona     string              `json:"persona,omitempty"`
    Modules     []string            `json:"modules,omitempty"`
    Gather      map[string][]string `json:"gather,omitempty"`
    LLMProfile  string              `json:"llm_profile,omitempty"`
    Keywords    []string            `json:"keywords,omitempty"`
}

// Memory types
type AssistantMemory struct {
    Entries map[string]string `json:"entries"`
}

// Template types
type AssistantTemplate struct {
    Name        string   `json:"name"`
    DisplayName string   `json:"display_name"`
    Persona     string   `json:"persona"`
    Modules     []string `json:"modules,omitempty"`
    LLMProfile  string   `json:"llm_profile"`
    Keywords    []string `json:"keywords,omitempty"`
}

type TemplateOverrides struct {
    Name        string `json:"name,omitempty"`
    DisplayName string `json:"display_name,omitempty"`
}
```

### API Endpoints

| Method | Endpoint | Client Method |
|--------|----------|---------------|
| GET | `/assistants/{name}` | `GetAssistant(name)` |
| POST | `/assistants` | `CreateAssistant(req)` |
| PUT | `/assistants/{name}` | `UpdateAssistant(name, req)` |
| DELETE | `/assistants/{name}` | `DeleteAssistant(name)` |
| GET | `/assistants/{name}/memory` | `GetAssistantMemory(name)` |
| PUT | `/assistants/{name}/memory` | `UpdateAssistantMemory(name, mem)` |
| DELETE | `/assistants/{name}/history` | `ClearAssistantHistory(name)` |
| GET | `/modules/{name}/assistants` | `ListModuleAssistantTemplates(module)` |
| POST | `/modules/{name}/assistants/{template}/create` | `CreateAssistantFromTemplate(module, template, overrides)` |

### Module Type Update

Add `Tools []string` to the existing Module struct:

```go
type Module struct {
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Enabled     bool     `json:"enabled"`
    Version     string   `json:"version"`
    Tools       []string `json:"tools"`  // NEW
}
```

## Validation

How do we know this phase is complete?

- [ ] Can call `GetAssistant("name")` and get full assistant details
- [ ] Can call `CreateAssistant(req)` and create a new assistant
- [ ] Can call `UpdateAssistant(name, req)` and update an assistant
- [ ] Can call `DeleteAssistant(name)` and delete an assistant
- [ ] Can call `GetAssistantMemory(name)` and get memory entries
- [ ] Can call `UpdateAssistantMemory(name, mem)` and update memory
- [ ] Can call `ClearAssistantHistory(name)` and clear history
- [ ] Can call `ListModuleAssistantTemplates(module)` and get templates
- [ ] Can call `CreateAssistantFromTemplate(module, template, nil)` and create from template
- [ ] `ListModules()` returns Tools field populated
- [ ] All methods follow existing error handling patterns (parseError)
