package vault

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/pmqueiroz/toolbelt/internal/files"
)

const KeySize = 32

var (
	magic          = []byte("tbv1")
	ErrNotFound    = errors.New("secret not found")
	ErrCorrupt     = errors.New("secret store is corrupt or was encrypted with another key")
	ErrInvalidKey  = fmt.Errorf("data key must be %d bytes", KeySize)
	ErrEmptyKey    = errors.New("secret key must not be empty")
	ErrEmptySecret = errors.New("secret value must not be empty")
)

type Connection struct {
	Token          string `json:"token"`
	Region         string `json:"region"`
	OrganizationID string `json:"organizationId"`
	ProjectID      string `json:"projectId"`
	ProjectName    string `json:"projectName"`
}

type AIConnection struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIKey   string `json:"apiKey,omitempty"`
	BaseURL  string `json:"baseUrl,omitempty"`
}

type Entry struct {
	Value string `json:"value"`
	ID    string `json:"id,omitempty"`
}

type Store struct {
	Connection *Connection      `json:"connection,omitempty"`
	AI         *AIConnection    `json:"ai,omitempty"`
	Entries    map[string]Entry `json:"entries"`
}

func (s *Store) Connected() bool {
	return s.Connection != nil
}

func (s *Store) Get(key string) (Entry, error) {
	entry, ok := s.Entries[key]
	if !ok {
		return Entry{}, fmt.Errorf("%w: %s", ErrNotFound, key)
	}
	return entry, nil
}

func (s *Store) Set(key string, entry Entry) error {
	if key == "" {
		return ErrEmptyKey
	}
	if entry.Value == "" {
		return ErrEmptySecret
	}
	if s.Entries == nil {
		s.Entries = map[string]Entry{}
	}
	s.Entries[key] = entry
	return nil
}

func (s *Store) Keys() []string {
	keys := make([]string, 0, len(s.Entries))
	for key := range s.Entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func Load(path string, key []byte) (*Store, error) {
	sealed, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &Store{Entries: map[string]Entry{}}, nil
	}
	if err != nil {
		return nil, err
	}
	plaintext, err := open(key, sealed)
	if err != nil {
		return nil, err
	}
	store := &Store{}
	if err := json.Unmarshal(plaintext, store); err != nil {
		return nil, ErrCorrupt
	}
	if store.Entries == nil {
		store.Entries = map[string]Entry{}
	}
	return store, nil
}

func (s *Store) Save(path string, key []byte) error {
	plaintext, err := json.Marshal(s)
	if err != nil {
		return err
	}
	sealed, err := seal(key, plaintext)
	if err != nil {
		return err
	}
	return writeAtomic(path, sealed)
}

func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func seal(key, plaintext []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	header := append(bytes.Clone(magic), nonce...)
	return gcm.Seal(header, nonce, plaintext, magic), nil
}

func open(key, sealed []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if len(sealed) < len(magic)+gcm.NonceSize() || !bytes.Equal(sealed[:len(magic)], magic) {
		return nil, ErrCorrupt
	}
	body := sealed[len(magic):]
	plaintext, err := gcm.Open(nil, body[:gcm.NonceSize()], body[gcm.NonceSize():], magic)
	if err != nil {
		return nil, ErrCorrupt
	}
	return plaintext, nil
}

func writeAtomic(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return files.WriteAtomic(path, content, 0o600)
}
