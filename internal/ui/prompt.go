package ui

import (
	"context"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/mattn/go-isatty"
)

func Interactive() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
}

func Confirm(title string) bool {
	confirmed := false
	err := huh.NewConfirm().Title(title).Value(&confirmed).Run()
	return err == nil && confirmed
}

func Spin(title string, action func(context.Context) error) error {
	if !Interactive() {
		Info(title)
		return action(context.Background())
	}
	return spinner.New().
		Type(spinner.MiniDot).
		Title(" " + title).
		ActionWithErr(action).
		Run()
}
