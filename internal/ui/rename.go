package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mousems/cc-tmux-menu/internal/tmux"
)

func (m Model) updateRename(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = stateSubMenu
			return m, nil
		case "enter":
			newName := strings.TrimSpace(m.textInput.Value())
			if newName != "" {
				tmux.RenameSession(m.renameTarget, newName)
			}
			m.state = stateMainMenu
			return m, func() tea.Msg { return refreshMsg{} }
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) viewRename() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Rename session: "))
	b.WriteString(greenStyle.Render(m.renameTarget))
	b.WriteString("\n\n")
	b.WriteString("  ")
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")
	b.WriteString(hintStyle.Render("  Enter to confirm, Esc to cancel"))
	b.WriteString("\n")

	return b.String()
}
