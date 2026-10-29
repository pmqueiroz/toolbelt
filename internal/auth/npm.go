package auth

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pmqueiroz/toolbelt/internal/files"
	"github.com/pmqueiroz/toolbelt/internal/plan"
	"github.com/pmqueiroz/toolbelt/internal/secret"
)

const (
	TokenSettingsURL = "https://github.com/settings/tokens/new"
	npmTokenPrefix   = "//npm.pkg.github.com/:_authToken="
)

func NPM(home, token string) []plan.Action {
	npmrc := filepath.Join(home, ".npmrc")
	masked, err := secret.Mask(token)
	if err != nil {
		masked = "****"
	}
	return []plan.Action{{
		Description: fmt.Sprintf("save github packages token %s to %s", masked, files.Pretty(npmrc, home)),
		Run: func(context.Context) error {
			existing, err := os.ReadFile(npmrc)
			if err != nil && !os.IsNotExist(err) {
				return err
			}
			return os.WriteFile(npmrc, []byte(UpsertNpmToken(string(existing), token)), 0o600)
		},
	}}
}

func UpsertNpmToken(npmrc, token string) string {
	var kept []string
	for _, line := range strings.Split(strings.TrimRight(npmrc, "\n"), "\n") {
		if line != "" && !strings.HasPrefix(strings.TrimSpace(line), npmTokenPrefix) {
			kept = append(kept, line)
		}
	}
	return strings.Join(append(kept, npmTokenPrefix+token), "\n") + "\n"
}
