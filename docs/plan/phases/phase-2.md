# Phase 2: List View Enhancement

> **Depends on:** Phase 1 (Client Layer)
> **Enables:** Phase 3.1 (Builder State & Display)
>
> See: [Full Plan](../plan.md)

## Goal

Add create, edit, and delete capabilities to the existing workflows list modal.

## Key Deliverables

- `[n]` hotkey to create new workflow (enters builder)
- `[e]` hotkey to edit selected workflow (loads and enters builder)
- `[d]` hotkey to delete with confirmation
- Delete confirmation dialog
- Message types for workflow operations

## Files to Modify

- `internal/ui/modal/workflows.go` — Add hotkeys and state for CRUD operations
- `internal/app/messages.go` — Add message types (if not already in modal file)

## Implementation Notes

### Add View State

The modal needs to track whether we're in list view or builder view:

```go
type workflowsView int

const (
    viewList workflowsView = iota
    viewBuilder
    viewDeleteConfirm
)

type WorkflowsModal struct {
    client    *client.Client
    workflows []client.Workflow
    selected  int
    loading   bool
    error     string

    // New fields
    view          workflowsView
    builder       *WorkflowBuilder  // nil when in list view
    deleteConfirm bool              // showing delete confirmation
}
```

### Hotkey Handling

In the `Update` method, add handlers for new keys (when in list view):

```go
case "n":
    // Create new workflow - enter builder
    m.builder = NewWorkflowBuilder(true)  // isNew = true
    m.view = viewBuilder
    return m, m.builder.Init()

case "e":
    // Edit selected workflow
    if len(m.workflows) > 0 {
        m.loading = true
        return m, m.loadWorkflowForEdit(m.workflows[m.selected].Name)
    }

case "d":
    // Delete selected workflow
    if len(m.workflows) > 0 {
        m.view = viewDeleteConfirm
    }
```

### Load Workflow for Edit

```go
func (m *WorkflowsModal) loadWorkflowForEdit(name string) tea.Cmd {
    return func() tea.Msg {
        wf, err := m.client.GetWorkflow(name)
        return WorkflowLoadedMsg{Workflow: wf, Error: err}
    }
}

// Handle the loaded message
case WorkflowLoadedMsg:
    m.loading = false
    if msg.Error != nil {
        m.error = msg.Error.Error()
    } else {
        m.builder = NewWorkflowBuilder(false)  // isNew = false
        m.builder.LoadWorkflow(msg.Workflow)
        m.view = viewBuilder
        return m, m.builder.Init()
    }
```

### Delete Confirmation

Simple confirmation dialog:

```go
func (m *WorkflowsModal) renderDeleteConfirm() string {
    wf := m.workflows[m.selected]
    return fmt.Sprintf(
        "Delete workflow '%s'?\n\n[y] Yes, delete  [n] No, cancel",
        wf.Name,
    )
}

// In Update, when view == viewDeleteConfirm:
case "y":
    m.loading = true
    return m, m.deleteWorkflow(m.workflows[m.selected].Name)
case "n", "esc":
    m.view = viewList
```

### Message Types

```go
type WorkflowLoadedMsg struct {
    Workflow *client.Workflow
    Error    error
}

type WorkflowDeletedMsg struct {
    Name  string
    Error error
}

type WorkflowSavedMsg struct {
    Name  string
    IsNew bool
    Error error
}
```

### View Routing

Update the `View()` method to route based on current view:

```go
func (m *WorkflowsModal) View() string {
    switch m.view {
    case viewBuilder:
        return m.builder.View()
    case viewDeleteConfirm:
        return m.renderDeleteConfirm()
    default:
        return m.renderList()  // existing list rendering
    }
}
```

### Update Hints

Add new hotkeys to the hint line:

```go
// Change from:
lines = append(lines, legendStyle.Render("  Use #workflow to run  [r] Refresh"))

// To:
lines = append(lines, legendStyle.Render("  [n]ew  [e]dit  [d]elete  [r]efresh"))
```

## Validation

- [ ] `[n]` opens builder in create mode (empty workflow)
- [ ] `[e]` loads selected workflow and opens builder in edit mode
- [ ] `[d]` shows confirmation dialog
- [ ] `[y]` in confirmation deletes workflow and refreshes list
- [ ] `[n]` or `[Esc]` in confirmation returns to list
- [ ] After delete, selection adjusts appropriately (doesn't go out of bounds)
- [ ] Hotkeys are disabled when list is empty (except [n])
- [ ] Error states display appropriately (load error, delete error)
