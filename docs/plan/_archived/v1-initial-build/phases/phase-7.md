# Phase 7: Background Tasks

> **Depends on:** Phase 6.2 (Resource Modals)
> **Enables:** Project complete
>
> See: [Full Plan](../plan.md)

## Goal

Trigger workflows in the background and track their status with a task indicator and modal.

## Key Deliverables

- `#{workflow}` triggers workflow, returns immediately
- Track running tasks in app state
- Poll `/runs` periodically for status updates
- Status bar shows: `2 running` or `2 running · 1 failed`
- `/tasks` modal (view running, completed, failed tasks)
- Notifications when tasks complete or fail

## Files to Create

- `internal/ui/modal/tasks.go` — Tasks modal
- `internal/client/runs.go` — Runs/tasks endpoints

## Files to Modify

- `internal/app/app.go` — Task state, polling, status display
- `internal/app/messages.go` — Task-related messages
- `internal/ui/status/status.go` — Add task counts
- `internal/ui/chat/input.go` — Handle # prefix as workflow trigger

## Dependencies

**Internal:** Modal framework from Phase 6.1, client from Phase 2

**External:** None

## Implementation Notes

### Task State

```go
type TaskState struct {
    Running   []Run
    Completed []Run  // Recent completions
    Failed    []Run  // Recent failures
}

type Run struct {
    ID        string
    Workflow  string
    Status    string  // "running", "completed", "failed", "cancelled"
    StartedAt time.Time
    EndedAt   time.Time
    Error     string
}
```

### Triggering Workflows

When user types `#morning_briefing`:

```go
func (m *Model) triggerWorkflow(name string) tea.Cmd {
    return func() tea.Msg {
        runID, err := m.client.RunWorkflow(name)
        if err != nil {
            return WorkflowErrorMsg{Name: name, Error: err}
        }
        return WorkflowStartedMsg{Name: name, RunID: runID}
    }
}
```

Show confirmation in chat: "Started workflow: morning_briefing"

### Polling

Poll `/runs` every few seconds while tasks are running:

```go
func (m Model) pollTasks() tea.Cmd {
    return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
        return PollTasksMsg{}
    })
}

// In Update:
case PollTasksMsg:
    if len(m.tasks.Running) > 0 {
        return tea.Batch(
            m.fetchTaskStatus(),
            m.pollTasks(),  // Continue polling
        )
    }
```

Stop polling when no tasks are running.

### Status Bar Update

Extend status bar to show task counts:

```
Connected to hub                          2 running · 1 failed
```

Format:
- `N running` when tasks are active
- `N running · M failed` when there are failures
- Nothing when no tasks and no failures

```go
func (s *StatusBar) taskIndicator() string {
    var parts []string

    if len(s.tasks.Running) > 0 {
        parts = append(parts, fmt.Sprintf("%d running", len(s.tasks.Running)))
    }

    if len(s.tasks.Failed) > 0 {
        parts = append(parts,
            lipgloss.NewStyle().
                Foreground(theme.Error).
                Render(fmt.Sprintf("%d failed", len(s.tasks.Failed))))
    }

    return strings.Join(parts, " · ")
}
```

### Notifications

When a task completes or fails, show inline notification:

```
┌─────────────────────────────────────────┐
│                                         │
│  ... chat messages ...                  │
│                                         │
│  ✓ Workflow completed: morning_briefing │  ← Success notification
│                                         │
│  ✗ Workflow failed: backup_notes        │  ← Error notification
│                                         │
└─────────────────────────────────────────┘
```

These appear as system messages in the chat, styled differently from user/hub messages.

### Tasks Modal

```
Tasks                                      [Esc to close]

Running:
  ● morning_briefing    Started 2 min ago

Completed:
  ✓ weekly_report       Completed 10 min ago
  ✓ sync_calendar       Completed 1 hour ago

Failed:
  ✗ backup_notes        Error: Connection timeout
                        Failed 30 min ago

  [Enter] View details  [c] Cancel running  [Esc] Close
```

Features:
- Three sections: Running, Completed, Failed
- Enter expands details (show output if available)
- `c` cancels selected running task
- Shows relative timestamps

### Task Details View

When Enter pressed on a task:

```
Task: morning_briefing                     [Esc to go back]

Status:    Completed
Started:   2025-12-24 08:00:00
Ended:     2025-12-24 08:00:45
Duration:  45s

Output:
  Gathered calendar events: 3
  Gathered emails: 12
  Generated briefing successfully

  [Esc] Back
```

### Clearing Failed Tasks

Failed tasks should clear after some time or when acknowledged:
- View details clears the "new" state
- `/tasks` modal has option to clear all
- Failures older than 1 hour auto-clear from indicator (but stay in history)

## Validation

How do we know this phase is complete?

- [ ] `#workflow_name` triggers workflow
- [ ] Confirmation message appears in chat
- [ ] Status bar shows "1 running"
- [ ] Polling updates status every few seconds
- [ ] When complete, notification appears in chat
- [ ] Status bar updates (running count decreases)
- [ ] Failed workflow shows error notification
- [ ] Failed count appears in status bar
- [ ] `/tasks` opens tasks modal
- [ ] Tasks modal shows running, completed, failed sections
- [ ] Can view task details
- [ ] Can cancel running task
- [ ] `c` in modal cancels selected task
- [ ] Polling stops when no tasks running
