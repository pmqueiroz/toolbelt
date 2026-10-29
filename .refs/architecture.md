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
  `checkout`, `hot`, `benchmark`, `env`, `secret`, `dots`, `update`, `uninstall`, `version`), wired in `root.go`.
  `root.go` runs everything under a context cancelled by Ctrl+C.
- `internal/ui/` — the only place terminal styling lives: `log` wrapper
  (kitchen timestamps, like the old `gum log`), lipgloss styles,
  `Confirm`, `Spin` (falls back to a plain log line without a TTY), `Bar`
  (a `█░` bar for a 0–1 fraction, clamped) and `Panel` (rounded titled box).
- `internal/run/` — `Runner` executes commands. `Run` sends output to the
  log file (or the terminal too with `--verbose`), `Output` captures stdout,
  `Attached` wires stdin/stdout for interactive commands (git pull, gh).
- `internal/plan/` — `Action{Description, Run}` and `Dedupe`.
- `internal/pkgmgr/` — `Manager` interface with `brew` (linux, darwin) and
  `winget` (windows).
- `internal/recipes/` — installable units (see below).
- `internal/dots/` — embedded dot files (`files/`), their per-OS targets and
  the link + merge sync (`sync.go`).
- `internal/merge/` — pure line-based three-way merge (`Three`) and conflict
  marker detection (`HasMarkers`), used by dot sync.
- `internal/shellrc/` — the managed block in shell rc files (`blocks/`).
- `internal/auth/` — ssh key, npm token, gpg signing key.
- `internal/files/` — compare / write-with-backup / atomic write / `~` pretty
  paths, shared by dots, shellrc and vault.
- `internal/secret/` — masking used by `tb env`, `tb secret list` and the npm plan line.
  `Mask` keeps the first/last 4 chars and rejects values under 8; `Redact`
  stars short values fully; `SplitEnv` splits a `NAME=value` entry (names
  starting with `=`, Windows drive vars, split on the next `=`);
  `MaskEnviron` redacts every entry of `os.Environ()`; `Compact` is `Redact`
  with the hidden middle capped at 6 stars, so a huge value (`LS_COLORS`)
  stays one short cell.
- `tb env` with no argument (`internal/cli/env.go`): when stdout is a
  terminal it renders a rounded lipgloss table (NAME / VALUE, sorted by name,
  values through `secret.Compact`, `Wrap(false)`); the table keeps its
  natural width and is only capped to the terminal width when wider, so
  overflowing cells are truncated with `…` instead of wrapping. When stdout
  is not a terminal (piped, redirected) it prints the plain `MaskEnviron`
  lines with the full star count, unchanged for scripts.
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

`tb setup` refuses to run without a TTY. The bootstrap scripts don't start
it: they install `tb` (and brew on linux/macOS) and print the command, so
installing never changes the machine's config by surprise.

## tb uninstall

`internal/cli/uninstall.go` builds `[]setup.Phase` and reuses
`setup.Render` / `setup.Execute`, so it shows the plan and asks before
changing anything, like `tb setup`. Without a TTY it refuses unless `--yes`.

1. **shell sources** — `shellrc.RemovalPlan` checks every rc file `tb setup`
   could have written (`~/.bashrc` and `~/.zshrc` regardless of `$SHELL`, or
   each PowerShell profile) and strips the toolbelt block and the legacy
   dotfiles block with `shellrc.Remove`, backing the file up first. Files
   without a block get no action. `Apply` uses `Remove` too, so both agree on
   what the block is.
2. **secrets** — only with `--purge`: `vault.Purge` (see
   [secret.md](./secret.md#command-behavior)). Without it, an existing store
   is kept and a warning says so *before* the confirm, because once the
   binary is gone there is no `tb` left to purge with.
3. **binary** — the resolved `os.Executable()` (symlinks followed). Unix just
   unlinks it while it runs. Windows can't delete a running exe but can
   rename it, so it is moved to `%TEMP%\tb_uninstalled_<timestamp>.exe` for
   the OS temp cleanup to collect. Last on purpose: a failure earlier leaves
   `tb` around to retry.

What `tb setup` installed (packages, dot files, ssh/gpg keys, `~/.npmrc`,
`~/.gitconfig.local`) is left alone, including the dot store under
`<UserConfigDir>/toolbelt/dots` the targets link to (removing it would
leave dangling links): those are the machine's config now, not
tb's, and other tools depend on them.

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

Current recipes: `essentials`, `node`, `pnpm`, `code`, `claude` (default)
and `apps`, `android` (macOS/Windows only, because brew casks don't exist on
linux).

`CaskOnLinux` marks the exception: a cask that ships a plain binary artifact
and installs fine with Linuxbrew. `claude` (Claude Code, cask `claude-code`,
winget `Anthropic.ClaudeCode`) is one, so it installs on all three OSes.
Its config is the `claude-settings.json` dot, not part of the recipe, so it
syncs like every other dot.
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
`.tmux.conf` is skipped on Windows.

`claude-settings.json` targets `~/.claude/settings.json` (Claude Code's user
settings, same path on every OS). It sets `attribution.commit` to `""`,
which drops Claude's co-author trailer from commits (PR attribution stays
default), enables the `context7` plugin from `claude-plugins-official`
(Anthropic's marketplace, known to Claude Code out of the box, so no
`extraKnownMarketplaces` entry), and points `statusLine` at `npx -y ccstatusline@latest`
([ccstatusline](https://github.com/sirmalloc/ccstatusline)). npx comes from
the `node` recipe. Claude Code
rewrites `settings.json` itself (`/model`, `/config`); if it replaces the link
with a regular file, sync treats that file as local and merges it.

`ccstatusline-settings.json` is ccstatusline's layout (widgets, colors),
taken from its TUI export with `exportedBy` dropped (its own import strips
that key too). It targets `~/.config/ccstatusline/settings.json` under the
home dir on every OS, not `os.UserConfigDir()`, because ccstatusline builds
the path from `os.homedir()`. When ccstatusline saves from its TUI it
resolves the symlink and renames its temp file over the store copy, so the
link survives and the change is a local edit that sync merges.

### Store and links

Dots are not copied to their targets. The rendered file lives in the store,
`<UserConfigDir>/toolbelt/dots/<file>` (`Paths.Store`), and the target
(`~/.gitconfig`, VS Code `settings.json`, …) is a symlink to it. Editing the
target edits the store copy, which is what sync merges against.

Store layout:

- `<file>` — the live copy the target links to (local edits land here).
- `.base/<file>` — the exact upstream render tb last wrote: the merge base.
- `vars.json` — `username` / `email` used to render, so `tb dots sync` can
  re-render without the setup form. The gpg passphrase is never saved (no
  dot uses it).

The link is created as a temp symlink renamed over the target, so the target
is never missing mid-write. If `os.Symlink` fails (Windows without developer
mode) the target gets a plain copy instead; sync then treats that copy as the
local version, same as a link an editor replaced with a regular file.

### Sync (`dots.Plan`, used by `tb setup` and `tb dots sync`)

Per dot, with *upstream* = this binary's render, *base* = `.base/<file>`,
*local* = the live copy if the target links to it (or is missing), else the
target file's own content:

| situation | result |
|---|---|
| target exists, base == upstream | nothing (local edits kept, no action) |
| no local file | write upstream, link |
| local == upstream | write upstream, link if needed |
| no base (first adoption of a foreign file) | backup target, write upstream, link |
| base present | `merge.Three(base, local, upstream)`, write result, link |
| local still has `<<<<<<< local` / `>>>>>>> toolbelt` markers | refuse; action fails asking to resolve |

Writes are atomic (`files.WriteAtomic`) and ordered backup → live → base →
link. A crash between live and base is safe: the next merge sees the same
change on both sides, which the merge accepts without conflict.

Backups (`<target>.<timestamp>.bak`, holding *local*) are made when local
content would otherwise be lost: first adoption, any conflict, and a merge
into a target that was a regular file. A clean merge through the link needs
none: the result contains every local line change.

Conflicts follow git: the live file gets
`<<<<<<< local` / `=======` / `>>>>>>> toolbelt` hunks, base advances to
upstream, and the action returns an error (exit 1) naming the file. Until
the markers are gone, sync refuses to touch that file, so an unresolved file
is never merged again. Because base already moved, resolving by keeping
either side (or both) merges cleanly next time.

`merge.Three` is a diff3 over lines: LCS-match base to each side, emit
stable runs, and per changed hunk take the side that changed (identical
changes on both sides are not a conflict). Pure Go, no `git merge-file`, so
it works before git is installed and on Windows.

### Updates

`tb update` syncs dots after replacing the binary: it runs the *new*
binary's `tb dots sync` (`run.Runner.Attached`), because the running process
still has the old embedded files. When already up to date it syncs
in-process, which also repairs missing links. A sync failure makes `tb
update` exit 1 but the binary is already updated. Without `vars.json` (dots
step never run with this version) sync is a no-op that says to run
`tb setup`.

`.gitconfig` includes `~/.gitconfig.local`. Machine-specific git settings
(`user.signingkey`, Windows `gpg.program`) go there, so syncing dots never
wipes the signing key.

## Shell sources

`shellrc.Targets` picks `~/.bashrc` and/or `~/.zshrc` (whichever exist, plus
the one matching `$SHELL`, bash as fallback) or, on Windows, the
`$PROFILE.CurrentUserCurrentHost` of each installed PowerShell (`pwsh`,
`powershell`). The block between `# ---BEGIN TOOLBELT---` and
`# ---END TOOLBELT---` is replaced in place; the legacy
`# ---BEGIN DOTFILES SOURCE---` block from the old bash installer is removed.
Block contents: `~/.local/bin` on PATH, brew shellenv, mise activate, zoxide
init, `tb` completion (Cobra's built-in `tb completion <shell>`, eval'd at
shell start), history size, `glo`/`repo`, and aliases from `pr`, `commit`,
`checkout`, `hot` to `tb`. In zsh the completion line runs `compinit` first
when `compdef` isn't defined yet (no framework like oh-my-zsh loaded it).
Completion is bound to `tb` only; the aliases don't complete. `tb env` gets no alias on purpose: it would shadow
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

With no argument, `tb checkout` targets `gitx.DefaultBranch`: it reads
`refs/remotes/origin/HEAD`, which only `git clone` creates. Repos made with
`git init` + `git remote add` lack it, so on failure it runs
`git remote set-head origin --auto` (asks the remote, writes the ref) and
retries, then falls back to a local `main` or `master`.

`tb hot` polls size+mtime of each file every 500ms and runs the command via
`sh -c` or `powershell -Command`.

## Benchmark

`tb benchmark` generates uuids for 5s, 10s or a custom time and estimates how
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
- Dots dropped from `dots.All` in a later version keep their link and store
  copy; sync only visits current dots.
- Machines set up before dot linking keep copied files until `tb setup` runs
  the dots step again (no `vars.json`, so `tb dots sync` can't render).
- `tb uninstall` on Windows leaves the install dir entry that `install.ps1`
  added to the user `Path`, and the empty install dir.
