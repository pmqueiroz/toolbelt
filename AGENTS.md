# Agent Instructions

## .refs — feature & architecture notes

Working references for how this project is built. Code is the truth; these
explain the *why* and the gotchas.

- [conventions.md](./.refs/conventions.md) — authoring rules: **no comments in
  code** (docs go in `.refs/`), reusable-component discipline. Read first.
- [architecture.md](./.refs/architecture.md) — `tb` package layout, the
  `tb setup` form → plan → confirm → execute flow, recipes (the extension
  point), dots, shell block, auth, git helpers, known gaps.
- [secret.md](./.refs/secret.md) — `tb secret`: Bitwarden Secrets Manager
  protocol port (pure Go, no cgo), encrypted local mirror, keyring key,
  merge rules.
- [ask.md](./.refs/ask.md) — `tb ask`: server-driven answer blocks (JSON
  schema → `ui` components), providers (Anthropic SDK + OpenAI-compatible),
  `connect`, run-a-suggested-command flow.
- [benchmark.md](./.refs/benchmark.md) — `tb benchmark`: uuid generation,
  base64-prefix bit bucket, find-all / fill-memory / fill-disk estimates.
- [ci-cd.md](./.refs/ci-cd.md) — CI matrix, GoReleaser releases, GitHub
  Pages one-liners, the names the installers and `tb update` depend on.
- [incidents/](./.refs/incidents/README.md) — postmortems + standing rules;
  **read before any migration or destructive op**.

Add more `.refs/` files as the project grows a subsystem worth explaining
(e.g. `observability.md`, `economy.md`, `legal.md`) — one file per concern,
linked from this list.

## Operating Protocol

*The Self-Maintaining Loop:* If you create a new feature, make changes to an
existing one, or modify app lifecycle process, *your final task before
completing the job is to update the corresponding markdown file in the
`.refs/` directory*. The documentation must perfectly reflect the code in the
app.

# Agent Skills
Custom skills and operational workflows are stored in `.agents/skills/`.
When handling tasks matching the definitions inside `.agents/skills/<skill-name>/SKILL.md`, read the corresponding skill file and adhere to its instructions.

CLAUDE Only: Everytime loading the skills create a mirror in .claude/skills with each skill pointing to the original one. Also add a .gitignore file inside .claude/skills with "*"
