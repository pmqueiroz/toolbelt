package pkgmgr

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSupports(t *testing.T) {
	formula := Package{Name: "rg", Brew: "ripgrep", Winget: "BurntSushi.ripgrep.MSVC"}
	cask := Package{Name: "slack", Brew: "slack", Cask: true, Winget: "SlackTechnologies.Slack"}
	brewOnly := Package{Name: "tmux", Brew: "tmux"}
	linuxOnly := Package{Name: "xsel", Brew: "xsel", OS: []string{"linux"}}

	cases := []struct {
		name string
		m    Manager
		goos string
		p    Package
		want bool
	}{
		{"formula on linux", &brew{goos: "linux"}, "linux", formula, true},
		{"cask on darwin", &brew{goos: "darwin"}, "darwin", cask, true},
		{"cask on linux", &brew{goos: "linux"}, "linux", cask, false},
		{"cask on windows", &winget{}, "windows", cask, true},
		{"brew only on windows", &winget{}, "windows", brewOnly, false},
		{"linux only on darwin", &brew{goos: "darwin"}, "darwin", linuxOnly, false},
		{"linux only on linux", &brew{goos: "linux"}, "linux", linuxOnly, true},
	}
	for _, c := range cases {
		if got := Supports(c.m, c.goos, c.p); got != c.want {
			t.Errorf("%s: Supports = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestPrependPathPutsManagerDirsFirstWithoutDuplicates(t *testing.T) {
	sep := string(os.PathListSeparator)
	brewDir := filepath.Dir(filepath.FromSlash("/opt/homebrew/bin/brew"))
	t.Setenv("PATH", strings.Join([]string{"/usr/bin", brewDir}, sep))

	PrependPath(&brew{bin: filepath.Join(brewDir, "brew")})

	want := strings.Join([]string{brewDir, "/usr/bin"}, sep)
	if got := os.Getenv("PATH"); got != want {
		t.Errorf("PATH = %q, want %q", got, want)
	}
}
