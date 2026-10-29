package run

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunLogsCommandAndOutput(t *testing.T) {
	var log bytes.Buffer
	r := New(false, &log)

	if err := r.Run(context.Background(), "go", "version"); err != nil {
		t.Fatal(err)
	}

	got := log.String()
	if !strings.Contains(got, "running command: go version") {
		t.Errorf("missing command line in log: %q", got)
	}
	if !strings.Contains(got, "go version go") {
		t.Errorf("missing command output in log: %q", got)
	}
}

func TestOutputTrimsStdout(t *testing.T) {
	r := New(false, nil)

	out, err := r.Output(context.Background(), "go", "env", "GOOS")
	if err != nil {
		t.Fatal(err)
	}
	if out == "" || strings.ContainsAny(out, "\n ") {
		t.Errorf("unexpected output %q", out)
	}
}

func TestRunWrapsError(t *testing.T) {
	r := New(false, nil)

	err := r.Run(context.Background(), "go", "definitely-not-a-command")
	if err == nil || !strings.Contains(err.Error(), "go definitely-not-a-command") {
		t.Errorf("expected wrapped error, got %v", err)
	}
}
