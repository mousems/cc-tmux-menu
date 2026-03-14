package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mousems/cc-tmux-menu/internal/tmux"
)

func (m Model) updateSubMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "h", "left", "esc":
			m.state = stateMainMenu
			return m, nil
		case "up", "k":
			m.subCursor--
			if m.subCursor < 0 {
				m.subCursor = 2
			}
		case "down", "j":
			m.subCursor++
			if m.subCursor > 2 {
				m.subCursor = 0
			}
		case "enter":
			return m.handleSubMenuSelect()
		}
	case sessionExitMsg:
		m.state = stateMainMenu
		return m, func() tea.Msg { return refreshMsg{} }
	}
	return m, nil
}

func (m Model) handleSubMenuSelect() (tea.Model, tea.Cmd) {
	switch m.subCursor {
	case 0: // Attach
		c := tmux.AttachSessionCmd(m.subTarget)
		return m, tea.Exec(&execWrapper{c}, func(err error) tea.Msg {
			return sessionExitMsg{err}
		})
	case 1: // Rename
		m.state = stateRename
		m.renameTarget = m.subTarget
		m.textInput.Reset()
		if p, ok := m.previews[m.subTarget]; ok && p.Name != "" {
			m.textInput.SetValue(p.Name)
		}
		m.textInput.Focus()
		return m, textinput.Blink
	case 2: // Kill
		tmux.KillSession(m.subTarget)
		m.state = stateMainMenu
		return m, func() tea.Msg { return refreshMsg{} }
	}
	return m, nil
}

func (m Model) viewSubMenu() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Session: "))
	b.WriteString(greenStyle.Render(m.subTarget))
	b.WriteString("  ")
	b.WriteString(hintStyle.Render("(←/h back, ↑↓/jk select, Enter confirm)"))
	b.WriteString("\n\n")

	items := []string{
		greenStyle.Render("Attach") + "       " + dimStyle.Render("connect to "+m.subTarget),
		yellowStyle.Render("Rename") + "       " + dimStyle.Render("rename session"),
		redStyle.Render("Kill") + "         " + dimStyle.Render("kill session"),
	}

	for i, item := range items {
		if i == m.subCursor {
			b.WriteString("  " + selectedStyle.Render("▸") + " " + item + "\n")
		} else {
			b.WriteString("    " + item + "\n")
		}
	}

	if p, ok := m.previews[m.subTarget]; ok && p.Summary != "" {
		b.WriteString("\n")
		b.WriteString(previewStyle.Render("Preview: " + p.Summary))
		b.WriteString("\n")
	}

	return b.String()
}
