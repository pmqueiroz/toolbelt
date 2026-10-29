package recipes

import (
	"slices"
	"testing"
)

func TestClaudeInstallsOnEveryOS(t *testing.T) {
	cases := map[string]string{
		"linux":   "install claude (brew claude-code)",
		"darwin":  "install claude (brew claude-code)",
		"windows": "install claude (winget Anthropic.ClaudeCode)",
	}
	for goos, want := range cases {
		env, _ := fakeEnv(goos)
		if got := descriptions(claude.Plan(env)); !slices.Equal(got, []string{want}) {
			t.Errorf("%s: descriptions = %v, want [%s]", goos, got, want)
		}
	}
}
