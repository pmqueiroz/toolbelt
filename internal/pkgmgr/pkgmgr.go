package pkgmgr

import (
	"context"
	"os"
	"path/filepath"
	"slices"

	"github.com/pmqueiroz/toolbelt/internal/run"
)

type Package struct {
	Name   string
	Brew   string
	Cask   bool
	Winget string
	OS     []string
}

type Manager interface {
	Name() string
	ID(p Package) string
	Installed(ctx context.Context, p Package) bool
	Install(ctx context.Context, p Package) error
	BinDirs() []string
}

func Detect(goos string, r *run.Runner) (Manager, error) {
	if goos == "windows" {
		return newWinget(r)
	}
	return newBrew(goos, r)
}

func Supports(m Manager, goos string, p Package) bool {
	if len(p.OS) > 0 && !slices.Contains(p.OS, goos) {
		return false
	}
	return m.ID(p) != ""
}

func PrependPath(m Manager) {
	dirs := append(m.BinDirs(), filepath.SplitList(os.Getenv("PATH"))...)
	os.Setenv("PATH", joinPath(dirs))
}

func joinPath(dirs []string) string {
	var unique []string
	for _, d := range dirs {
		if d != "" && !slices.Contains(unique, d) {
			unique = append(unique, d)
		}
	}
	out := ""
	for i, d := range unique {
		if i > 0 {
			out += string(os.PathListSeparator)
		}
		out += d
	}
	return out
}
