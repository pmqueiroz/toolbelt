package cli

import (
	"fmt"
	"os"

	"github.com/pmqueiroz/toolbelt/internal/secret"
	"github.com/spf13/cobra"
)

func newEnvCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "env [value]",
		Short: "Print a secret masked, or every environment variable masked when no value is given",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if len(args) == 0 {
				for _, line := range secret.MaskEnviron(os.Environ()) {
					fmt.Fprintln(out, line)
				}
				return nil
			}
			masked, err := secret.Mask(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(out, masked)
			return nil
		},
	}
}
