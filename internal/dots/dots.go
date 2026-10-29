package dots

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/pmqueiroz/toolbelt/internal/files"
	"github.com/pmqueiroz/toolbelt/internal/plan"
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
}

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
	return Paths{Home: home, Config: config}, nil
}

func Render(content string, vars Vars) string {
	return strings.NewReplacer(
		"{{username}}", vars.Username,
		"{{email}}", vars.Email,
		"{{passphrase}}", vars.Passphrase,
	).Replace(content)
}

func Plan(goos string, paths Paths, vars Vars, now time.Time) ([]plan.Action, error) {
	var actions []plan.Action
	for _, d := range All {
		if len(d.OS) > 0 && !slices.Contains(d.OS, goos) {
			continue
		}
		raw, err := embedded.ReadFile("files/" + d.File)
		if err != nil {
			return nil, err
		}
		content := []byte(Render(string(raw), vars))
		target := d.Target(paths)

		state, err := files.Compare(target, content)
		if err != nil {
			return nil, err
		}
		if state == files.Same {
			continue
		}

		description := fmt.Sprintf("write %s to %s", d.File, files.Pretty(target, paths.Home))
		if state == files.Differs {
			description += fmt.Sprintf(" (backup %s)", files.Pretty(files.BackupPath(target, now), paths.Home))
		}
		actions = append(actions, plan.Action{
			Description: description,
			Run: func(context.Context) error {
				return files.WriteWithBackup(target, content, now)
			},
		})
	}
	return actions, nil
}
