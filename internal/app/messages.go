package app

import tea "github.com/charmbracelet/bubbletea"

// Custom message types for the hub-tui application.
// Additional messages will be added as features are implemented.

// SetProgramMsg is sent to set the tea.Program reference for streaming.
type SetProgramMsg struct {
	Program *tea.Program
}

// QuitHintExpiredMsg is sent when the Ctrl+C hint timer expires.
type QuitHintExpiredMsg struct{}

// LoginResultMsg is sent when a login attempt completes.
type LoginResultMsg struct {
	Success   bool
	Token     string
	ExpiresAt string
	Error     string
}

// HealthCheckMsg is sent when a health check completes.
type HealthCheckMsg struct {
	Success bool
	Error   string
}

// StreamChunkMsg is sent when a chunk of streaming response arrives.
type StreamChunkMsg struct {
	Content string
}

// StreamDoneMsg is sent when streaming is complete.
type StreamDoneMsg struct {
	Error error
}

// RouteMsg is sent when routing info is received from /ask.
type RouteMsg struct {
	Type   string // "assistant", "workflow", "module", etc.
	Target string // Name of the target
}

// CacheRefreshMsg is sent when cache refresh completes.
type CacheRefreshMsg struct {
	Success    bool
	Error      string
	Assistants []string
	Workflows  []string
	Modules    []string
}
