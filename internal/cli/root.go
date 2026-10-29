package cli

import (
	"context"
	"os"
	"os/signal"

	"github.com/pmqueiroz/toolbelt/internal/run"
	"github.com/pmqueiroz/toolbelt/internal/ui"
	"github.com/spf13/cobra"
)

type Build struct {
	Version string
	Commit  string
	Date    string
}

func Execute(build Build) {
	root := &cobra.Command{
		Use:           "tb",
		Short:         "toolbelt: machine setup and everyday git helpers",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		newVersionCmd(build),
		newPRCmd(),
		newCommitCmd(),
		newCheckoutCmd(),
		newHotCmd(),
		newPenvCmd(),
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := root.ExecuteContext(ctx)
	stop()
	if err != nil {
		ui.Error(err.Error())
		os.Exit(1)
	}
}

func quietRunner() *run.Runner {
	return run.New(false, nil)
}
