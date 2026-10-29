package run

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Runner struct {
	Verbose bool
	Log     io.Writer
	mu      sync.Mutex
}

func New(verbose bool, log io.Writer) *Runner {
	if log == nil {
		log = io.Discard
	}
	return &Runner{Verbose: verbose, Log: log}
}

func OpenLogFile() (*os.File, error) {
	name := fmt.Sprintf("toolbelt_%s.log", time.Now().Format("20060102_150405"))
	return os.Create(filepath.Join(os.TempDir(), name))
}

func (r *Runner) Run(ctx context.Context, name string, args ...string) error {
	cmd := r.command(ctx, name, args...)
	if r.Verbose {
		cmd.Stdout = io.MultiWriter(os.Stdout, r.Log)
		cmd.Stderr = io.MultiWriter(os.Stderr, r.Log)
	} else {
		cmd.Stdout = r.Log
		cmd.Stderr = r.Log
	}
	return wrap(cmd.Run(), name, args)
}

func (r *Runner) Output(ctx context.Context, name string, args ...string) (string, error) {
	cmd := r.command(ctx, name, args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = r.Log
	err := cmd.Run()
	return strings.TrimSpace(stdout.String()), wrap(err, name, args)
}

func (r *Runner) Attached(ctx context.Context, name string, args ...string) error {
	cmd := r.command(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return wrap(cmd.Run(), name, args)
}

func (r *Runner) Logf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fmt.Fprintf(r.Log, format+"\n", args...)
}

func (r *Runner) command(ctx context.Context, name string, args ...string) *exec.Cmd {
	r.Logf("running command: %s", strings.Join(append([]string{name}, args...), " "))
	return exec.CommandContext(ctx, name, args...)
}

func Exists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func wrap(err error, name string, args []string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
}
