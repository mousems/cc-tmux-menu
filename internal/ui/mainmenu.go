package ui

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mousems/cc-tmux-menu/internal/config"
	"github.com/mousems/cc-tmux-menu/internal/preview"
	"github.com/mousems/cc-tmux-menu/internal/remote"
	"github.com/mousems/cc-tmux-menu/internal/tmux"
)

type viewState int

const (
	stateMainMenu viewState = iota
	stateSubMenu
	stateRename
)

// Messages
type previewMsg struct {
	SessionName string
	Preview     *preview.Result
}

type sessionExitMsg struct{ err error }

type connModeMsg struct{ mode remote.ConnMode }

type refreshMsg struct{}

// execWrapper wraps *exec.Cmd to satisfy tea.ExecCommand interface.
type execWrapper struct {
	cmd *exec.Cmd
}

func (e *execWrapper) Run() error            { return e.cmd.Run() }
func (e *execWrapper) SetStdin(r io.Reader)  { e.cmd.Stdin = r }
func (e *execWrapper) SetStdout(w io.Writer) { e.cmd.Stdout = w }
func (e *execWrapper) SetStderr(w io.Writer) { e.cmd.Stderr = w }

// Model is the root TUI model.
type Model struct {
	cfg   *config.Config
	state viewState

	// Main menu
	sessions []tmux.Session
	cursor   int
	fixedTop int // number of fixed items before session list
	previews map[string]*preview.Result
	loading  map[string]bool
	connMode remote.ConnMode

	// Terminal size & scrolling
	termHeight int
	scrollOff  int // first visible session index in scrollable area

	// Sub menu
	subTarget string
	subCursor int

	// Rename
	textInput    textinput.Model
	renameTarget string

	quitting bool
}

// NewModel creates the initial model.
func NewModel(cfg *config.Config) Model {
	ti := textinput.New()
	ti.Placeholder = "new name"
	ti.CharLimit = 50

	return Model{
		cfg:       cfg,
		state:     stateMainMenu,
		previews:  make(map[string]*preview.Result),
		loading:   make(map[string]bool),
		textInput: ti,
	}
}

func (m Model) Init() tea.Cmd {
	return func() tea.Msg { return refreshMsg{} }
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateMainMenu:
		return m.updateMainMenu(msg)
	case stateSubMenu:
		return m.updateSubMenu(msg)
	case stateRename:
		return m.updateRename(msg)
	}
	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	switch m.state {
	case stateMainMenu:
		return m.viewMainMenu()
	case stateSubMenu:
		return m.viewSubMenu()
	case stateRename:
		return m.viewRename()
	}
	return ""
}

// === Main Menu ===

func (m Model) totalMenuItems() int {
	return m.fixedTop + len(m.sessions) + 2 // +2 for Exit and Disconnect
}

func (m Model) updateMainMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case refreshMsg:
		sessions, _ := tmux.ListSessions()
		m.sessions = sessions
		m.cursor = 0
		m.previews = make(map[string]*preview.Result)
		m.loading = make(map[string]bool)
		m.connMode = remote.ConnChecking

		m.fixedTop = 3
		if m.cfg.RemoteHost != "" {
			m.fixedTop = 4
		}

		m.scrollOff = 0

		var cmds []tea.Cmd
		if m.cfg.RemoteHost != "" {
			cmds = append(cmds, m.checkConnectivity())
		}
		for _, s := range sessions {
			sname := s.Name
			m.loading[sname] = true
			cmds = append(cmds, m.loadPreview(sname))
		}
		return m, tea.Batch(cmds...)

	case connModeMsg:
		m.connMode = msg.mode
		return m, nil

	case previewMsg:
		m.previews[msg.SessionName] = msg.Preview
		delete(m.loading, msg.SessionName)
		return m, nil

	case sessionExitMsg:
		return m, func() tea.Msg { return refreshMsg{} }

	case tea.WindowSizeMsg:
		m.termHeight = msg.Height
		return m, nil

	case tea.KeyMsg:
		total := m.totalMenuItems()
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = total - 1
			}
		case "down", "j":
			m.cursor++
			if m.cursor >= total {
				m.cursor = 0
			}
		case "enter":
			return m.handleMainMenuSelect()
		}
		m.adjustScroll()
	}
	return m, nil
}

func (m Model) handleMainMenuSelect() (tea.Model, tea.Cmd) {
	total := m.totalMenuItems()
	nsessions := len(m.sessions)

	exitIdx := total - 2
	disconnIdx := total - 1

	remoteIdx := -1
	var newIdx, resumeIdx, shellIdx int
	if m.cfg.RemoteHost != "" {
		remoteIdx = 0
		newIdx = 1
		resumeIdx = 2
		shellIdx = 3
	} else {
		newIdx = 0
		resumeIdx = 1
		shellIdx = 2
	}

	switch {
	case m.cursor == exitIdx:
		m.quitting = true
		return m, tea.Quit

	case m.cursor == disconnIdx:
		m.quitting = true
		ppid := os.Getppid()
		if p, err := os.FindProcess(ppid); err == nil {
			p.Signal(syscall.SIGHUP)
		}
		return m, tea.Quit

	case m.cursor == remoteIdx && m.cfg.RemoteHost != "":
		if m.connMode == remote.ConnUnreachable || m.connMode == remote.ConnChecking {
			return m, nil // block if unreachable or still checking
		}
		var c *exec.Cmd
		if m.connMode == remote.ConnTunnel && m.cfg.RemoteGatewayWAN != "" {
			// DNAT: ssh -p wanPort user@gatewayWAN
			target := m.cfg.RemoteGatewayWAN
			if m.cfg.RemoteUser != "" {
				target = m.cfg.RemoteUser + "@" + target
			}
			port := "22"
			if m.cfg.RemoteGatewayWANPort > 0 {
				port = fmt.Sprintf("%d", m.cfg.RemoteGatewayWANPort)
			}
			c = exec.Command("ssh", "-p", port, target)
		} else {
			// Direct: ssh user@host
			target := m.cfg.RemoteHost
			if m.cfg.RemoteUser != "" {
				target = m.cfg.RemoteUser + "@" + target
			}
			c = exec.Command("ssh", target)
		}
		return m, tea.Exec(&execWrapper{c}, func(err error) tea.Msg {
			return sessionExitMsg{err}
		})

	case m.cursor == newIdx:
		name := tmux.RandomName()
		c := tmux.NewSessionCmd(name, m.cfg.WorkDir, m.cfg.ClaudeCmd)
		return m, tea.Exec(&execWrapper{c}, func(err error) tea.Msg {
			return sessionExitMsg{err}
		})

	case m.cursor == resumeIdx:
		name := tmux.RandomName()
		c := tmux.NewSessionCmd(name, m.cfg.WorkDir, m.cfg.ClaudeCmd, "-r")
		return m, tea.Exec(&execWrapper{c}, func(err error) tea.Msg {
			return sessionExitMsg{err}
		})

	case m.cursor == shellIdx:
		name := tmux.RandomName()
		c := tmux.NewSessionCmd(name, m.cfg.WorkDir)
		return m, tea.Exec(&execWrapper{c}, func(err error) tea.Msg {
			return sessionExitMsg{err}
		})

	default:
		sessionIdx := m.cursor - m.fixedTop
		if sessionIdx >= 0 && sessionIdx < nsessions {
			// Block selection if preview still loading
			sname := m.sessions[sessionIdx].Name
			if m.loading[sname] {
				return m, nil
			}
			m.state = stateSubMenu
			m.subTarget = sname
			m.subCursor = 0
		}
	}

	return m, nil
}

func (m Model) checkConnectivity() tea.Cmd {
	cfg := m.cfg
	return func() tea.Msg {
		mode := remote.DetectConnMode(cfg.RemoteHost, cfg.RemoteGatewayLAN, cfg.RemoteGatewayWAN, cfg.RemoteGatewayWANPort)
		return connModeMsg{mode: mode}
	}
}

func (m Model) loadPreview(sessionName string) tea.Cmd {
	cfg := m.cfg
	return func() tea.Msg {
		pcfg := &preview.Config{
			APIKey:   cfg.OpenRouterAPIKey,
			Model:    cfg.OpenRouterModel,
			Lang:     cfg.SummaryLang,
			CacheTTL: cfg.PreviewCacheTTL,
		}
		p := preview.CaptureAndSummarize(sessionName, pcfg)
		return previewMsg{SessionName: sessionName, Preview: p}
	}
}

// visibleSessionCount returns how many sessions fit in the scrollable area.
func (m Model) visibleSessionCount() int {
	if m.termHeight <= 0 {
		return len(m.sessions) // no height info yet, show all
	}
	// Lines used by non-session parts:
	//   title + blank line = 2
	//   fixed top items = m.fixedTop
	//   bottom items (Exit + Disconnect) = 2
	//   preview/warnings + trailing blank = 2
	//   scroll indicator (if needed) = 1
	overhead := 2 + m.fixedTop + 2 + 2 + 1
	avail := m.termHeight - overhead
	if avail < 1 {
		avail = 1
	}
	if avail >= len(m.sessions) {
		return len(m.sessions)
	}
	return avail
}

// adjustScroll keeps the cursor visible in the scrollable session area.
func (m *Model) adjustScroll() {
	nsessions := len(m.sessions)
	if nsessions == 0 {
		m.scrollOff = 0
		return
	}
	visible := m.visibleSessionCount()
	sessionIdx := m.cursor - m.fixedTop
	// cursor is not in session area
	if sessionIdx < 0 || sessionIdx >= nsessions {
		return
	}
	if sessionIdx < m.scrollOff {
		m.scrollOff = sessionIdx
	}
	if sessionIdx >= m.scrollOff+visible {
		m.scrollOff = sessionIdx - visible + 1
	}
	// clamp
	if m.scrollOff < 0 {
		m.scrollOff = 0
	}
	if m.scrollOff > nsessions-visible {
		m.scrollOff = nsessions - visible
	}
}

func (m Model) viewMainMenu() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("tmux sessions"))
	b.WriteString("  ")
	b.WriteString(hintStyle.Render("(↑↓/jk select, Enter confirm, q quit)"))
	b.WriteString("\n\n")

	idx := 0

	// Remote SSH
	if m.cfg.RemoteHost != "" {
		label := m.cfg.RemoteHost
		if m.cfg.RemoteUser != "" {
			label = m.cfg.RemoteUser + "@" + label
		}
		var status string
		switch m.connMode {
		case remote.ConnDirect:
			status = greenStyle.Render("● direct")
		case remote.ConnTunnel:
			status = yellowStyle.Render("● tunnel")
		case remote.ConnUnreachable:
			status = redStyle.Render("● offline")
		default:
			status = dimStyle.Render("● checking...")
		}
		item := cyanStyle.Render("Remote SSH") + "    " + dimStyle.Render(label) + " " + status
		b.WriteString(m.renderItem(idx, item))
		idx++
	}

	// New Claude
	item := yellowStyle.Render("New Claude") + "    " + dimStyle.Render(fmt.Sprintf("new session (%s)", m.cfg.ClaudeCmd))
	b.WriteString(m.renderItem(idx, item))
	idx++

	// Resume Claude
	item = yellowStyle.Render("Resume Claude") + " " + dimStyle.Render(fmt.Sprintf("continue last (%s -r)", m.cfg.ClaudeCmd))
	b.WriteString(m.renderItem(idx, item))
	idx++

	// Shell
	item = yellowStyle.Render("Shell") + "         " + dimStyle.Render("plain shell")
	b.WriteString(m.renderItem(idx, item))
	idx++

	// Sessions (scrollable)
	nsessions := len(m.sessions)
	visible := m.visibleSessionCount()
	showScrollUp := m.scrollOff > 0
	showScrollDown := m.scrollOff+visible < nsessions

	if showScrollUp {
		b.WriteString("    " + dimStyle.Render(fmt.Sprintf("  ▲ %d more above", m.scrollOff)) + "\n")
	}

	for i := m.scrollOff; i < m.scrollOff+visible && i < nsessions; i++ {
		s := m.sessions[i]
		attStr := ""
		if s.Attached {
			attStr = " " + dimStyle.Render("(attached)")
		}
		item = greenStyle.Render(fmt.Sprintf("%-12s", s.Name)) + " " + dimStyle.Render(s.Age) + attStr
		b.WriteString(m.renderItem(m.fixedTop+i, item))
	}

	if showScrollDown {
		remaining := nsessions - m.scrollOff - visible
		b.WriteString("    " + dimStyle.Render(fmt.Sprintf("  ▼ %d more below", remaining)) + "\n")
	}

	// Exit
	exitIdx := m.fixedTop + nsessions
	item = dimStyle.Render("Exit           stay in shell")
	b.WriteString(m.renderItem(exitIdx, item))

	// Disconnect
	disconnIdx := exitIdx + 1
	item = dimStyle.Render("Disconnect     exit SSH")
	b.WriteString(m.renderItem(disconnIdx, item))

	// Preview / warnings line
	b.WriteString("\n")
	sessionIdx := m.cursor - m.fixedTop
	if sessionIdx >= 0 && sessionIdx < nsessions {
		sname := m.sessions[sessionIdx].Name
		if p, ok := m.previews[sname]; ok {
			summary := p.Summary
			if len(summary) > 70 {
				summary = summary[:70] + "…"
			}
			b.WriteString(previewStyle.Render("Preview: " + summary))
		} else {
			b.WriteString(previewStyle.Render("Preview: ⏳ loading..."))
		}
	} else if warnings := m.cfg.Warnings(); len(warnings) > 0 {
		for _, w := range warnings {
			b.WriteString(warnStyle.Render("⚠ " + w))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	return b.String()
}

func (m Model) renderItem(idx int, label string) string {
	if idx == m.cursor {
		return "  " + selectedStyle.Render("▸") + " " + label + "\n"
	}
	return "    " + label + "\n"
}
