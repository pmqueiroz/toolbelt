package pkgmgr

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pmqueiroz/toolbelt/internal/run"
)

var ErrWingetMissing = errors.New("winget is not installed, install App Installer from the Microsoft Store first")

var wingetCommon = []string{"--exact", "--accept-source-agreements", "--disable-interactivity"}

type winget struct {
	bin string
	r   *run.Runner
}

func newWinget(r *run.Runner) (*winget, error) {
	bin, err := exec.LookPath("winget")
	if err != nil {
		return nil, ErrWingetMissing
	}
	return &winget{bin: bin, r: r}, nil
}

func (w *winget) Name() string { return "winget" }

func (w *winget) ID(p Package) string { return p.Winget }

func (w *winget) Installed(ctx context.Context, p Package) bool {
	args := append([]string{"list", "--id", p.Winget}, wingetCommon...)
	_, err := w.r.Output(ctx, w.bin, args...)
	return err == nil
}

func (w *winget) Install(ctx context.Context, p Package) error {
	args := append([]string{"install", "--id", p.Winget, "--silent", "--accept-package-agreements"}, wingetCommon...)
	return w.r.Run(ctx, w.bin, args...)
}

func (w *winget) BinDirs() []string {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		return nil
	}
	return []string{filepath.Join(local, "Microsoft", "WinGet", "Links")}
}
