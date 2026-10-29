package recipes

import (
	"slices"
	"testing"
)

func TestMissingExtensionsIsCaseInsensitive(t *testing.T) {
	installed := []string{"daltonmenezes.aura-theme", "eamodio.gitlens", ""}
	wanted := []string{"DaltonMenezes.aura-theme", "eamodio.gitlens", "rust-lang.rust-analyzer"}

	got := missingExtensions(installed, wanted)

	if !slices.Equal(got, []string{"rust-lang.rust-analyzer"}) {
		t.Errorf("missing = %v", got)
	}
}

func TestCodeInstallsVSCodeOnlyWhereThePackageManagerCan(t *testing.T) {
	cases := map[string]int{"linux": 1, "darwin": 2, "windows": 2}
	for goos, want := range cases {
		env, _ := fakeEnv(goos)
		if got := len(code.Plan(env)); got != want {
			t.Errorf("%s: %d actions, want %d", goos, got, want)
		}
	}
}
