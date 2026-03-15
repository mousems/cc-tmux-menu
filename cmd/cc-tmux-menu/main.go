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
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Println("cc-tmux-menu " + version)
			os.Exit(0)
		case "--setup":
			config.RunSetup()
			os.Exit(0)
		case "--help", "-h":
			fmt.Println("cc-tmux-menu — interactive tmux session manager for Claude Code")
			fmt.Println()
			fmt.Println("Usage: cc-tmux-menu [flags]")
			fmt.Println()
			fmt.Println("Flags:")
			fmt.Println("  --help, -h       Show this help")
			fmt.Println("  --version, -v    Show version")
			fmt.Println("  --setup          Run interactive configuration wizard")
			fmt.Println()
			fmt.Println("Navigation:")
			fmt.Println("  ↑/↓, j/k         Move cursor")
			fmt.Println("  Enter             Select / confirm")
			fmt.Println("  ←/h, Esc          Back (in sub-menu)")
			fmt.Println("  q, Ctrl+C         Quit")
			fmt.Println()
			fmt.Println("Config: ~/.config/cc-tmux-menu/config.toml")
			fmt.Println("Repo:   https://github.com/mousems/cc-tmux-menu")
			os.Exit(0)
		}
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
