package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const maskVisible = 8

func newPenvCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "penv <value>",
		Short: "Print a secret with everything but its first and last 4 characters masked",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			masked, err := mask(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), masked)
			return nil
		},
	}
}

func mask(value string) (string, error) {
	runes := []rune(value)
	if len(runes) < maskVisible {
		return "", errors.New("env var is too short")
	}
	edge := maskVisible / 2
	return string(runes[:edge]) + strings.Repeat("*", len(runes)-maskVisible) + string(runes[len(runes)-edge:]), nil
}
