package chat

import (
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/ui/theme"
)

// Role represents who sent a message.
type Role string

const (
	RoleUser   Role = "user"
	RoleHub    Role = "hub"
	RoleSystem Role = "system"
)

// Message symbols
const (
	UserSymbol      = "›"
	HubSymbol       = "●"
	SystemSymbol    = "›"
	StreamingCursor = "▌"
)

// Message represents a single chat message.
type Message struct {
	Role      Role
	Content   string
	Timestamp time.Time
	Streaming bool // True while response is being received
}

// NewUserMessage creates a new user message.
func NewUserMessage(content string) Message {
	return Message{
		Role:      RoleUser,
		Content:   content,
		Timestamp: time.Now(),
	}
}

// NewHubMessage creates a new hub message (initially empty for streaming).
func NewHubMessage() Message {
	return Message{
		Role:      RoleHub,
		Content:   "",
		Timestamp: time.Now(),
		Streaming: true,
	}
}

// NewSystemMessage creates a new system message.
func NewSystemMessage(content string) Message {
	return Message{
		Role:      RoleSystem,
		Content:   content,
		Timestamp: time.Now(),
	}
}

// AppendContent adds content to the message (used for streaming).
func (m *Message) AppendContent(chunk string) {
	m.Content += chunk
}

// FinishStreaming marks the message as complete.
func (m *Message) FinishStreaming() {
	m.Streaming = false
}

// Message styles
var (
	userSymbolStyle = lipgloss.NewStyle().
			Foreground(theme.Accent).
			Bold(true)

	userContentStyle = lipgloss.NewStyle().
				Foreground(theme.TextPrimary)

	hubSymbolStyle = lipgloss.NewStyle().
			Foreground(theme.TextPrimary).
			Bold(true)

	systemSymbolStyle = lipgloss.NewStyle().
				Foreground(theme.TextSecondary)

	systemContentStyle = lipgloss.NewStyle().
				Foreground(theme.TextSecondary)

	streamingStyle = lipgloss.NewStyle().
			Foreground(theme.Warning)
)

// Custom glamour style JSON - based on "dark" but with no left margin/indent
var glamourStyle = []byte(`{
	"document": {
		"block_prefix": "",
		"block_suffix": "",
		"margin": 0,
		"indent": 0
	},
	"paragraph": {
		"margin": 0
	},
	"heading": {
		"block_suffix": "",
		"margin": 0,
		"bold": true
	},
	"h1": {
		"prefix": "# ",
		"bold": true
	},
	"h2": {
		"prefix": "## ",
		"bold": true
	},
	"h3": {
		"prefix": "### ",
		"bold": true
	},
	"h4": {
		"prefix": "#### "
	},
	"h5": {
		"prefix": "##### "
	},
	"h6": {
		"prefix": "###### "
	},
	"list": {
		"level_indent": 2
	},
	"item": {
		"block_prefix": " "
	},
	"enumeration": {
		"block_prefix": ". "
	},
	"code_block": {
		"margin": 0,
		"chroma": {
			"theme": "dracula"
		}
	},
	"code": {
		"color": "203"
	},
	"emph": {
		"italic": true
	},
	"strong": {
		"bold": true
	},
	"link": {
		"color": "39",
		"underline": true
	},
	"link_text": {
		"color": "39",
		"bold": true
	}
}`)

// renderMarkdown renders markdown content using glamour.
// Uses a custom style with no left margin to fit our message layout.
func renderMarkdown(content string, width int) string {
	r, err := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes(glamourStyle),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return content
	}

	rendered, err := r.Render(content)
	if err != nil {
		return content
	}

	// Trim leading/trailing newlines glamour adds
	return strings.Trim(rendered, "\n")
}

// View renders the message.
func (m Message) View(width int) string {
	switch m.Role {
	case RoleUser:
		return m.renderUser(width)
	case RoleHub:
		return m.renderHub(width)
	case RoleSystem:
		return m.renderSystem(width)
	}
	return ""
}

func (m Message) renderUser(width int) string {
	symbol := userSymbolStyle.Render(UserSymbol)
	content := userContentStyle.
		Width(width - 4).
		Render(m.Content)

	// Indent continuation lines
	lines := strings.Split(content, "\n")
	var result strings.Builder
	for i, line := range lines {
		if i == 0 {
			result.WriteString(symbol + "  " + line)
		} else {
			result.WriteString("\n   " + line)
		}
	}
	return result.String()
}

func (m Message) renderHub(width int) string {
	symbol := hubSymbolStyle.Render(HubSymbol)

	content := m.Content
	if m.Streaming {
		content += streamingStyle.Render(StreamingCursor)
	} else if content != "" {
		// Render markdown only after streaming is complete
		content = renderMarkdown(content, width-4)
	}

	// Indent all content under the symbol
	lines := strings.Split(content, "\n")
	var result strings.Builder
	for i, line := range lines {
		if i == 0 {
			result.WriteString(symbol + "  " + line)
		} else {
			result.WriteString("\n   " + line)
		}
	}
	return result.String()
}

func (m Message) renderSystem(width int) string {
	symbol := systemSymbolStyle.Render(SystemSymbol)
	content := systemContentStyle.
		Width(width - 4).
		Render(m.Content)

	lines := strings.Split(content, "\n")
	var result strings.Builder
	for i, line := range lines {
		if i == 0 {
			result.WriteString(symbol + "  " + line)
		} else {
			result.WriteString("\n   " + line)
		}
	}
	return result.String()
}
