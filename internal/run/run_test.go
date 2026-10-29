package run

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
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

func TestRetryStopsAtTheFirstSuccess(t *testing.T) {
	calls := 0
	err := Retry(context.Background(), 3, 0, func() error {
		calls++
		if calls < 2 {
			return errors.New("503")
		}
		return nil
	})
	if err != nil || calls != 2 {
		t.Fatalf("Retry = %v after %d calls; want nil after 2", err, calls)
	}
}

func TestRetryReturnsTheLastErrorAfterEveryAttempt(t *testing.T) {
	calls := 0
	err := Retry(context.Background(), 3, 0, func() error {
		calls++
		return fmt.Errorf("attempt %d", calls)
	})
	if err == nil || err.Error() != "attempt 3" || calls != 3 {
		t.Fatalf("Retry = %v after %d calls", err, calls)
	}
}

func TestRetryStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	err := Retry(ctx, 3, time.Hour, func() error {
		calls++
		return errors.New("503")
	})
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatalf("Retry = %v after %d calls", err, calls)
	}
}
