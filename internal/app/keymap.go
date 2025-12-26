package app

import "github.com/charmbracelet/bubbletea"

// Key binding constants
const (
	KeyCtrlC = "ctrl+c"
	KeyCtrlL = "ctrl+l"
	KeyEsc   = "esc"
)

// IsQuit checks if the key message is Ctrl+C
func IsQuit(msg tea.KeyMsg) bool {
	return msg.String() == KeyCtrlC
}

// IsRedraw checks if the key message is Ctrl+L
func IsRedraw(msg tea.KeyMsg) bool {
	return msg.String() == KeyCtrlL
}

// IsCancel checks if the key message is Escape
func IsCancel(msg tea.KeyMsg) bool {
	return msg.String() == KeyEsc
}
