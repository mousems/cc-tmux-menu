package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	bold   = "\033[1m"
	cyan   = "\033[36m"
	green  = "\033[32m"
	yellow = "\033[33m"
	dim    = "\033[2m"
	reset  = "\033[0m"
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

	// Shell integration
	fmt.Printf("%sAuto-launch on new terminal?%s %s(starts cc-tmux-menu when you open a shell)%s\n", bold, reset, dim, reset)
	fmt.Printf("  [Y/n] ")
	shellAnswer, _ := reader.ReadString('\n')
	shellAnswer = strings.TrimSpace(strings.ToLower(shellAnswer))

	if shellAnswer != "n" && shellAnswer != "no" {
		snippet := "\n# cc-tmux-menu: auto-launch (only outside tmux, interactive shell)\nif [[ -z \"$TMUX\" && $- == *i* ]] && command -v cc-tmux-menu &>/dev/null; then\n  cc-tmux-menu\nfi\n"

		shellRC := detectShellRC()
		if shellRC != "" {
			// Check if already added
			existing, _ := os.ReadFile(shellRC)
			if strings.Contains(string(existing), "cc-tmux-menu") {
				fmt.Printf("  %s!%s Already in %s%s%s\n", yellow, reset, cyan, shellRC, reset)
			} else {
				f, err := os.OpenFile(shellRC, os.O_APPEND|os.O_WRONLY, 0644)
				if err == nil {
					f.WriteString(snippet)
					f.Close()
					fmt.Printf("  %s%s✓ Added to %s%s%s\n", green, bold, cyan, shellRC, reset)
				} else {
					fmt.Printf("  Could not write to %s: %v\n", shellRC, err)
					printManualSnippet()
				}
			}
		} else {
			fmt.Println("  Shell profile not found. Add manually:")
			printManualSnippet()
		}
	}

	fmt.Println()
	return true
}

func printManualSnippet() {
	fmt.Printf("\n  %s# Add to your shell profile (.zshrc, .bashrc, etc.)%s\n", dim, reset)
	fmt.Printf("  %sif [[ -z \"$TMUX\" && $- == *i* ]] && command -v cc-tmux-menu &>/dev/null; then%s\n", cyan, reset)
	fmt.Printf("  %s  cc-tmux-menu%s\n", cyan, reset)
	fmt.Printf("  %sfi%s\n", cyan, reset)
}

func detectShellRC() string {
	home, _ := os.UserHomeDir()

	// Check current shell
	shell := os.Getenv("SHELL")
	switch {
	case strings.HasSuffix(shell, "/zsh"):
		return filepath.Join(home, ".zshrc")
	case strings.HasSuffix(shell, "/bash"):
		return filepath.Join(home, ".bashrc")
	case strings.HasSuffix(shell, "/fish"):
		return filepath.Join(home, ".config", "fish", "config.fish")
	}

	// Fallback: check which files exist
	for _, rc := range []string{".zshrc", ".bashrc"} {
		path := filepath.Join(home, rc)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
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
