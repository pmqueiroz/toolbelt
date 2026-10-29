package recipes

import (
	"github.com/pmqueiroz/toolbelt/internal/pkgmgr"
	"github.com/pmqueiroz/toolbelt/internal/plan"
)

var essentialPackages = []pkgmgr.Package{
	{Name: "xsel", Brew: "xsel", OS: []string{"linux"}},
	{Name: "rg", Brew: "ripgrep", Winget: "BurntSushi.ripgrep.MSVC"},
	{Name: "fastfetch", Brew: "fastfetch", Winget: "Fastfetch-cli.Fastfetch"},
	{Name: "tmux", Brew: "tmux"},
	{Name: "zoxide", Brew: "zoxide", Winget: "ajeetdsouza.zoxide"},
	{Name: "jq", Brew: "jq", Winget: "jqlang.jq"},
	{Name: "gh", Brew: "gh", Winget: "GitHub.cli"},
	{Name: "glow", Brew: "glow", Winget: "charmbracelet.glow"},
	{Name: "fzf", Brew: "fzf", Winget: "junegunn.fzf"},
	{Name: "gpg", Brew: "gnupg", Winget: "GnuPG.GnuPG"},
}

var essentials = Recipe{
	Name:    "essentials",
	Default: true,
	Plan: func(env Env) []plan.Action {
		return installPackages(env, essentialPackages...)
	},
}
