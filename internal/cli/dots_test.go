package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pmqueiroz/toolbelt/internal/dots"
)

func TestSyncDotsDoesNothingBeforeSetup(t *testing.T) {
	root := t.TempDir()
	paths := dots.Paths{Home: filepath.Join(root, "home"), Config: filepath.Join(root, "config"), Store: filepath.Join(root, "store")}

	if err := syncDots(context.Background(), "linux", paths, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(paths.Store); !os.IsNotExist(err) {
		t.Errorf("store should not be created, stat err = %v", err)
	}
}
