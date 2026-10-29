package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFingerprintChangesOnWrite(t *testing.T) {
	file := filepath.Join(t.TempDir(), "watched")
	missing := fingerprint([]string{file})

	if err := os.WriteFile(file, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	created := fingerprint([]string{file})
	if created == missing {
		t.Error("fingerprint did not change when file was created")
	}

	later := time.Now().Add(time.Second)
	if err := os.WriteFile(file, []byte("ab"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(file, later, later); err != nil {
		t.Fatal(err)
	}
	if fingerprint([]string{file}) == created {
		t.Error("fingerprint did not change when file was modified")
	}
}
