package vault

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func testKey(fill byte) []byte {
	return bytes.Repeat([]byte{fill}, KeySize)
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "toolbelt", "secrets.enc")
	store := &Store{Connection: &Connection{Token: "tok", ProjectID: "p"}}
	if err := store.Set("GITHUB_TOKEN", Entry{Value: "ghp_123", ID: "id-1"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(path, testKey(1)); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path, testKey(1))
	if err != nil {
		t.Fatal(err)
	}
	entry, err := loaded.Get("GITHUB_TOKEN")
	if err != nil || entry != (Entry{Value: "ghp_123", ID: "id-1"}) {
		t.Fatalf("Get = %+v, %v", entry, err)
	}
	if !loaded.Connected() || loaded.Connection.Token != "tok" {
		t.Fatalf("connection = %+v", loaded.Connection)
	}
}

func TestSaveDoesNotWritePlaintext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.enc")
	store := &Store{}
	store.Set("NAME", Entry{Value: "plaintext-value"})
	if err := store.Save(path, testKey(1)); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if bytes.Contains(raw, []byte("plaintext-value")) || bytes.Contains(raw, []byte("NAME")) {
		t.Fatal("store file contains plaintext")
	}
}

func TestSaveRestrictsPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permissions")
	}
	path := filepath.Join(t.TempDir(), "secrets.enc")
	if err := (&Store{}).Save(path, testKey(1)); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v; want 0600", info.Mode().Perm())
	}
}

func TestLoadWithAnotherKeyFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.enc")
	if err := (&Store{}).Save(path, testKey(1)); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path, testKey(2)); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Load with another key = %v; want ErrCorrupt", err)
	}
}

func TestLoadOfAMissingFileIsEmpty(t *testing.T) {
	store, err := Load(filepath.Join(t.TempDir(), "none.enc"), testKey(1))
	if err != nil || store.Connected() || len(store.Keys()) != 0 {
		t.Fatalf("Load = %+v, %v", store, err)
	}
}

func TestGetOfAMissingKeyFails(t *testing.T) {
	if _, err := (&Store{}).Get("NOPE"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get = %v; want ErrNotFound", err)
	}
}

func TestSetRejectsEmptyKeyOrValue(t *testing.T) {
	store := &Store{}
	if err := store.Set("", Entry{Value: "v"}); err != ErrEmptyKey {
		t.Fatalf("Set empty key = %v", err)
	}
	if err := store.Set("k", Entry{}); err != ErrEmptySecret {
		t.Fatalf("Set empty value = %v", err)
	}
}

func TestKeysAreSorted(t *testing.T) {
	store := &Store{}
	for _, key := range []string{"b", "c", "a"} {
		store.Set(key, Entry{Value: "v"})
	}
	if got := store.Keys(); len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Fatalf("Keys = %v", got)
	}
}
