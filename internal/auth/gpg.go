package auth

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pmqueiroz/toolbelt/internal/dots"
	"github.com/pmqueiroz/toolbelt/internal/files"
	"github.com/pmqueiroz/toolbelt/internal/plan"
	"github.com/pmqueiroz/toolbelt/internal/run"
)

const GPGSettingsURL = "https://github.com/settings/gpg/new"

var ErrGPGMissing = errors.New("gpg is not installed, select the essentials recipe or install GnuPG")

//go:embed gpg_template.gpg
var gpgTemplate string

var windowsGPGDirs = []string{
	`C:\Program Files\GnuPG\bin`,
	`C:\Program Files (x86)\GnuPG\bin`,
	`C:\Program Files (x86)\gnupg\bin`,
}

func GitconfigLocal(home string) string {
	return filepath.Join(home, ".gitconfig.local")
}

func GPG(goos, home string, vars dots.Vars, r *run.Runner) []plan.Action {
	local := GitconfigLocal(home)
	return []plan.Action{{
		Description: fmt.Sprintf("create gpg key for %s (reuse if one exists) and set it as git signing key in %s", vars.Email, files.Pretty(local, home)),
		Run: func(ctx context.Context) error {
			bin, err := gpgBin(goos)
			if err != nil {
				return err
			}
			id, err := ensureGPGKey(ctx, r, bin, vars)
			if err != nil {
				return err
			}
			if err := r.Run(ctx, "git", "config", "--file", local, "user.signingkey", id); err != nil {
				return err
			}
			if goos == "windows" {
				return r.Run(ctx, "git", "config", "--file", local, "gpg.program", bin)
			}
			return nil
		},
	}}
}

func GPGPublicKey(ctx context.Context, goos string, r *run.Runner, email string) (string, error) {
	bin, err := gpgBin(goos)
	if err != nil {
		return "", err
	}
	id, err := secretKeyID(ctx, r, bin, email)
	if err != nil {
		return "", err
	}
	if id == "" {
		return "", fmt.Errorf("no gpg key for %s", email)
	}
	return r.Output(ctx, bin, "--armor", "--export", id)
}

func ensureGPGKey(ctx context.Context, r *run.Runner, bin string, vars dots.Vars) (string, error) {
	if id, err := secretKeyID(ctx, r, bin, vars.Email); err == nil && id != "" {
		r.Logf("gpg key %s already exists for %s. reusing", id, vars.Email)
		return id, nil
	}

	params, err := os.CreateTemp("", "toolbelt-gpg-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(params.Name())
	_, err = params.WriteString(dots.Render(gpgTemplate, vars))
	if closeErr := params.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", err
	}

	if err := r.Run(ctx, bin, "--batch", "--gen-key", params.Name()); err != nil {
		return "", err
	}

	id, err := secretKeyID(ctx, r, bin, vars.Email)
	if err == nil && id == "" {
		err = fmt.Errorf("gpg key for %s not found after generating it", vars.Email)
	}
	return id, err
}

func secretKeyID(ctx context.Context, r *run.Runner, bin, email string) (string, error) {
	out, err := r.Output(ctx, bin, "--list-secret-keys", "--with-colons", email)
	if err != nil {
		return "", nil
	}
	return LastSecretKeyID(out), nil
}

func LastSecretKeyID(colons string) string {
	id := ""
	for _, line := range strings.Split(colons, "\n") {
		fields := strings.Split(line, ":")
		if len(fields) > 4 && fields[0] == "sec" {
			id = fields[4]
		}
	}
	return id
}

func gpgBin(goos string) (string, error) {
	if bin, err := exec.LookPath("gpg"); err == nil {
		return bin, nil
	}
	if goos == "windows" {
		for _, dir := range windowsGPGDirs {
			bin := filepath.Join(dir, "gpg.exe")
			if _, err := os.Stat(bin); err == nil {
				return bin, nil
			}
		}
	}
	return "", ErrGPGMissing
}
