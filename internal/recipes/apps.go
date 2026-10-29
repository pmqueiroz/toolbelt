package recipes

import (
	"github.com/pmqueiroz/toolbelt/internal/pkgmgr"
	"github.com/pmqueiroz/toolbelt/internal/plan"
)

var desktopApps = []pkgmgr.Package{
	{Name: "slack", Brew: "slack", Cask: true, Winget: "SlackTechnologies.Slack"},
	{Name: "warp", Brew: "warp", Cask: true, Winget: "Warp.Warp"},
	{Name: "firefox", Brew: "firefox", Cask: true, Winget: "Mozilla.Firefox"},
	{Name: "obs", Brew: "obs", Cask: true, Winget: "OBSProject.OBSStudio"},
	{Name: "figma", Brew: "figma", Cask: true, Winget: "Figma.Figma"},
}

var androidStudio = pkgmgr.Package{Name: "android-studio", Brew: "android-studio", Cask: true, Winget: "Google.AndroidStudio"}

var apps = Recipe{
	Name: "apps",
	OS:   []string{"darwin", "windows"},
	Plan: func(env Env) []plan.Action {
		return installPackages(env, desktopApps...)
	},
}

var android = Recipe{
	Name: "android",
	OS:   []string{"darwin", "windows"},
	Plan: func(env Env) []plan.Action {
		return installPackages(env, androidStudio)
	},
}
