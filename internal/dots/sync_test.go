package dots

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

var (
	now   = time.Date(2026, 10, 28, 23, 50, 0, 0, time.UTC)
	later = now.Add(time.Hour)
	peam  = Vars{Username: "peam", Email: "peam@example.com"}
)

func testPaths(t *testing.T) Paths {
	root := t.TempDir()
	return Paths{
		Home:   filepath.Join(root, "home"),
		Config: filepath.Join(root, "config"),
		Store:  filepath.Join(root, "config", "toolbelt", "dots"),
	}
}

func requireSymlinks(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := os.Symlink(filepath.Join(dir, "a"), filepath.Join(dir, "b")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
}

func descriptions(t *testing.T, goos string, paths Paths, vars Vars, at time.Time) []string {
	t.Helper()
	actions, err := Plan(goos, paths, vars, at)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, a := range actions {
		out = append(out, a.Description)
	}
	return out
}

func apply(t *testing.T, paths Paths, vars Vars, at time.Time) []error {
	t.Helper()
	actions, err := Plan("linux", paths, vars, at)
	if err != nil {
		t.Fatal(err)
	}
	var errs []error
	for _, a := range actions {
		if err := a.Run(context.Background()); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func mustApply(t *testing.T, paths Paths, vars Vars, at time.Time) {
	t.Helper()
	if errs := apply(t, paths, vars, at); len(errs) > 0 {
		t.Fatal(errs)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func write(t *testing.T, path, content string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPlanPerOS(t *testing.T) {
	paths := testPaths(t)

	linux := descriptions(t, "linux", paths, peam, now)
	windows := descriptions(t, "windows", paths, peam, now)

	if len(linux) != 6 {
		t.Errorf("linux plan = %v", linux)
	}
	if slices.ContainsFunc(windows, func(d string) bool { return strings.Contains(d, ".tmux.conf") }) {
		t.Errorf("windows plan should skip tmux: %v", windows)
	}
	if !slices.ContainsFunc(linux, func(d string) bool { return strings.HasPrefix(d, "write .gitconfig to ") }) {
		t.Errorf("linux plan = %v", linux)
	}
}

func TestPlanLinksTargetsToTheStoreAndBacksUpForeignFiles(t *testing.T) {
	requireSymlinks(t)
	paths := testPaths(t)
	gitconfig := filepath.Join(paths.Home, ".gitconfig")
	write(t, gitconfig, "old")

	mustApply(t, paths, peam, now)

	dest, err := os.Readlink(gitconfig)
	if err != nil || dest != filepath.Join(paths.Store, ".gitconfig") {
		t.Fatalf("gitconfig link = %q, %v", dest, err)
	}
	if got := read(t, gitconfig); !strings.Contains(got, "name = peam") || strings.Contains(got, "{{") {
		t.Errorf("gitconfig not rendered: %s", got)
	}
	if got := read(t, gitconfig+".20261028_235000.bak"); got != "old" {
		t.Errorf("backup = %q", got)
	}
	if _, err := os.Readlink(filepath.Join(paths.Config, "Code", "User", "settings.json")); err != nil {
		t.Errorf("vscode settings not linked: %v", err)
	}
	if vars, err := LoadVars(paths); err != nil || vars.Username != "peam" || vars.Email != "peam@example.com" {
		t.Errorf("LoadVars = %+v, %v", vars, err)
	}

	if again := descriptions(t, "linux", paths, peam, now); len(again) != 0 {
		t.Errorf("second plan should be empty, got %v", again)
	}
}

func TestPlanKeepsLocalEditsWhenUpstreamIsUnchanged(t *testing.T) {
	requireSymlinks(t)
	paths := testPaths(t)
	mustApply(t, paths, peam, now)
	gitconfig := filepath.Join(paths.Home, ".gitconfig")
	write(t, gitconfig, read(t, gitconfig)+"[local]\n\tkept = true\n")

	if again := descriptions(t, "linux", paths, peam, later); len(again) != 0 {
		t.Errorf("plan should leave local edits alone, got %v", again)
	}
}

func TestPlanMergesUpstreamChangesWithLocalEdits(t *testing.T) {
	requireSymlinks(t)
	paths := testPaths(t)
	mustApply(t, paths, peam, now)
	gitconfig := filepath.Join(paths.Home, ".gitconfig")
	write(t, gitconfig, read(t, gitconfig)+"[local]\n\tkept = true\n")

	renamed := Vars{Username: "pedro", Email: peam.Email}
	got := descriptions(t, "linux", paths, renamed, later)
	if !slices.ContainsFunc(got, func(d string) bool {
		return strings.HasPrefix(d, "merge .gitconfig into ") && strings.HasSuffix(d, "keeping local edits")
	}) {
		t.Errorf("plan = %v", got)
	}
	mustApply(t, paths, renamed, later)

	merged := read(t, gitconfig)
	if !strings.Contains(merged, "name = pedro") || !strings.Contains(merged, "kept = true") {
		t.Errorf("merged gitconfig = %s", merged)
	}
	if _, err := os.Stat(gitconfig + ".20261029_005000.bak"); err == nil {
		t.Error("a clean merge through the link should not need a backup")
	}
}

func TestPlanWritesConflictMarkersAndRefusesToSyncUntilResolved(t *testing.T) {
	requireSymlinks(t)
	paths := testPaths(t)
	mustApply(t, paths, peam, now)
	gitconfig := filepath.Join(paths.Home, ".gitconfig")
	local := strings.Replace(read(t, gitconfig), "name = peam", "name = local", 1)
	write(t, gitconfig, local)

	errs := apply(t, paths, Vars{Username: "upstream", Email: peam.Email}, later)
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "1 conflict(s)") {
		t.Fatalf("errs = %v", errs)
	}
	conflicted := read(t, gitconfig)
	for _, want := range []string{"<<<<<<< local", "name = local", "=======", "name = upstream", ">>>>>>> toolbelt"} {
		if !strings.Contains(conflicted, want) {
			t.Errorf("conflicted gitconfig misses %q: %s", want, conflicted)
		}
	}
	if got := read(t, gitconfig+".20261029_005000.bak"); got != local {
		t.Errorf("backup = %q", got)
	}

	errs = apply(t, paths, Vars{Username: "newer", Email: peam.Email}, later)
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "resolve the conflict markers") {
		t.Fatalf("errs = %v", errs)
	}
	if read(t, gitconfig) != conflicted {
		t.Error("an unresolved file was changed")
	}

	write(t, gitconfig, local)
	mustApply(t, paths, Vars{Username: "upstream", Email: peam.Email}, later)
	if !strings.Contains(read(t, gitconfig), "name = local") {
		t.Error("resolved local choice was overwritten")
	}
}

func TestPlanRelinksAFileAnEditorReplacedAndKeepsItsEdits(t *testing.T) {
	requireSymlinks(t)
	paths := testPaths(t)
	mustApply(t, paths, peam, now)
	gitconfig := filepath.Join(paths.Home, ".gitconfig")
	edited := read(t, gitconfig) + "[local]\n\tkept = true\n"
	os.Remove(gitconfig)
	write(t, gitconfig, edited)

	mustApply(t, paths, Vars{Username: "pedro", Email: peam.Email}, later)

	if _, err := os.Readlink(gitconfig); err != nil {
		t.Errorf("gitconfig not relinked: %v", err)
	}
	merged := read(t, gitconfig)
	if !strings.Contains(merged, "name = pedro") || !strings.Contains(merged, "kept = true") {
		t.Errorf("merged gitconfig = %s", merged)
	}
	if got := read(t, gitconfig+".20261029_005000.bak"); got != edited {
		t.Errorf("backup = %q", got)
	}
}

func TestPlanRecreatesAMissingLink(t *testing.T) {
	requireSymlinks(t)
	paths := testPaths(t)
	mustApply(t, paths, peam, now)
	gitconfig := filepath.Join(paths.Home, ".gitconfig")
	os.Remove(gitconfig)

	got := descriptions(t, "linux", paths, peam, later)
	if len(got) != 1 || !strings.Contains(got[0], "link") {
		t.Fatalf("plan = %v", got)
	}
	mustApply(t, paths, peam, later)
	if _, err := os.Readlink(gitconfig); err != nil {
		t.Errorf("gitconfig not relinked: %v", err)
	}
}

func TestLoadVarsWithoutSetupIsNotManaged(t *testing.T) {
	if _, err := LoadVars(testPaths(t)); !errors.Is(err, ErrNotManaged) {
		t.Errorf("err = %v, want ErrNotManaged", err)
	}
}

func TestDecideAdoptsForeignFilesWithABackup(t *testing.T) {
	o := decide(state{upstream: []byte("new"), local: []byte("mine"), hasLocal: true, targetFile: true})
	if string(o.content) != "new" || !o.backup {
		t.Errorf("outcome = %+v", o)
	}
}
