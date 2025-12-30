# Phase 4: App Integration

> **Depends on:** Phase 2 (Modal List View), Phase 3 (Modal Edit/Create)
> **Enables:** Feature complete
>
> See: [Full Plan](../plan.md)

## Goal

Wire the LLM modal into the app, register the `/llm` slash command, handle async messages, and update the help modal.

## Key Deliverables

- `/llm` slash command registered and working
- All LLM modal messages handled in app.go
- Help modal updated with `/llm` command
- End-to-end testing of the feature

## Files to Modify

- `internal/app/app.go` — Register command, handle messages
- `internal/ui/modal/help.go` — Add `/llm` to command list

## Dependencies

**Internal:**
- `internal/ui/modal/llm.go` (Phases 2-3)
- Existing app structure and patterns

**External:** None

## Implementation Notes

### Register Slash Command

In `app.go`, add `/llm` to the slash command handling:

```go
case "/llm":
    m.modal = modal.NewLLMModal(m.client)
    return m, m.modal.Init()
```

### Handle Async Messages

Add cases in `app.go` `Update()` for all LLM modal messages:

```go
case modal.LLMProfilesLoadedMsg:
    if m.modal != nil {
        var cmd tea.Cmd
        m.modal, cmd = m.modal.Update(msg)
        return m, cmd
    }

case modal.LLMProfileTestedMsg:
    if m.modal != nil {
        var cmd tea.Cmd
        m.modal, cmd = m.modal.Update(msg)
        return m, cmd
    }

case modal.LLMProfileDeletedMsg:
    if m.modal != nil {
        var cmd tea.Cmd
        m.modal, cmd = m.modal.Update(msg)
        return m, cmd
    }

case modal.LLMDefaultSetMsg:
    if m.modal != nil {
        var cmd tea.Cmd
        m.modal, cmd = m.modal.Update(msg)
        return m, cmd
    }

case modal.LLMProfileSavedMsg:
    if m.modal != nil {
        var cmd tea.Cmd
        m.modal, cmd = m.modal.Update(msg)
        return m, cmd
    }
```

### Update Help Modal

In `internal/ui/modal/help.go`, add `/llm` to the commands list:

```go
// In the commands section, add:
{"/llm", "Manage LLM profiles"},
```

Place it logically near other configuration commands like `/integrations`, `/modules`.

### Autocomplete (if applicable)

If slash commands are statically defined for autocomplete, add `/llm` to that list. Check `internal/ui/chat/input.go` for the commands list.

### Testing Checklist

1. Type `/llm` and press Enter → modal opens
2. Modal shows list of profiles (or empty state)
3. Navigation with j/k works
4. `t` tests profile, shows result
5. `s` sets default, star indicator moves
6. `d` deletes profile
7. `n` opens create form
8. Create new profile → appears in list
9. `Enter` on profile → edit form with values
10. Edit and save → changes reflected
11. Rename → old name gone, new name appears
12. `Esc` from list closes modal
13. `/help` shows `/llm` command

## Validation

How do we know this phase is complete?

- [ ] `/llm` command opens the LLM profiles modal
- [ ] All async messages (loaded, tested, deleted, saved, default set) handled correctly
- [ ] Modal closes properly and returns to chat
- [ ] `/help` modal lists `/llm` command with description
- [ ] Autocomplete includes `/llm` (if applicable)
- [ ] Full end-to-end flow works: list → create → edit → delete → set default
- [ ] No console errors or panics
- [ ] Code compiles with `go build ./...`
- [ ] Code passes `go vet ./...`
