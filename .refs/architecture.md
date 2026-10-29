# Architecture

toolbelt is one Go binary, `tb`, that sets up a Linux, macOS or Windows
machine (`tb setup`) and carries everyday git helpers (`tb pr`, `tb checkout`,
…). Mental model: every setup concern *describes* what it would do as a list
of `plan.Action`s; `tb setup` collects them, shows them, and only runs them
after confirmation.

Code is the truth. This file explains the *why* behind the shape it takes.

## Layout

- `cmd/tb/main.go` — entry point. `version`/`commit`/`date` are stamped by
  GoReleaser ldflags; a plain `go build` reports `dev`.
- `internal/cli/` — one cobra command per file (`setup`, `pr`, `commit`,
  `checkout`, `hot`, `benchmark`, `env`, `secret`, `update`, `version`), wired in `root.go`.
  `root.go` runs everything under a context cancelled by Ctrl+C.
- `internal/ui/` — the only place terminal styling lives: `log` wrapper
  (kitchen timestamps, like the old `gum log`), lipgloss styles,
  `Confirm`, `Spin` (falls back to a plain log line without a TTY).
- `internal/run/` — `Runner` executes commands. `Run` sends output to the
  log file (or the terminal too with `--verbose`), `Output` captures stdout,
  `Attached` wires stdin/stdout for interactive commands (git pull, gh).
- `internal/plan/` — `Action{Description, Run}` and `Dedupe`.
- `internal/pkgmgr/` — `Manager` interface with `brew` (linux, darwin) and
  `winget` (windows).
- `internal/recipes/` — installable units (see below).
- `internal/dots/` — embedded dot files (`files/`) and their per-OS targets.
- `internal/shellrc/` — the managed block in shell rc files (`blocks/`).
- `internal/auth/` — ssh key, npm token, gpg signing key.
- `internal/files/` — compare / write-with-backup / `~` pretty paths, shared
  by dots and shellrc.
- `internal/secret/` — masking used by `tb env`, `tb secret list` and the npm plan line.
  `Mask` keeps the first/last 4 chars and rejects values under 8; `Redact`
  stars short values fully; `MaskEnviron` masks each `NAME=value` of
  `os.Environ()` (names starting with `=`, Windows drive vars, split on the
  next `=`) for `tb env` with no argument.
- `internal/bitwarden/` — pure-Go Bitwarden Secrets Manager client: access
  token parsing, key derivation, EncString crypto, login and the project /
  secret endpoints (see [secret.md](./secret.md)).
- `internal/vault/` — the `tb secret` local store: AES-GCM file, data key in
  the OS keyring (file fallback), pure `Merge` of local and remote entries.
- `internal/bench/` — `tb benchmark`: uuid workers, the base64-indexed bit
  bucket, estimates and per-OS memory/disk size (see
  [benchmark.md](./benchmark.md)).
- `internal/gitx/` — git/gh queries and the pure rules behind `tb checkout`.
- `internal/setup/` — the `tb setup` flow: form, plan, execute, summary.
- `scripts/` — `install.sh` / `install.ps1` bootstrap one-liners, served by
  GitHub Pages at `https://toolbelt.pmqueiroz.dev` (`CNAME` file included).
- `dots/keychron_k6_pro.json` — keyboard config kept for reference; nothing
  installs it.

## tb setup

1. **Form** (`setup.Ask`): one huh form, nothing runs before it is
   submitted. Identity is prefilled from the global git config. Groups for
   recipes, npm token and gpg passphrase are hidden unless their step is
   selected. There are no skip flags; unselecting a step is the skip.
2. **Plan** (`setup.BuildPlan`): `Answers` + `Context` → ordered `[]Phase`:
   shell sources → dependencies → dot files → ssh key → npm token → gpg key.
   Steps whose work is already done contribute no actions (identical dot,
   existing ssh key, rc block already current), so a rerun shows only real
   changes. Building the plan only reads; it never writes.
3. **Confirm**: `setup.Render` shows every `Description`; "no" exits with
   nothing changed, which doubles as a dry run.
4. **Execute**: each action under a spinner; failures are collected, not
   fatal, and reported with the log path at the end (exit code 1).
5. **Summary**: done banner plus ssh/gpg public keys and the GitHub settings
   URLs to paste them into.

The plan shown is the plan run: both read the same `[]Phase`. Descriptions
must never contain secrets (`TestRenderListsEveryActionAndHidesSecrets`).

`tb setup` refuses to run without a TTY; the bootstrap scripts reattach
`/dev/tty` because their own stdin is the pipe.

## The extension point: recipes

`Recipe{Name, Default, OS, Plan func(Env) []plan.Action}` in
`internal/recipes/`, one file per recipe, registered in order in `All()`
(`recipe.go`). Order is install order.

Packages are declared once with both IDs:
`pkgmgr.Package{Name, Brew, Cask, Winget, OS}`. `installPackages` drops
packages the current manager can't install (casks on linux, brew-only tools
on windows, `OS`-restricted ones), and at run time skips a package when its
`Name` is already on PATH or the manager reports it installed — so tools
installed some other way (apt, manual) are not reinstalled.

Adding a recipe: create `internal/recipes/<name>.go` with a `Recipe` var,
add it to `All()`, give every package both a brew name and a winget ID
(verify both exist), add a `<name>_test.go`.

`code` installs extensions one at a time, each retried 3 times with 2s/4s
backoff (`run.Retry`): the marketplace answers 503 now and then, and one
flaky extension used to fail the whole action.

Current recipes: `essentials`, `node`, `pnpm`, `code` (default) and `apps`,
`android` (macOS/Windows only, because brew casks don't exist on linux).
`node` and `pnpm` both start with "install mise"; `plan.Dedupe` keeps one.

Why mise instead of asdf: asdf has no Windows support; mise runs everywhere
with the same commands. Why fastfetch: neofetch was removed from Homebrew.

## Package managers

- brew is found on PATH or at `/opt/homebrew`, `/usr/local`,
  `/home/linuxbrew/.linuxbrew`, `~/.linuxbrew`. `tb` does not install brew:
  that needs sudo and an interactive prompt, so `scripts/install.sh` does it
  before `tb setup`.
- winget is assumed present (Windows 10 1809+/11).
- `PrependPath` puts the manager's bin dir first in this process's PATH so
  tools installed mid-run (mise, gpg) are usable by later actions.
- winget doesn't always put a portable package's alias in `WinGet\Links`;
  it often edits the user PATH in the registry instead ("path environment
  variable modified, restart your shell"), which the running process never
  sees. So after every winget install, `RefreshPath` appends the persisted
  user + machine `Path` (`HKCU\Environment`, `HKLM\...\Session
  Manager\Environment`, `%VAR%` expanded) to this process's PATH. Without
  it `mise use` failed right after `winget install jdx.mise`. No-op off
  Windows.

A missing manager only fails the plan if the dependencies step is selected.

## Dot files

Embedded with `//go:embed all:files` (`all:` because they are dotfiles).
`{{username}}`, `{{email}}`, `{{passphrase}}` are replaced by `dots.Render`.
Targets use `os.UserConfigDir()`, which gives the right VS Code folder per OS.
`.tmux.conf` is skipped on Windows. Changed files are backed up to
`<file>.<timestamp>.bak`; identical files are left alone.

`.gitconfig` includes `~/.gitconfig.local`. Machine-specific git settings
(`user.signingkey`, Windows `gpg.program`) go there, so rerunning the dots
step never wipes the signing key.

## Shell sources

`shellrc.Targets` picks `~/.bashrc` and/or `~/.zshrc` (whichever exist, plus
the one matching `$SHELL`, bash as fallback) or, on Windows, the
`$PROFILE.CurrentUserCurrentHost` of each installed PowerShell (`pwsh`,
`powershell`). The block between `# ---BEGIN TOOLBELT---` and
`# ---END TOOLBELT---` is replaced in place; the legacy
`# ---BEGIN DOTFILES SOURCE---` block from the old bash installer is removed.
Block contents: `~/.local/bin` on PATH, brew shellenv, mise activate, zoxide
init, history size, `glo`/`repo`, and aliases from `pr`, `commit`,
`checkout`, `hot` to `tb`. `tb env` gets no alias on purpose: it would shadow
the system `env` (`env FOO=1 cmd`, `env -i`). Every tool line is guarded so a
missing tool doesn't break the shell.

## Auth

- ssh: `ssh-keygen -t ed25519` only when `~/.ssh/id_ed25519` is missing.
- npm: writes `//npm.pkg.github.com/:_authToken=` into `~/.npmrc` (mode
  0600), replacing a previous line. No `npm-cli-login`, so no Node needed.
- gpg: reuses an existing secret key for the email, otherwise generates one
  from `auth/gpg_template.gpg` via a temp file that is always removed (it
  holds the passphrase). On Windows, git is pointed at GnuPG's `gpg.exe`
  because Git for Windows ships its own gpg.

## Git helpers

`tb checkout` deletes every other local branch. A branch checked out in a
worktree is kept when it has uncommitted changes, was never pushed, or has
unpushed commits (`gitx.BranchState.KeepReason`); otherwise the worktree is
removed first. A failed checkout stops; a failed pull only warns.

`tb hot` polls size+mtime of each file every 500ms and runs the command via
`sh -c` or `powershell -Command`.

## Benchmark

`tb benchmark` generates uuids for 1m, 5m or a custom time and estimates how
long finding all of them, or filling RAM and disk with them, would take.
Details in [benchmark.md](./benchmark.md). No shell alias: `benchmark` is
not a daily command.

## Secrets

`tb secret` keeps secrets in an encrypted local store and, once `connect`ed,
mirrors one Bitwarden Secrets Manager project. Details in
[secret.md](./secret.md). It has no shell alias for the same reason `tb env`
has none: a bare `secret` is too generic a name to claim.

## Known gaps

- Not yet exercised on real Windows or macOS beyond CI unit tests: winget
  installs, PowerShell profile block, gpg on Windows.
- `tb commit`/`tb pr` interactive paths have no automated test.
- `apps`/`android` have no linux equivalent (no flatpak support).
- On linux the `code` recipe needs VS Code already installed.
