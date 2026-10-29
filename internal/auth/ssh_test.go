package auth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pmqueiroz/toolbelt/internal/run"
)

func TestSSHSkipsExistingKey(t *testing.T) {
	home := t.TempDir()
	r := run.New(false, nil)

	if got := SSH(home, "peam@example.com", r); len(got) != 1 || got[0].Description != "create ssh key ~/.ssh/id_ed25519" {
		t.Fatalf("missing key plan = %v", got)
	}

	os.MkdirAll(filepath.Join(home, ".ssh"), 0o700)
	os.WriteFile(sshKeyPath(home), []byte("private"), 0o600)
	os.WriteFile(sshKeyPath(home)+".pub", []byte("ssh-ed25519 AAAA peam@example.com\n"), 0o644)

	if got := SSH(home, "peam@example.com", r); len(got) != 0 {
		t.Errorf("existing key plan = %v", got)
	}
	if key, err := SSHPublicKey(home); err != nil || key != "ssh-ed25519 AAAA peam@example.com" {
		t.Errorf("SSHPublicKey = %q, %v", key, err)
	}
}
