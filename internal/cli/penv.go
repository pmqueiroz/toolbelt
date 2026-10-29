package cli

import (
	"fmt"

	"github.com/pmqueiroz/toolbelt/internal/secret"
	"github.com/spf13/cobra"
)

func newPenvCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "penv <value>",
		Short: "Print a secret with everything but its first and last 4 characters masked",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			masked, err := secret.Mask(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), masked)
			return nil
		},
	}
}
