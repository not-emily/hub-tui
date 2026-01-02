package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/pxp/hub-tui/internal/client"
)

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

// AuthExpiredMsg is sent when an API call fails due to expired/invalid token.
type AuthExpiredMsg struct{}

// WorkflowStartedMsg is sent when a workflow is successfully triggered.
type WorkflowStartedMsg struct {
	Name  string
	RunID string
}

// WorkflowErrorMsg is sent when a workflow fails to start.
type WorkflowErrorMsg struct {
	Name  string
	Error string
}

// PollTasksMsg triggers a task status poll.
type PollTasksMsg struct{}

// TaskStatusMsg is sent when task status is fetched.
type TaskStatusMsg struct {
	Runs  []Run
	Error error
}

// TaskCompletedMsg is sent when a running task completes.
type TaskCompletedMsg struct {
	Workflow string
	Success  bool
	Error    string
}

// TaskCancelledMsg is sent when a task is cancelled.
type TaskCancelledMsg struct {
	RunID string
	Error error
}

// Run mirrors client.Run for use in messages.
type Run struct {
	ID             string
	Workflow       string
	Status         string
	StartedAt      string
	EndedAt        string
	Error          string
	Result         *RunResult
	NeedsAttention bool
}

// RunResult mirrors client.RunResult.
type RunResult struct {
	WorkflowName string
	Success      bool
	Steps        []StepResult
	Error        string
}

// StepResult mirrors client.StepResult.
type StepResult struct {
	StepName string
	Success  bool
	Output   interface{}
	Error    string
}

// AskNeedsInputMsg indicates the API needs more input (parameter collection).
type AskNeedsInputMsg struct {
	Target string
	Schema *client.ParamSchema
}

// AskExecutedMsg indicates the API executed successfully.
type AskExecutedMsg struct {
	Target string
	Result *client.ExecuteResult
}

// AskErrorMsg indicates an API error.
type AskErrorMsg struct {
	Target string
	Error  *client.AskError
}
