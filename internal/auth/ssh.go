package auth

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pmqueiroz/toolbelt/internal/files"
	"github.com/pmqueiroz/toolbelt/internal/plan"
	"github.com/pmqueiroz/toolbelt/internal/run"
)

const SSHSettingsURL = "https://github.com/settings/ssh/new"

func sshKeyPath(home string) string {
	return filepath.Join(home, ".ssh", "id_ed25519")
}

func SSH(home, email string, r *run.Runner) []plan.Action {
	key := sshKeyPath(home)
	if _, err := os.Stat(key); err == nil {
		return nil
	}
	return []plan.Action{{
		Description: fmt.Sprintf("create ssh key %s", files.Pretty(key, home)),
		Run: func(ctx context.Context) error {
			if err := os.MkdirAll(filepath.Dir(key), 0o700); err != nil {
				return err
			}
			return r.Run(ctx, "ssh-keygen", "-t", "ed25519", "-C", email, "-f", key, "-q", "-N", "")
		},
	}}
}

func SSHPublicKey(home string) (string, error) {
	data, err := os.ReadFile(sshKeyPath(home) + ".pub")
	return strings.TrimSpace(string(data)), err
}
