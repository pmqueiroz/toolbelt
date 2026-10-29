package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/pmqueiroz/toolbelt/internal/ui"
	"github.com/spf13/cobra"
)

const releaseRepo = "pmqueiroz/toolbelt"

var errDevBuild = errors.New("this is a dev build, install a release with the one-liner from the readme to enable updates")

func newUpdateCmd(build Build) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update tb to the latest release",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return update(cmd.Context(), build.Version)
		},
	}
}

func update(ctx context.Context, current string) error {
	if current == "dev" {
		return errDevBuild
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	})
	if err != nil {
		return err
	}

	var latest *selfupdate.Release
	var found bool
	err = ui.Spin("checking latest release", func(ctx context.Context) error {
		latest, found, err = updater.DetectLatest(ctx, selfupdate.ParseSlug(releaseRepo))
		return err
	})
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no release found for this os/arch in %s", releaseRepo)
	}
	if latest.LessOrEqual(current) {
		ui.Info("tb is up to date", "version", current)
		return nil
	}

	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return err
	}
	if err := ui.Spin("updating to "+latest.Version(), func(ctx context.Context) error {
		return updater.UpdateTo(ctx, latest, exe)
	}); err != nil {
		return err
	}
	ui.Info("updated tb", "from", current, "to", latest.Version())
	return nil
}
