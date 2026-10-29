package recipes

import (
	"slices"
	"testing"

	"github.com/pmqueiroz/toolbelt/internal/plan"
)

func TestNodeToolChainIncludesPnpm(t *testing.T) {
	env, _ := fakeEnv("darwin")

	got := descriptions(node.Plan(env))

	want := []string{"install mise (brew mise)", "mise use -g node@lts", "mise use -g pnpm@latest"}
	if !slices.Equal(got, want) {
		t.Errorf("descriptions = %v, want %v", got, want)
	}
}

func TestLanguagesShareOneMiseInstall(t *testing.T) {
	env, _ := fakeEnv("windows")

	var actions []plan.Action
	for _, l := range Languages() {
		actions = append(actions, l.Plan(env)...)
	}
	got := descriptions(plan.Dedupe(actions))

	want := []string{
		"install mise (winget jdx.mise)",
		"mise use -g node@lts",
		"mise use -g pnpm@latest",
		"mise use -g rust@latest",
		"mise use -g go@latest",
		"mise use -g python@latest",
	}
	if !slices.Equal(got, want) {
		t.Errorf("descriptions = %v, want %v", got, want)
	}
}
