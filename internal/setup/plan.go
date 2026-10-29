package setup

import (
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/pmqueiroz/toolbelt/internal/auth"
	"github.com/pmqueiroz/toolbelt/internal/dots"
	"github.com/pmqueiroz/toolbelt/internal/pkgmgr"
	"github.com/pmqueiroz/toolbelt/internal/plan"
	"github.com/pmqueiroz/toolbelt/internal/recipes"
	"github.com/pmqueiroz/toolbelt/internal/run"
	"github.com/pmqueiroz/toolbelt/internal/shellrc"
	"github.com/pmqueiroz/toolbelt/internal/ui"
)

type Context struct {
	GOOS   string
	Paths  dots.Paths
	Shell  shellrc.Env
	Pkg    pkgmgr.Manager
	PkgErr error
	Run    *run.Runner
	Now    time.Time
}

type Phase struct {
	Title   string
	Actions []plan.Action
}

func BuildPlan(a Answers, c Context) ([]Phase, error) {
	var phases []Phase
	add := func(title string, actions []plan.Action) {
		if len(actions) > 0 {
			phases = append(phases, Phase{Title: title, Actions: actions})
		}
	}

	if a.Has(StepSources) {
		actions, err := shellrc.Plan(c.Shell, c.Now)
		if err != nil {
			return nil, err
		}
		add("shell sources", actions)
	}

	if a.Has(StepDependencies) && len(a.Recipes) > 0 {
		if c.PkgErr != nil {
			return nil, c.PkgErr
		}
		add("dependencies", recipeActions(c, recipes.ForOS(c.GOOS), a.Recipes))
	}

	if a.Has(StepLanguages) && len(a.Languages) > 0 {
		if c.PkgErr != nil {
			return nil, c.PkgErr
		}
		add("language tool chains", recipeActions(c, recipes.LanguagesForOS(c.GOOS), a.Languages))
	}

	if a.Has(StepDots) {
		actions, err := dots.Plan(c.GOOS, c.Paths, a.Vars(), c.Now)
		if err != nil {
			return nil, err
		}
		add("dot files", actions)
	}

	if a.Has(StepSSH) {
		add("ssh key", auth.SSH(c.Paths.Home, a.Email, c.Run))
	}
	if a.Has(StepNPM) {
		add("npm token", auth.NPM(c.Paths.Home, a.NPMToken))
	}
	if a.Has(StepGPG) {
		add("gpg key", auth.GPG(c.GOOS, c.Paths.Home, a.Vars(), c.Run))
	}

	return phases, nil
}

func recipeActions(c Context, available []recipes.Recipe, chosen []string) []plan.Action {
	env := recipes.Env{GOOS: c.GOOS, Pkg: c.Pkg, Run: c.Run}
	var actions []plan.Action
	for _, r := range available {
		if slices.Contains(chosen, r.Name) {
			actions = append(actions, r.Plan(env)...)
		}
	}
	return plan.Dedupe(actions)
}

func Count(phases []Phase) int {
	n := 0
	for _, p := range phases {
		n += len(p.Actions)
	}
	return n
}

func Render(phases []Phase) string {
	var b strings.Builder
	b.WriteString(ui.Title.Render("plan"))
	for _, p := range phases {
		b.WriteString("\n\n" + ui.Title.Render(p.Title))
		for _, a := range p.Actions {
			b.WriteString("\n  • " + a.Description)
		}
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.Pink).
		Padding(1, 2).
		Render(b.String())
}
