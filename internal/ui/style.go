package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	Pink    = lipgloss.Color("212")
	Rose    = lipgloss.Color("211")
	Purple  = lipgloss.Color("#BD93F9")
	Magenta = lipgloss.Color("#FF79C6")
	Gold    = lipgloss.Color("222")
	Blue    = lipgloss.Color("12")
)

var (
	Title = lipgloss.NewStyle().Foreground(Pink).Bold(true)
	Faint = lipgloss.NewStyle().Foreground(Rose).Italic(true).Faint(true)
	Link  = lipgloss.NewStyle().Foreground(Gold)
	List  = lipgloss.NewStyle().Foreground(Blue).Margin(0, 1)

	keyBox = lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("10")).
		Margin(0, 1).
		Padding(1)

	branchName = lipgloss.NewStyle().Bold(true).Underline(true).Foreground(Purple)
	prHash     = lipgloss.NewStyle().Foreground(Magenta)
)

func KeyBox(key string) string {
	return keyBox.Render(key)
}

func Branch(name string) string {
	return "🌿 " + branchName.Render(name)
}

func PR(number string) string {
	return prHash.Render("#") + branchName.Render(number)
}

func Println(s string) {
	fmt.Println(s)
}
