package dots

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

var now = time.Date(2026, 10, 28, 23, 50, 0, 0, time.UTC)

func testPaths(t *testing.T) Paths {
	root := t.TempDir()
	return Paths{Home: filepath.Join(root, "home"), Config: filepath.Join(root, "config")}
}

func descriptions(t *testing.T, goos string, paths Paths) []string {
	t.Helper()
	actions, err := Plan(goos, paths, Vars{Username: "peam", Email: "peam@example.com"}, now)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, a := range actions {
		out = append(out, a.Description)
	}
	return out
}

func TestRender(t *testing.T) {
	got := Render("name = {{username}}\nemail = {{email}}\npass = {{passphrase}}", Vars{"peam", "p@e.com", "s3cret"})
	if got != "name = peam\nemail = p@e.com\npass = s3cret" {
		t.Errorf("Render = %q", got)
	}
}

func TestEveryDotIsEmbedded(t *testing.T) {
	for _, d := range All {
		if _, err := embedded.ReadFile("files/" + d.File); err != nil {
			t.Errorf("%s: %v", d.File, err)
		}
	}
}

func TestPlanPerOS(t *testing.T) {
	paths := testPaths(t)

	linux := descriptions(t, "linux", paths)
	windows := descriptions(t, "windows", paths)

	if len(linux) != 4 {
		t.Errorf("linux plan = %v", linux)
	}
	if slices.ContainsFunc(windows, func(d string) bool { return strings.Contains(d, ".tmux.conf") }) {
		t.Errorf("windows plan should skip tmux: %v", windows)
	}
	if !slices.Contains(linux, "write .gitconfig to ~/.gitconfig") {
		t.Errorf("linux plan = %v", linux)
	}
}

func TestPlanRunWritesRenderedFilesAndBacksUpChanged(t *testing.T) {
	paths := testPaths(t)
	gitconfig := filepath.Join(paths.Home, ".gitconfig")
	os.MkdirAll(paths.Home, 0o755)
	os.WriteFile(gitconfig, []byte("old"), 0o644)

	actions, err := Plan("linux", paths, Vars{Username: "peam", Email: "peam@example.com"}, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range actions {
		if err := a.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
	}

	written, _ := os.ReadFile(gitconfig)
	if !strings.Contains(string(written), "name = peam") || strings.Contains(string(written), "{{") {
		t.Errorf("gitconfig not rendered: %s", written)
	}
	if _, err := os.Stat(gitconfig + ".20261028_235000.bak"); err != nil {
		t.Errorf("missing backup: %v", err)
	}
	if _, err := os.Stat(filepath.Join(paths.Config, "Code", "User", "settings.json")); err != nil {
		t.Errorf("missing vscode settings: %v", err)
	}

	if again := descriptions(t, "linux", paths); len(again) != 0 {
		t.Errorf("second plan should be empty, got %v", again)
	}
}
