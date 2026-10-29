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

## Shell style

- Bash only, `#!/usr/bin/env bash` shebang, `shopt -s expand_aliases` at the
  top of any file that relies on aliases from `dots/aliases.sh`.
- 3-space indent in `install.sh`, `dots/aliases.sh` and most recipes (some
  recipes use 2; match the file being edited).
- User-facing output goes through `gum_log <level>` / `gum style`, never bare
  `echo` (exception: the pre-`gum` bootstrap at the top of `install.sh`).
- Log messages are lower-case.
- Wrap noisy commands in `_` so their output lands in the logfile unless
  `--verbose`.
- Anything rendered from `dots/` uses `{{placeholder}}` + `render_string`;
  never hardcode the user's name or email in a dot.

## Documentation lives in `.refs/`

Every explanation of code behavior or business rule goes in the matching
`.refs/` file (see [AGENTS.md](../AGENTS.md) for the map). When you add or
change a feature, updating the `.refs/` doc is **part of the task**, not a
follow-up — the Operating Protocol in AGENTS.md is binding.

## Helpers must be reusable, and reused

One concept = one helper. Shared shell helpers live in `dots/aliases.sh`
(available both in the user's shell and inside `install.sh`); installer-only
helpers live in `install.sh`. Before writing a new function, check both. If a
recipe needs something another recipe already does, move it to a shared
helper instead of copy-pasting it.

Each installable unit is its own recipe (see
[architecture.md](./architecture.md#the-extension-point-recipes)); don't grow
install logic for a new tool inline in `install.sh`.
