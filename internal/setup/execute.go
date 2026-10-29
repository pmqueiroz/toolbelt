package setup

import (
	"context"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/pmqueiroz/toolbelt/internal/auth"
	"github.com/pmqueiroz/toolbelt/internal/ui"
)

type Failure struct {
	Description string
	Err         error
}

func Execute(ctx context.Context, phases []Phase, verbose bool) []Failure {
	var failures []Failure
	for _, p := range phases {
		ui.Info(p.Title)
		for _, a := range p.Actions {
			if ctx.Err() != nil {
				return append(failures, Failure{a.Description, ctx.Err()})
			}
			var err error
			if verbose {
				ui.Info(a.Description)
				err = a.Run(ctx)
			} else {
				err = ui.Spin(a.Description, a.Run)
			}
			if err != nil {
				ui.Error(a.Description, "err", err)
				failures = append(failures, Failure{a.Description, err})
			}
		}
	}
	return failures
}

func Summary(ctx context.Context, a Answers, c Context, failures []Failure, logPath string) error {
	ui.Println("\n" + lipgloss.JoinVertical(lipgloss.Center,
		ui.Title.Render("DONE!"),
		ui.Faint.Render("do not forget to gimme a star on github"),
		ui.Faint.Render("https://github.com/pmqueiroz/toolbelt"),
	))

	if a.Has(StepSSH) {
		if key, err := auth.SSHPublicKey(c.Paths.Home); err == nil {
			showKey("copy this ssh key", key, auth.SSHSettingsURL)
		}
	}
	if a.Has(StepGPG) {
		if key, err := auth.GPGPublicKey(ctx, c.GOOS, c.Run, a.Email); err == nil {
			showKey("copy this gpg key", key, auth.GPGSettingsURL)
		}
	}

	if len(failures) == 0 {
		return nil
	}
	for _, f := range failures {
		ui.Error(f.Description, "err", f.Err)
	}
	return fmt.Errorf("%d action(s) failed, full log at %s", len(failures), logPath)
}

func showKey(title, key, url string) {
	ui.Println(lipgloss.JoinVertical(lipgloss.Center,
		lipgloss.NewStyle().Foreground(ui.Pink).Margin(1).Render(title),
		ui.KeyBox(key),
		lipgloss.JoinHorizontal(lipgloss.Center,
			lipgloss.NewStyle().Foreground(ui.Pink).Margin(1).Render("and paste on"),
			ui.Link.Render(url),
		),
	))
}
