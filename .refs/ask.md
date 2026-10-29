# tb ask

`tb ask <question>` sends a natural-language question to an AI provider and
renders a short answer in the terminal, Warp-style: a one-line definition, a
command to run, a snippet. `tb ask connect` picks the provider, key and
model.

## Server-driven UI

The model never returns prose for the terminal to print as-is. It returns
JSON, a list of typed blocks, and `tb` decides how each block looks with its
own components (`internal/ui/answer.go`). So the look stays consistent across
providers and models, and a new block type is a schema + renderer change in
one place.

```json
{"blocks":[{"type":"command","text":"note","code":"ls -la","language":"","items":[]}]}
```

| type | fields used | styled render (`ui.*`) | plain render |
|---|---|---|---|
| `text` | `text` | `Paragraph` (wrapped) | the text |
| `command` | `code`, optional note in `text` | `CommandLine`: `$ cmd` per line, faint note | the command only |
| `code` | `code`, `language` | `CodeBlock`: left bar, faint language | the code |
| `list` | `items` | `Bullets` | `- item` lines |
| `warning` | `text` | `Caution` (`⚠`, gold) | `warning: text` |

Every block carries all five fields, unused ones empty. That is what strict
JSON-schema modes require (OpenAI `strict: true` needs every property in
`required` and `additionalProperties: false`), and the same schema
(`ai.Schema`) is sent to every provider.

Brevity is enforced by the system prompt (`ai.System`): shortest correct
answer, prefer one block, at most three, commands for the user's OS and
shell (`runtime.GOOS`, `$SHELL` base name, `powershell` on Windows).

### Parsing is lenient (`ai.Parse`, pure)

Structured output is requested, but OpenRouter models and older Ollama
models may ignore it. So `Parse` takes the text between the first `{` and the
last `}` (drops ```` ```json ```` fences and chatter), drops blocks with an
unknown type or nothing to show, and trims fields. If nothing valid is left,
the raw reply becomes one `text` block: the user always sees an answer
instead of a JSON error.

## Output

- **stdout is a terminal**: styled render, width = terminal width capped at
  100. Then, if the answer has `command` blocks and stdin is a terminal, a
  select offers `run <command>` for each (first preselected) or `done`.
  Picking one opens it in an input (a multi-line text area when the command
  has newlines) to edit before running; esc cancels. It runs through
  `run.Runner.Attached` with `sh -c` / `powershell -Command` (the same
  `shellCommand` as `tb hot`). Nothing runs without that two-step
  confirmation.
- **stdout is not a terminal**: the plain render, for piping. Command notes
  are dropped so `tb ask ... | sh` style use gets only the command.

## Input

- Arguments are joined with spaces into the question.
- stdin not a terminal + arguments: stdin is context
  (`cat err.log | tb ask why does this fail`). Only the last
  `ai.MaxContextBytes` (16 KiB) are sent, with a warning: the end of a log is
  where the error is, and answers are meant to be cheap and short.
- stdin not a terminal + no arguments: stdin is the question.
- terminal + no arguments: an input prompt asks for it.

`ask connect` is a subcommand, so a question whose first word is literally
`connect` must be quoted. Flags in a question (`tb ask what does -v do`)
are parsed by cobra; quote them or put them after `--`.

The shell block adds an `ask` alias. In zsh it is `noglob tb ask`, because
`?` in `what is the moon?` is a glob and zsh aborts with "no matches found";
bash leaves unmatched globs as-is. PowerShell gets `function ask`.

## Providers (`internal/ai`)

`ai.Client` has two methods: `Ask` (one question, raw reply text) and
`Models` (list ids; `connect` uses it both to pick a model and to check the
key works).

| provider | client | base url | key env |
|---|---|---|---|
| `anthropic` | official Go SDK (`anthropic-sdk-go/v2`) | SDK default | `ANTHROPIC_API_KEY` |
| `openai` | OpenAI-compatible HTTP | `api.openai.com/v1` | `OPENAI_API_KEY` |
| `gemini` | OpenAI-compatible HTTP | `generativelanguage.googleapis.com/v1beta/openai` | `GEMINI_API_KEY` |
| `openrouter` | OpenAI-compatible HTTP | `openrouter.ai/api/v1` | `OPENROUTER_API_KEY` |
| `ollama` | OpenAI-compatible HTTP | `localhost:11434/v1` | none |

- **Anthropic** uses the SDK, as Anthropic recommends over raw HTTP. It
  sends `output_config.format` (`json_schema`, structured outputs) and
  `output_config.effort: low`: answers are short, so deep thinking would only
  add latency. The SDK sends the key as `Authorization: Bearer`. A
  `stop_reason: refusal` becomes `ai.ErrRefused`. Default model:
  `claude-opus-5-5`, listed first in the picker; pick `claude-haiku-5-5` in
  `connect` for the cheapest and fastest answers. The SDK costs about 11 MB
  of binary size (15.9 → 26.9 MB) because its generated API types are large.
- **Everything else** is one hand-written `chat/completions` client
  (`openai.go`) with `response_format: json_schema, strict: true`. Gemini,
  OpenRouter and Ollama all serve that API, so one client covers them;
  `--base-url` points it at any other compatible server (LM Studio, vLLM, a
  proxy). Gemini lists ids as `models/<id>`; the prefix is stripped. A
  `refusal` field becomes `ai.ErrRefused`; non-2xx is `*ai.APIError` with
  the body.

## Storage

The connection (`vault.AIConnection`: provider, model, api key, base url) is
the `ai` field of the `tb secret` store (`secrets.enc`, see
[secret.md](./secret.md#storage)): the key is a secret, and reusing the store
means one data key, one lock-out guard and `tb uninstall --purge` covers it.
It is never pushed to Bitwarden: only `entries` are mirrored.

## connect resolution

- **provider**: `--provider`, else a select (stored one preselected), else
  (no TTY) the stored one, else an error.
- **key** (not for ollama): `--key`, then the provider's env var, then the
  stored key *if the stored provider is the same*, then a password prompt.
  Switching providers never reuses the other provider's key.
- **base url**: `--base-url`, then the stored one for the same provider, then
  the preset.
- **model**: `--model` as given (not checked against the list: some servers
  serve aliases they don't list), else a select over `Models()` (stored one
  preselected, the preset default first), else (no TTY) the stored model if
  still listed, then the preset default, then the first listed.

## Known gaps

- One question, one answer: no conversation history, no streaming.
- No refusal fallback model: a refused question is an error.
- Live calls to providers aren't covered by CI; clients are tested against
  `httptest` servers.
