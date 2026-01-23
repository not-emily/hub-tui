# Phase 4: Dependencies Tab UI

> **Depends on:** Phase 1 (admin status), Phase 2 (tabs component), Phase 3 (dependency API methods)
> **Enables:** Phase 5 (integration config check reuses install logic)
>
> See: [Full Plan](../plan.md)

## Goal

Add Dependencies tab to integrations modal with full dependency management UI.

## Key Deliverables

- Tab bar in integrations modal ("Integrations" | "Dependencies")
- Dependencies table view (Integration | Tool | Status | Current | Required | Actions)
- Install button for missing dependencies (admin only)
- Update button for outdated dependencies (admin only)
- "Contact admin" message for non-admin users
- Refresh functionality ([r] key)
- Success/error feedback after install/update
- Loading states during operations

## Files to Modify

- `internal/ui/modal/integrations.go` — Add tabs, dependencies view, install/update flows

## Dependencies

**Internal:**
- Phase 1: `app.isAdmin` field
- Phase 2: `components.Tabs`
- Phase 3: `client.GetDependencies()`, `client.InstallDependency()`

**External:** None

## Implementation Notes

### IntegrationsModal Changes

Add fields:
```go
type IntegrationsModal struct {
    // ... existing fields ...

    // Tab state
    tabs *components.Tabs

    // Dependencies state
    dependencies    []client.Dependency
    depLoading      bool
    depError        string
    depInstalling   string // Name of dependency being installed (empty if none)
}
```

### Constructor Update

```go
func NewIntegrationsModal(client *client.Client, isAdmin bool) *IntegrationsModal {
    return &IntegrationsModal{
        // ... existing initialization ...
        tabs: components.NewTabs([]string{"Integrations", "Dependencies"}),
        // ... rest of initialization ...
    }
}
```

### Update Logic - Tab Switching

When in list view, handle Tab/Shift+Tab:
```go
case tea.KeyMsg:
    if m.view == viewList {
        switch msg.String() {
        case "tab":
            m.tabs.Next()
            // If switching to Dependencies tab and not loaded, fetch
            if m.tabs.ActiveIndex() == 1 && m.dependencies == nil {
                return m, m.fetchDependencies()
            }
            return m, nil
        case "shift+tab":
            m.tabs.Previous()
            return m, nil
        case "r":
            // Refresh dependencies if on Dependencies tab
            if m.tabs.ActiveIndex() == 1 {
                return m, m.fetchDependencies()
            }
        }
    }
```

### Fetch Dependencies Command

```go
func (m *IntegrationsModal) fetchDependencies() tea.Cmd {
    return func() tea.Msg {
        deps, err := m.client.GetDependencies()
        return DependenciesLoadedMsg{Dependencies: deps, Err: err}
    }
}
```

### Handle Dependencies Loaded

```go
case DependenciesLoadedMsg:
    m.depLoading = false
    if msg.Err != nil {
        m.depError = msg.Err.Error()
        return m, nil
    }
    m.dependencies = msg.Dependencies
    m.depError = ""
    return m, nil
```

### Install Dependency Command

```go
func (m *IntegrationsModal) installDependency(name string) tea.Cmd {
    return func() tea.Msg {
        // Install with "latest" version (could be configurable later)
        result, err := m.client.InstallDependency(name, "latest")
        if err != nil {
            return DependencyInstalledMsg{Name: name, Err: err}
        }
        return DependencyInstalledMsg{Name: name, Status: &result.Status, Err: nil}
    }
}
```

### Handle Dependency Installed

```go
case DependencyInstalledMsg:
    m.depInstalling = ""
    if msg.Err != nil {
        m.depError = fmt.Sprintf("Failed to install %s: %s", msg.Name, msg.Err.Error())
        return m, nil
    }
    // Update dependency in list
    for i, dep := range m.dependencies {
        if dep.Name == msg.Name {
            m.dependencies[i] = *msg.Status
            break
        }
    }
    m.depError = ""
    return m, nil
```

### View Rendering - Tab Bar

```go
func (m *IntegrationsModal) View() string {
    var b strings.Builder

    if m.view == viewList {
        // Render tabs at top
        b.WriteString(m.tabs.View())
        b.WriteString("\n\n")

        // Render content based on active tab
        switch m.tabs.ActiveIndex() {
        case 0:
            b.WriteString(m.renderIntegrationsList())
        case 1:
            b.WriteString(m.renderDependenciesView())
        }
    } else {
        // Config views don't show tabs
        switch m.view {
        case viewConfigLLM:
            b.WriteString(m.renderLLMConfig())
        case viewConfigAPIKey:
            b.WriteString(m.renderAPIKeyConfig())
        }
    }

    return b.String()
}
```

### Dependencies View

```go
func (m *IntegrationsModal) renderDependenciesView() string {
    var b strings.Builder

    titleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.TextPrimary)
    b.WriteString(titleStyle.Render("Dependencies"))
    b.WriteString("\n\n")

    if m.depLoading {
        b.WriteString("Loading dependencies...\n")
        return b.String()
    }

    if m.depError != "" {
        errorStyle := lipgloss.NewStyle().Foreground(theme.Error)
        b.WriteString(errorStyle.Render("Error: " + m.depError))
        b.WriteString("\n")
        return b.String()
    }

    if len(m.dependencies) == 0 {
        b.WriteString("No dependencies configured.\n")
        return b.String()
    }

    // Table header
    headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.TextSecondary)
    b.WriteString(headerStyle.Render(
        fmt.Sprintf("%-15s %-10s %-15s %-10s %-10s %s",
            "Integration", "Tool", "Status", "Current", "Required", "Actions")))
    b.WriteString("\n")
    b.WriteString(strings.Repeat("─", m.width))
    b.WriteString("\n")

    // Table rows
    for _, dep := range m.dependencies {
        status := m.dependencyStatusString(dep)
        actions := m.dependencyActionsString(dep)

        row := fmt.Sprintf("%-15s %-10s %-15s %-10s %-10s %s",
            dep.Integration,
            dep.Name,
            status,
            dep.CurrentVersion,
            dep.MinVersion,
            actions)
        b.WriteString(row)
        b.WriteString("\n")
    }

    b.WriteString("\n")

    // Show installing status if in progress
    if m.depInstalling != "" {
        loadingStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary).Italic(true)
        b.WriteString(loadingStyle.Render(fmt.Sprintf("Installing %s...", m.depInstalling)))
        b.WriteString("\n")
    }

    // Hints
    hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
    if m.isAdmin {
        b.WriteString(hintStyle.Render("[r] Refresh  [Enter] Install/Update"))
    } else {
        b.WriteString(hintStyle.Render("[r] Refresh"))
    }

    return b.String()
}

func (m *IntegrationsModal) dependencyStatusString(dep client.Dependency) string {
    if !dep.Installed {
        return "✗ Not installed"
    }
    if !dep.UpToDate {
        return "⚠️ Outdated"
    }
    return "✓ Up to date"
}

func (m *IntegrationsModal) dependencyActionsString(dep client.Dependency) string {
    if !m.isAdmin {
        if !dep.Installed {
            return "Contact admin to install"
        }
        return ""
    }

    if !dep.Installed {
        return "[Enter] Install"
    }
    if !dep.UpToDate {
        return "[Enter] Update"
    }
    return ""
}
```

### Dependencies List Navigation

When on Dependencies tab, allow selecting dependencies with up/down and Enter to install:
```go
// Add to IntegrationsModal
selectedDepIndex int

// In Update()
case tea.KeyMsg:
    if m.view == viewList && m.tabs.ActiveIndex() == 1 {
        switch msg.String() {
        case "up", "k":
            if m.selectedDepIndex > 0 {
                m.selectedDepIndex--
            }
            return m, nil
        case "down", "j":
            if m.selectedDepIndex < len(m.dependencies)-1 {
                m.selectedDepIndex++
            }
            return m, nil
        case "enter":
            if m.isAdmin && len(m.dependencies) > 0 {
                dep := m.dependencies[m.selectedDepIndex]
                if !dep.Installed || !dep.UpToDate {
                    m.depInstalling = dep.Name
                    return m, m.installDependency(dep.Name)
                }
            }
            return m, nil
        }
    }
```

Update rendering to highlight selected dependency.

### Non-Admin Experience

When `isAdmin` is false:
- Dependencies tab is still visible (read-only view)
- No install/update buttons shown
- Actions column shows "Contact admin to install" for missing deps
- Enter key does nothing on dependencies list

### Error Scenarios

Handle these error cases with clear messages:
- **Network failure**: "Failed to fetch dependencies: {error}"
- **403 Forbidden**: "Admin permissions required"
- **Installation failure**: "Failed to install {name}: {error}"
- **Unsupported platform**: Display error message from API

## Validation

How do we know this phase is complete?

- [ ] Integrations modal shows tab bar with "Integrations" and "Dependencies" tabs
- [ ] Tab/Shift+Tab switches between tabs
- [ ] Dependencies tab loads and displays dependencies table
- [ ] Table shows: Integration, Tool, Status (with icons), Current/Required versions, Actions
- [ ] Admin sees Install button for missing dependencies
- [ ] Admin sees Update button for outdated dependencies
- [ ] Non-admin sees "Contact admin" message instead of action buttons
- [ ] Clicking Install triggers installation (shows loading state)
- [ ] After successful install, dependency status updates to "✓ Up to date"
- [ ] After failed install, error message displayed
- [ ] [r] key refreshes dependencies list
- [ ] Error handling works for all failure scenarios (network, 403, platform)
- [ ] Manual test: Install sage as admin, verify it works end-to-end
- [ ] Manual test: View as non-admin, verify read-only behavior
