package setup

import (
	"slices"

	"github.com/pmqueiroz/toolbelt/internal/dots"
)

type Step string

const (
	StepSources      Step = "sources"
	StepDependencies Step = "dependencies"
	StepLanguages    Step = "languages"
	StepDots         Step = "dots"
	StepSSH          Step = "ssh"
	StepNPM          Step = "npm"
	StepGPG          Step = "gpg"
)

var Steps = []struct {
	Step  Step
	Label string
}{
	{StepSources, "shell sources (bash, zsh, powershell profile)"},
	{StepDependencies, "dependencies (recipes)"},
	{StepLanguages, "language tool chains (node, rust, go, python)"},
	{StepDots, "dot files (git, tmux, vscode)"},
	{StepSSH, "github ssh key"},
	{StepNPM, "github packages npm token"},
	{StepGPG, "gpg signing key"},
}

type Answers struct {
	Username      string
	Email         string
	Steps         []Step
	Recipes       []string
	Languages     []string
	NPMToken      string
	GPGPassphrase string
}

func (a Answers) Has(step Step) bool {
	return slices.Contains(a.Steps, step)
}

func (a Answers) Vars() dots.Vars {
	return dots.Vars{Username: a.Username, Email: a.Email, Passphrase: a.GPGPassphrase}
}
