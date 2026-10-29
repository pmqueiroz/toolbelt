package vault

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "toolbelt"
	keyringUser    = "secret-store"
)

var (
	ErrKeyringItemMissing = errors.New("keyring item missing")
	ErrKeyUnavailable     = errors.New("the secret store exists but its key is not in the os keyring nor in the key file")
)

type Keyring interface {
	Get(service, user string) (string, error)
	Set(service, user, secret string) error
}

type KeyringDeleter interface {
	Delete(service, user string) error
}

type Paths struct {
	Store   string
	KeyFile string
}

func DefaultPaths() (Paths, error) {
	config, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, err
	}
	dir := filepath.Join(config, "toolbelt")
	return Paths{Store: filepath.Join(dir, "secrets.enc"), KeyFile: filepath.Join(dir, "secret.key")}, nil
}

type OSKeyring struct{}

func (OSKeyring) Get(service, user string) (string, error) {
	secret, err := keyring.Get(service, user)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrKeyringItemMissing
	}
	return secret, err
}

func (OSKeyring) Set(service, user, secret string) error {
	return keyring.Set(service, user, secret)
}

func (OSKeyring) Delete(service, user string) error {
	err := keyring.Delete(service, user)
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrKeyringItemMissing
	}
	return err
}

func Purge(ring KeyringDeleter, paths Paths) error {
	_, keyFileErr := os.Stat(paths.KeyFile)
	keyLivedInFile := keyFileErr == nil

	for _, path := range []string{paths.Store, paths.KeyFile} {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	removeIfEmpty(filepath.Dir(paths.Store))

	err := ring.Delete(keyringService, keyringUser)
	if err == nil || errors.Is(err, ErrKeyringItemMissing) || keyLivedInFile {
		return nil
	}
	return err
}

func removeIfEmpty(dir string) {
	entries, err := os.ReadDir(dir)
	if err == nil && len(entries) == 0 {
		os.Remove(dir)
	}
}

type DataKey struct {
	Bytes    []byte
	FromFile bool
	Created  bool
}

func LoadDataKey(ring Keyring, paths Paths) (DataKey, error) {
	encoded, err := ring.Get(keyringService, keyringUser)
	if err == nil {
		key, err := decodeKey(encoded)
		return DataKey{Bytes: key}, err
	}
	keyringReachable := errors.Is(err, ErrKeyringItemMissing)

	if encoded, err := os.ReadFile(paths.KeyFile); err == nil {
		key, err := decodeKey(string(encoded))
		return DataKey{Bytes: key, FromFile: true}, err
	} else if !errors.Is(err, fs.ErrNotExist) {
		return DataKey{}, err
	}

	if _, err := os.Stat(paths.Store); err == nil {
		return DataKey{}, ErrKeyUnavailable
	}

	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return DataKey{}, err
	}
	encoded = base64.StdEncoding.EncodeToString(key)
	if keyringReachable && ring.Set(keyringService, keyringUser, encoded) == nil {
		return DataKey{Bytes: key, Created: true}, nil
	}
	if err := writeAtomic(paths.KeyFile, []byte(encoded+"\n")); err != nil {
		return DataKey{}, err
	}
	return DataKey{Bytes: key, FromFile: true, Created: true}, nil
}

func decodeKey(encoded string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil || len(key) != KeySize {
		return nil, ErrInvalidKey
	}
	return key, nil
}
