package shellrc

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var now = time.Date(2026, 10, 28, 23, 50, 0, 0, time.UTC)

func noProfile(string) (string, bool) { return "", false }

func TestApplyAppendsBlockAfterUserContent(t *testing.T) {
	got := Apply("export FOO=1\n\n\n", "alias a=b\n")

	want := "export FOO=1\n\n# ---BEGIN TOOLBELT---\nalias a=b\n# ---END TOOLBELT---\n"
	if got != want {
		t.Errorf("Apply = %q, want %q", got, want)
	}
}

func TestApplyReplacesExistingBlockInPlace(t *testing.T) {
	existing := "export FOO=1\n\n# ---BEGIN TOOLBELT---\nalias old=1\n# ---END TOOLBELT---\n"

	got := Apply(existing, "alias new=1")

	if strings.Contains(got, "old") || strings.Count(got, beginMarker) != 1 || !strings.Contains(got, "alias new=1") {
		t.Errorf("Apply = %q", got)
	}
}

func TestApplyRemovesLegacyDotfilesBlock(t *testing.T) {
	existing := "a\n# ---BEGIN DOTFILES SOURCE---\nsource \"$HOME\"/dots-aliases.sh\n# ---END DOTFILES SOURCE---\nb\n"

	got := Apply(existing, "x")

	if strings.Contains(got, "dots-aliases") || strings.Contains(got, legacyBeginMarker) {
		t.Errorf("legacy block not removed: %q", got)
	}
	if !strings.HasPrefix(got, "a\nb\n") {
		t.Errorf("user lines lost: %q", got)
	}
}

func TestApplyIsIdempotent(t *testing.T) {
	once := Apply("export FOO=1\n", "alias a=b")
	if twice := Apply(once, "alias a=b"); twice != once {
		t.Errorf("second Apply changed content:\n%q\n%q", once, twice)
	}
}

func TestTargets(t *testing.T) {
	home := t.TempDir()
	env := Env{GOOS: "darwin", Home: home, Shell: "/bin/zsh", Profile: noProfile}

	got := Targets(env)
	if len(got) != 1 || got[0].Block != "zsh.sh" {
		t.Errorf("zsh user without rc files: %v", got)
	}

	os.WriteFile(filepath.Join(home, ".bashrc"), nil, 0o644)
	if got := Targets(env); len(got) != 2 {
		t.Errorf("zsh user with .bashrc: %v", got)
	}

	if got := Targets(Env{GOOS: "linux", Home: t.TempDir(), Profile: noProfile}); len(got) != 1 || got[0].Block != "bash.sh" {
		t.Errorf("unknown shell falls back to bash: %v", got)
	}
}

func TestTargetsWindowsUsesPowershellProfiles(t *testing.T) {
	env := Env{GOOS: "windows", Profile: func(bin string) (string, bool) {
		return `C:\Users\peam\Documents\` + bin + `\profile.ps1`, bin == "pwsh"
	}}

	got := Targets(env)
	if len(got) != 1 || got[0].Block != "profile.ps1" || !strings.Contains(got[0].Path, "pwsh") {
		t.Errorf("Targets = %v", got)
	}
}

func TestPlanWritesBlockThenHasNothingLeftToDo(t *testing.T) {
	home := t.TempDir()
	bashrc := filepath.Join(home, ".bashrc")
	os.WriteFile(bashrc, []byte("export FOO=1\n"), 0o644)
	env := Env{GOOS: "linux", Home: home, Profile: noProfile}

	actions, err := Plan(env, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || !strings.HasPrefix(actions[0].Description, "update toolbelt block in ~/.bashrc") {
		t.Fatalf("actions = %v", actions)
	}
	if err := actions[0].Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	written, _ := os.ReadFile(bashrc)
	if !strings.HasPrefix(string(written), "export FOO=1\n") || !strings.Contains(string(written), "alias checkout='tb checkout'") {
		t.Errorf(".bashrc = %s", written)
	}

	again, err := Plan(env, now)
	if err != nil || len(again) != 0 {
		t.Errorf("second plan = %v, %v", again, err)
	}
}
