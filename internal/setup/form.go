package setup

import (
	"errors"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/pmqueiroz/toolbelt/internal/auth"
	"github.com/pmqueiroz/toolbelt/internal/recipes"
)

var ErrAborted = huh.ErrUserAborted

func Ask(defaults Answers, available []recipes.Recipe) (Answers, error) {
	a := defaults
	a.Steps = allSteps()
	a.Recipes = defaultRecipes(available)

	stepOptions := make([]huh.Option[Step], 0, len(Steps))
	for _, s := range Steps {
		stepOptions = append(stepOptions, huh.NewOption(s.Label, s.Step).Selected(true))
	}

	recipeOptions := make([]huh.Option[string], 0, len(available))
	for _, r := range available {
		recipeOptions = append(recipeOptions, huh.NewOption(r.Name, r.Name).Selected(r.Default))
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("username").Value(&a.Username).Validate(required("username")),
			huh.NewInput().Title("email").Placeholder("email@mail.com").Value(&a.Email).Validate(validEmail),
		),
		huh.NewGroup(
			huh.NewMultiSelect[Step]().
				Title("what should setup do?").
				Options(stepOptions...).
				Value(&a.Steps).
				Validate(func(s []Step) error {
					if len(s) == 0 {
						return errors.New("select at least one step")
					}
					return nil
				}),
		),
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("recipes to install").
				Options(recipeOptions...).
				Value(&a.Recipes),
		).WithHideFunc(func() bool { return !a.Has(StepDependencies) }),
		huh.NewGroup(
			huh.NewNote().Title("github packages token").Description("generate a classic token with read:packages at\n"+auth.TokenSettingsURL),
			huh.NewInput().Title("token").Placeholder("ghp_...").EchoMode(huh.EchoModePassword).Value(&a.NPMToken).Validate(required("token")),
		).WithHideFunc(func() bool { return !a.Has(StepNPM) }),
		huh.NewGroup(
			huh.NewInput().Title("gpg key passphrase").EchoMode(huh.EchoModePassword).Value(&a.GPGPassphrase).Validate(required("passphrase")),
		).WithHideFunc(func() bool { return !a.Has(StepGPG) }),
	)

	if err := form.Run(); err != nil {
		return Answers{}, err
	}
	a.Username = strings.TrimSpace(a.Username)
	a.Email = strings.TrimSpace(a.Email)
	return a, nil
}

func allSteps() []Step {
	steps := make([]Step, 0, len(Steps))
	for _, s := range Steps {
		steps = append(steps, s.Step)
	}
	return steps
}

func defaultRecipes(available []recipes.Recipe) []string {
	var names []string
	for _, r := range available {
		if r.Default {
			names = append(names, r.Name)
		}
	}
	return names
}

func required(name string) func(string) error {
	return func(v string) error {
		if strings.TrimSpace(v) == "" {
			return errors.New(name + " is required")
		}
		return nil
	}
}

func validEmail(v string) error {
	v = strings.TrimSpace(v)
	at := strings.Index(v, "@")
	if at < 1 || at == len(v)-1 {
		return errors.New("enter a valid email")
	}
	return nil
}
