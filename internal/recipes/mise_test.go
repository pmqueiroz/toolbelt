package recipes

import (
	"slices"
	"testing"

	"github.com/pmqueiroz/toolbelt/internal/plan"
)

func TestNodeAndPnpmShareOneMiseInstall(t *testing.T) {
	env, _ := fakeEnv("darwin")

	got := descriptions(plan.Dedupe(append(node.Plan(env), pnpm.Plan(env)...)))

	want := []string{"install mise (brew mise)", "mise use -g node@lts", "mise use -g pnpm@latest"}
	if !slices.Equal(got, want) {
		t.Errorf("descriptions = %v, want %v", got, want)
	}
}
