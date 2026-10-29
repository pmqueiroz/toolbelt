package recipes

import (
	"slices"
	"testing"
)

func TestEssentialsPerOS(t *testing.T) {
	cases := []struct {
		goos    string
		include []string
		exclude []string
	}{
		{"linux", []string{"install xsel (brew xsel)", "install tmux (brew tmux)", "install gpg (brew gnupg)"}, nil},
		{"darwin", []string{"install tmux (brew tmux)", "install rg (brew ripgrep)"}, []string{"install xsel (brew xsel)"}},
		{"windows", []string{"install gpg (winget GnuPG.GnuPG)"}, []string{"install tmux (winget )", "install xsel (winget )"}},
	}
	for _, c := range cases {
		env, _ := fakeEnv(c.goos)
		got := descriptions(essentials.Plan(env))
		for _, want := range c.include {
			if !slices.Contains(got, want) {
				t.Errorf("%s: missing %q in %v", c.goos, want, got)
			}
		}
		for _, unwanted := range c.exclude {
			if slices.Contains(got, unwanted) {
				t.Errorf("%s: unexpected %q", c.goos, unwanted)
			}
		}
	}
}

func TestEssentialsWindowsSkipsBrewOnlyPackages(t *testing.T) {
	env, _ := fakeEnv("windows")
	if got := len(essentials.Plan(env)); got != len(essentialPackages)-2 {
		t.Errorf("windows essentials has %d actions, want %d", got, len(essentialPackages)-2)
	}
}
