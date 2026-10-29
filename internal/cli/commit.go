package cli

import (
	"errors"
	"strings"

	"github.com/pmqueiroz/toolbelt/internal/gitx"
	"github.com/pmqueiroz/toolbelt/internal/ui"
	"github.com/spf13/cobra"
)

func newCommitCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "commit [git commit flags] [message]",
		Short:              "Stage everything and commit, titled after the branch when no message is given",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			r := quietRunner()
			git := gitx.New(r)

			if err := git.EnsureRepo(ctx); err != nil {
				return err
			}

			options, message := splitCommitArgs(args)
			if message == "" {
				branch, err := git.CurrentBranch(ctx)
				if err != nil {
					return err
				}
				message = gitx.CommitTitle(branch)
			}

			if err := r.Run(ctx, "git", "add", "."); err != nil {
				return err
			}

			ui.Info("commiting files with message " + ui.Branch(message))
			staged, err := git.StagedFiles(ctx)
			if err != nil {
				return err
			}
			ui.Println(ui.List.Render(strings.Join(staged, "\n")))

			if !ui.Confirm("confirm?") {
				_ = r.Run(ctx, "git", "reset")
				return errors.New("exiting...")
			}

			return r.Attached(ctx, "git", append([]string{"commit", "-m", message}, options...)...)
		},
	}
}

func splitCommitArgs(args []string) (options []string, message string) {
	var words []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			options = append(options, arg)
		} else {
			words = append(words, arg)
		}
	}
	return options, strings.Join(words, " ")
}
