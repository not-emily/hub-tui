# Phase 2: Modal Routing + api_key

> **Depends on:** Phase 1 (Client Layer Updates)
> **Enables:** Phase 3 (LLM Config View)
>
> See: [Full Plan](../plan.md)

## Goal

Add config-type routing to the integrations modal and validate that existing `api_key` flow still works correctly.

## Key Deliverables

- Config type switch in modal when user selects "configure"
- `api_key` type routes to existing profile selection → form flow
- `llm` type shows placeholder (implemented in Phase 3)
- Unknown types show error message

## Files to Modify

- `internal/ui/modal/integrations.go` — Add config type routing logic

## Dependencies

**Internal:**
- Phase 1 complete (client returns `ConfigType` field)

**External:** None

## Implementation Notes

### Config Type Routing

When user presses Enter on an integration in the list, check its `ConfigType`:

```go
func (m *IntegrationsModal) enterConfigMode(integration Integration) (Modal, tea.Cmd) {
    switch integration.ConfigType {
    case "api_key":
        return m.enterAPIKeyConfig(integration)
    case "llm":
        return m.enterLLMConfig(integration) // Placeholder for now
    default:
        m.error = fmt.Sprintf("Unknown config type: %s", integration.ConfigType)
        return m, nil
    }
}
```

### api_key Flow

The existing flow should work as-is, just needs to be wrapped in `enterAPIKeyConfig`:

1. `enterAPIKeyConfig` → shows profile selection (existing profiles + "New profile")
2. User selects profile → shows form with fields from `integration.Fields`
3. User fills form, saves → calls `ConfigureAPIKey` (or existing `ConfigureIntegration`)
4. Test integration works as before

Rename or reorganize existing methods as needed to fit this pattern.

### llm Placeholder

For now, `enterLLMConfig` can just set an informational message:

```go
func (m *IntegrationsModal) enterLLMConfig(integration Integration) (Modal, tea.Cmd) {
    m.view = viewConfigLLM
    m.configIntegration = integration
    // Actual implementation in Phase 3
    return m, nil
}
```

And `viewLLM()` can return a placeholder:

```go
func (m *IntegrationsModal) viewLLM() string {
    return "LLM configuration coming in Phase 3..."
}
```

### Status Display Update

Update the list view to show config-type-aware status:

```go
func (m *IntegrationsModal) formatStatus(i Integration) string {
    if !i.Configured {
        return "✗ not configured"
    }
    switch i.ConfigType {
    case "llm":
        return fmt.Sprintf("✓ %d providers · %d profiles", i.ProviderCount, i.ProfileCount)
    default:
        return "✓ configured"
    }
}
```

## Validation

- [ ] `go build` succeeds
- [ ] Selecting a `api_key` integration (e.g., Notion) opens existing config flow
- [ ] Can still configure an api_key integration (full flow works)
- [ ] Can still test an api_key integration
- [ ] Selecting an `llm` integration shows placeholder message
- [ ] Selecting an unknown config type shows error
- [ ] Status column shows appropriate format based on config type
