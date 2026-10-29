package cli

import (
	"context"
	"errors"
	"testing"
)

func TestUpdateRefusesDevBuild(t *testing.T) {
	if err := update(context.Background(), "dev"); !errors.Is(err, errDevBuild) {
		t.Errorf("err = %v, want errDevBuild", err)
	}
}
