# Phase 3: Modal Edit/Create

> **Depends on:** Phase 2 (Modal List View)
> **Enables:** Phase 4 (App Integration)
>
> See: [Full Plan](../plan.md)

## Goal

Add create and edit views to the LLM modal, allowing users to create new profiles, edit existing ones, and rename profiles.

## Key Deliverables

- Edit/create view with form fields
- Create new profile flow
- Edit existing profile flow
- Rename profile support
- Form validation and error handling

## Files to Modify

- `internal/ui/modal/llm.go` — Add edit/create view

## Dependencies

**Internal:**
- `internal/ui/components/form.go` (Form component)
- Phase 2 modal structure

**External:** None

## Implementation Notes

### View State Addition

```go
type llmView int

const (
    viewList llmView = iota
    viewEdit
)

// Add to LLMModal struct:
type LLMModal struct {
    // ... existing fields from Phase 2

    // Edit mode
    editName     string  // original name (empty for create)
    editIsNew    bool    // true if creating, false if editing
    form         *components.Form
    saving       bool
}
```

### Message Types

```go
type LLMProfileSavedMsg struct {
    Name  string
    IsNew bool
    Error error
}
```

### Form Fields

| Field | Key | Required | Notes |
|-------|-----|----------|-------|
| Name | `name` | Yes | Profile name (alphanumeric, underscore, hyphen) |
| Integration | `integration` | Yes | e.g., `openai`, `claude`, `ollama` |
| Integration Profile | `profile` | No | Defaults to `default` if empty |
| Model | `model` | Yes | e.g., `gpt-4o`, `claude-sonnet-4-20250514` |

### Create Flow

1. User presses `n` in list view
2. Modal switches to edit view with empty form
3. `editIsNew = true`, `editName = ""`
4. User fills form and presses Enter
5. Call `CreateLLMProfile(formName, config)`
6. On success, return to list view and refresh

### Edit Flow

1. User presses `Enter` on selected profile
2. Modal switches to edit view with pre-filled form
3. `editIsNew = false`, `editName = selectedProfileName`
4. User modifies fields and presses Enter
5. If name changed: include `Name` in config (triggers rename)
6. Call `UpdateLLMProfile(editName, config)`
7. On success, return to list view and refresh

### Key Bindings (Edit View)

| Key | Action |
|-----|--------|
| `Tab` | Next field |
| `Shift+Tab` | Previous field |
| `Enter` | Submit form (if on last field) or next field |
| `Esc` | Cancel and return to list |

### Form Pre-population (Edit Mode)

```go
func (m *LLMModal) enterEditMode(profileName string) {
    profile := m.profiles.Profiles[profileName]

    fields := []components.FormField{
        {Label: "Name", Key: "name", Value: profileName},
        {Label: "Integration", Key: "integration", Value: profile.Integration},
        {Label: "Integration Profile", Key: "profile", Value: profile.Profile},
        {Label: "Model", Key: "model", Value: profile.Model},
    }

    m.form = components.NewForm("Edit Profile", fields)
    m.editName = profileName
    m.editIsNew = false
    m.view = viewEdit
}
```

### Validation

- Name: Required, alphanumeric + underscore/hyphen only
- Integration: Required
- Model: Required
- Profile: Optional (defaults to "default")

Display validation errors inline or below form.

### Pattern Reference

Follow `internal/ui/modal/integrations.go` `viewConfigure` for:
- Form initialization and rendering
- Saving state management
- Error display
- Navigation back to list on success

## Validation

How do we know this phase is complete?

- [ ] `n` opens create view with empty form
- [ ] `Enter` on profile opens edit view with pre-filled values
- [ ] Form displays all four fields (Name, Integration, Profile, Model)
- [ ] Tab navigation between fields works
- [ ] Creating a new profile calls API and refreshes list
- [ ] Editing a profile calls API and refreshes list
- [ ] Renaming (changing name field) works correctly
- [ ] `Esc` cancels and returns to list
- [ ] Saving errors display in the form view
- [ ] Code compiles with `go build ./...`
- [ ] Code passes `go vet ./...`
