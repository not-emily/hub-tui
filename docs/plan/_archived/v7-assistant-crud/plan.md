# v7-assistants-management: Manage AI Assistants from the TUI

> **Status:** Planning complete | Last updated: 2026-01-23
>
> Phase files: [phases/](phases/)

## Overview

Add full assistant management capabilities to hub-tui, allowing users to view, create, edit, and delete AI assistants without leaving the terminal. Currently, users can only list and chat with assistants - any management requires direct API calls via curl.

This feature builds on the existing modal pattern (like IntegrationsModal, WorkflowsModal) to provide a keyboard-driven interface for assistant CRUD operations, including module/tool access configuration, core memory management, and creation from module templates.

## Core Vision

- **Keyboard-first**: All operations accessible via keyboard shortcuts, following existing TUI patterns
- **Progressive disclosure**: List view → Detail view → Edit/Memory sub-views to avoid overwhelming users
- **Transparency**: Make all assistant configuration visible and editable, including memory
- **Module-aware**: Assistants can be scoped to specific modules and auto-gather context from tools

## Requirements

### Must Have
- List all assistants with status indicators
- View assistant details (name, persona, profile, keywords, modules)
- Create new assistant via form
- Create assistant from module template
- Edit existing assistant (all fields except name)
- Delete assistant with confirmation
- Configure module access and gather tools per assistant
- View and edit core memory (in dedicated sub-view)
- Clear conversation history with confirmation

### Nice to Have
- View conversation history (read-only)
- Duplicate assistant (clone with new name)
- Test assistant chat from within modal

### Out of Scope
- Full chat interface (already exists in main view)
- Module template management (modules modal)
- LLM profile management (integrations modal)

## Constraints

- **Tech stack**: Go, Bubble Tea, Lip Gloss (existing stack)
- **Patterns**: Must follow existing Modal interface, tea.Cmd, message routing
- **Keyboard-only**: No mouse support
- **API**: All endpoints already exist in hub-core

## Success Metrics

- Can CRUD assistants entirely from TUI
- Can create from module templates with one-click
- Can view/edit core memory without leaving TUI
- Can configure module access and gather tools
- No regressions in existing assistant chat functionality

## Architecture Decisions

### 1. Single Modal with View States
**Choice:** One AssistantsModal with internal view state machine
**Rationale:** Follows IntegrationsModal pattern which handles complex flows well
**Trade-offs:** Single file may get large, but keeps related code together

### 2. Form Patterns
**Choice:** Reuse form patterns from integrations modal (text inputs, dropdowns, textareas)
**Rationale:** Consistent UX, proven patterns
**Trade-offs:** May need to extend for module/gather multi-select

### 3. Memory as Sub-view
**Choice:** Separate memory view accessed via [m] from detail view
**Rationale:** Keeps main UI clean while making memory fully accessible
**Trade-offs:** Extra navigation step to edit memory

### 4. Block Creation Without LLM Profile
**Choice:** Require at least one LLM profile before allowing assistant creation
**Rationale:** An assistant without a profile wouldn't be functional
**Trade-offs:** New users must configure AI integration first

## Project Structure

```
internal/
├── client/
│   └── assistants.go     # Expand with CRUD, memory, template methods
└── ui/modal/
    └── assistants.go     # New modal (list, detail, edit, memory, templates)
```

### Key Files
- `internal/client/assistants.go` — HTTP client methods for assistant API
- `internal/ui/modal/assistants.go` — Main modal implementation
- `internal/client/modules.go` — Update Module struct to include Tools field

## Core Interfaces

### View States
```
viewList (default)
  └── select → viewDetail
viewDetail
  ├── [e] → viewEdit
  ├── [m] → viewMemory
  ├── [h] → confirm → clear history
  └── [d] → confirm → delete
viewEdit (form)
  └── submit → viewDetail
viewMemory
  └── edit/save → viewDetail
viewCreate (form)
  └── submit → viewList
viewTemplateSelect
  └── select → viewTemplateCreate → viewList
```

### Key Bindings
| Context | Key | Action |
|---------|-----|--------|
| List | `j/k` or `↑/↓` | Navigate |
| List | `Enter` | View detail |
| List | `n` | New assistant |
| List | `t` | Create from template |
| List | `r` | Refresh |
| Detail | `e` | Edit |
| Detail | `m` | Memory |
| Detail | `h` | Clear history |
| Detail | `d` | Delete |
| Form | `Tab/↓` | Next field |
| Form | `Shift+Tab/↑` | Previous field |
| Form | `Enter` | Submit (on last field) |
| Any | `Esc` | Back / Cancel |

### Client Methods to Add
```go
// CRUD
GetAssistant(name string) (*Assistant, error)
CreateAssistant(req *CreateAssistantRequest) (*Assistant, error)
UpdateAssistant(name string, req *UpdateAssistantRequest) (*Assistant, error)
DeleteAssistant(name string) error

// Memory
GetAssistantMemory(name string) (*AssistantMemory, error)
UpdateAssistantMemory(name string, memory *AssistantMemory) error

// History
ClearAssistantHistory(name string) error

// Templates
ListModuleAssistantTemplates(module string) ([]AssistantTemplate, error)
CreateAssistantFromTemplate(module, template string, overrides *TemplateOverrides) (*Assistant, error)
```

### Assistant Form Fields
| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `name` | text | Yes | Slug, lowercase alphanumeric + hyphens |
| `display_name` | text | Yes | Human-readable name |
| `llm_profile` | dropdown | Yes | From configured LLM profiles |
| `persona` | textarea | Yes | Multi-line, scrollable (5-8 lines) |
| `keywords` | text | No | Comma-separated |
| `modules` | multi-select | No | Checkboxes for enabled modules |
| `gather` | nested | No | Per-module tool checkboxes |

## Implementation Phases

| Phase | Name | Scope | Depends On | Key Outputs |
|-------|------|-------|------------|-------------|
| 1 | Client Layer | CRUD + memory + template methods | — | `assistants.go` expanded |
| 2 | List & Detail Views | Modal with list and read-only detail | Phase 1 | Basic navigation |
| 3 | Create & Edit | Forms with modules/gather selection | Phase 2 | Full CRUD |
| 4 | Memory Management | View/edit core memory sub-view | Phase 3 | Memory UI |
| 5 | Templates | Create from module templates | Phase 3 | Template flow |

### Critical Path
Phases 1-3 are sequential and required for basic functionality. Phases 4-5 can be done in parallel after Phase 3.

### Phase Details
- [Phase 1: Client Layer](phases/phase-1.md)
- [Phase 2: List & Detail Views](phases/phase-2.md)
- [Phase 3: Create & Edit Forms](phases/phase-3.md)
- [Phase 4: Memory Management](phases/phase-4.md)
- [Phase 5: Module Templates](phases/phase-5.md)

## Tech Stack

| Category | Choice | Notes |
|----------|--------|-------|
| Language | Go | Existing |
| TUI Framework | Bubble Tea | Existing |
| Styling | Lip Gloss | Existing |
| HTTP Client | net/http | Existing patterns in client/ |

## Future Considerations

- **Conversation history viewer**: Could add read-only history view in a future iteration
- **Assistant duplication**: Clone existing assistant with new name
- **Import/export**: YAML/JSON import/export of assistant configs
- **Assistant testing**: In-modal chat test before saving
