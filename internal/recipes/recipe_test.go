package recipes

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/pmqueiroz/toolbelt/internal/pkgmgr"
	"github.com/pmqueiroz/toolbelt/internal/plan"
	"github.com/pmqueiroz/toolbelt/internal/run"
)

type fakeManager struct {
	goos      string
	installed map[string]bool
	installs  []string
}

func (f *fakeManager) Name() string {
	if f.goos == "windows" {
		return "winget"
	}
	return "brew"
}

func (f *fakeManager) ID(p pkgmgr.Package) string {
	switch {
	case f.goos == "windows":
		return p.Winget
	case p.Cask && f.goos != "darwin" && !(p.CaskOnLinux && f.goos == "linux"):
		return ""
	}
	return p.Brew
}

func (f *fakeManager) Installed(_ context.Context, p pkgmgr.Package) bool {
	return f.installed[p.Name]
}

func (f *fakeManager) Install(_ context.Context, p pkgmgr.Package) error {
	f.installs = append(f.installs, p.Name)
	return nil
}

func (f *fakeManager) BinDirs() []string { return nil }

func fakeEnv(goos string) (Env, *fakeManager) {
	m := &fakeManager{goos: goos, installed: map[string]bool{}}
	return Env{GOOS: goos, Pkg: m, Run: run.New(false, nil)}, m
}

func descriptions(actions []plan.Action) []string {
	var out []string
	for _, a := range actions {
		out = append(out, a.Description)
	}
	return out
}

func names(rs []Recipe) []string {
	var out []string
	for _, r := range rs {
		out = append(out, r.Name)
	}
	return out
}

func TestForOS(t *testing.T) {
	cases := map[string][]string{
		"linux":   {"essentials", "node", "pnpm", "code", "claude"},
		"darwin":  {"essentials", "node", "pnpm", "code", "claude", "apps", "android"},
		"windows": {"essentials", "node", "pnpm", "code", "claude", "apps", "android"},
	}
	for goos, want := range cases {
		if got := names(ForOS(goos)); !slices.Equal(got, want) {
			t.Errorf("ForOS(%s) = %v, want %v", goos, got, want)
		}
	}
}

func TestInstallPackagesSkipsAlreadyInstalled(t *testing.T) {
	env, m := fakeEnv("linux")
	m.installed["fzf-like"] = true

	actions := installPackages(env,
		pkgmgr.Package{Name: "fzf-like", Brew: "fzf-like"},
		pkgmgr.Package{Name: "not-on-path", Brew: "not-on-path"},
	)
	for _, a := range actions {
		if err := a.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
	}

	if !slices.Equal(m.installs, []string{"not-on-path"}) {
		t.Errorf("installs = %v, want [not-on-path]", m.installs)
	}
}

func TestInstallPackagesDescribesManagerAndID(t *testing.T) {
	env, _ := fakeEnv("windows")

	got := descriptions(installPackages(env, pkgmgr.Package{Name: "rg", Brew: "ripgrep", Winget: "BurntSushi.ripgrep.MSVC"}))

	if !slices.Equal(got, []string{"install rg (winget BurntSushi.ripgrep.MSVC)"}) {
		t.Errorf("descriptions = %v", got)
	}
}

func TestEveryPackageHasAnIDOnEverySupportedOS(t *testing.T) {
	for _, goos := range []string{"linux", "darwin", "windows"} {
		env, _ := fakeEnv(goos)
		for _, r := range ForOS(goos) {
			for _, d := range descriptions(r.Plan(env)) {
				if strings.HasSuffix(d, " )") || strings.Contains(d, "  ") {
					t.Errorf("%s/%s: action with empty id: %q", goos, r.Name, d)
				}
			}
		}
	}
}

func TestInstallPackagesSkipsCommandsAlreadyOnPath(t *testing.T) {
	env, m := fakeEnv("linux")

	for _, a := range installPackages(env, pkgmgr.Package{Name: "go", Brew: "go"}) {
		if err := a.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
	}

	if len(m.installs) != 0 {
		t.Errorf("installs = %v, want none", m.installs)
	}
}
