package chat

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pxp/hub-tui/internal/ui/theme"
)

const (
	scrollPageSize = 10
)

// Model is the chat view component.
type Model struct {
	messages     []Message
	input        Input
	autocomplete Autocomplete
	width        int
	height       int
	scrollPos    int  // Current scroll position (0 = bottom)
	autoScroll   bool // Whether to auto-scroll on new messages
	inContext    bool // Whether in assistant context (for input border)
}

// New creates a new chat model.
func New() Model {
	return Model{
		messages:     make([]Message, 0),
		input:        NewInput(),
		autocomplete: NewAutocomplete(),
		autoScroll:   true,
	}
}

// SetSize sets the chat view dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.input.SetWidth(width)
	m.autocomplete.SetWidth(width)
}

// SetInContext sets whether chat is in assistant context (affects input border).
func (m *Model) SetInContext(inContext bool) {
	m.inContext = inContext
}

// AddUserMessage adds a user message to the chat.
func (m *Model) AddUserMessage(content string) {
	m.messages = append(m.messages, NewUserMessage(content))
	if m.autoScroll {
		m.scrollPos = 0
	}
}

// AddHubMessage adds a new hub message (for streaming).
func (m *Model) AddHubMessage() {
	m.messages = append(m.messages, NewHubMessage())
	if m.autoScroll {
		m.scrollPos = 0
	}
}

// AddSystemMessage adds a system message to the chat.
func (m *Model) AddSystemMessage(content string) {
	m.messages = append(m.messages, NewSystemMessage(content))
	if m.autoScroll {
		m.scrollPos = 0
	}
}

// ClearMessages clears all messages from the chat.
func (m *Model) ClearMessages() {
	m.messages = make([]Message, 0)
	m.scrollPos = 0
	m.autoScroll = true
}

// AppendToLastMessage appends content to the last message.
func (m *Model) AppendToLastMessage(chunk string) {
	if len(m.messages) > 0 {
		m.messages[len(m.messages)-1].AppendContent(chunk)
	}
}

// FinishLastMessage marks the last message as complete.
func (m *Model) FinishLastMessage() {
	if len(m.messages) > 0 {
		m.messages[len(m.messages)-1].FinishStreaming()
	}
}

// ReplaceLastMessageContent replaces the content of the last message.
func (m *Model) ReplaceLastMessageContent(content string) {
	if len(m.messages) > 0 {
		m.messages[len(m.messages)-1].Content = content
	}
}

// MessageCount returns the number of messages.
func (m Model) MessageCount() int {
	return len(m.messages)
}

// UpdateMessageContent updates the content of a message at the given index.
func (m *Model) UpdateMessageContent(idx int, content string) {
	if idx >= 0 && idx < len(m.messages) {
		m.messages[idx].Content = content
	}
}

// IsStreaming returns true if currently receiving a response.
func (m Model) IsStreaming() bool {
	if len(m.messages) == 0 {
		return false
	}
	return m.messages[len(m.messages)-1].Streaming
}

// InputValue returns the current input text.
func (m Model) InputValue() string {
	return m.input.Value()
}

// ClearInput clears the input.
func (m *Model) ClearInput() {
	m.input.Reset()
}

// FocusInput focuses the input.
func (m *Model) FocusInput() {
	m.input.Focus()
}

// ShowAutocomplete shows the autocomplete menu with suggestions.
func (m *Model) ShowAutocomplete(prefix InputPrefix, partial string, suggestions []string) {
	m.autocomplete.Show(prefix, partial, suggestions)
}

// HideAutocomplete hides the autocomplete menu.
func (m *Model) HideAutocomplete() {
	m.autocomplete.Hide()
}

// IsAutocompleteVisible returns true if autocomplete is showing.
func (m Model) IsAutocompleteVisible() bool {
	return m.autocomplete.IsVisible()
}

// AutocompleteUp moves autocomplete selection up.
func (m *Model) AutocompleteUp() {
	m.autocomplete.MoveUp()
}

// AutocompleteDown moves autocomplete selection down.
func (m *Model) AutocompleteDown() {
	m.autocomplete.MoveDown()
}

// AutocompletePrefix returns the current autocomplete prefix type.
func (m Model) AutocompletePrefix() InputPrefix {
	return m.autocomplete.Prefix()
}

// CompleteInput completes the input with the selected suggestion.
func (m *Model) CompleteInput() {
	if !m.autocomplete.IsVisible() {
		return
	}

	selected := m.autocomplete.Selected()
	if selected == "" {
		return
	}

	// Get the prefix character
	var prefixChar string
	switch m.autocomplete.Prefix() {
	case PrefixAssistant:
		prefixChar = "@"
	case PrefixWorkflow:
		prefixChar = "#"
	case PrefixCommand:
		prefixChar = "/"
	}

	// Set input to prefix + selected
	m.input.SetValue(prefixChar + selected)
	m.autocomplete.Hide()
}

// GetInputPrefix returns the current input prefix and partial text.
func (m Model) GetInputPrefix() (InputPrefix, string) {
	return DetectPrefix(m.input.Value())
}

// Update handles input events.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			// Scroll up if input is empty or at beginning
			if m.input.IsEmpty() {
				m.scrollUp(1)
				return m, nil
			}
		case "down":
			// Scroll down
			if m.input.IsEmpty() {
				m.scrollDown(1)
				return m, nil
			}
		case "pgup":
			m.scrollUp(scrollPageSize)
			return m, nil
		case "pgdown":
			m.scrollDown(scrollPageSize)
			return m, nil
		case "home":
			// Scroll to top
			m.scrollPos = m.maxScroll()
			m.autoScroll = false
			return m, nil
		case "end":
			// Scroll to bottom
			m.scrollPos = 0
			m.autoScroll = true
			return m, nil
		}
	}

	// Update input
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) scrollUp(lines int) {
	m.scrollPos += lines
	maxScroll := m.maxScroll()
	if m.scrollPos > maxScroll {
		m.scrollPos = maxScroll
	}
	m.autoScroll = false
}

func (m *Model) scrollDown(lines int) {
	m.scrollPos -= lines
	if m.scrollPos < 0 {
		m.scrollPos = 0
	}
	if m.scrollPos == 0 {
		m.autoScroll = true
	}
}

func (m Model) maxScroll() int {
	totalLines := m.countMessageLines()
	visibleLines := m.messagesHeight()
	if totalLines <= visibleLines {
		return 0
	}
	return totalLines - visibleLines
}

func (m Model) countMessageLines() int {
	total := 0
	for _, msg := range m.messages {
		rendered := msg.View(m.width)
		total += strings.Count(rendered, "\n") + 1
		total++ // Add spacing between messages
	}
	return total
}

func (m Model) messagesHeight() int {
	// Total height minus input area (3 lines typically) minus status bar (1 line)
	inputHeight := strings.Count(m.input.View(), "\n") + 1
	return m.height - inputHeight - 1
}

// View renders the chat view.
func (m Model) View() string {
	return m.ViewWithHeight(m.height)
}

// ViewMessagesOnly renders just the messages area (no input).
func (m Model) ViewMessagesOnly(height int) string {
	return m.renderMessages(height)
}

// ViewInputOnly renders just the input area.
func (m Model) ViewInputOnly() string {
	inputView := m.input.View()
	if m.inContext {
		lineStyle := lipgloss.NewStyle().Foreground(theme.Accent)
		line := lineStyle.Render(strings.Repeat("─", m.width))
		inputView = "\n" + line + "\n" + inputView + "\n" + line
	}
	return inputView
}

// ViewWithHeight renders the chat view with a specific height.
func (m Model) ViewWithHeight(height int) string {
	// Calculate messages height based on provided height
	inputHeight := strings.Count(m.input.View(), "\n") + 1
	messagesHeight := height - inputHeight - 1

	// Account for autocomplete menu height
	autocompleteView := m.autocomplete.View()
	if autocompleteView != "" {
		autocompleteLines := strings.Count(autocompleteView, "\n") + 1
		messagesHeight -= autocompleteLines
	}

	// Render messages
	messagesView := m.renderMessages(messagesHeight)

	// Render input (with colored lines if in assistant context)
	inputView := m.input.View()
	if m.inContext {
		// Create horizontal line in accent color
		lineStyle := lipgloss.NewStyle().Foreground(theme.Accent)
		line := lineStyle.Render(strings.Repeat("─", m.width))
		inputView = "\n" + line + "\n" + inputView + "\n" + line
	}

	// Build the view
	if autocompleteView != "" {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			messagesView,
			autocompleteView,
			inputView,
		)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		messagesView,
		inputView,
	)
}

func (m Model) renderMessages(height int) string {
	if len(m.messages) == 0 {
		// Show placeholder when no messages
		placeholder := lipgloss.NewStyle().
			Foreground(theme.TextSecondary).
			Italic(true).
			Render("No messages yet. Type something to start chatting.")

		return lipgloss.Place(
			m.width,
			height,
			lipgloss.Center,
			lipgloss.Center,
			placeholder,
		)
	}

	// Render all messages
	var lines []string
	for i, msg := range m.messages {
		rendered := msg.View(m.width)
		lines = append(lines, rendered)
		if i < len(m.messages)-1 {
			lines = append(lines, "") // Spacing between messages
		}
	}

	allLines := strings.Join(lines, "\n")
	splitLines := strings.Split(allLines, "\n")

	// Apply scrolling
	totalLines := len(splitLines)
	if totalLines <= height {
		// No scrolling needed, pad to fill height
		for len(splitLines) < height {
			splitLines = append([]string{""}, splitLines...)
		}
	} else {
		// Apply scroll position
		start := totalLines - height - m.scrollPos
		if start < 0 {
			start = 0
		}
		end := start + height
		if end > totalLines {
			end = totalLines
		}
		splitLines = splitLines[start:end]
	}

	return strings.Join(splitLines, "\n")
}
