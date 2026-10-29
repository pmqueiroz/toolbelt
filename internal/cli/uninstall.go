package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/pmqueiroz/toolbelt/internal/files"
	"github.com/pmqueiroz/toolbelt/internal/plan"
	"github.com/pmqueiroz/toolbelt/internal/setup"
	"github.com/pmqueiroz/toolbelt/internal/shellrc"
	"github.com/pmqueiroz/toolbelt/internal/ui"
	"github.com/pmqueiroz/toolbelt/internal/vault"
	"github.com/spf13/cobra"
)

var errUninstallNeedsConfirmation = errors.New("tb uninstall needs an interactive terminal to confirm, pass --yes to skip the prompt")

type uninstallEnv struct {
	goos       string
	shell      shellrc.Env
	secrets    vault.Paths
	keyring    vault.KeyringDeleter
	executable string
	tempDir    string
	now        time.Time
}

func defaultUninstallEnv() (uninstallEnv, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return uninstallEnv{}, err
	}
	secrets, err := vault.DefaultPaths()
	if err != nil {
		return uninstallEnv{}, err
	}
	exe, err := os.Executable()
	if err != nil {
		return uninstallEnv{}, err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return uninstallEnv{
		goos:       runtime.GOOS,
		shell:      shellrc.DefaultEnv(runtime.GOOS, home),
		secrets:    secrets,
		keyring:    vault.OSKeyring{},
		executable: exe,
		tempDir:    os.TempDir(),
		now:        time.Now(),
	}, nil
}

func newUninstallCmd() *cobra.Command {
	var purge, yes bool
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove tb and its shell block; --purge also deletes the secret store",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			env, err := defaultUninstallEnv()
			if err != nil {
				return err
			}
			return runUninstall(cmd.Context(), env, purge, yes)
		},
	}
	cmd.Flags().BoolVar(&purge, "purge", false, "also delete the local secret store and its os keyring key")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the confirmation prompt")
	return cmd
}

func runUninstall(ctx context.Context, env uninstallEnv, purge, yes bool) error {
	phases, err := uninstallPlan(env, purge)
	if err != nil {
		return err
	}

	ui.Println(setup.Render(phases))
	if _, err := os.Stat(env.secrets.Store); !purge && err == nil {
		ui.Warn("the secret store is kept, pass --purge to delete it with tb", "path", files.Pretty(env.secrets.Store, env.shell.Home))
	}
	if !yes {
		if !ui.Interactive() {
			return errUninstallNeedsConfirmation
		}
		if !ui.Confirm("uninstall tb?") {
			ui.Warn("uninstall cancelled. nothing changed")
			return nil
		}
	}

	if failures := setup.Execute(ctx, phases, false); len(failures) > 0 {
		return fmt.Errorf("%d uninstall step(s) failed", len(failures))
	}
	ui.Info("tb uninstalled. packages, dot files and keys from tb setup were left in place")
	return nil
}

func uninstallPlan(env uninstallEnv, purge bool) ([]setup.Phase, error) {
	shellActions, err := shellrc.RemovalPlan(env.shell, env.now)
	if err != nil {
		return nil, err
	}

	var phases []setup.Phase
	if len(shellActions) > 0 {
		phases = append(phases, setup.Phase{Title: "shell sources", Actions: shellActions})
	}
	if purge {
		phases = append(phases, setup.Phase{Title: "secrets", Actions: []plan.Action{{
			Description: fmt.Sprintf("delete secret store %s and its os keyring key", files.Pretty(filepath.Dir(env.secrets.Store), env.shell.Home)),
			Run: func(context.Context) error {
				return vault.Purge(env.keyring, env.secrets)
			},
		}}})
	}
	phases = append(phases, setup.Phase{Title: "binary", Actions: []plan.Action{{
		Description: "delete " + files.Pretty(env.executable, env.shell.Home),
		Run: func(context.Context) error {
			return removeExecutable(env)
		},
	}}})
	return phases, nil
}

func removeExecutable(env uninstallEnv) error {
	if env.goos != "windows" {
		return os.Remove(env.executable)
	}
	parked := filepath.Join(env.tempDir, fmt.Sprintf("tb_uninstalled_%s.exe", env.now.Format("20060102_150405")))
	return os.Rename(env.executable, parked)
}
