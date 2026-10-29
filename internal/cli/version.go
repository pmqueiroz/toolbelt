package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

func newVersionCmd(build Build) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the tb version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "tb %s (%s, %s) %s/%s\n",
				build.Version, build.Commit, build.Date, runtime.GOOS, runtime.GOARCH)
		},
	}
}
