package cli

import (
	"context"
	"errors"
	"runtime"
	"time"

	"github.com/pmqueiroz/toolbelt/internal/dots"
	"github.com/pmqueiroz/toolbelt/internal/pkgmgr"
	"github.com/pmqueiroz/toolbelt/internal/recipes"
	"github.com/pmqueiroz/toolbelt/internal/run"
	"github.com/pmqueiroz/toolbelt/internal/setup"
	"github.com/pmqueiroz/toolbelt/internal/shellrc"
	"github.com/pmqueiroz/toolbelt/internal/ui"
	"github.com/spf13/cobra"
)

var errNoTerminal = errors.New("tb setup needs an interactive terminal")

func newSetupCmd() *cobra.Command {
	var verbose bool
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Set up this machine: shell, packages, dot files and github keys",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSetup(cmd.Context(), verbose)
		},
	}
	cmd.Flags().BoolVar(&verbose, "verbose", false, "show the output of every command")
	return cmd
}

func runSetup(ctx context.Context, verbose bool) error {
	if !ui.Interactive() {
		return errNoTerminal
	}
	ui.SetVerbose(verbose)

	logFile, err := run.OpenLogFile()
	if err != nil {
		return err
	}
	defer logFile.Close()
	r := run.New(verbose, logFile)

	paths, err := dots.DefaultPaths()
	if err != nil {
		return err
	}
	c := setup.Context{
		GOOS:  runtime.GOOS,
		Paths: paths,
		Shell: shellrc.DefaultEnv(runtime.GOOS, paths.Home),
		Run:   r,
		Now:   time.Now(),
	}
	c.Pkg, c.PkgErr = pkgmgr.Detect(c.GOOS, r)
	if c.PkgErr == nil {
		pkgmgr.PrependPath(c.Pkg)
	}

	ui.Println(ui.Banner("TOOLBELT", "starting setup", "made with <3 by peam"))

	answers, err := setup.Ask(gitIdentity(ctx, r), recipes.ForOS(c.GOOS))
	if errors.Is(err, setup.ErrAborted) {
		ui.Warn("setup aborted. nothing changed")
		return nil
	}
	if err != nil {
		return err
	}

	phases, err := setup.BuildPlan(answers, c)
	if err != nil {
		return err
	}
	if setup.Count(phases) == 0 {
		ui.Info("everything is already set up. nothing to do")
		return setup.Summary(ctx, answers, c, nil, logFile.Name())
	}

	ui.Println(setup.Render(phases))
	if !ui.Confirm("apply this plan?") {
		ui.Warn("plan not applied. nothing changed")
		return nil
	}

	failures := setup.Execute(ctx, phases, verbose)
	return setup.Summary(ctx, answers, c, failures, logFile.Name())
}

func gitIdentity(ctx context.Context, r *run.Runner) setup.Answers {
	name, _ := r.Output(ctx, "git", "config", "--global", "user.name")
	email, _ := r.Output(ctx, "git", "config", "--global", "user.email")
	return setup.Answers{Username: name, Email: email}
}
