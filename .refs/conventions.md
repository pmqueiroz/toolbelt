# Coding conventions & authoring rules

How code is written in this project. These are hard rules, not suggestions. This file is about **how to
author** code and where documentation lives.

## No comments in code

Code carries **no comments**. Do not:

- Explain a function with a comment above it.
- Annotate a line because it "might need documenting".
- Leave TODO/NOTE/why-notes inline.

If something needs explaining — a business rule, a non-obvious decision, a
gotcha — it belongs in `.refs/`, **always**. The code says *what*; `.refs/`
says *why*. A comment is a signal that a `.refs/` entry is missing: write the
entry instead.

Name things well enough that the code reads without prose. Extract a
well-named function or variable instead of writing a comment.

Compiler directives are not comments and are allowed: `//go:embed` is the
only one in use.

## Go style

- `gofmt` is enforced by CI.
- Tests: one `<file>_test.go` per source file, in the same package so
  unexported functions can be tested directly. Name tests after the behavior
  (`TestApplyRemovesLegacyDotfilesBlock`), not the function alone.
- Keep logic that decides things pure and separate from code that touches
  the machine (`shellrc.Apply`, `gitx.BranchState.KeepReason`,
  `auth.UpsertNpmToken`), so it can be tested without a shell, git repo or
  TTY. Inject what varies (`shellrc.Env.Profile`, `dots.Paths`, fake
  `pkgmgr.Manager`) instead of reading globals.
- Paths in tests go through `filepath.Join` / `filepath.FromSlash`; CI runs
  on Windows.
- User-facing output goes through `internal/ui` (`ui.Info`, `ui.Warn`, …),
  never bare `fmt.Print` (exception: command results meant for piping, like
  `tb version` and `tb penv`, which write to `cmd.OutOrStdout()`).
- Log and error messages are lower-case.
- External commands go through `run.Runner`, never `exec.Command` directly,
  so they are logged (exception: `shellrc.powershellProfile`, which runs
  while planning, before there is a runner).
- Plan action descriptions never contain secrets; mask with
  `secret.Mask`.
- Anything rendered from `internal/dots/files/` uses `{{placeholder}}` +
  `dots.Render`; never hardcode the user's name or email in a dot.

## Documentation lives in `.refs/`

Every explanation of code behavior or business rule goes in the matching
`.refs/` file (see [AGENTS.md](../AGENTS.md) for the map). When you add or
change a feature, updating the `.refs/` doc is **part of the task**, not a
follow-up — the Operating Protocol in AGENTS.md is binding.

## Helpers must be reusable, and reused

One concept = one helper. Before writing a new function, check the shared
packages: `internal/files` (compare, backup, pretty paths), `internal/run`,
`internal/ui`, `internal/secret`, `internal/plan`. If a second package needs
something another one already does, move it to a shared package instead of
copy-pasting it.

Each installable unit is its own recipe (see
[architecture.md](./architecture.md#the-extension-point-recipes)); don't grow
install logic for a new tool inside `internal/setup`.
