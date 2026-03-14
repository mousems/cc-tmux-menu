package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mousems/cc-tmux-menu/internal/config"
	"github.com/mousems/cc-tmux-menu/internal/ui"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println("cc-tmux-menu " + version)
		os.Exit(0)
	}

	if len(os.Args) > 1 && os.Args[1] == "--setup" {
		config.RunSetup()
		os.Exit(0)
	}

	// If already inside tmux, exit silently
	if os.Getenv("TMUX") != "" {
		os.Exit(0)
	}

	cfg := config.Load()

	// First-time setup: no config file found
	if cfg.LoadedFrom == "" {
		if config.RunSetup() {
			cfg = config.Load()
		}
	}

	m := ui.NewModel(cfg)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
