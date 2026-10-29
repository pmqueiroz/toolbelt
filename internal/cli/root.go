package cli

import (
	"os"

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

	root.AddCommand(newVersionCmd(build))

	if err := root.Execute(); err != nil {
		ui.Error(err.Error())
		os.Exit(1)
	}
}
