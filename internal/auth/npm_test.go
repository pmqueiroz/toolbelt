package auth

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpsertNpmToken(t *testing.T) {
	cases := map[string]string{
		"":                                       "//npm.pkg.github.com/:_authToken=new\n",
		"registry=https://registry.npmjs.org/\n": "registry=https://registry.npmjs.org/\n//npm.pkg.github.com/:_authToken=new\n",
		"a=1\n//npm.pkg.github.com/:_authToken=old\nb=2\n": "a=1\nb=2\n//npm.pkg.github.com/:_authToken=new\n",
	}
	for in, want := range cases {
		if got := UpsertNpmToken(in, "new"); got != want {
			t.Errorf("UpsertNpmToken(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNPMMasksTokenAndWritesPrivateFile(t *testing.T) {
	home := t.TempDir()
	token := "ghp_abcdefghijklmnop"

	actions := NPM(home, token)
	if strings.Contains(actions[0].Description, token) || !strings.Contains(actions[0].Description, "ghp_************mnop") {
		t.Errorf("description leaks or misses token: %q", actions[0].Description)
	}
	if err := actions[0].Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(home, ".npmrc"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 && os.PathSeparator == '/' {
		t.Errorf(".npmrc mode = %v, want private", info.Mode().Perm())
	}
}
