package app

// Custom message types for the hub-tui application.
// Additional messages will be added as features are implemented.

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
