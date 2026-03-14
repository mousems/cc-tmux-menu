package preview

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mousems/cc-tmux-menu/internal/tmux"
)

// CacheDir is where preview summaries are cached.
var CacheDir = filepath.Join(os.TempDir(), "cc-tmux-menu-preview-cache")

// Result holds a preview summary and suggested name.
type Result struct {
	Summary string
	Name    string
}

// Config holds settings needed for preview generation.
type Config struct {
	APIKey   string
	Model    string
	Lang     string
	CacheTTL int
}

var (
	ansiRe   = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)
	oscRe    = regexp.MustCompile(`\x1b\][^\x07]*\x07`)
	hlineRe  = regexp.MustCompile(`^[\s─━═╌╍┄┅┈┉]*$`)
	blockRe  = regexp.MustCompile(`[▀▁▂▃▄▅▆▇█▉▊▋▌▍▎▏▐░▒▓▔▕▖▗▘▙▚▛▜▝▞▟]`)
)

// isHLine checks if a string is purely whitespace/line-drawing characters.
func isHLine(s string) bool {
	cleaned := strings.Map(func(r rune) rune {
		switch {
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			return -1
		case r == '─' || r == '━' || r == '═' || r == '╌' || r == '╍' || r == '┄' || r == '┅' || r == '┈' || r == '┉':
			return -1
		default:
			return r
		}
	}, s)
	return cleaned == ""
}

// filterContent strips ANSI codes and Claude Code UI elements.
func filterContent(raw string) string {
	raw = ansiRe.ReplaceAllString(raw, "")
	raw = oscRe.ReplaceAllString(raw, "")

	var lines []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if hlineRe.MatchString(line) {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if strings.Contains(line, "accept edits on") ||
			strings.Contains(line, "⏵") ||
			(strings.Contains(line, "ctx: ") && strings.Contains(line, "%")) ||
			strings.Contains(line, "✻") ||
			strings.Contains(line, "Claude Code v") ||
			blockRe.MatchString(line) ||
			trimmed == "❯" {
			continue
		}
		lines = append(lines, line)
	}

	// Remove last 2 lines (often prompts)
	if len(lines) > 2 {
		lines = lines[:len(lines)-2]
	}

	return strings.Join(lines, "\n")
}

// LoadFromCache tries to load a cached preview. Returns nil on cache miss.
func LoadFromCache(sessionName string, cacheTTL int) *Result {
	summaryPath := filepath.Join(CacheDir, sessionName+".summary")
	namePath := filepath.Join(CacheDir, sessionName+".name")

	info, err := os.Stat(summaryPath)
	if err != nil || info.Size() == 0 {
		return nil
	}

	if time.Since(info.ModTime()) > time.Duration(cacheTTL)*time.Second {
		return nil
	}

	summary, err := os.ReadFile(summaryPath)
	if err != nil {
		return nil
	}

	summaryStr := strings.TrimSpace(string(summary))
	if isHLine(summaryStr) {
		return nil
	}

	var nameStr string
	if nameData, err := os.ReadFile(namePath); err == nil {
		nameStr = strings.TrimSpace(string(nameData))
	}

	return &Result{Summary: summaryStr, Name: nameStr}
}

func saveCache(sessionName string, p *Result) {
	os.MkdirAll(CacheDir, 0755)
	os.WriteFile(filepath.Join(CacheDir, sessionName+".summary"), []byte(p.Summary), 0644)
	os.WriteFile(filepath.Join(CacheDir, sessionName+".name"), []byte(p.Name), 0644)
}

// CaptureAndSummarize captures a session's pane content and generates an AI summary.
func CaptureAndSummarize(sessionName string, cfg *Config) *Result {
	if p := LoadFromCache(sessionName, cfg.CacheTTL); p != nil {
		return p
	}

	raw, err := tmux.CapturePaneLast150(sessionName)
	if err != nil || strings.TrimSpace(raw) == "" {
		p := &Result{Summary: "(no output)", Name: ""}
		saveCache(sessionName, p)
		return p
	}

	filtered := filterContent(raw)
	if filtered == "" {
		p := &Result{Summary: "(no output)", Name: ""}
		saveCache(sessionName, p)
		return p
	}

	result := aiSummarize(filtered, cfg)
	if result != "" {
		lines := strings.SplitN(result, "\n", 2)
		summary := strings.TrimSpace(lines[0])
		var name string
		if len(lines) > 1 {
			name = strings.TrimSpace(lines[1])
		}
		if !isHLine(summary) {
			p := &Result{Summary: summary, Name: name}
			saveCache(sessionName, p)
			return p
		}
	}

	// Fallback: last meaningful line
	fallback := lastMeaningfulLine(filtered)
	if fallback == "" {
		fallback = "(session active)"
	}
	p := &Result{Summary: fallback, Name: ""}
	saveCache(sessionName, p)
	return p
}

func lastMeaningfulLine(content string) string {
	symbolOnlyRe := regexp.MustCompile(`^[\s─━═╌╍┄┅┈┉●✻▀▁▂▃▄▅▆▇█▉▊▋▌▍▎▏▐░▒▓▔▕▖▗▘▙▚▛▜▝▞▟]*$`)
	lines := strings.Split(content, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" && !symbolOnlyRe.MatchString(line) {
			return line
		}
	}
	return ""
}

type openRouterRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openRouterResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func aiSummarize(content string, cfg *Config) string {
	if cfg.APIKey == "" {
		return ""
	}

	if len(content) > 2000 {
		content = content[:2000]
	}

	prompt := fmt.Sprintf(`Analyze this terminal session output. Return exactly 2 lines:
Line 1: Brief summary of what this session is doing (max 30 chars, in %s)
Line 2: A descriptive session name for tmux (in %s, max 20 chars, no spaces — use CamelCase or hyphens)

The session name should follow "Action+Target" pattern. Good examples:
  API-Refactor, CI-Pipeline-Fix, DB-Migration, Auth-Debug, Docs-Update, Test-Coverage, Deploy-Staging
Bad examples (too vague):
  GitHub, Claude, 更新, 程式碼, 查看

Plain text only, no markdown, no prefixes, no quotes.

%s`, cfg.Lang, cfg.Lang, content)

	reqBody := openRouterRequest{
		Model:       cfg.Model,
		Messages:    []chatMessage{{Role: "user", Content: prompt}},
		MaxTokens:   100,
		Temperature: 0.3,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return ""
	}

	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var result openRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}

	if len(result.Choices) > 0 {
		return strings.TrimSpace(result.Choices[0].Message.Content)
	}
	return ""
}
