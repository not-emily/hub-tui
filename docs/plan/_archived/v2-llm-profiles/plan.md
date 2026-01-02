# LLM Profile Management: Add /llm modal to hub-tui

> **Status:** Planning complete | Last updated: 2025-12-30
>
> Phase files: [phases/](phases/)

## Overview

Add a `/llm` modal to hub-tui for managing LLM profiles. LLM profiles are named configurations that map to a specific integration + model combination, allowing users to decouple their assistants and workflows from specific providers. For example, a "default" profile might use GPT-4o today but could be switched to Claude tomorrow without updating any workflows.

The modal follows established patterns from `/integrations` and `/modules` modals, providing list, create, edit, and delete functionality with test connectivity support.

## Core Vision

- **Provider Agnostic**: Profiles abstract away the underlying integration, making it easy to switch providers
- **Consistent Patterns**: Follow existing modal conventions for familiarity
- **Clear Feedback**: Show connection status, test results, and warnings for misconfigured profiles

## Requirements

### Must Have
- List all LLM profiles with name, model, provider (integration + profile), default indicator
- Create new profile with name, integration, integration profile, model
- Edit existing profile configuration
- Rename profile
- Delete profile (API handles protection if in use)
- Test profile connectivity with latency display
- Set default profile
- Warning indicator for profiles with unconfigured integrations
- Update `/help` modal with new `/llm` command

### Nice to Have
- Model suggestions dropdown (requires hub-core `/llm/providers` endpoint)
- Show which assistants/workflows use each profile

### Out of Scope
- Integration configuration (use `/integrations` modal)
- Creating new integrations from within `/llm` modal

## Constraints

- **Tech stack**: Go, Bubble Tea, Lip Gloss (existing stack)
- **API**: Hub-core endpoints already exist
- **Patterns**: Must follow existing modal patterns for consistency

## Success Metrics

- `/llm` command opens modal and displays profiles
- All CRUD operations work correctly
- Test shows latency in milliseconds
- Default profile indicated with star
- Unconfigured integrations show warning

## Architecture Decisions

### 1. Single Modal with Views
**Choice:** Use multi-view modal pattern (list → edit/create) like integrations modal
**Rationale:** Proven pattern, familiar navigation, keeps code organized
**Trade-offs:** Slightly more complex than separate modals, but more cohesive UX

### 2. Display Format for Provider
**Choice:** Show `integration (profile)` format, e.g., `openai (default)` or `openai (work)`
**Rationale:** Users need to know which integration profile is used, especially with multiple accounts
**Trade-offs:** Slightly longer display, but more informative

### 3. Delete Protection
**Choice:** Let hub-core API handle delete protection and return errors
**Rationale:** Keeps logic centralized, TUI just displays API response
**Trade-offs:** Requires hub-core to implement this check

## Project Structure

```
internal/
├── client/
│   └── llm.go           # NEW: LLM profile API client
└── ui/
    └── modal/
        └── llm.go       # NEW: LLM profiles modal
```

### Files to Modify
- `internal/app/app.go` — register `/llm` command, handle async messages
- `internal/ui/modal/help.go` — add `/llm` to command list

## Core Interfaces

### Client Types

```go
// LLMProfile represents an LLM profile configuration.
type LLMProfile struct {
    Integration string `json:"integration"`          // e.g., "openai", "claude", "ollama"
    Profile     string `json:"profile,omitempty"`    // integration profile, defaults to "default"
    Model       string `json:"model"`                // e.g., "gpt-4o", "claude-sonnet-4-20250514"
}

// LLMProfileList is the response from GET /llm/profiles.
type LLMProfileList struct {
    Profiles       map[string]LLMProfile `json:"profiles"`
    DefaultProfile string                `json:"default_profile"`
}

// LLMTestResult is the response from POST /llm/profiles/{name}/test.
type LLMTestResult struct {
    Success   bool   `json:"success"`
    Model     string `json:"model"`
    LatencyMs int    `json:"latency_ms"`
    Error     string `json:"error,omitempty"`
}

// LLMProfileConfig is the request body for PUT /llm/profiles/{name}.
type LLMProfileConfig struct {
    Name        string `json:"name,omitempty"`  // for rename
    Integration string `json:"integration"`
    Profile     string `json:"profile,omitempty"`
    Model       string `json:"model"`
}
```

### Client Methods

```go
func (c *Client) ListLLMProfiles() (*LLMProfileList, error)
func (c *Client) CreateLLMProfile(name string, config LLMProfileConfig) error
func (c *Client) UpdateLLMProfile(name string, config LLMProfileConfig) error
func (c *Client) DeleteLLMProfile(name string) error
func (c *Client) TestLLMProfile(name string) (*LLMTestResult, error)
func (c *Client) SetDefaultLLMProfile(name string) error
```

### Modal Views

1. **List View**: Table showing profiles with selection, test results, action hints
2. **Edit/Create View**: Form with Name, Integration, Integration Profile, Model fields

## Implementation Phases

| Phase | Name | Scope | Depends On | Key Outputs |
|-------|------|-------|------------|-------------|
| 1 | Client Layer | API client methods | — | `internal/client/llm.go` |
| 2 | Modal List View | Display, test, delete, set default | Phase 1 | `internal/ui/modal/llm.go` (partial) |
| 3 | Modal Edit/Create | Form for create/edit/rename | Phase 2 | `internal/ui/modal/llm.go` (complete) |
| 4 | App Integration | Wire command, messages, help | Phases 2-3 | Updates to `app.go`, `help.go` |

### Critical Path
All phases are sequential. Each builds on the previous.

### Phase Details
- [Phase 1: Client Layer](phases/phase-1.md)
- [Phase 2: Modal List View](phases/phase-2.md)
- [Phase 3: Modal Edit/Create](phases/phase-3.md)
- [Phase 4: App Integration](phases/phase-4.md)

## Tech Stack

| Category | Choice | Notes |
|----------|--------|-------|
| Language | Go | Existing |
| TUI Framework | Bubble Tea | Existing |
| Styling | Lip Gloss | Existing |
| HTTP Client | net/http | Existing pattern |

## Future Considerations

- Model suggestions dropdown when hub-core adds `/llm/providers` endpoint
- Show usage count (which assistants/workflows use each profile)
- Bulk operations (test all, delete unused)
