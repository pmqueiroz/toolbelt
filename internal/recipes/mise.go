package recipes

import (
	"context"

	"github.com/pmqueiroz/toolbelt/internal/pkgmgr"
	"github.com/pmqueiroz/toolbelt/internal/plan"
)

var misePackage = pkgmgr.Package{Name: "mise", Brew: "mise", Winget: "jdx.mise"}

var node = Recipe{
	Name:    "node",
	Default: true,
	Plan: func(env Env) []plan.Action {
		return miseUse(env, "node@lts")
	},
}

var pnpm = Recipe{
	Name:    "pnpm",
	Default: true,
	Plan: func(env Env) []plan.Action {
		return miseUse(env, "pnpm@latest")
	},
}

func miseUse(env Env, tool string) []plan.Action {
	return append(installPackages(env, misePackage), plan.Action{
		Description: "mise use -g " + tool,
		Run: func(ctx context.Context) error {
			return env.Run.Run(ctx, "mise", "use", "--global", "--yes", tool)
		},
	})
}
