package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	barFilled = lipgloss.NewStyle().Foreground(Pink)
	barEmpty  = lipgloss.NewStyle().Foreground(Rose).Faint(true)
	panel     = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Purple).
			Padding(0, 1)
)

func Bar(width int, fraction float64) string {
	filled := int(min(max(fraction, 0), 1)*float64(width) + 0.5)
	return barFilled.Render(strings.Repeat("█", filled)) +
		barEmpty.Render(strings.Repeat("░", width-filled))
}

func Panel(title, body string) string {
	return panel.Render(Title.Render(title) + "\n\n" + body)
}
