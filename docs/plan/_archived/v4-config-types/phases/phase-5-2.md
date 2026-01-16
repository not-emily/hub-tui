# Phase 5.2: LLM Profile Operations

> **Depends on:** Phase 5.1 (LLM Profile Form)
> **Enables:** Phase 6 (Cleanup & Integration)
>
> See: [Full Plan](../plan.md)

## Goal

Add delete, test, and set-default operations for LLM profiles from the list view.

## Key Deliverables

- Delete profile with double-press confirmation
- Test profile and show result
- Set profile as default
- Visual feedback for operations

## Files to Modify

- `internal/ui/modal/integrations_llm.go` — Add profile operations

## Dependencies

**Internal:**
- Phase 5.1 complete (profile form works)
- Client methods: `DeleteLLMProfile`, `TestLLMProfile`, `SetDefaultLLMProfile`
- Confirmation component from Phase 4

**External:** None

## Implementation Notes

### Delete Profile

Reuse the confirmation pattern from Phase 4:

```go
case "d":
    item := m.llmItems[m.llmSelected]

    if item.Type == llmItemProviderAccount {
        // Provider delete (from Phase 4)
        key := fmt.Sprintf("provider:%s/%s", item.Provider, item.Account)
        if execute, cmd := m.llmConfirm.Check(key, item.Account); execute {
            return m, m.deleteProvider(item.Provider, item.Account)
        } else if cmd != nil {
            return m, cmd
        }
    } else if item.Type == llmItemProfile {
        // Profile delete
        key := fmt.Sprintf("profile:%s", item.Profile.Name)
        if execute, cmd := m.llmConfirm.Check(key, item.Profile.Name); execute {
            return m, m.deleteProfile(item.Profile.Name)
        } else if cmd != nil {
            return m, cmd
        }
    }
```

Delete and refresh:

```go
func (m *IntegrationsModal) deleteProfile(name string) tea.Cmd {
    return func() tea.Msg {
        err := m.client.DeleteLLMProfile(m.llmIntegration.Name, name)
        if err != nil {
            return llmErrorMsg{err: err}
        }
        return llmProfileDeletedMsg{}
    }
}

case llmProfileDeletedMsg:
    return m, m.loadLLMData()
```

### Test Profile

Add state for test result:

```go
llmTesting     bool
llmTestResult  *LLMTestResult
llmTestProfile string
```

Handle `[t]` key:

```go
case "t":
    item := m.llmItems[m.llmSelected]
    if item.Type == llmItemProfile {
        m.llmTesting = true
        m.llmTestProfile = item.Profile.Name
        m.llmTestResult = nil
        return m, m.testProfile(item.Profile.Name)
    }

func (m *IntegrationsModal) testProfile(name string) tea.Cmd {
    return func() tea.Msg {
        result, err := m.client.TestLLMProfile(m.llmIntegration.Name, name)
        if err != nil {
            return llmProfileTestedMsg{err: err}
        }
        return llmProfileTestedMsg{result: result}
    }
}

case llmProfileTestedMsg:
    m.llmTesting = false
    if msg.err != nil {
        m.llmError = msg.err.Error()
        return m, nil
    }
    m.llmTestResult = msg.result
    return m, nil
```

### Display Test Result

Show result in the view:

```go
func (m *IntegrationsModal) viewLLM() string {
    // ... existing render ...

    // Show test result
    if m.llmTesting {
        b.WriteString(fmt.Sprintf("\n\nTesting %s...", m.llmTestProfile))
    } else if m.llmTestResult != nil {
        if m.llmTestResult.Success {
            b.WriteString(fmt.Sprintf("\n\n✓ %s: %dms",
                m.llmTestResult.Model, m.llmTestResult.LatencyMs))
        } else {
            b.WriteString(fmt.Sprintf("\n\n✗ %s: %s",
                m.llmTestResult.Model, m.llmTestResult.Error))
        }
    }

    // ... rest of render ...
}
```

Clear test result on navigation:

```go
case "j", "down", "k", "up":
    m.llmTestResult = nil // Clear stale result
    // ... navigation logic ...
```

### Set Default

Handle `[s]` key:

```go
case "s":
    item := m.llmItems[m.llmSelected]
    if item.Type == llmItemProfile && !item.Profile.IsDefault {
        return m, m.setDefaultProfile(item.Profile.Name)
    }

func (m *IntegrationsModal) setDefaultProfile(name string) tea.Cmd {
    return func() tea.Msg {
        err := m.client.SetDefaultLLMProfile(m.llmIntegration.Name, name)
        if err != nil {
            return llmErrorMsg{err: err}
        }
        return llmDefaultSetMsg{}
    }
}

case llmDefaultSetMsg:
    return m, m.loadLLMData() // Refresh to update ★ indicator
```

### Key Hints

Update the key hints in the view to show available actions:

```go
func (m *IntegrationsModal) viewLLM() string {
    // ... render lists ...

    // Key hints based on selection
    var hints []string
    hints = append(hints, "[a] Add provider", "[n] New profile", "[r] Refresh")

    if m.llmSelected >= 0 && m.llmSelected < len(m.llmItems) {
        item := m.llmItems[m.llmSelected]
        if item.Type == llmItemProviderAccount {
            hints = append(hints, "[d] Delete")
        } else if item.Type == llmItemProfile {
            hints = append(hints, "[Enter] Edit", "[t] Test", "[d] Delete")
            if !item.Profile.IsDefault {
                hints = append(hints, "[s] Set default")
            }
        }
    }

    b.WriteString("\n\n" + strings.Join(hints, "  "))

    // ... confirmation and test result ...
}
```

### Error Handling

Display errors and allow dismissal:

```go
// In view
if m.llmError != "" {
    b.WriteString(fmt.Sprintf("\n\nError: %s", m.llmError))
}

// Clear error on any key
case tea.KeyMsg:
    if m.llmError != "" {
        m.llmError = ""
    }
    // ... rest of key handling ...
```

## Validation

- [ ] `go build` succeeds
- [ ] `[d]` on profile shows confirmation hint
- [ ] Second `[d]` deletes profile
- [ ] Profile removed from list after delete
- [ ] `[t]` on profile shows "Testing..." message
- [ ] Test result shows success with latency
- [ ] Test result shows failure with error message
- [ ] Test result clears on navigation
- [ ] `[s]` on non-default profile sets it as default
- [ ] `[s]` not shown for already-default profile
- [ ] ★ indicator moves to new default after setting
- [ ] Key hints update based on selection (provider account vs profile)
- [ ] Errors display and can be dismissed
