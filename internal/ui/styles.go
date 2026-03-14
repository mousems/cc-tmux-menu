package ui

import "github.com/charmbracelet/lipgloss"

var (
	greenStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	yellowStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	redStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	cyanStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	titleStyle    = lipgloss.NewStyle().Bold(true)
	hintStyle     = lipgloss.NewStyle().Faint(true)
	dimStyle      = lipgloss.NewStyle().Faint(true)
	previewStyle  = lipgloss.NewStyle().Faint(true)
	warnStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
)
