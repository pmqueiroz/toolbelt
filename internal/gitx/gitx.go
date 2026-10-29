package gitx

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/pmqueiroz/toolbelt/internal/run"
)

var ErrNotRepo = errors.New("not a git repository")

type Git struct {
	r *run.Runner
}

func New(r *run.Runner) *Git {
	return &Git{r: r}
}

func CommitTitle(branch string) string {
	return strings.NewReplacer("/", ": ", "-", " ").Replace(branch)
}

func (g *Git) EnsureRepo(ctx context.Context) error {
	if _, err := g.r.Output(ctx, "git", "rev-parse", "--is-inside-work-tree"); err != nil {
		return ErrNotRepo
	}
	return nil
}

func (g *Git) CurrentBranch(ctx context.Context) (string, error) {
	return g.r.Output(ctx, "git", "branch", "--show-current")
}

var ErrNoDefaultBranch = errors.New("could not determine default branch")

func (g *Git) DefaultBranch(ctx context.Context) (string, error) {
	if branch, err := g.originHead(ctx); err == nil {
		return branch, nil
	}
	if _, err := g.r.Output(ctx, "git", "remote", "set-head", "origin", "--auto"); err == nil {
		if branch, err := g.originHead(ctx); err == nil {
			return branch, nil
		}
	}
	for _, branch := range []string{"main", "master"} {
		if _, err := g.r.Output(ctx, "git", "rev-parse", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
			return branch, nil
		}
	}
	return "", ErrNoDefaultBranch
}

func (g *Git) originHead(ctx context.Context) (string, error) {
	ref, err := g.r.Output(ctx, "git", "symbolic-ref", "refs/remotes/origin/HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(ref, "refs/remotes/origin/"), nil
}

func (g *Git) BranchForPR(ctx context.Context, number string) (string, error) {
	return g.r.Output(ctx, "gh", "pr", "view", number, "--json", "headRefName", "--jq", ".headRefName")
}

func (g *Git) LocalBranches(ctx context.Context) ([]string, error) {
	out, err := g.r.Output(ctx, "git", "for-each-ref", "--format=%(refname:short)", "refs/heads/")
	if err != nil {
		return nil, err
	}
	return lines(out), nil
}

func (g *Git) StagedFiles(ctx context.Context) ([]string, error) {
	out, err := g.r.Output(ctx, "git", "diff", "--name-only", "--cached")
	if err != nil {
		return nil, err
	}
	return lines(out), nil
}

func (g *Git) Worktrees(ctx context.Context) (map[string]string, error) {
	out, err := g.r.Output(ctx, "git", "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return ParseWorktrees(out), nil
}

func ParseWorktrees(porcelain string) map[string]string {
	byBranch := map[string]string{}
	path := ""
	for _, line := range strings.Split(porcelain, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "branch refs/heads/"):
			byBranch[strings.TrimPrefix(line, "branch refs/heads/")] = path
		}
	}
	return byBranch
}

func (g *Git) BranchState(ctx context.Context, branch, worktree string) (BranchState, error) {
	status, err := g.r.Output(ctx, "git", "-C", worktree, "status", "--porcelain")
	if err != nil {
		return BranchState{}, err
	}
	ref := "refs/heads/" + branch
	upstream, err := g.r.Output(ctx, "git", "for-each-ref", "--format=%(upstream:short)", ref)
	if err != nil {
		return BranchState{}, err
	}
	track, err := g.r.Output(ctx, "git", "for-each-ref", "--format=%(upstream:track)", ref)
	if err != nil {
		return BranchState{}, err
	}
	state := BranchState{Dirty: status != "", Upstream: upstream, Gone: track == "[gone]"}
	if upstream != "" && !state.Gone {
		count, err := g.r.Output(ctx, "git", "rev-list", "--count", upstream+".."+branch)
		if err != nil {
			return BranchState{}, err
		}
		state.Ahead, _ = strconv.Atoi(count)
	}
	return state, nil
}

type BranchState struct {
	Dirty    bool
	Upstream string
	Gone     bool
	Ahead    int
}

func (s BranchState) KeepReason() string {
	switch {
	case s.Dirty:
		return "uncommitted changes"
	case s.Upstream == "":
		return "never pushed"
	case !s.Gone && s.Ahead > 0:
		return "unpushed commits"
	}
	return ""
}

func lines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
