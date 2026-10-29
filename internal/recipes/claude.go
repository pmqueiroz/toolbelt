package recipes

import (
	"github.com/pmqueiroz/toolbelt/internal/pkgmgr"
	"github.com/pmqueiroz/toolbelt/internal/plan"
)

var claudePackage = pkgmgr.Package{Name: "claude", Brew: "claude-code", Cask: true, CaskOnLinux: true, Winget: "Anthropic.ClaudeCode"}

var claude = Recipe{
	Name:    "claude",
	Default: true,
	Plan: func(env Env) []plan.Action {
		return installPackages(env, claudePackage)
	},
}
