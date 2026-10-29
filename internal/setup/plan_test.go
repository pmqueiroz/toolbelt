package setup

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pmqueiroz/toolbelt/internal/dots"
	"github.com/pmqueiroz/toolbelt/internal/pkgmgr"
	"github.com/pmqueiroz/toolbelt/internal/run"
	"github.com/pmqueiroz/toolbelt/internal/shellrc"
)

type fakeManager struct{}

func (fakeManager) Name() string                                   { return "brew" }
func (fakeManager) ID(p pkgmgr.Package) string                     { return p.Brew }
func (fakeManager) Installed(context.Context, pkgmgr.Package) bool { return false }
func (fakeManager) Install(context.Context, pkgmgr.Package) error  { return nil }
func (fakeManager) BinDirs() []string                              { return nil }

func testContext(t *testing.T) Context {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	os.MkdirAll(home, 0o755)
	return Context{
		GOOS:  "linux",
		Paths: dots.Paths{Home: home, Config: filepath.Join(root, "config"), Store: filepath.Join(root, "store")},
		Shell: shellrc.Env{GOOS: "linux", Home: home, Profile: func(string) (string, bool) { return "", false }},
		Pkg:   fakeManager{},
		Run:   run.New(false, nil),
		Now:   time.Date(2026, 10, 28, 23, 50, 0, 0, time.UTC),
	}
}

func everything() Answers {
	return Answers{
		Username:      "peam",
		Email:         "peam@example.com",
		Steps:         allSteps(),
		Recipes:       []string{"essentials", "node", "pnpm"},
		NPMToken:      "ghp_abcdefghijklmnop",
		GPGPassphrase: "s3cret",
	}
}

func titles(phases []Phase) []string {
	var out []string
	for _, p := range phases {
		out = append(out, p.Title)
	}
	return out
}

func TestBuildPlanOrdersPhasesLikeTheOldInstaller(t *testing.T) {
	phases, err := BuildPlan(everything(), testContext(t))
	if err != nil {
		t.Fatal(err)
	}

	got := strings.Join(titles(phases), ",")
	if got != "shell sources,dependencies,dot files,ssh key,npm token,gpg key" {
		t.Errorf("phases = %s", got)
	}
}

func TestBuildPlanOnlyIncludesChosenSteps(t *testing.T) {
	a := everything()
	a.Steps = []Step{StepDots, StepSSH}

	phases, err := BuildPlan(a, testContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(titles(phases), ","); got != "dot files,ssh key" {
		t.Errorf("phases = %s", got)
	}
}

func TestBuildPlanDedupesSharedRecipeActions(t *testing.T) {
	a := everything()
	a.Steps = []Step{StepDependencies}
	a.Recipes = []string{"node", "pnpm"}

	phases, err := BuildPlan(a, testContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if n := Count(phases); n != 3 {
		t.Errorf("dependencies actions = %d, want 3 (mise once, node, pnpm)", n)
	}
}

func TestBuildPlanFailsWhenPackageManagerMissing(t *testing.T) {
	c := testContext(t)
	c.Pkg, c.PkgErr = nil, pkgmgr.ErrBrewMissing

	_, err := BuildPlan(everything(), c)
	if !errors.Is(err, pkgmgr.ErrBrewMissing) {
		t.Errorf("err = %v", err)
	}

	a := everything()
	a.Steps = []Step{StepDots}
	if _, err := BuildPlan(a, c); err != nil {
		t.Errorf("dots-only plan should not need a package manager: %v", err)
	}
}

func TestRenderListsEveryActionAndHidesSecrets(t *testing.T) {
	a := everything()
	phases, err := BuildPlan(a, testContext(t))
	if err != nil {
		t.Fatal(err)
	}

	out := Render(phases)
	for _, p := range phases {
		for _, action := range p.Actions {
			if !strings.Contains(out, action.Description) {
				t.Errorf("render misses %q", action.Description)
			}
		}
	}
	if strings.Contains(out, a.NPMToken) || strings.Contains(out, a.GPGPassphrase) {
		t.Error("render leaks a secret")
	}
}
