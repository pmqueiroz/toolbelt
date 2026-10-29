package pkgmgr

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pmqueiroz/toolbelt/internal/run"
)

var ErrBrewMissing = errors.New("brew is not installed, install it from https://brew.sh first")

var brewLocations = []string{
	"/opt/homebrew/bin/brew",
	"/usr/local/bin/brew",
	"/home/linuxbrew/.linuxbrew/bin/brew",
}

type brew struct {
	bin  string
	goos string
	r    *run.Runner
}

func newBrew(goos string, r *run.Runner) (*brew, error) {
	bin, err := findBrew()
	if err != nil {
		return nil, err
	}
	return &brew{bin: bin, goos: goos, r: r}, nil
}

func findBrew() (string, error) {
	if bin, err := exec.LookPath("brew"); err == nil {
		return bin, nil
	}
	candidates := brewLocations
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".linuxbrew", "bin", "brew"))
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", ErrBrewMissing
}

func (b *brew) Name() string { return "brew" }

func (b *brew) ID(p Package) string {
	if p.Cask && b.goos != "darwin" {
		return ""
	}
	return p.Brew
}

func (b *brew) Installed(ctx context.Context, p Package) bool {
	_, err := b.r.Output(ctx, b.bin, "list", b.kind(p), b.ID(p))
	return err == nil
}

func (b *brew) Install(ctx context.Context, p Package) error {
	return b.r.Run(ctx, b.bin, "install", b.kind(p), b.ID(p))
}

func (b *brew) BinDirs() []string {
	return []string{filepath.Dir(b.bin)}
}

func (b *brew) kind(p Package) string {
	if p.Cask {
		return "--cask"
	}
	return "--formula"
}
