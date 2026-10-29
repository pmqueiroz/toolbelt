package cli

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/pmqueiroz/toolbelt/internal/dots"
	"github.com/pmqueiroz/toolbelt/internal/ui"
	"github.com/spf13/cobra"
)

func newDotsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dots",
		Short: "Manage the dot files tb links into place",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "sync",
		Short: "Merge this version's dot files into yours, keeping local edits",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			paths, err := dots.DefaultPaths()
			if err != nil {
				return err
			}
			return syncDots(cmd.Context(), runtime.GOOS, paths, time.Now())
		},
	})
	return cmd
}

func syncDots(ctx context.Context, goos string, paths dots.Paths, now time.Time) error {
	vars, err := dots.LoadVars(paths)
	if errors.Is(err, dots.ErrNotManaged) {
		ui.Info("no dot files to sync, run tb setup to link them")
		return nil
	}
	if err != nil {
		return err
	}
	actions, err := dots.Plan(goos, paths, vars, now)
	if err != nil {
		return err
	}
	if len(actions) == 0 {
		ui.Info("dot files are up to date")
		return nil
	}
	failed := 0
	for _, a := range actions {
		if err := a.Run(ctx); err != nil {
			ui.Error(err.Error())
			failed++
			continue
		}
		ui.Info(a.Description)
	}
	if failed > 0 {
		return fmt.Errorf("%d dot file(s) need attention", failed)
	}
	return nil
}
