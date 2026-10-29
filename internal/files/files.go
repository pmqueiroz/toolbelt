package files

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type State int

const (
	Missing State = iota
	Same
	Differs
)

func Compare(path string, content []byte) (State, error) {
	current, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Missing, nil
	}
	if err != nil {
		return 0, err
	}
	if bytes.Equal(current, content) {
		return Same, nil
	}
	return Differs, nil
}

func BackupPath(path string, now time.Time) string {
	return fmt.Sprintf("%s.%s.bak", path, now.Format("20060102_150405"))
}

func WriteWithBackup(path string, content []byte, now time.Time) error {
	state, err := Compare(path, content)
	if err != nil {
		return err
	}
	switch state {
	case Same:
		return nil
	case Differs:
		if err := copyFile(path, BackupPath(path, now)); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func WriteAtomic(path string, content []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func Pretty(path, home string) string {
	if home != "" && strings.HasPrefix(path, home) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}

func copyFile(from, to string) error {
	data, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	return os.WriteFile(to, data, 0o644)
}
