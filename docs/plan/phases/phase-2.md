# Phase 2: Modal List View

> **Depends on:** Phase 1 (Client Layer)
> **Enables:** Phase 3 (Modal Edit/Create)
>
> See: [Full Plan](../plan.md)

## Goal

Create the LLM profiles modal with list view, supporting display, test, delete, and set default operations.

## Key Deliverables

- `LLMModal` struct implementing `Modal` interface
- List view showing profiles in table format
- Test profile functionality with latency display
- Delete profile with confirmation/error handling
- Set default profile
- Warning indicator for unconfigured integrations

## Files to Create

- `internal/ui/modal/llm.go` — LLM profiles modal (list view only this phase)

## Dependencies

**Internal:**
- `internal/client/llm.go` (Phase 1)
- `internal/ui/modal/modal.go` (Modal interface)
- `internal/ui/theme/theme.go` (styling)

**External:** None

## Implementation Notes

### Modal Structure

```go
type LLMModal struct {
    client   *client.Client
    profiles *client.LLMProfileList
    selected int
    loading  bool
    error    string

    // View state
    view llmView  // viewList or viewEdit (Phase 3)

    // Test state
    testing    bool
    testResult *client.LLMTestResult

    // Delete state
    deleting bool
}
```

### Message Types

```go
type LLMProfilesLoadedMsg struct {
    Profiles *client.LLMProfileList
    Error    error
}

type LLMProfileTestedMsg struct {
    Name   string
    Result *client.LLMTestResult
    Error  error
}

type LLMProfileDeletedMsg struct {
    Name  string
    Error error
}

type LLMDefaultSetMsg struct {
    Name  string
    Error error
}
```

### List View Display

Table columns:
1. **Name** — Profile name with `★` prefix if default, highlight if selected
2. **Model** — Model name (e.g., `gpt-4o`)
3. **Provider** — Format as `integration (profile)`, e.g., `openai (default)`

Warning indicator: Show `⚠` next to provider if test fails or integration unconfigured.

### Key Bindings (List View)

| Key | Action |
|-----|--------|
| `j`/`↓` | Move selection down |
| `k`/`↑` | Move selection up |
| `Enter` | Edit selected profile (Phase 3) |
| `n` | New profile (Phase 3) |
| `t` | Test selected profile |
| `d` | Delete selected profile |
| `s` | Set selected as default |
| `r` | Refresh list |
| `Esc` | Close modal |

### Test Result Display

After testing, show below the list:
- Success: `✓ Connected (245ms)` in green
- Failure: `✗ Connection failed: <error>` in red

### Pattern Reference

Follow `internal/ui/modal/integrations.go` for:
- Modal interface implementation (`Init`, `Update`, `View`, `Title`)
- Async message handling pattern
- Loading/error states
- Key binding structure
- Styling with theme colors

## Validation

How do we know this phase is complete?

- [ ] `LLMModal` implements `Modal` interface
- [ ] `Init()` triggers profile loading
- [ ] List view displays profiles with Name, Model, Provider columns
- [ ] Default profile shows `★` indicator
- [ ] `j`/`k` navigation works
- [ ] `t` tests profile and shows result with latency
- [ ] `d` deletes profile (displays API error if protected)
- [ ] `s` sets profile as default
- [ ] `r` refreshes the list
- [ ] `Esc` returns `nil` to close modal
- [ ] Loading and error states display correctly
- [ ] Code compiles with `go build ./...`
- [ ] Code passes `go vet ./...`
