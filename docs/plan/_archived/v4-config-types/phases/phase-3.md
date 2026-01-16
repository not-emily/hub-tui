# Phase 3: LLM Config View

> **Depends on:** Phase 2 (Modal Routing + api_key)
> **Enables:** Phase 4 (LLM Provider Management)
>
> See: [Full Plan](../plan.md)

## Goal

Display providers and profiles in a navigable list view with continuous navigation.

## Key Deliverables

- New `integrations_llm.go` file in modal layer
- `viewConfigLLM` state with providers + profiles sections
- Indented list rendering (provider headers non-selectable)
- Continuous j/k navigation through both sections
- Data loading on entry
- Refresh and back navigation

## Files to Create

- `internal/ui/modal/integrations_llm.go` — LLM config type views and logic

## Files to Modify

- `internal/ui/modal/integrations.go` — Wire up LLM view routing

## Dependencies

**Internal:**
- Phase 2 complete (routing calls `enterLLMConfig`)
- Client methods from Phase 1 (`ListLLMProviders`, `ListLLMProfiles`)

**External:** None

## Implementation Notes

### View State Constants

Add to integrations modal (offset to avoid collision with existing states):

```go
const (
    viewConfigLLM integrationsView = iota + 100
    viewLLMProviderForm
    viewLLMProfileForm
)
```

### LLM State Fields

Add fields to `IntegrationsModal` struct (or create embedded struct):

```go
// LLM config state
llmIntegration  Integration       // current integration being configured
llmProviders    []ProviderAccount // loaded providers
llmProfiles     []LLMProfile      // loaded profiles
llmItems        []llmListItem     // flattened list for navigation
llmSelected     int               // current selection index
llmLoading      bool
llmError        string
```

### Flattened List Model

Create a unified list for navigation:

```go
type llmItemType int

const (
    llmItemProviderAccount llmItemType = iota
    llmItemProfile
)

type llmListItem struct {
    Type     llmItemType
    Provider string // for provider accounts
    Account  string // for provider accounts
    Profile  *LLMProfile // for profiles
}
```

Build this list from providers and profiles:

```go
func (m *IntegrationsModal) buildLLMItems() {
    m.llmItems = nil

    // Add provider accounts
    for _, p := range m.llmProviders {
        for _, acct := range p.Accounts {
            m.llmItems = append(m.llmItems, llmListItem{
                Type:     llmItemProviderAccount,
                Provider: p.Provider,
                Account:  acct,
            })
        }
    }

    // Add profiles
    for i := range m.llmProfiles {
        m.llmItems = append(m.llmItems, llmListItem{
            Type:    llmItemProfile,
            Profile: &m.llmProfiles[i],
        })
    }
}
```

### View Rendering

Render the indented list with section headers:

```
Providers
  OpenAI
  > • default                    ← selected
    • work
  Anthropic
    • personal

─────────────────────────────────

Profiles
  ★ default    openai/default · gpt-4o
    fast       openai/default · gpt-4o-mini

[a] Add provider  [n] New profile  [r] Refresh
```

Key rendering logic:
- Track current provider to insert headers when provider changes
- Highlight selected item with `>`
- Show `★` for default profile
- Provider headers are just visual (not in `llmItems`)

```go
func (m *IntegrationsModal) viewLLM() string {
    var b strings.Builder

    b.WriteString("Providers\n")
    currentProvider := ""
    providerAccountIdx := 0

    for i, item := range m.llmItems {
        if item.Type == llmItemProviderAccount {
            // Insert provider header if changed
            if item.Provider != currentProvider {
                currentProvider = item.Provider
                displayName := m.getProviderDisplayName(item.Provider)
                b.WriteString(fmt.Sprintf("  %s\n", displayName))
            }

            // Render account
            cursor := "  "
            if i == m.llmSelected {
                cursor = "> "
            }
            b.WriteString(fmt.Sprintf("  %s• %s\n", cursor, item.Account))
            providerAccountIdx++
        }
    }

    b.WriteString("\n─────────────────────────────────\n\n")
    b.WriteString("Profiles\n")

    for i, item := range m.llmItems {
        if item.Type == llmItemProfile {
            cursor := "  "
            if i == m.llmSelected {
                cursor = "> "
            }
            defaultMark := "  "
            if item.Profile.IsDefault {
                defaultMark = "★ "
            }
            b.WriteString(fmt.Sprintf("%s%s%s    %s/%s · %s\n",
                cursor, defaultMark, item.Profile.Name,
                item.Profile.Provider, item.Profile.Account, item.Profile.Model))
        }
    }

    b.WriteString("\n[a] Add provider  [n] New profile  [r] Refresh")

    return b.String()
}
```

### Navigation

Handle j/k/up/down for continuous navigation:

```go
func (m *IntegrationsModal) updateLLM(msg tea.Msg) (Modal, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "j", "down":
            if m.llmSelected < len(m.llmItems)-1 {
                m.llmSelected++
            }
        case "k", "up":
            if m.llmSelected > 0 {
                m.llmSelected--
            }
        case "r":
            return m, m.loadLLMData()
        case "esc":
            m.view = viewList
            return m, nil
        // a, n, Enter, d, t, s — Phase 4 and 5
        }
    }
    return m, nil
}
```

### Data Loading

Load providers and profiles on entry:

```go
func (m *IntegrationsModal) enterLLMConfig(integration Integration) (Modal, tea.Cmd) {
    m.view = viewConfigLLM
    m.llmIntegration = integration
    m.llmLoading = true
    return m, m.loadLLMData()
}

func (m *IntegrationsModal) loadLLMData() tea.Cmd {
    return func() tea.Msg {
        providers, err := m.client.ListLLMProviders(m.llmIntegration.Name)
        if err != nil {
            return llmDataLoadedMsg{err: err}
        }

        profileList, err := m.client.ListLLMProfiles(m.llmIntegration.Name)
        if err != nil {
            return llmDataLoadedMsg{err: err}
        }

        return llmDataLoadedMsg{
            providers: providers,
            profiles:  profileList.Profiles,
        }
    }
}
```

### Empty State

Handle case where there are no providers or profiles yet:

```
Providers
  (none configured)

─────────────────────────────────

Profiles
  (none configured)

[a] Add provider  [n] New profile  [r] Refresh
```

## Validation

- [ ] `go build` succeeds
- [ ] Opening LLM integration shows providers section with accounts
- [ ] Opening LLM integration shows profiles section
- [ ] Provider accounts grouped under provider headers
- [ ] Provider headers are visual only (cursor skips them)
- [ ] j/k navigation moves through all items continuously
- [ ] Empty state displays correctly when no providers/profiles
- [ ] `[r]` refreshes the data
- [ ] `[Esc]` returns to integration list
- [ ] Default profile shows ★ indicator
