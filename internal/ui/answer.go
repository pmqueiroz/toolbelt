package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	prompt      = lipgloss.NewStyle().Foreground(Pink).Bold(true)
	commandLine = lipgloss.NewStyle().Foreground(Gold).Bold(true)
	codeBox     = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(Purple).
			PaddingLeft(1)
	language = lipgloss.NewStyle().Foreground(Purple).Faint(true)
	bullet   = lipgloss.NewStyle().Foreground(Magenta)
	caution  = lipgloss.NewStyle().Foreground(Gold)
)

func Paragraph(text string, width int) string {
	return lipgloss.NewStyle().Width(width).Render(text)
}

func CommandLine(command, note string, width int) string {
	lines := strings.Split(command, "\n")
	for i, line := range lines {
		lines[i] = prompt.Render("$ ") + commandLine.Render(line)
	}
	out := strings.Join(lines, "\n")
	if note != "" {
		out += "\n" + Faint.Width(width).Render(note)
	}
	return out
}

func CodeBlock(lang, code string) string {
	body := code
	if lang != "" {
		body = language.Render(lang) + "\n" + code
	}
	return codeBox.Render(body)
}

func Bullets(items []string, width int) string {
	lines := make([]string, len(items))
	for i, item := range items {
		lines[i] = lipgloss.JoinHorizontal(lipgloss.Top, bullet.Render("• "), lipgloss.NewStyle().Width(max(width-2, 1)).Render(item))
	}
	return strings.Join(lines, "\n")
}

func Caution(text string, width int) string {
	return caution.Width(width).Render("⚠ " + text)
}
