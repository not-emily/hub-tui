package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pxp/hub-tui/internal/app"
	"github.com/pxp/hub-tui/internal/config"
)

func main() {
	// Load config (creates empty config if file doesn't exist)
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Save config to ensure the config file exists
	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	// Create the app model
	model := app.New(cfg)

	// Create the program
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Set program reference for streaming (via a startup command)
	go func() {
		// Small delay to ensure program is running
		p.Send(app.SetProgramMsg{Program: p})
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running app: %v\n", err)
		os.Exit(1)
	}
}
