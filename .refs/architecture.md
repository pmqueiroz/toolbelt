# Architecture

Personal dotfiles: one interactive bash installer (`install.sh`) that sets up
a fresh macOS/Linux machine via Homebrew, installs selectable "recipes",
copies templated config files into place, and walks through GitHub/npm/GPG
auth. Mental model: `install.sh` is the host and owns the flow and UI;
a recipe owns only installing one thing.

Code is the truth. This file explains the *why* behind the shape it takes.

## Layout

- `install.sh` — entry point. Bootstraps `gum` and `coreutils` via brew,
  sources every recipe, parses `--flag` options, then runs the phases in
  order: credentials prompt → sudo auth → aliases → bash sources →
  dependencies → dots → (confirm) ssh → npm token → gpg → `post_install`.
- `recipes/<name>.sh` — one file per installable unit, each defining a single
  `install_<name>` function. Sourced by `install.sh`; never executed alone.
- `dots/` — config files copied to their target locations:
  - `aliases.sh` — shell aliases/functions (`commit`, `checkout`, `pr`,
    `hot`, `checksum`, `gum_log`, …). Copied to `~/dots-aliases.sh`, and also
    sourced by `install.sh` itself, so the installer depends on its helpers
    (`gum_log`, `checksum`, the `date`→`gdate` alias on macOS).
  - `sources.sh` — block appended to `~/.bashrc` (sources the aliases file,
    zoxide init, history size).
  - `.gitconfig`, `.tmux.config`, `vscode-settings.json`, `keybindings.json`
    — mapped to their target paths in the `dots` associative array in
    `install.sh`.
  - `gpg_template.gpg`, `gnome-terminal-profile.dconf` — templates consumed
    by the gpg phase and the `gnome_terminal` recipe.
  - `keychron_k6_pro.json` — keyboard config, not installed by the script.
- `bundler.rb` — inlines every `source`d file into `install.sh` so it can run
  standalone via `curl | bash`. Run only by CI (see [ci-cd.md](./ci-cd.md)).
- `.github/assets/` — logos for the readme.

## The extension point: recipes

Contract: a file `recipes/<name>.sh` defining `install_<name>`. The installer
calls it with `eval install_"$dep"` for each chosen dependency, then re-sources
`~/.bashrc` so later recipes see PATH changes (e.g. `asdf` before `node`).

Recipes may use the installer's globals: `_` (run a command, logging output
to the logfile unless `--verbose`), `gum_log`, `render_string`, `$user_name`,
`$user_email`, and the `./tmp` scratch dir.

Adding a recipe:
1. Create `recipes/<name>.sh` with `install_<name>`. Hyphens in the file name
   are fine; the function name and choice name use underscores
   (`gnome-terminal.sh` → `install_gnome_terminal`).
2. Add `source recipes/<name>.sh` near the top of `install.sh`.
3. Add `<name>` to `dependencies` (preselected) or `additional_dependencies`
   (opt-in) in the dependencies phase. Order in the arrays is install order.

## Shared state

Everything is bash globals in `install.sh`: `options` (parsed flags),
`logfile`, `user_name`, `user_email`, `passphrase` (sudo password, reused as
the GPG key passphrase). The sudo timestamp is primed once by `auth` and reset
in `post_install`.

## Reusable mechanics

- **Templating** — `render_string key value …` replaces `{{key}}` in stdin.
  Used for every file in `dots/` and the gpg/dconf templates. Placeholders in
  use: `{{username}}`, `{{email}}`, `{{passphrase}}`.
- **Managed `.bashrc` block** — `sources.sh` lives between
  `# ---BEGIN DOTFILES SOURCE---` / `# ---END DOTFILES SOURCE---` markers.
  On rerun, the installed block's md5 is compared to `dots/sources.sh`; if it
  differs, `.bashrc` is backed up to `~/.bashrc.<timestamp>.bak` and the block
  is replaced. Idempotent otherwise.
- **Local vs. remote source** — when run inside the git checkout, files are
  copied from the repo; otherwise they're fetched from
  `raw.githubusercontent.com/pmqueiroz/dotfiles/master`.
- **Logging** — `gum_log <level>`; output of wrapped commands goes to
  `peam_dotfiles_<timestamp>.log` (gitignored).

## Known gaps

- `~/dots-aliases.sh` is only written if it doesn't exist, so alias changes
  don't reach an existing machine on rerun. Delete the file first.
- The dots phase writes the rendered file from `./dots/` even after fetching
  the remote copy, so the remote-fetch branch is effectively broken when not
  run from a checkout.
- `node` recipe pins Node 18.20.2 (EOL).
- `install_code` exits the whole installer if `code` isn't on PATH.
- Requires bash 4+ (associative arrays, `mapfile`); stock macOS bash is 3.2.
