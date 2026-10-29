---
name: commit
description: Creates Conventional Commits — small atomic commits, title-only by default (no body), with a body reserved exclusively for breaking-change notes and co-authoring trailers. Encodes the project's commit type rules. Use whenever staging and committing changes.
---

# Skill: Commit Creator

## 🎯 Purpose
Turn working-tree changes into clean Conventional Commits that match the
project conventions: small and atomic, a **title only** in the common case,
and a body used *only* for breaking-change notes or co-authoring trailers.

## 📥 Arguments
All optional — derive from the diff when not given.
* **`type`** / **`scope`** / **`subject`** — the Conventional Commit pieces.

## 🗂 Pre-Execution
1. **Read the diff.** `git status --short` then `git diff` (staged + unstaged).
   The message is derived from what actually changed — never guess.
2. **Split into atomic commits.** One commit = one coherent change that builds
   and passes on its own. If the working tree mixes concerns (e.g. types vs.
   styles vs. tests, or two unrelated fixes), stage and commit them separately
   with `git add <specific paths>` rather than `git add -A`. Several small
   commits are preferable to one mixed blob.

## 📝 Message Format

### Title (always)
```
<type>(<scope>): <imperative subject, lower-case, no trailing period>
```
* Imperative mood ("add", "migrate", "fix" — not "added"/"fixes").
* Lower-case after the colon, no full stop at the end.
* `scope` is the area touched: a command (`setup`, `pr`, `commit`,
  `checkout`, `hot`, `benchmark`, `env`, `secret`, `update`), a package (`pkgmgr`, `recipes`,
  `dots`, `merge`, `shellrc`, `auth`, `ui`, `run`, `bitwarden`, `vault`, `bench`), `install` (bootstrap scripts), a
  recipe name (`essentials`, `node`, `code`, …) or a dot (`gitconfig`,
  `vscode`, `tmux`). Omit it when the change spans several areas.

### Body — avoid it
The commit is **title-only by default.** Do not write an explanatory body to
describe *what* or *why* — that reasoning belongs in the PR description, not the
commit. A body is allowed **only** for:

1. **Breaking-change notes** — a `BREAKING CHANGE:` footer describing the break.
2. **Co-authoring trailers** — e.g.
   `Co-Authored-By: <agent name> <noreply@example.com>`

If neither applies, commit with the title alone:
```bash
git commit -m "feat(essentials): install bat"
```

When a footer/trailer *is* needed:
```bash
git commit -m "$(cat <<'EOF'
feat(install)!: rename --skip-dots to --skip-config

BREAKING CHANGE: `--skip-dots` is removed; use `--skip-config`.

Co-Authored-By: <agent name> <noreply@example.com>
EOF
)"
```

## 🏷 Type Rules (project conventions)
* `feat` / `fix` — user-facing behaviour change.
* `chore` — dependency/version bumps, tooling, cleanup, asset changes.
* `test` — reserved for **new** test coverage only.
* `perf` — reserved for real runtime performance improvements only.
* `docs`, `refactor`, `style`, `ci`, `build` — standard Conventional Commits
  meanings.

### The `!` breaking-change marker — guard
Add `!` (e.g. `feat(x)!:`) **only for a true break.** Re-exporting a renamed or
deprecated symbol with a deprecated alias is **not** a break — no `!`, no
`BREAKING CHANGE:` footer. The `!` and the footer go together when used.

## ✅ Verification
* The staged paths are exactly the ones belonging to this commit.
* Title is Conventional-Commit-shaped; no stray body.
* `!`/`BREAKING CHANGE:` present only for genuine breaks.

## 📤 Expected Output
* One or more atomic commits, each title-only unless a break note or co-author
  trailer is required.

## ⚠️ Rules and Constraints
* **Never** `git add -A` blindly when the tree mixes concerns — stage per
  commit.
* **Never** add a body just to explain the change — that's the PR's job.
* Commit text is written normally (full prose), even when the chat is terse.
* Don't push here.
