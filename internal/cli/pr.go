package cli

import (
	"github.com/pmqueiroz/toolbelt/internal/gitx"
	"github.com/spf13/cobra"
)

func newPRCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "pr [gh pr create flags]",
		Short:              "Open the current branch's PR, or create one titled after the branch",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			r := quietRunner()
			git := gitx.New(r)

			if _, err := r.Output(ctx, "gh", "pr", "view", "--json", "number", "-q", ".number"); err == nil {
				return r.Attached(ctx, "gh", "pr", "view", "--web")
			}

			branch, err := git.CurrentBranch(ctx)
			if err != nil {
				return err
			}
			create := append([]string{"pr", "create", "--assignee", "@me", "--title", gitx.CommitTitle(branch), "--web"}, args...)
			return r.Attached(ctx, "gh", create...)
		},
	}
}
