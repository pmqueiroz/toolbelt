package recipes

import (
	"context"
	"fmt"
	"slices"

	"github.com/pmqueiroz/toolbelt/internal/pkgmgr"
	"github.com/pmqueiroz/toolbelt/internal/plan"
	"github.com/pmqueiroz/toolbelt/internal/run"
)

type Env struct {
	GOOS string
	Pkg  pkgmgr.Manager
	Run  *run.Runner
}

type Recipe struct {
	Name    string
	Default bool
	OS      []string
	Plan    func(env Env) []plan.Action
}

func All() []Recipe {
	return []Recipe{
		essentials,
		node,
		pnpm,
		code,
		apps,
		android,
	}
}

func ForOS(goos string) []Recipe {
	var supported []Recipe
	for _, r := range All() {
		if r.Supports(goos) {
			supported = append(supported, r)
		}
	}
	return supported
}

func (r Recipe) Supports(goos string) bool {
	return len(r.OS) == 0 || slices.Contains(r.OS, goos)
}

func installPackages(env Env, pkgs ...pkgmgr.Package) []plan.Action {
	var actions []plan.Action
	for _, p := range pkgs {
		if !pkgmgr.Supports(env.Pkg, env.GOOS, p) {
			continue
		}
		actions = append(actions, plan.Action{
			Description: fmt.Sprintf("install %s (%s %s)", p.Name, env.Pkg.Name(), env.Pkg.ID(p)),
			Run: func(ctx context.Context) error {
				if run.Exists(p.Name) || env.Pkg.Installed(ctx, p) {
					env.Run.Logf("%s already installed. skipping", p.Name)
					return nil
				}
				return env.Pkg.Install(ctx, p)
			},
		})
	}
	return actions
}
