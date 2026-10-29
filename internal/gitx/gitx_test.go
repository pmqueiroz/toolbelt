package gitx

import (
	"reflect"
	"testing"
)

func TestCommitTitle(t *testing.T) {
	cases := map[string]string{
		"feat/add-tb-cli":    "feat: add tb cli",
		"fix/aliases/prune":  "fix: aliases: prune",
		"master":             "master",
		"chore/bump-node-24": "chore: bump node 24",
	}
	for branch, want := range cases {
		if got := CommitTitle(branch); got != want {
			t.Errorf("CommitTitle(%q) = %q, want %q", branch, got, want)
		}
	}
}

func TestParseWorktrees(t *testing.T) {
	porcelain := `worktree /repo
HEAD 1111111111111111111111111111111111111111
branch refs/heads/master

worktree /repo/.worktrees/feat
HEAD 2222222222222222222222222222222222222222
branch refs/heads/feat/x

worktree /repo/.worktrees/detached
HEAD 3333333333333333333333333333333333333333
detached
`
	want := map[string]string{
		"master": "/repo",
		"feat/x": "/repo/.worktrees/feat",
	}
	if got := ParseWorktrees(porcelain); !reflect.DeepEqual(got, want) {
		t.Errorf("ParseWorktrees = %v, want %v", got, want)
	}
}

func TestKeepReason(t *testing.T) {
	cases := []struct {
		name  string
		state BranchState
		want  string
	}{
		{"dirty", BranchState{Dirty: true, Upstream: "origin/x"}, "uncommitted changes"},
		{"never pushed", BranchState{}, "never pushed"},
		{"ahead", BranchState{Upstream: "origin/x", Ahead: 2}, "unpushed commits"},
		{"gone upstream", BranchState{Upstream: "origin/x", Gone: true}, ""},
		{"in sync", BranchState{Upstream: "origin/x"}, ""},
	}
	for _, c := range cases {
		if got := c.state.KeepReason(); got != c.want {
			t.Errorf("%s: KeepReason() = %q, want %q", c.name, got, c.want)
		}
	}
}
