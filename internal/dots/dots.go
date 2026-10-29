package dots

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
)

//go:embed all:files
var embedded embed.FS

type Vars struct {
	Username   string
	Email      string
	Passphrase string
}

type Paths struct {
	Home   string
	Config string
	Store  string
}

func (p Paths) live(file string) string { return filepath.Join(p.Store, file) }
func (p Paths) base(file string) string { return filepath.Join(p.Store, ".base", file) }
func (p Paths) vars() string            { return filepath.Join(p.Store, "vars.json") }

type Dot struct {
	File   string
	OS     []string
	Target func(p Paths) string
}

var All = []Dot{
	{File: ".gitconfig", Target: func(p Paths) string { return filepath.Join(p.Home, ".gitconfig") }},
	{File: ".tmux.conf", OS: []string{"linux", "darwin"}, Target: func(p Paths) string { return filepath.Join(p.Home, ".tmux.conf") }},
	{File: "vscode-settings.json", Target: func(p Paths) string { return filepath.Join(p.Config, "Code", "User", "settings.json") }},
	{File: "keybindings.json", Target: func(p Paths) string { return filepath.Join(p.Config, "Code", "User", "keybindings.json") }},
	{File: "claude-settings.json", Target: func(p Paths) string { return filepath.Join(p.Home, ".claude", "settings.json") }},
	{File: "ccstatusline-settings.json", Target: func(p Paths) string { return filepath.Join(p.Home, ".config", "ccstatusline", "settings.json") }},
}

func DefaultPaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}
	config, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, err
	}
	return Paths{Home: home, Config: config, Store: filepath.Join(config, "toolbelt", "dots")}, nil
}

func Render(content string, vars Vars) string {
	return strings.NewReplacer(
		"{{username}}", vars.Username,
		"{{email}}", vars.Email,
		"{{passphrase}}", vars.Passphrase,
	).Replace(content)
}
