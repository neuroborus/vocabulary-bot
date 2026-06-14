# AGENTS.md

Operational guide for AI coding agents working in `vocabulary-bot`.

Read this file first. Load product docs from `docs/` only when the task touches that area.

## Documentation Gates

- `README.md` — human-facing project overview, structure, and commands.
- `RHYTHM.md` — chronological decision log. Add meaningful repo decisions there, newest section first.
- `docs/README.md` — index for product/reference docs and when to read each file.
- `docs/product/DEVELOPMENT_PLAN.md` — read when changing the overall vocabulary model, sync flow, Telegram commands, logging, scheduling, or MVP scope.
- `docs/product/POCKETBOOK_GOALS.md` — read before changing PocketBook auth, fetching, parsing, fallback import, fixtures, or PocketBook source anchors.
- `docs/product/SPREADSHEET_GOALS.md` — read before changing Google Sheets config, row parsing, metadata writeback, or spreadsheet source anchors.
- `.agents/skills/finalization/SKILL.md` — post-change checklist. It is plain markdown with optional skill metadata; any agent can read it before staging or drafting a commit.

## Project Shape

This is one Go application, not a public Go library.

```text
cmd/vocabulary-bot/          executable entrypoint
internal/app/                composition root
internal/config/             environment configuration
internal/vocabulary/         domain model, normalization, merge rules
internal/source/             source adapter contract
internal/source/pocketbook/  PocketBook source boundary
internal/source/spreadsheet/ Google Sheets source boundary and row parser
internal/storage/            persistence adapters
internal/sync/               source sync orchestration
internal/telegram/           Telegram command/notifier boundary
internal/logging/            file logger setup
internal/review/             review scheduling helpers
docs/                        product/reference docs
test/fixtures/               sanitized external payload fixtures
```

Do not add `pkg/` until another Go module has a real need to import this project.

## Dependency Direction

Keep dependencies pointed inward:

```text
cmd -> internal/app
internal/app -> internal/sync, internal/source/*, internal/storage/*, internal/vocabulary
internal/sync -> internal/source, internal/vocabulary
internal/source/* -> internal/vocabulary
internal/storage/* -> internal/vocabulary
internal/vocabulary -> standard library only
```

`internal/vocabulary` must not import MongoDB, Telegram, Google Sheets, PocketBook, logging, scheduler, or app composition packages.

## Boundary Rules

- `cmd/vocabulary-bot` stays thin: context cancellation, `app.Run`, final error handling.
- `internal/app` wires config, logging, storage, source adapters, and use-case services.
- Source adapters return `vocabulary.Draft`; they do not merge, persist, schedule, or notify.
- `internal/sync` orchestrates source order and passes drafts to common merge logic.
- Storage adapters implement repository interfaces; storage-specific details stay out of domain code.
- Telegram command handlers call application services; they do not duplicate sync or merge logic.

## Vocabulary Invariants

- PocketBook and Google Sheets write into one shared vocabulary entity through `vocabulary.Draft`.
- `translations` and `contexts` are independent arrays. Do not infer a translation/context pair automatically.
- Raw visible forms are always preserved in `forms`.
- Lookup matching uses structural keys, not a hardcoded or configurable article/preposition/particle whitelist.
- Display data is not destructively normalized.
- `displayWord` should prefer the best observed phrase:
  - richer/more complete forms over bare forms;
  - leading articles `a`, `an`, `the` among otherwise similar forms;
  - trailing auxiliary/context words after the main word when that is the observed form.
- Ambiguous lookup matches are skipped/reported, not auto-merged.

## Source Rules

- PocketBook Cloud is unofficial. Keep the adapter defensive and fixture-driven.
- PocketBook errors should not stop Google Sheets sync when the remaining source can still run.
- Google Sheets is an input adapter, not a database.
- Spreadsheet metadata writeback, if added, may update metadata columns only; do not rewrite user-owned `word`, `translations`, `contexts`, `note`, or `tags`.
- Sync order remains PocketBook first, Google Sheets second.
- Full sync must be idempotent.

## Config, Secrets, Logs

- Runtime config belongs in `internal/config`.
- Update `.env.example` when adding or renaming env vars.
- Never commit production secrets, Telegram tokens, Google service account JSON, PocketBook credentials, refresh tokens, cookies, or raw credential-bearing payloads.
- Logs are written in English and default to `$TMPDIR/vocabulary-bot/logs/vocabulary.log` (override with `LOG_PATH`).
- Do not log secrets or full raw provider responses that may contain credentials.

## Go Conventions

- Prefer small packages and clear exported names.
- Define interfaces at consumer boundaries when possible.
- Return errors with context at package boundaries.
- Avoid catch-all `utils` or `helpers` packages; place pure functions next to the domain they serve.
- Add comments only for non-obvious intent, trade-offs, or constraints.
- Keep dependencies minimal. Prefer the standard library until an external package has a clear payoff.

## Validation

Use the project commands:

```bash
make fmt
make test
make vet
make run
```

In this sandbox the default Go build cache may be read-only, so use a temp cache when needed:

```bash
GOCACHE=/tmp/vocabulary-bot-go-cache go test ./...
GOCACHE=/tmp/vocabulary-bot-go-cache go vet ./...
GOCACHE=/tmp/vocabulary-bot-go-cache go run ./cmd/vocabulary-bot
```

Run tests for code changes. Documentation-only changes do not require Go tests unless they affect generated artifacts or examples.

## Git Hygiene

- You may be in a dirty worktree. Do not revert user changes.
- Keep edits scoped to the task.
- Do not run `git commit` or `git push` unless the user explicitly asks in the current message.

## Optional Tool Integrations

Tool-specific metadata may exist under `.agents/`. Keep those files readable as normal markdown so agents that do not support the metadata can still use them.
