package files

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

var now = time.Date(2026, 10, 28, 23, 50, 0, 0, time.UTC)

func TestWriteWithBackupCreatesMissingFileAndParents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b", "settings.json")

	if err := WriteWithBackup(path, []byte("new"), now); err != nil {
		t.Fatal(err)
	}

	assertContent(t, path, "new")
	assertMissing(t, BackupPath(path, now))
}

func TestWriteWithBackupBacksUpDifferentContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".gitconfig")
	os.WriteFile(path, []byte("old"), 0o644)

	if err := WriteWithBackup(path, []byte("new"), now); err != nil {
		t.Fatal(err)
	}

	assertContent(t, path, "new")
	assertContent(t, filepath.Join(filepath.Dir(path), ".gitconfig.20261028_235000.bak"), "old")
}

func TestWriteWithBackupLeavesIdenticalFileAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".gitconfig")
	os.WriteFile(path, []byte("same"), 0o644)

	if err := WriteWithBackup(path, []byte("same"), now); err != nil {
		t.Fatal(err)
	}

	assertMissing(t, BackupPath(path, now))
}

func TestPretty(t *testing.T) {
	if got := Pretty("/home/peam/.bashrc", "/home/peam"); got != "~/.bashrc" {
		t.Errorf("Pretty = %q", got)
	}
	if got := Pretty("/etc/hosts", "/home/peam"); got != "/etc/hosts" {
		t.Errorf("Pretty = %q", got)
	}
}

func assertContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("%s = %q, want %q", path, got, want)
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Errorf("%s should not exist", path)
	}
}
