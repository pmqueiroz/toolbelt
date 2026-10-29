package cli

import (
	"context"
	"fmt"
	"regexp"

	"github.com/pmqueiroz/toolbelt/internal/gitx"
	"github.com/pmqueiroz/toolbelt/internal/run"
	"github.com/pmqueiroz/toolbelt/internal/ui"
	"github.com/spf13/cobra"
)

var prNumber = regexp.MustCompile(`^[0-9]+$`)

func newCheckoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "checkout [branch|pr-number]",
		Short: "Switch to a branch (default branch when empty), pull, and prune every other local branch",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			r := quietRunner()
			git := gitx.New(r)

			if err := git.EnsureRepo(ctx); err != nil {
				return err
			}

			target, err := resolveCheckoutTarget(ctx, git, args)
			if err != nil {
				return err
			}

			current, err := git.CurrentBranch(ctx)
			if err != nil {
				return err
			}
			if current != target {
				ui.Info("checking out to " + ui.Branch(target))
				if err := r.Attached(ctx, "git", "checkout", "--quiet", target); err != nil {
					return err
				}
			}

			ui.Info("updating current branch")
			if err := r.Attached(ctx, "git", "pull", "--rebase", "--quiet"); err != nil {
				ui.Warn(err.Error())
			}

			return pruneBranches(ctx, r, git, target)
		},
	}
}

func resolveCheckoutTarget(ctx context.Context, git *gitx.Git, args []string) (string, error) {
	if len(args) == 0 {
		return git.DefaultBranch(ctx)
	}
	if !prNumber.MatchString(args[0]) {
		return args[0], nil
	}
	ui.Info("fetching branch to " + ui.PR(args[0]))
	branch, err := git.BranchForPR(ctx, args[0])
	if err != nil || branch == "" {
		return "", fmt.Errorf("could not determine branch to %s", ui.PR(args[0]))
	}
	return branch, nil
}

func pruneBranches(ctx context.Context, r *run.Runner, git *gitx.Git, keep string) error {
	branches, err := git.LocalBranches(ctx)
	if err != nil {
		return err
	}
	var others []string
	for _, b := range branches {
		if b != keep {
			others = append(others, b)
		}
	}
	if len(others) == 0 {
		return nil
	}

	ui.Info("deleting other branches. was:")
	for _, b := range others {
		ui.Println(ui.Branch(b))
	}

	worktrees, err := git.Worktrees(ctx)
	if err != nil {
		return err
	}

	for _, branch := range others {
		path, inWorktree := worktrees[branch]
		if !inWorktree {
			if err := r.Run(ctx, "git", "branch", "-q", "-D", branch); err != nil {
				ui.Error(err.Error())
			}
			continue
		}

		state, err := git.BranchState(ctx, branch, path)
		if err != nil {
			ui.Error(err.Error())
			continue
		}
		if reason := state.KeepReason(); reason != "" {
			ui.Warn(fmt.Sprintf("keeping %s: %s (worktree %s)", branch, reason, path))
			continue
		}

		ui.Info("removing worktree " + path)
		if err := r.Run(ctx, "git", "worktree", "remove", path); err != nil {
			ui.Error(err.Error())
			continue
		}
		if err := r.Run(ctx, "git", "branch", "-q", "-D", branch); err != nil {
			ui.Error(err.Error())
		}
	}
	return nil
}
