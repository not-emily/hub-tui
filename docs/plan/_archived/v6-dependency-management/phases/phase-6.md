# Phase 6: Hub Self-Update

> **Depends on:** Phase 3 (hub update API methods)
> **Enables:** Admin can update hub-core from TUI
>
> See: [Full Plan](../plan.md)

## Goal

Add hub-core version check and self-update functionality to the Dependencies tab.

## Key Deliverables

- Hub version section in Dependencies tab
- Check for updates button
- Update available indicator with release link
- "Update Now" button with confirmation dialog
- Graceful 404 handling (repo is private)
- Reconnection flow after hub restart

## Files to Modify

- `internal/ui/modal/integrations.go` — Add hub update UI to Dependencies tab

## Dependencies

**Internal:**
- Phase 3: `client.GetHubUpdates()`, `client.ApplyHubUpdate()`

**External:** None

## Implementation Notes

### IntegrationsModal State

Add fields:
```go
type IntegrationsModal struct {
    // ... existing fields ...

    // Hub update state
    hubUpdateInfo      *client.HubUpdateInfo
    hubUpdateLoading   bool
    hubUpdateError     string
    hubUpdateConfirm   bool // Confirmation dialog state
}
```

### Check Hub Updates Command

```go
func (m *IntegrationsModal) checkHubUpdates() tea.Cmd {
    return func() tea.Msg {
        info, err := m.client.GetHubUpdates()
        return HubUpdatesLoadedMsg{UpdateInfo: info, Err: err}
    }
}
```

### Handle Hub Updates Loaded

```go
case HubUpdatesLoadedMsg:
    m.hubUpdateLoading = false
    if msg.Err != nil {
        // Check if 404 (repo is private)
        if apiErr, ok := msg.Err.(*client.APIError); ok && apiErr.StatusCode == 404 {
            m.hubUpdateError = "Automatic updates not available (repository is private)"
        } else {
            m.hubUpdateError = msg.Err.Error()
        }
        return m, nil
    }
    m.hubUpdateInfo = msg.UpdateInfo
    m.hubUpdateError = ""
    return m, nil
```

### Apply Hub Update Command

```go
func (m *IntegrationsModal) applyHubUpdate() tea.Cmd {
    return func() tea.Msg {
        result, err := m.client.ApplyHubUpdate()
        if err != nil {
            return HubUpdateAppliedMsg{Success: false, Err: err}
        }
        return HubUpdateAppliedMsg{Success: result.Success, Message: result.Message, Err: nil}
    }
}
```

### Handle Hub Update Applied

```go
case HubUpdateAppliedMsg:
    if msg.Err != nil {
        m.hubUpdateError = fmt.Sprintf("Update failed: %s", msg.Err.Error())
        return m, nil
    }

    // Update initiated, server will restart
    // Show message and wait for reconnection
    m.hubUpdateError = ""
    m.view = viewHubUpdating
    return m, nil
```

### Hub Version Section in Dependencies View

Add to `renderDependenciesView()` after dependencies table:

```go
// Hub Version section
b.WriteString("\n")
b.WriteString(strings.Repeat("─", m.width))
b.WriteString("\n\n")

sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.TextPrimary)
b.WriteString(sectionStyle.Render("Hub Version"))
b.WriteString("\n\n")

if m.hubUpdateLoading {
    b.WriteString("Checking for updates...\n")
} else if m.hubUpdateError != "" {
    errorStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary).Italic(true)
    b.WriteString(errorStyle.Render(m.hubUpdateError))
    b.WriteString("\n")
} else if m.hubUpdateInfo != nil {
    currentStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
    b.WriteString(currentStyle.Render(fmt.Sprintf("Current: %s", m.hubUpdateInfo.CurrentVersion)))
    b.WriteString("\n")

    if m.hubUpdateInfo.UpdateAvailable {
        updateStyle := lipgloss.NewStyle().Foreground(theme.Success)
        b.WriteString(updateStyle.Render(fmt.Sprintf("Latest: %s (update available)", m.hubUpdateInfo.LatestVersion)))
        b.WriteString("\n\n")

        linkStyle := lipgloss.NewStyle().Foreground(theme.Link).Underline(true)
        b.WriteString("Release: ")
        b.WriteString(linkStyle.Render(m.hubUpdateInfo.ReleaseURL))
        b.WriteString("\n\n")

        if m.isAdmin {
            if m.hubUpdateConfirm {
                warningStyle := lipgloss.NewStyle().Foreground(theme.Warning)
                b.WriteString(warningStyle.Render("⚠️  Server will restart. Press Enter again to confirm."))
            } else {
                hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
                b.WriteString(hintStyle.Render("[u] Update Now"))
            }
        }
    } else {
        upToDateStyle := lipgloss.NewStyle().Foreground(theme.Success)
        b.WriteString(upToDateStyle.Render("✓ Up to date"))
        b.WriteString("\n")
    }
} else {
    // Not loaded yet
    hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
    if m.isAdmin {
        b.WriteString(hintStyle.Render("[c] Check for updates"))
    } else {
        b.WriteString("Version check available to administrators only")
    }
    b.WriteString("\n")
}
```

### Key Handling for Hub Updates

```go
case tea.KeyMsg:
    if m.view == viewList && m.tabs.ActiveIndex() == 1 {
        switch msg.String() {
        case "c":
            // Check for hub updates
            if m.isAdmin {
                m.hubUpdateLoading = true
                return m, m.checkHubUpdates()
            }
            return m, nil
        case "u":
            // Apply hub update (with confirmation)
            if m.isAdmin && m.hubUpdateInfo != nil && m.hubUpdateInfo.UpdateAvailable {
                if m.hubUpdateConfirm {
                    // Confirmed, apply update
                    m.hubUpdateConfirm = false
                    return m, m.applyHubUpdate()
                } else {
                    // First press, show confirmation
                    m.hubUpdateConfirm = true
                    // Clear confirmation after 3 seconds
                    return m, func() tea.Msg {
                        time.Sleep(3 * time.Second)
                        return HubUpdateConfirmExpiredMsg{}
                    }
                }
            }
            return m, nil
        }
    }

case HubUpdateConfirmExpiredMsg:
    m.hubUpdateConfirm = false
    return m, nil
```

### Hub Updating View

Show a special view while hub is updating:

```go
func (m *IntegrationsModal) renderHubUpdatingView() string {
    var b strings.Builder

    titleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.TextPrimary)
    b.WriteString(titleStyle.Render("Hub Update in Progress"))
    b.WriteString("\n\n")

    textStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
    b.WriteString(textStyle.Render("Hub-core is updating and will restart momentarily."))
    b.WriteString("\n")
    b.WriteString(textStyle.Render("You will be disconnected and automatically reconnect when the server is ready."))
    b.WriteString("\n\n")

    loadingStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary).Italic(true)
    b.WriteString(loadingStyle.Render("Waiting for server to restart..."))

    return b.String()
}
```

After showing this view, the TUI's health check loop should detect the disconnection and attempt to reconnect. Once reconnected, return to normal state.

### Auto-load Hub Version

When Dependencies tab is first opened, automatically check hub version (in addition to dependencies):

```go
// When switching to Dependencies tab
if m.tabs.ActiveIndex() == 1 {
    var cmds []tea.Cmd

    if m.dependencies == nil {
        cmds = append(cmds, m.fetchDependencies())
    }

    if m.hubUpdateInfo == nil && m.isAdmin {
        m.hubUpdateLoading = true
        cmds = append(cmds, m.checkHubUpdates())
    }

    return m, tea.Batch(cmds...)
}
```

### Error Scenarios

Handle these cases:
- **404 (repo is private)**: Show "Automatic updates not available (repository is private)"
- **Network failure**: Show "Failed to check for updates: {error}"
- **403 Forbidden**: Show "Admin permissions required"
- **Update fails**: Show error message, don't enter updating view
- **Server restart fails to reconnect**: TUI's health check should show disconnected state, retry connection

## Validation

How do we know this phase is complete?

- [ ] Dependencies tab shows "Hub Version" section below dependencies table
- [ ] Shows current version on load (admin only)
- [ ] [c] key checks for updates
- [ ] If update available, shows latest version and release link
- [ ] [u] key shows confirmation message ("Server will restart. Press Enter again to confirm.")
- [ ] Second press of [u] applies update
- [ ] After applying update, shows "Updating" view
- [ ] TUI reconnects after server restarts (existing health check handles this)
- [ ] 404 error handled gracefully with "repository is private" message
- [ ] Non-admin sees "Version check available to administrators only"
- [ ] Manual test (if repo is public): Check for updates, apply update, verify reconnection
- [ ] Manual test (if repo is private): Check for updates, verify 404 handled gracefully
- [ ] Confirmation expires after 3 seconds if not pressed again
