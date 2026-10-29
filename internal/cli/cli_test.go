package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestSplitCommitArgs(t *testing.T) {
	options, message := splitCommitArgs([]string{"--no-verify", "fix", "the", "thing", "-S"})
	if !reflect.DeepEqual(options, []string{"--no-verify", "-S"}) {
		t.Errorf("options = %v", options)
	}
	if message != "fix the thing" {
		t.Errorf("message = %q", message)
	}

	options, message = splitCommitArgs(nil)
	if options != nil || message != "" {
		t.Errorf("empty args gave %v %q", options, message)
	}
}

func TestMask(t *testing.T) {
	cases := map[string]string{
		"abcdefgh":     "abcdefgh",
		"abcd1234wxyz": "abcd****wxyz",
	}
	for in, want := range cases {
		got, err := mask(in)
		if err != nil || got != want {
			t.Errorf("mask(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := mask("short"); err == nil {
		t.Error("expected error for short value")
	}
}

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
