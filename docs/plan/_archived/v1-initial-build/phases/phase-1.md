# Phase 1: Foundation

> **Depends on:** None
> **Enables:** Phase 2 (API Client & Connection)
>
> See: [Full Plan](../plan.md)

## Goal

Set up the project scaffolding with a basic Bubble Tea app that compiles, runs, and exits cleanly.

## Key Deliverables

- Project directory structure
- Go module with dependencies (Bubble Tea, Lip Gloss)
- Config file loading and saving
- Basic Bubble Tea app (starts, shows placeholder, quits)
- Theme foundation with dark color palette
- Build and run scripts

## Files to Create

- `cmd/hub-tui/main.go` — Entry point, initializes app
- `internal/app/app.go` — Root Bubble Tea model (placeholder)
- `internal/app/keymap.go` — Key binding definitions
- `internal/app/messages.go` — Custom message types (placeholder)
- `internal/config/config.go` — Config struct, load/save functions
- `internal/ui/theme/theme.go` — Lip Gloss styles, colors
- `scripts/build.sh` — Build the binary
- `scripts/run.sh` — Run for development
- `go.mod` — Module definition

## Dependencies

**Internal:** None (first phase)

**External:**
- `github.com/charmbracelet/bubbletea` — TUI framework
- `github.com/charmbracelet/lipgloss` — Styling

## Implementation Notes

### Config Structure

```go
type Config struct {
    ServerURL string `json:"server_url"`
    Token     string `json:"token,omitempty"`
    TokenExp  string `json:"token_expires,omitempty"`
}
```

Config file location: `~/.config/hub-tui/config.json`

On first run, config won't exist. The app should handle this gracefully (Phase 2 will prompt for server URL).

### Theme Colors

Dark theme using grays (not pure black):

```go
var (
    Background    = lipgloss.Color("#1a1a1a")  // Dark gray
    Surface       = lipgloss.Color("#2a2a2a")  // Slightly lighter
    TextPrimary   = lipgloss.Color("#e0e0e0")  // Light gray text
    TextSecondary = lipgloss.Color("#888888")  // Muted text
    Accent        = lipgloss.Color("#7c9fc7")  // Soft blue accent
    Error         = lipgloss.Color("#d46a6a")  // Soft red
    Success       = lipgloss.Color("#6ad47c")  // Soft green
)
```

### Basic App Structure

The root model should:
1. Initialize with terminal size
2. Handle window resize events
3. Support double Ctrl+C to quit (with hint message)
4. Show a placeholder message ("hub-tui" or similar)

```go
type Model struct {
    width, height int
    quitting      bool
    ctrlCPressed  bool  // For double Ctrl+C detection
}
```

### Build Script

```bash
#!/bin/bash
set -e
cd "$(dirname "$0")/.."
go build -o bin/hub-tui ./cmd/hub-tui
echo "Built: bin/hub-tui"
```

## Validation

How do we know this phase is complete?

- [ ] `go mod tidy` succeeds with no errors
- [ ] `./scripts/build.sh` produces `bin/hub-tui`
- [ ] `./scripts/run.sh` launches the TUI
- [ ] TUI shows placeholder content
- [ ] Single Ctrl+C shows "Ctrl+C again to quit" hint
- [ ] Double Ctrl+C exits cleanly
- [ ] Config file is created at `~/.config/hub-tui/config.json` on first run
- [ ] Restarting reads existing config file
