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
		return miseUse(env, "node@lts", "pnpm@latest")
	},
}

var rust = Recipe{
	Name: "rust",
	Plan: func(env Env) []plan.Action {
		return miseUse(env, "rust@latest")
	},
}

var golang = Recipe{
	Name: "go",
	Plan: func(env Env) []plan.Action {
		return miseUse(env, "go@latest")
	},
}

var python = Recipe{
	Name: "python",
	Plan: func(env Env) []plan.Action {
		return miseUse(env, "python@latest")
	},
}

func miseUse(env Env, tools ...string) []plan.Action {
	actions := installPackages(env, misePackage)
	for _, tool := range tools {
		actions = append(actions, plan.Action{
			Description: "mise use -g " + tool,
			Run: func(ctx context.Context) error {
				return env.Run.Run(ctx, "mise", "use", "--global", "--yes", tool)
			},
		})
	}
	return actions
}
