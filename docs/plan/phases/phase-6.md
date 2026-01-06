# Phase 6: Cleanup & Integration

> **Depends on:** Phase 5.2 (LLM Profile Operations)
> **Enables:** Project complete
>
> See: [Full Plan](../plan.md)

## Goal

Remove old code, ensure `/integrations` is the single entry point, and update documentation.

## Key Deliverables

- Remove `/llm` command from app
- Delete old `client/llm.go` file
- Delete old `modal/llm.go` file
- Update help modal
- Update PROJECT_PROGRESS.md
- Verify clean build with no dead code

## Files to Delete

- `internal/client/llm.go` — Replaced by `integrations_llm.go`
- `internal/ui/modal/llm.go` — Logic moved to `integrations_llm.go`

## Files to Modify

- `internal/app/app.go` — Remove `/llm` command handler
- `internal/ui/modal/help.go` — Remove `/llm` from help, update `/integrations` description
- `.claude/PROJECT_PROGRESS.md` — Update with completion status

## Dependencies

**Internal:**
- All previous phases complete
- Full LLM config flow working through integrations modal

**External:** None

## Implementation Notes

### Remove /llm Command

In `app.go`, find and remove the `/llm` command handler:

```go
// Remove this case from the command switch
case "/llm":
    return m.openLLMModal()
```

Also remove any associated methods like `openLLMModal()`.

### Delete Old Files

```bash
rm internal/client/llm.go
rm internal/ui/modal/llm.go
```

After deletion, run `go build` to identify any remaining references. Fix import statements and remove any orphaned code.

### Update Help Modal

In `help.go`, update the command list:

**Remove:**
```
/llm         Manage LLM profiles
```

**Update description for /integrations:**
```
/integrations    Manage external service connections and AI providers
```

### Check for Dead Code

After removing old files, look for:

1. **Unused imports** in files that previously imported llm.go
2. **Unused message types** (e.g., old LLMProfilesLoadedMsg)
3. **Unused methods** on Modal or Client structs

Run:
```bash
go build ./...
go vet ./...
```

### Update Status Bar

If the status bar shows LLM-specific info (profile count, etc.), verify it still works or update it to use the new data source.

Check `internal/ui/status/` for any LLM references.

### Update PROJECT_PROGRESS.md

```markdown
## Current Focus
Config-type-aware Integration Configuration - COMPLETE

## Completed This Week
- v4-config-types implementation
  - Phase 1: Client layer updates (Integration.ConfigType, integrations_llm.go)
  - Phase 2: Modal routing + api_key flow
  - Phase 3: LLM config view (providers + profiles list)
  - Phase 4: LLM provider management (add/delete)
  - Phase 5.1: LLM profile form (cascading dropdowns, pagination)
  - Phase 5.2: LLM profile operations (delete, test, set default)
  - Phase 6: Cleanup (removed /llm command and old files)

## Future Enhancements
- oauth config type
- email_pass config type
```

### Archive Plan Files

Move completed plan to archive:

```bash
mkdir -p docs/plan/_archived/v4-config-types
mv docs/plan/plan.md docs/plan/_archived/v4-config-types/
mv docs/plan/phases docs/plan/_archived/v4-config-types/
```

### Final Testing Checklist

Before marking complete, verify:

1. `/integrations` command works
2. api_key integrations can be configured
3. LLM integration shows provider/profile view
4. Can add provider account
5. Can create LLM profile with cascading dropdowns
6. Can delete provider and profile
7. Can test profile
8. Can set default profile
9. `/llm` command shows "unknown command" error
10. Help modal shows correct commands
11. No build warnings or errors

## Validation

- [ ] `go build ./...` succeeds with no errors
- [ ] `go vet ./...` reports no issues
- [ ] `/llm` command no longer works
- [ ] `/integrations` is the only entry point for all integration config
- [ ] Help modal updated (no `/llm`, updated `/integrations` description)
- [ ] No unused imports or dead code
- [ ] PROJECT_PROGRESS.md updated
- [ ] Plan files archived
- [ ] All functionality tested end-to-end
