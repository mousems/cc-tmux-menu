package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	bold  = "\033[1m"
	cyan  = "\033[36m"
	green = "\033[32m"
	dim   = "\033[2m"
	reset = "\033[0m"
)

// RunSetup runs an interactive first-time configuration wizard.
// Returns true if config was created, false if user skipped.
func RunSetup() bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\n%sWelcome to cc-tmux-menu!%s\n", bold, reset)
	fmt.Printf("%sNo config file found. Set up now? [Y/n]%s ", dim, reset)

	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer == "n" || answer == "no" {
		fmt.Printf("%sSkipped. You can run %scc-tmux-menu --setup%s%s later.%s\n\n", dim, cyan, reset, dim, reset)
		return false
	}

	fmt.Println()

	// API key
	fmt.Printf("%sOpenRouter API Key%s %s(for AI session summaries, optional)%s\n", bold, reset, dim, reset)
	fmt.Printf("  Get one at: %shttps://openrouter.ai/keys%s\n", cyan, reset)
	fmt.Printf("  Press Enter to skip: ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)
	fmt.Println()

	// Work directory
	fmt.Printf("%sWork directory%s %s(for new Claude Code sessions)%s\n", bold, reset, dim, reset)
	fmt.Printf("  Press Enter for home directory: ")
	workDir, _ := reader.ReadString('\n')
	workDir = strings.TrimSpace(workDir)
	if workDir == "" {
		workDir = "~"
	}
	fmt.Println()

	// Language
	fmt.Printf("%sSummary language%s %s(en, zh-TW, ja, ko, ...)%s\n", bold, reset, dim, reset)
	fmt.Printf("  Press Enter for en: ")
	lang, _ := reader.ReadString('\n')
	lang = strings.TrimSpace(lang)
	if lang == "" {
		lang = "en"
	}
	fmt.Println()

	// Write config
	configDir := DefaultConfigDir()
	configPath := filepath.Join(configDir, "config.toml")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating config dir: %v\n", err)
		return false
	}

	var b strings.Builder
	b.WriteString("# cc-tmux-menu configuration\n\n")
	b.WriteString(fmt.Sprintf("openrouter_api_key = %q\n", apiKey))
	b.WriteString("openrouter_model = \"google/gemini-2.0-flash-001\"\n")
	b.WriteString(fmt.Sprintf("work_dir = %q\n", workDir))
	b.WriteString(fmt.Sprintf("summary_lang = %q\n", lang))
	b.WriteString("claude_cmd = \"claude\"\n")
	b.WriteString("preview_cache_ttl = 1800\n")

	if err := os.WriteFile(configPath, []byte(b.String()), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing config: %v\n", err)
		return false
	}

	fmt.Printf("%s%s✓ Config saved to %s%s%s\n\n", green, bold, cyan, configPath, reset)
	return true
}

// DefaultConfigDir returns the default config directory path.
func DefaultConfigDir() string {
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		home, _ := os.UserHomeDir()
		xdgConfig = filepath.Join(home, ".config")
	}
	return filepath.Join(xdgConfig, "cc-tmux-menu")
}
