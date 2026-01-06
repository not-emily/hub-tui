package chat

import "strings"

// InputPrefix identifies what type of input the user is entering.
type InputPrefix int

const (
	PrefixNone      InputPrefix = iota
	PrefixAssistant             // @
	PrefixWorkflow              // #
	PrefixCommand               // /
)

// Command represents a parsed slash command.
type Command struct {
	Name string
	Args string
}

// KnownCommands is the list of valid slash commands.
var KnownCommands = []string{
	"exit",
	"clear",
	"help",
	"hub",
	"refresh",
	"modules",
	"integrations",
	"workflows",
	"tasks",
	"settings",
}

// DetectPrefix returns the prefix type and the text after the prefix.
func DetectPrefix(input string) (InputPrefix, string) {
	input = strings.TrimSpace(input)
	if len(input) == 0 {
		return PrefixNone, ""
	}

	switch input[0] {
	case '@':
		return PrefixAssistant, input[1:]
	case '#':
		return PrefixWorkflow, input[1:]
	case '/':
		return PrefixCommand, input[1:]
	}
	return PrefixNone, input
}

// ParseCommand parses a slash command from input.
// Returns nil if the input is not a command.
func ParseCommand(input string) *Command {
	prefix, rest := DetectPrefix(input)
	if prefix != PrefixCommand {
		return nil
	}

	parts := strings.SplitN(rest, " ", 2)
	cmd := &Command{Name: strings.ToLower(parts[0])}
	if len(parts) > 1 {
		cmd.Args = parts[1]
	}
	return cmd
}

// IsValidCommand checks if a command name is known.
func IsValidCommand(name string) bool {
	for _, c := range KnownCommands {
		if c == name {
			return true
		}
	}
	return false
}
