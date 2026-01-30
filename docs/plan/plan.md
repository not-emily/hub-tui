# v9-workflow-builder: Visual Workflow Builder for TUI

> **Status:** Planning complete | Last updated: 2026-01-29
>
> Phase files: [phases/](phases/)

## Overview

The workflow builder enables hub-tui users to create and edit workflows through a visual interface without writing JSON. Users can select tools from available modules/integrations, configure parameters with dynamic forms, test steps to see real output, and build data flow between steps using a field picker.

This feature builds on the new workflow builder API in hub-core, which provides endpoints for tool discovery, step testing, schedule preview, transform generation, and validation.

## Core Vision

- **Progressive disclosure**: Start simple, reveal complexity as needed. Users see basic options first, advanced features when required.
- **Learn by doing**: Test steps and see real outputs to understand data flow. Immediate feedback builds understanding.
- **Transparent transforms**: Users see and understand what transformations do. No hidden magic - show the generated jq for transparency.
- **Keyboard-first**: Fast navigation without mouse, consistent with rest of TUI. Standard j/k navigation, modal hotkeys.

## Requirements

### Must Have
- Workflow list view with create/edit/delete
- Workflow metadata editing (name, trigger, output)
- Trigger configuration (manual or scheduled with friendly form)
- Step list with add/edit/delete/reorder
- Tool picker (hierarchical browser by type/source)
- Dynamic parameter forms from tool schemas
- Profile selection for integration tools
- Step testing with output display
- Field picker with smart extraction (not raw JSON)
- Transform steps via preset forms (Filter, Extract, Sort, First, Last, Count)
- Workflow validation before save
- Available variables display and tracking

### Nice to Have
- Duplicate workflow
- Import/export workflow JSON
- Undo/redo within builder
- Variable reference autocomplete in text fields
- Step output caching (don't re-test if params unchanged)

### Out of Scope
- **Workflow execution from TUI** — Already exists via `#workflow` trigger in chat
- **Workflow run history/logs** — Different feature, separate modal
- **Visual flow diagram** — Too complex for TUI, step list is sufficient
- **Custom jq editing** — Transform presets only; advanced users edit JSON files
- **Webhook triggers** — API doesn't support yet
- **Conditional branching** — Not supported by workflow engine currently

## Constraints

- **Tech stack**: Go, Bubble Tea, Lip Gloss (existing TUI stack)
- **Terminal size**: Must work on 80x24 minimum
- **API dependency**: Requires hub-core with workflow builder endpoints
- **Patterns**: Follow existing modal patterns (assistants, integrations)

## Success Metrics

- User can create a multi-step workflow entirely through TUI
- User can test steps and use outputs to build data flow
- User can add transform steps without knowing jq
- Validation catches errors before save with clear messages
- Editing existing workflows works seamlessly

## Architecture Decisions

### 1. Option B Layout: Step List with Detail View
**Choice:** Main view shows step list, Enter opens step detail for editing
**Rationale:** Matches existing modal patterns (assistants), provides context while editing, works on standard terminals
**Trade-offs:** More navigation than inline editing, but clearer mental model

### 2. Single Builder State Struct
**Choice:** All workflow state in one `WorkflowBuilder` struct with view enum for navigation
**Rationale:** Easy to serialize for validation/save, matches assistants modal pattern, clear state ownership
**Trade-offs:** Struct grows with features, but keeps related state together

### 3. Flat View Enum (Not Nested Modals)
**Choice:** `BuilderView` enum with explicit transitions between views
**Rationale:** Simpler than modal stack, each view knows where "back" goes, easier to reason about
**Trade-offs:** Less flexible than true modal stack, but sufficient for this use case

### 4. Smart Field Extraction (Not Raw JSON)
**Choice:** Field picker shows extracted values with friendly names, generates paths automatically
**Rationale:** Users care about values ("Task 1"), not structure (`properties.Name.title[0].plain_text`)
**Trade-offs:** Heuristics may miss edge cases; Tab toggles raw tree fallback

### 5. Transforms as User-Facing Steps
**Choice:** Expose transform operations (Filter, Extract, etc.) as explicit steps users add
**Rationale:** Transparent - users understand data flow; matches Shortcuts/Zapier mental model
**Trade-offs:** More steps in workflow, but clearer than hidden transformations

### 6. Reusable Picker Components
**Choice:** TreePicker and FieldPicker as components in `ui/components/`
**Rationale:** Clean separation, testable in isolation, potential reuse in other features
**Trade-offs:** Slight overhead of component abstraction

### 7. Array Field Selection Asks User
**Choice:** When selecting from array data, ask "Apply to: [First item] [All items]"
**Rationale:** Explicit is better than guessing; user understands the difference
**Trade-offs:** Extra interaction, but prevents confusion

## Project Structure

```
internal/
├── client/
│   └── workflows.go              # EXTEND - full Workflow struct, CRUD, builder API
│
├── ui/
│   ├── modal/
│   │   ├── workflows.go          # EXTEND - [n]ew/[e]dit/[d]elete, view routing
│   │   ├── workflows_builder.go  # NEW - builder state, step list, metadata
│   │   ├── workflows_step.go     # NEW - step detail form
│   │   ├── workflows_schedule.go # NEW - trigger/schedule form
│   │   └── workflows_transform.go# NEW - transform preset forms
│   │
│   └── components/
│       ├── treepicker.go         # NEW - hierarchical navigation component
│       └── fieldpicker.go        # NEW - extracted field picker component
```

### Key Files
- `internal/client/workflows.go` — Workflow struct, CRUD, and builder API methods
- `internal/ui/modal/workflows.go` — Modal shell, list view, view routing to builder
- `internal/ui/modal/workflows_builder.go` — WorkflowBuilder state, step list view
- `internal/ui/components/treepicker.go` — Reusable hierarchical picker
- `internal/ui/components/fieldpicker.go` — Reusable field extraction picker

## Core Interfaces

### WorkflowBuilder State

```go
type BuilderView int

const (
    ViewList BuilderView = iota
    ViewStepDetail
    ViewToolPicker
    ViewFieldPicker
    ViewTransformPicker
    ViewTransformForm
    ViewTriggerForm
    ViewValidation
)

type WorkflowBuilder struct {
    // Identity
    IsNew        bool
    OriginalName string

    // Workflow data
    Name        string
    Description string
    Trigger     TriggerConfig
    Steps       []WorkflowStep
    Output      string

    // Editing state
    View          BuilderView
    SelectedStep  int
    EditingStep   *WorkflowStep
    StepOutput    interface{}

    // Sub-components
    ToolPicker    *components.TreePicker
    FieldPicker   *components.FieldPicker
    TransformForm *TransformForm

    // Cached
    Tools    *client.ToolsResponse
    Profiles map[string][]string

    // State
    Dirty   bool
    Error   string
    Loading bool
}
```

### Component Contracts

```go
// TreePicker - hierarchical navigation
func NewTreePicker(nodes []TreeNode) *TreePicker
func (t *TreePicker) Update(msg tea.Msg) (selected *TreeNode, cmd tea.Cmd)
func (t *TreePicker) View() string

// FieldPicker - field extraction from JSON
func NewFieldPicker(stepName string, output interface{}) *FieldPicker
func (f *FieldPicker) Update(msg tea.Msg) (selectedPath string, cmd tea.Cmd)
func (f *FieldPicker) View() string
func (f *FieldPicker) ToggleRawMode()
```

### Client Methods

```go
// CRUD
func (c *Client) GetWorkflow(name string) (*Workflow, error)
func (c *Client) CreateWorkflow(wf *Workflow) error
func (c *Client) UpdateWorkflow(name string, wf *Workflow) error
func (c *Client) DeleteWorkflow(name string) error

// Builder API
func (c *Client) GetBuilderTools() (*ToolsResponse, error)
func (c *Client) TestStep(req *StepTestRequest) (*StepTestResult, error)
func (c *Client) PreviewSchedule(req *ScheduleRequest) (*SchedulePreview, error)
func (c *Client) PreviewTransform(req *TransformRequest) (*TransformPreview, error)
func (c *Client) ValidateWorkflow(wf *Workflow) (*ValidationResult, error)
```

## Implementation Phases

| Phase | Name | Scope | Depends On | Key Outputs |
|-------|------|-------|------------|-------------|
| 1 | Client Layer | API methods for builder + CRUD | — | Extended `workflows.go` |
| 2 | List View Enhancement | [n]ew/[e]dit/[d]elete hotkeys | Phase 1 | Working CRUD from list |
| 3.1 | Builder State & Display | Builder struct, step list view | Phase 2 | Read-only step display |
| 3.2 | Builder Editing | Metadata form, step CRUD, reorder | Phase 3.1 | Full step management |
| 4 | Trigger Form | Type radio, schedule config, preview | Phase 3.2 | Trigger configuration |
| 5 | Tool Picker | TreePicker component, tool browser | Phase 3.2 | Tool selection |
| 6 | Step Detail Form | Dynamic params, profile dropdown | Phase 5 | Step configuration |
| 7.1 | Step Testing | Test step, display output | Phase 6 | Step execution |
| 7.2 | Field Picker | Smart extraction, array handling | Phase 7.1 | Data flow building |
| 8 | Transform Forms | All 6 transform presets | Phase 7.2 | Transform steps |
| 9 | Validation & Polish | Validate endpoint, error display | Phase 8 | Production-ready |

### Critical Path

```
Phase 1 → Phase 2 → Phase 3.1 → Phase 3.2 → Phase 4
                                    ↓
                               Phase 5 → Phase 6 → Phase 7.1 → Phase 7.2 → Phase 8 → Phase 9
```

### Phase Details
- [Phase 1: Client Layer](phases/phase-1.md)
- [Phase 2: List View Enhancement](phases/phase-2.md)
- [Phase 3.1: Builder State & Display](phases/phase-3-1.md)
- [Phase 3.2: Builder Editing](phases/phase-3-2.md)
- [Phase 4: Trigger Form](phases/phase-4.md)
- [Phase 5: Tool Picker](phases/phase-5.md)
- [Phase 6: Step Detail Form](phases/phase-6.md)
- [Phase 7.1: Step Testing](phases/phase-7-1.md)
- [Phase 7.2: Field Picker](phases/phase-7-2.md)
- [Phase 8: Transform Forms](phases/phase-8.md)
- [Phase 9: Validation & Polish](phases/phase-9.md)

## Tech Stack

| Category | Choice | Notes |
|----------|--------|-------|
| Language | Go | Existing codebase |
| TUI Framework | Bubble Tea | Elm architecture, existing |
| Styling | Lip Gloss | Existing theme system |
| API Client | net/http | Existing client pattern |

## Future Considerations

Items explicitly deferred but architecturally supported:

- **Webhook triggers** — TriggerConfig struct can accommodate when API supports
- **Conditional branching** — Step list could support branch markers when engine supports
- **Workflow templates** — Builder could load from template instead of blank
- **Step library** — Save/reuse configured steps across workflows
