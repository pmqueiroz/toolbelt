package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pmqueiroz/toolbelt/internal/setup"
	"github.com/pmqueiroz/toolbelt/internal/shellrc"
	"github.com/pmqueiroz/toolbelt/internal/vault"
)

type missingKeyring struct{}

func (missingKeyring) Delete(string, string) error { return vault.ErrKeyringItemMissing }

func testUninstallEnv(t *testing.T, goos string) uninstallEnv {
	home := t.TempDir()
	exe := filepath.Join(home, "bin", "tb")
	if err := os.MkdirAll(filepath.Dir(exe), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(home, "config", "toolbelt")
	return uninstallEnv{
		goos:       goos,
		shell:      shellrc.Env{GOOS: goos, Home: home, Profile: func(string) (string, bool) { return "", false }},
		secrets:    vault.Paths{Store: filepath.Join(config, "secrets.enc"), KeyFile: filepath.Join(config, "secret.key")},
		keyring:    missingKeyring{},
		executable: exe,
		tempDir:    t.TempDir(),
		now:        time.Date(2026, 10, 29, 10, 0, 0, 0, time.UTC),
	}
}

func phaseTitles(phases []setup.Phase) []string {
	var titles []string
	for _, p := range phases {
		titles = append(titles, p.Title)
	}
	return titles
}

func TestUninstallPlanKeepsSecretsUnlessPurging(t *testing.T) {
	env := testUninstallEnv(t, "linux")

	phases, err := uninstallPlan(env, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(phaseTitles(phases), ","); got != "binary" {
		t.Errorf("phases = %s, want binary", got)
	}

	phases, err = uninstallPlan(env, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(phaseTitles(phases), ","); got != "secrets,binary" {
		t.Errorf("phases = %s, want secrets,binary", got)
	}
}

func TestUninstallPlanRemovesShellBlockBeforeBinary(t *testing.T) {
	env := testUninstallEnv(t, "linux")
	bashrc := filepath.Join(env.shell.Home, ".bashrc")
	if err := os.WriteFile(bashrc, []byte(shellrc.Apply("export FOO=1\n", "alias a=b")), 0o644); err != nil {
		t.Fatal(err)
	}

	phases, err := uninstallPlan(env, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(phaseTitles(phases), ","); got != "shell sources,secrets,binary" {
		t.Fatalf("phases = %s", got)
	}

	if failures := setup.Execute(context.Background(), phases, true); len(failures) > 0 {
		t.Fatalf("failures = %+v", failures)
	}
	if got, _ := os.ReadFile(bashrc); string(got) != "export FOO=1\n" {
		t.Errorf("bashrc = %q", got)
	}
	if _, err := os.Stat(env.executable); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("executable still present: %v", err)
	}
}

func TestRemoveExecutableParksTheRunningBinaryOnWindows(t *testing.T) {
	env := testUninstallEnv(t, "windows")

	if err := removeExecutable(env); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(env.executable); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("executable still present: %v", err)
	}
	if _, err := os.Stat(filepath.Join(env.tempDir, "tb_uninstalled_20261029_100000.exe")); err != nil {
		t.Errorf("parked binary missing: %v", err)
	}
}
