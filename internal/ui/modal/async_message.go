package modal

import "github.com/pxp/hub-tui/internal/client"

// AsyncModalMessage is implemented by messages that are results of async operations
// (typically API calls) initiated by modal components. These messages are automatically
// forwarded to the modal and checked for auth errors.
//
// To add a new async message:
// 1. Define your message struct with an Error field (or Err for consistency with existing code)
// 2. Add the interface methods (see examples below)
// 3. The message will automatically be routed to the modal in app.go
//
// Example:
//
//	type MyNewMsg struct {
//	    Data  interface{}
//	    Error error
//	}
//	func (m MyNewMsg) IsAsyncModalMessage() {}
//	func (m MyNewMsg) AuthError() error     { return m.Error }
type AsyncModalMessage interface {
	// IsAsyncModalMessage is a marker method to identify async modal messages.
	IsAsyncModalMessage()

	// AuthError returns the error from this message, or nil if no error.
	// Used to check for authentication expiration.
	AuthError() error
}

// IsAuthError checks if an error is an authentication error.
// Convenience wrapper around client.IsAuthError for use in app.go.
func IsAuthError(err error) bool {
	return err != nil && client.IsAuthError(err)
}
