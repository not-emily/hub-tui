package components

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// DefaultConfirmTimeout is the default duration for confirmation timeouts.
const DefaultConfirmTimeout = 2 * time.Second

// ConfirmationExpiredMsg is sent when a confirmation timeout expires.
// This is a generic message type used by all confirmation flows.
type ConfirmationExpiredMsg struct {
	Key string // The action key (e.g., "delete", "dismiss", "quit")
	ID  string // Identifier for what was being confirmed
}

// Confirmation provides reusable double-press confirmation logic.
// Embed this in modals or components that need confirmation flows.
type Confirmation struct {
	pendingKey string
	pendingID  string
	timeout    time.Duration
}

// NewConfirmation creates a new Confirmation with default timeout.
func NewConfirmation() *Confirmation {
	return &Confirmation{
		timeout: DefaultConfirmTimeout,
	}
}

// WithTimeout sets a custom timeout duration.
func (c *Confirmation) WithTimeout(d time.Duration) *Confirmation {
	c.timeout = d
	return c
}

// Check handles the double-press confirmation logic.
// Returns (shouldExecute, cmd) where:
//   - shouldExecute=true means this is the second press, perform the action
//   - shouldExecute=false means this is the first press, cmd starts the timeout
//
// Usage:
//
//	if execute, cmd := c.confirm.Check("delete", itemID); execute {
//	    return m, m.doDelete(itemID)
//	} else if cmd != nil {
//	    return m, cmd
//	}
func (c *Confirmation) Check(key, id string) (bool, tea.Cmd) {
	if c.pendingKey == key && c.pendingID == id {
		// Second press - clear and signal to execute
		c.Clear()
		return true, nil
	}
	// First press - set pending and return timeout command
	c.pendingKey = key
	c.pendingID = id
	timeout := c.timeout
	if timeout == 0 {
		timeout = DefaultConfirmTimeout
	}
	return false, tea.Tick(timeout, func(t time.Time) tea.Msg {
		return ConfirmationExpiredMsg{Key: key, ID: id}
	})
}

// Clear resets the pending confirmation state.
// Call this on navigation, escape, or other actions that should cancel confirmation.
func (c *Confirmation) Clear() {
	c.pendingKey = ""
	c.pendingID = ""
}

// IsPending returns true if there's a pending confirmation for the given key.
// The id parameter is optional - if empty, checks if any confirmation is pending for the key.
func (c *Confirmation) IsPending(key string, id string) bool {
	if id == "" {
		return c.pendingKey == key
	}
	return c.pendingKey == key && c.pendingID == id
}

// IsPendingAny returns true if any confirmation is pending.
func (c *Confirmation) IsPendingAny() bool {
	return c.pendingKey != ""
}

// PendingKey returns the current pending action key.
func (c *Confirmation) PendingKey() string {
	return c.pendingKey
}

// PendingID returns the current pending identifier.
func (c *Confirmation) PendingID() string {
	return c.pendingID
}

// HandleExpired should be called when ConfirmationExpiredMsg is received.
// It clears the pending state only if it matches the expired message.
func (c *Confirmation) HandleExpired(msg ConfirmationExpiredMsg) {
	if c.pendingKey == msg.Key && c.pendingID == msg.ID {
		c.Clear()
	}
}
