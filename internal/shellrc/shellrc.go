package shellrc

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pmqueiroz/toolbelt/internal/files"
	"github.com/pmqueiroz/toolbelt/internal/plan"
)

//go:embed blocks
var blocks embed.FS

const (
	beginMarker       = "# ---BEGIN TOOLBELT---"
	endMarker         = "# ---END TOOLBELT---"
	legacyBeginMarker = "# ---BEGIN DOTFILES SOURCE---"
	legacyEndMarker   = "# ---END DOTFILES SOURCE---"
)

type Target struct {
	Path  string
	Block string
}

type Env struct {
	GOOS    string
	Home    string
	Shell   string
	Profile func(bin string) (string, bool)
}

func DefaultEnv(goos, home string) Env {
	return Env{GOOS: goos, Home: home, Shell: os.Getenv("SHELL"), Profile: powershellProfile}
}

func Targets(env Env) []Target {
	if env.GOOS == "windows" {
		var targets []Target
		for _, path := range profiles(env) {
			targets = append(targets, Target{Path: path, Block: "profile.ps1"})
		}
		return targets
	}

	bashrc := filepath.Join(env.Home, ".bashrc")
	zshrc := filepath.Join(env.Home, ".zshrc")
	useBash := exists(bashrc) || strings.HasSuffix(env.Shell, "bash")
	useZsh := exists(zshrc) || strings.HasSuffix(env.Shell, "zsh")

	var targets []Target
	if useBash || !useZsh {
		targets = append(targets, Target{Path: bashrc, Block: "bash.sh"})
	}
	if useZsh {
		targets = append(targets, Target{Path: zshrc, Block: "zsh.sh"})
	}
	return targets
}

func Apply(existing, block string) string {
	cleaned := strings.TrimRight(Remove(existing), "\n")
	managed := beginMarker + "\n" + strings.TrimRight(block, "\n") + "\n" + endMarker + "\n"
	if cleaned == "" {
		return managed
	}
	return cleaned + "\n\n" + managed
}

func Plan(env Env, now time.Time) ([]plan.Action, error) {
	var actions []plan.Action
	for _, t := range Targets(env) {
		block, err := blocks.ReadFile("blocks/" + t.Block)
		if err != nil {
			return nil, err
		}
		existing, err := readIfExists(t.Path)
		if err != nil {
			return nil, err
		}
		content := []byte(Apply(existing, string(block)))

		state, err := files.Compare(t.Path, content)
		if err != nil {
			return nil, err
		}
		pretty := files.Pretty(t.Path, env.Home)
		var description string
		switch state {
		case files.Same:
			continue
		case files.Missing:
			description = fmt.Sprintf("create %s with toolbelt block", pretty)
		case files.Differs:
			description = fmt.Sprintf("update toolbelt block in %s (backup %s)", pretty, files.Pretty(files.BackupPath(t.Path, now), env.Home))
		}

		path := t.Path
		actions = append(actions, plan.Action{
			Description: description,
			Run: func(context.Context) error {
				return files.WriteWithBackup(path, content, now)
			},
		})
	}
	return actions, nil
}

func Remove(existing string) string {
	cleaned := removeBlock(existing, legacyBeginMarker, legacyEndMarker)
	cleaned = removeBlock(cleaned, beginMarker, endMarker)
	cleaned = strings.TrimRight(cleaned, "\n")
	if cleaned == "" {
		return ""
	}
	return cleaned + "\n"
}

func RemovalPlan(env Env, now time.Time) ([]plan.Action, error) {
	var actions []plan.Action
	for _, path := range rcFiles(env) {
		existing, err := readIfExists(path)
		if err != nil {
			return nil, err
		}
		content := Remove(existing)
		if content == existing {
			continue
		}

		target := path
		actions = append(actions, plan.Action{
			Description: fmt.Sprintf("remove toolbelt block from %s (backup %s)", files.Pretty(path, env.Home), files.Pretty(files.BackupPath(path, now), env.Home)),
			Run: func(context.Context) error {
				return files.WriteWithBackup(target, []byte(content), now)
			},
		})
	}
	return actions, nil
}

func rcFiles(env Env) []string {
	if env.GOOS == "windows" {
		return profiles(env)
	}
	return []string{filepath.Join(env.Home, ".bashrc"), filepath.Join(env.Home, ".zshrc")}
}

func profiles(env Env) []string {
	var paths []string
	for _, bin := range []string{"pwsh", "powershell"} {
		if path, ok := env.Profile(bin); ok {
			paths = append(paths, path)
		}
	}
	return paths
}

func removeBlock(content, begin, end string) string {
	var kept []string
	inside := false
	for _, line := range strings.Split(content, "\n") {
		switch {
		case strings.TrimSpace(line) == begin:
			inside = true
		case strings.TrimSpace(line) == end:
			inside = false
		case !inside:
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

func readIfExists(path string) (string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	return string(data), err
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func powershellProfile(bin string) (string, bool) {
	if _, err := exec.LookPath(bin); err != nil {
		return "", false
	}
	out, err := exec.Command(bin, "-NoProfile", "-Command", "$PROFILE.CurrentUserCurrentHost").Output()
	path := strings.TrimSpace(string(out))
	if err != nil || path == "" {
		return "", false
	}
	return path, true
}
