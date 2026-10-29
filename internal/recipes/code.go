package recipes

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pmqueiroz/toolbelt/internal/pkgmgr"
	"github.com/pmqueiroz/toolbelt/internal/plan"
	"github.com/pmqueiroz/toolbelt/internal/run"
)

var ErrCodeMissing = errors.New("make sure vscode is installed and the code command is added to PATH")

var vscodePackage = pkgmgr.Package{Name: "vscode", Brew: "visual-studio-code", Cask: true, Winget: "Microsoft.VisualStudioCode"}

var codeExtensions = []string{
	"naumovs.color-highlight",
	"dracula-theme.theme-dracula",
	"DaltonMenezes.aura-theme",
	"pkief.material-icon-theme",
	"styled-components.vscode-styled-components",
	"wix.vscode-import-cost",
	"streetsidesoftware.code-spell-checker",
	"streetsidesoftware.code-spell-checker-portuguese-brazilian",
	"eamodio.gitlens",
	"moyu.snapcode",
	"formulahendry.auto-rename-tag",
	"dbaeumer.vscode-eslint",
	"yoavbls.pretty-ts-errors",
	"miguelsolorio.fluent-icons",
	"drcika.apc-extension",
	"rust-lang.rust-analyzer",
}

var code = Recipe{
	Name:    "code",
	Default: true,
	Plan: func(env Env) []plan.Action {
		return append(installPackages(env, vscodePackage), plan.Action{
			Description: fmt.Sprintf("install %d vscode extensions", len(codeExtensions)),
			Run: func(ctx context.Context) error {
				return installExtensions(ctx, env)
			},
		})
	},
}

func installExtensions(ctx context.Context, env Env) error {
	prependCodeBin(env.GOOS)
	if !run.Exists("code") {
		return ErrCodeMissing
	}

	listed, err := env.Run.Output(ctx, "code", "--list-extensions")
	if err != nil {
		return err
	}

	var failed []string
	for _, ext := range missingExtensions(strings.Split(listed, "\n"), codeExtensions) {
		if err := env.Run.Run(ctx, "code", "--install-extension", ext); err != nil {
			failed = append(failed, ext)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("failed to install extensions: %s", strings.Join(failed, ", "))
	}
	return nil
}

func missingExtensions(installed, wanted []string) []string {
	have := map[string]bool{}
	for _, ext := range installed {
		have[strings.ToLower(strings.TrimSpace(ext))] = true
	}
	var missing []string
	for _, ext := range wanted {
		if !have[strings.ToLower(ext)] {
			missing = append(missing, ext)
		}
	}
	return missing
}

func prependCodeBin(goos string) {
	dir := codeBinDir(goos)
	if dir == "" {
		return
	}
	os.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func codeBinDir(goos string) string {
	switch goos {
	case "darwin":
		return "/Applications/Visual Studio Code.app/Contents/Resources/app/bin"
	case "windows":
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			return filepath.Join(local, "Programs", "Microsoft VS Code", "bin")
		}
	}
	return ""
}
