package vault

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type fakeKeyring struct {
	items   map[string]string
	getErr  error
	setErr  error
	setCall int
}

func (f *fakeKeyring) Get(service, user string) (string, error) {
	if f.getErr != nil {
		return "", f.getErr
	}
	secret, ok := f.items[service+"/"+user]
	if !ok {
		return "", ErrKeyringItemMissing
	}
	return secret, nil
}

func (f *fakeKeyring) Set(service, user, secret string) error {
	f.setCall++
	if f.setErr != nil {
		return f.setErr
	}
	if f.items == nil {
		f.items = map[string]string{}
	}
	f.items[service+"/"+user] = secret
	return nil
}

func testPaths(t *testing.T) Paths {
	dir := t.TempDir()
	return Paths{Store: filepath.Join(dir, "secrets.enc"), KeyFile: filepath.Join(dir, "secret.key")}
}

func TestLoadDataKeyCreatesAndReusesAKeyringKey(t *testing.T) {
	ring := &fakeKeyring{}
	paths := testPaths(t)
	first, err := LoadDataKey(ring, paths)
	if err != nil || first.FromFile || !first.Created || len(first.Bytes) != KeySize {
		t.Fatalf("first = %+v, %v", first, err)
	}
	second, err := LoadDataKey(ring, paths)
	if err != nil || second.Created || !bytes.Equal(first.Bytes, second.Bytes) || ring.setCall != 1 {
		t.Fatalf("second = %+v, %v, set calls %d", second, err, ring.setCall)
	}
	if _, err := os.Stat(paths.KeyFile); err == nil {
		t.Fatal("key file written while the keyring works")
	}
}

func TestLoadDataKeyFallsBackToAFileWithoutKeyring(t *testing.T) {
	ring := &fakeKeyring{getErr: errors.New("no dbus")}
	paths := testPaths(t)
	first, err := LoadDataKey(ring, paths)
	if err != nil || !first.FromFile {
		t.Fatalf("first = %+v, %v", first, err)
	}
	second, err := LoadDataKey(ring, paths)
	if err != nil || !bytes.Equal(first.Bytes, second.Bytes) {
		t.Fatalf("second = %+v, %v", second, err)
	}
	if ring.setCall != 0 {
		t.Fatal("tried to write to an unreachable keyring")
	}
}

func TestLoadDataKeyFallsBackWhenTheKeyringRefusesWrites(t *testing.T) {
	ring := &fakeKeyring{setErr: errors.New("locked")}
	key, err := LoadDataKey(ring, testPaths(t))
	if err != nil || !key.FromFile {
		t.Fatalf("key = %+v, %v", key, err)
	}
}

func TestLoadDataKeyRefusesToReplaceTheKeyOfAnExistingStore(t *testing.T) {
	paths := testPaths(t)
	os.WriteFile(paths.Store, []byte("tbv1"), 0o600)
	if _, err := LoadDataKey(&fakeKeyring{getErr: errors.New("no dbus")}, paths); err != ErrKeyUnavailable {
		t.Fatalf("LoadDataKey = %v; want ErrKeyUnavailable", err)
	}
}

func TestLoadDataKeyRejectsAGarbledKey(t *testing.T) {
	ring := &fakeKeyring{items: map[string]string{keyringService + "/" + keyringUser: "short"}}
	if _, err := LoadDataKey(ring, testPaths(t)); err != ErrInvalidKey {
		t.Fatalf("LoadDataKey = %v; want ErrInvalidKey", err)
	}
}
