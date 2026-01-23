# Phase 5: Integration Config Dependency Check

> **Depends on:** Phase 1 (admin status), Phase 3 (dependency API), Phase 4 (install logic)
> **Enables:** Integration config flows are blocked until dependencies are satisfied
>
> See: [Full Plan](../plan.md)

## Goal

Check dependencies before allowing integration configuration, blocking config flow until dependencies are satisfied.

## Key Deliverables

- Dependency check when entering integration config
- Blocking modal/prompt for missing dependencies
- Install button (admin) or "Contact admin" message (non-admin)
- Automatic proceed to config after successful install
- Works for any integration, not just LLM

## Files to Modify

- `internal/ui/modal/integrations_llm.go` — Add dependency check before entering LLM config

## Dependencies

**Internal:**
- Phase 1: `app.isAdmin` field
- Phase 3: `client.GetDependencies()`
- Phase 4: Dependency install logic and messages

**External:** None

## Implementation Notes

### When to Check

Check dependencies when user selects an integration to configure. This happens in:
- `enterLLMConfig()` in `integrations_llm.go`
- Future: `enterAPIKeyConfig()` or other config entry points

### Dependency Check Logic

Before showing config UI, check if integration has unsatisfied dependencies:

```go
func (m *IntegrationsModal) enterLLMConfig(integration client.Integration) (Modal, tea.Cmd) {
    // Check dependencies first
    cmd := func() tea.Msg {
        deps, err := m.client.GetDependencies()
        if err != nil {
            return LLMErrorMsg{Err: err}
        }

        // Filter for this integration
        integrationDeps := filterDependenciesForIntegration(deps, integration.Name)

        // Check if any are unsatisfied
        if !areDependenciesSatisfied(integrationDeps) {
            return DependencyCheckFailedMsg{
                Integration: integration.Name,
                Dependencies: integrationDeps,
            }
        }

        // All satisfied, proceed to load LLM data
        return DependencyCheckPassedMsg{Integration: integration.Name}
    }

    m.view = viewCheckingDependencies // New view state
    return m, cmd
}
```

### Helper Functions

```go
// Filter dependencies for a specific integration
func filterDependenciesForIntegration(deps []client.Dependency, integrationName string) []client.Dependency {
    var filtered []client.Dependency
    for _, dep := range deps {
        if dep.Integration == integrationName {
            filtered = append(filtered, dep)
        }
    }
    return filtered
}

// Check if all dependencies are satisfied (installed and up to date)
func areDependenciesSatisfied(deps []client.Dependency) bool {
    for _, dep := range deps {
        if !dep.Installed || !dep.UpToDate {
            return false
        }
    }
    return true
}
```

### New Message Types

```go
type DependencyCheckFailedMsg struct {
    Integration  string
    Dependencies []client.Dependency
}

type DependencyCheckPassedMsg struct {
    Integration string
}
```

### Handle Dependency Check Results

```go
case DependencyCheckFailedMsg:
    // Store unsatisfied dependencies
    m.unsatisfiedDeps = msg.Dependencies
    m.view = viewDependencyBlocked
    return m, nil

case DependencyCheckPassedMsg:
    // Proceed to load integration data (existing logic)
    m.view = viewConfigLLM
    return m, m.loadLLMData()
```

### Dependency Blocked View

```go
func (m *IntegrationsModal) renderDependencyBlockedView() string {
    var b strings.Builder

    titleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Warning)
    b.WriteString(titleStyle.Render("⚠️  Dependencies Required"))
    b.WriteString("\n\n")

    textStyle := lipgloss.NewStyle().Foreground(theme.TextPrimary)
    b.WriteString(textStyle.Render(
        "The following dependencies must be installed before configuring this integration:"))
    b.WriteString("\n\n")

    // List unsatisfied dependencies
    for _, dep := range m.unsatisfiedDeps {
        status := "Not installed"
        if dep.Installed && !dep.UpToDate {
            status = fmt.Sprintf("Outdated (current: %s, required: %s)",
                dep.CurrentVersion, dep.MinVersion)
        }

        depStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
        b.WriteString(depStyle.Render(fmt.Sprintf("  • %s: %s", dep.Name, status)))
        b.WriteString("\n")
    }

    b.WriteString("\n")

    if m.isAdmin {
        // Show install button for admin
        if m.depInstalling != "" {
            loadingStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary).Italic(true)
            b.WriteString(loadingStyle.Render(fmt.Sprintf("Installing %s...", m.depInstalling)))
            b.WriteString("\n\n")
        }

        hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
        b.WriteString(hintStyle.Render("[Enter] Install missing dependencies  [Esc] Cancel"))
    } else {
        // Show contact admin message for non-admin
        messageStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary).Italic(true)
        b.WriteString(messageStyle.Render(
            "Please contact your administrator to install these dependencies."))
        b.WriteString("\n\n")

        hintStyle := lipgloss.NewStyle().Foreground(theme.TextSecondary)
        b.WriteString(hintStyle.Render("[Esc] Back"))
    }

    return b.String()
}
```

### Install from Blocked View

When admin presses Enter in blocked view, install all unsatisfied dependencies:

```go
case tea.KeyMsg:
    if m.view == viewDependencyBlocked {
        switch msg.String() {
        case "enter":
            if m.isAdmin && m.depInstalling == "" {
                // Install first unsatisfied dependency
                // (could install all in parallel, but sequential is simpler for v1)
                for _, dep := range m.unsatisfiedDeps {
                    if !dep.Installed || !dep.UpToDate {
                        m.depInstalling = dep.Name
                        return m, m.installDependency(dep.Name)
                    }
                }
            }
            return m, nil
        case "esc":
            m.view = viewList
            return m, nil
        }
    }

case DependencyInstalledMsg:
    m.depInstalling = ""
    if msg.Err != nil {
        m.depError = fmt.Sprintf("Failed to install %s: %s", msg.Name, msg.Err.Error())
        return m, nil
    }

    // Update unsatisfied deps list
    for i, dep := range m.unsatisfiedDeps {
        if dep.Name == msg.Name {
            m.unsatisfiedDeps[i] = *msg.Status
            break
        }
    }

    // Check if all deps now satisfied
    if areDependenciesSatisfied(m.unsatisfiedDeps) {
        // All satisfied! Proceed to config
        return m.enterLLMConfig(m.selectedIntegration)
    }

    // More deps to install
    return m, nil
```

### Integration with Existing Config Flow

The dependency check is transparent to existing config logic:
1. User selects integration
2. Check dependencies
3. If satisfied: Proceed to existing config flow (load providers, show form, etc.)
4. If not satisfied: Show blocking view with install button
5. After install succeeds: Automatically proceed to step 3

### Edge Cases

- **No dependencies for integration**: Skip check, proceed directly to config
- **All dependencies satisfied**: Skip blocking view, proceed directly to config
- **Multiple dependencies**: Install sequentially (could parallelize in future)
- **Installation fails**: Show error, stay in blocked view, allow retry

## Validation

How do we know this phase is complete?

- [ ] When entering integration config, dependencies are checked first
- [ ] If all dependencies satisfied, config flow proceeds normally (no change to existing behavior)
- [ ] If dependencies unsatisfied, blocking view is shown
- [ ] Blocking view lists all unsatisfied dependencies with status
- [ ] Admin sees [Enter] to install dependencies
- [ ] Non-admin sees "Contact administrator" message
- [ ] After successful install, config flow automatically proceeds
- [ ] After failed install, error message shown and user can retry
- [ ] [Esc] exits blocking view back to integrations list
- [ ] Manual test: Remove sage, try to configure AI integration, verify blocked
- [ ] Manual test: Install sage from blocking view, verify auto-proceed to config
- [ ] Manual test: Try as non-admin, verify "Contact admin" message
- [ ] Works for any integration (not hardcoded to LLM)
