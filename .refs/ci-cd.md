# CI/CD

GitHub Actions, `.github/workflows/cd.yaml`. Runs on every push to `master`.
No concurrency control.

Ruby is pinned to `3.1` via `ruby/setup-ruby@v1`. No local version pin file;
`bundler.rb` uses only the stdlib (`set`, `fileutils`).

## Jobs

- **`bundle`** — checks out `master`, runs `ruby bundler.rb` (inlines every
  `source`d recipe/dot into `install.sh`, dropping lines that start with `#`
  and keeping `$HOME`-relative sources as-is), commits the result as
  `update install`, and force-pushes it to the `release` branch with
  `ad-m/github-push-action`.

`release` is therefore a generated, single-file build and is what the
"More Unstable" `curl | bash` one-liner in the readme fetches. Never commit to
`release` by hand; it's overwritten on every push.

Note: the bundler drops *any* line starting with `#`, including the shebang
and anything inside heredocs, so don't rely on such lines surviving in the
bundled `install.sh`.

## Not set up yet

- No checks: no `shellcheck`, no test run, no dry-run of the installer.
  Nothing gates the push to `release`.
- `actions/checkout@v2` is outdated.
