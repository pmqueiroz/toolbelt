# CI/CD

GitHub Actions in `.github/workflows/`. Go version comes from `go.mod` via
`actions/setup-go` (`go-version-file`), so bumping `go.mod` bumps CI.

## Workflows

- **`ci.yaml`** — every PR and push to `main`; in-flight runs for the same
  ref are cancelled.
  - `test` — matrix on ubuntu, macos, windows: `go vet`, `go test`; gofmt
    check on linux only. Windows matters: path separators and profile logic
    differ, so tests compare paths with `filepath.FromSlash`.
  - `scripts` — `shellcheck scripts/install.sh`.
  - `snapshot` — `goreleaser release --snapshot --clean` to prove every
    target still builds. Needs `test`.
- **`release.yaml`** — on tag `v*`: tests, then GoReleaser publishes the
  GitHub release. `.goreleaser.yaml` builds linux/darwin/windows ×
  amd64/arm64 with `CGO_ENABLED=0`, archives `tb_<os>_<arch>.tar.gz`
  (`.zip` on windows) containing only the binary, plus `checksums.txt`.
- **`pages.yaml`** — on push to `main` touching `scripts/`: deploys
  `scripts/` to GitHub Pages. Custom domain `toolbelt.pmqueiroz.dev`
  (Cloudflare DNS, `CNAME toolbelt → pmqueiroz.github.io`, proxy **off** so
  GitHub can issue the certificate), HTTPS enforced.

## Contracts between pieces

These names are load-bearing; change them together or the installers and
`tb update` break:

- Archive names `tb_<os>_<arch>` — `scripts/install.sh`,
  `scripts/install.ps1` and go-selfupdate (suffix match) depend on them.
- `checksums.txt` — both installers and `tb update` verify against it.
- Repo `pmqueiroz/toolbelt` — `.goreleaser.yaml`, both scripts,
  `internal/cli/update.go`, and the module path.

Installers honour `TOOLBELT_BASE_URL` (point at a local `dist/` served over
http to test a snapshot), `TOOLBELT_INSTALL_DIR`, `TOOLBELT_SKIP_BREW`.

Installers only install: they never start `tb setup`, they print the
command to run instead (the full path when the install dir is not yet on
PATH, which `tb setup`'s shell block is what fixes on linux/macOS).

## Releasing

`git tag vX.Y.Z && git push origin vX.Y.Z`. Users get it from the
one-liners or `tb update`.

## Not set up yet

- No code signing or notarization: macOS Gatekeeper doesn't block binaries
  fetched with curl, and Windows SmartScreen doesn't apply to files fetched
  by `Invoke-WebRequest`, so it isn't needed for the one-liner path.
- No package manager distribution (brew tap, winget manifest).
- The old `release` branch on the remote (bundled bash installer) is no
  longer updated and can be deleted.
