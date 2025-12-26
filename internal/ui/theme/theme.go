package theme

import "github.com/charmbracelet/lipgloss"

// Colors - dark theme using grays (not pure black)
var (
	Background    = lipgloss.Color("#1a1a1a") // Dark gray
	Surface       = lipgloss.Color("#2a2a2a") // Slightly lighter
	TextPrimary   = lipgloss.Color("#e0e0e0") // Light gray text
	TextSecondary = lipgloss.Color("#888888") // Muted text
	Accent        = lipgloss.Color("#7c9fc7") // Soft blue accent
	Error         = lipgloss.Color("#d46a6a") // Soft red
	Success       = lipgloss.Color("#6ad47c") // Soft green
	Warning       = lipgloss.Color("#d4a96a") // Soft orange
)

// Base styles
var (
	BaseStyle = lipgloss.NewStyle().
			Background(Background).
			Foreground(TextPrimary)

	TitleStyle = lipgloss.NewStyle().
			Foreground(Accent).
			Bold(true)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(TextSecondary)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(Error)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(Success)

	WarningStyle = lipgloss.NewStyle().
			Foreground(Warning)

	HintStyle = lipgloss.NewStyle().
			Foreground(TextSecondary).
			Italic(true)
)
