package config

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config holds all cc-tmux-menu configuration.
type Config struct {
	OpenRouterAPIKey string `toml:"openrouter_api_key"`
	OpenRouterModel  string `toml:"openrouter_model"`
	WorkDir          string `toml:"work_dir"`
	SummaryLang      string `toml:"summary_lang"`
	ClaudeCmd        string `toml:"claude_cmd"`
	RemoteHost       string `toml:"remote_host"`
	RemoteUser       string `toml:"remote_user"`
	RemoteGatewayLAN    string `toml:"remote_gateway_lan"`
	RemoteGatewayWAN    string `toml:"remote_gateway_wan"`
	RemoteGatewayWANPort int   `toml:"remote_gateway_wan_port"`
	PreviewCacheTTL     int    `toml:"preview_cache_ttl"`

	// LoadedFrom is the config file path that was loaded, empty if none found.
	LoadedFrom string `toml:"-"`
}

func defaults() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		OpenRouterModel: "google/gemini-2.0-flash-001",
		WorkDir:         home,
		SummaryLang:     "en",
		ClaudeCmd:       "claude",
		PreviewCacheTTL: 1800,
	}
}

// Load searches for config in standard locations and returns merged config.
// Priority: script dir config.toml → script dir config → XDG config.toml → XDG config
func Load() *Config {
	cfg := defaults()

	exe, _ := os.Executable()
	scriptDir := filepath.Dir(exe)
	cwd, _ := os.Getwd()

	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		home, _ := os.UserHomeDir()
		xdgConfig = filepath.Join(home, ".config")
	}

	type candidate struct {
		path   string
		format string
	}
	candidates := []candidate{
		{filepath.Join(scriptDir, "config.toml"), "toml"},
		{filepath.Join(scriptDir, "config"), "shell"},
		{filepath.Join(cwd, "config.toml"), "toml"},
		{filepath.Join(cwd, "config"), "shell"},
		{filepath.Join(xdgConfig, "cc-tmux-menu", "config.toml"), "toml"},
		{filepath.Join(xdgConfig, "cc-tmux-menu", "config"), "shell"},
	}

	for _, c := range candidates {
		if _, err := os.Stat(c.path); err != nil {
			continue
		}
		var err error
		switch c.format {
		case "toml":
			err = loadTOML(c.path, cfg)
		case "shell":
			err = loadShell(c.path, cfg)
		}
		if err == nil {
			cfg.LoadedFrom = c.path
			break
		}
	}

	cfg.WorkDir = expandHome(cfg.WorkDir)
	return cfg
}

// Warnings returns a list of config issues the user should know about.
func (c *Config) Warnings() []string {
	var w []string
	if c.LoadedFrom == "" {
		w = append(w, "No config file found. Create ~/.config/cc-tmux-menu/config.toml")
	}
	if c.OpenRouterAPIKey == "" {
		w = append(w, "openrouter_api_key not set — AI preview disabled")
	}
	return w
}

func loadTOML(path string, cfg *Config) error {
	_, err := toml.DecodeFile(path, cfg)
	return err
}

var shellLineRe = regexp.MustCompile(`^(\w+)=["']?(.*?)["']?\s*$`)

func loadShell(path string, cfg *Config) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m := shellLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		key, val := m[1], m[2]
		switch key {
		case "OPENROUTER_API_KEY":
			cfg.OpenRouterAPIKey = val
		case "OPENROUTER_MODEL":
			cfg.OpenRouterModel = val
		case "WORK_DIR":
			cfg.WorkDir = val
		case "SUMMARY_LANG":
			cfg.SummaryLang = val
		case "CLAUDE_CMD":
			cfg.ClaudeCmd = val
		case "REMOTE_HOST":
			cfg.RemoteHost = val
		case "REMOTE_USER":
			cfg.RemoteUser = val
		case "REMOTE_GATEWAY_LAN":
			cfg.RemoteGatewayLAN = val
		case "REMOTE_GATEWAY_WAN":
			cfg.RemoteGatewayWAN = val
		case "REMOTE_GATEWAY_WAN_PORT":
			if v, err := strconv.Atoi(val); err == nil {
				cfg.RemoteGatewayWANPort = v
			}
		case "PREVIEW_CACHE_TTL":
			if v, err := strconv.Atoi(val); err == nil {
				cfg.PreviewCacheTTL = v
			}
		}
	}
	return scanner.Err()
}

func expandHome(path string) string {
	home, _ := os.UserHomeDir()
	switch {
	case path == "~":
		return home
	case strings.HasPrefix(path, "~/"):
		return filepath.Join(home, path[2:])
	case strings.HasPrefix(path, "$HOME"):
		return strings.Replace(path, "$HOME", home, 1)
	default:
		return path
	}
}
