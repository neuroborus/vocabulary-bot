---
name: finalization
description: Post-change finalization checklist for the Vocabulary Bot Go project. Use after completing feature, fix, refactor, or project-structure work before drafting a commit to keep the Go layout idiomatic, tests green, vocabulary merge rules correct, source boundaries clean, secrets safe, and docs aligned.
---

# Finalization — Vocabulary Bot

> Post-change checklist for `vocabulary-bot`.
>
> Run this after feature, fix, refactor, or scaffold work to keep the Go project
> coherent and aligned with the vocabulary import/review product rules.
>
> This file is the single finalization checklist. It is plain markdown with optional skill metadata.
>
> Last Updated: 2026-06-14

---

## 1. Tests for new functionality

- [ ] New behavior has focused Go coverage in the owning package.
- [ ] Use `package <pkg>` tests for unexported internals and `package <pkg>_test` tests for public behavior across package boundaries.
- [ ] Do not export internals just to make tests easier.
- [ ] Vocabulary domain changes have tests for:
  - normalization and compact lookup keys;
  - structural edge-token matching;
  - merge/create/update behavior;
  - form preservation and `displayWord` selection;
  - translation/context/note/tag deduplication;
  - ambiguous match handling.
- [ ] Spreadsheet changes have parser tests for headers, disabled rows, row-level errors, translations split by comma, and contexts split by dot.
- [ ] PocketBook parser/client changes use sanitized fixtures under `test/fixtures/pocketbook`.
- [ ] Review scheduler and Telegram command behavior have unit tests when business rules change.
- [ ] Obsolete tests are updated or removed.

---

## 2. Project boundary review

- [ ] `cmd/vocabulary-bot` stays a thin executable entrypoint: context setup, `app.Run`, final error handling.
- [ ] `internal/app` remains the composition root: load config, create logger, wire storage, sources, vocabulary service, sync service, Telegram/review later.
- [ ] `internal/vocabulary` remains source-agnostic and infrastructure-free. It must not import MongoDB, Telegram, Google Sheets, PocketBook, scheduler, or logging packages.
- [ ] `internal/source` defines the common adapter contract; concrete sources return `vocabulary.Draft`.
- [ ] Source adapters do not write to storage, do not merge vocabulary items, and do not send Telegram messages.
- [ ] `internal/sync` orchestrates sources and calls common vocabulary merge logic.
- [ ] `internal/storage/*` implements persistence boundaries and owns storage-specific details.
- [ ] `internal/telegram` owns Telegram command/notifier boundaries, not vocabulary merge logic.
- [ ] Do not add `pkg/` until another Go module has a real need to import this project.

---

## 3. Vocabulary domain review

- [ ] PocketBook and Google Sheets still write into one common `VocabularyItem` model through `vocabulary.Draft`.
- [ ] Raw visible forms are always preserved in `forms`.
- [ ] `translations` and `contexts` remain independent arrays; do not pair a translation with a specific context automatically.
- [ ] Lookup matching does not use a hardcoded or configurable article/preposition/particle whitelist.
- [ ] Structural lookup keys still include:
  - full compact form;
  - edge-token variants where appropriate;
  - no destructive changes to visible display data.
- [ ] `displayWord` prefers the best observed user-visible phrase:
  - richer/more complete forms over bare forms;
  - leading articles `a`, `an`, `the` when choosing among otherwise similar forms;
  - trailing auxiliary/context words after the main word when that is the observed form.
- [ ] Deduplication remains normalized but non-destructive:
  - forms by compact normalized form;
  - translations by trimmed lowercase text;
  - contexts/notes/tags by normalized text;
  - anchors by stable source metadata.
- [ ] Ambiguous merges are not auto-merged; they are surfaced through an error/summary/log path.

---

## 4. Source adapter review

- [ ] PocketBook stays an unofficial adapter with defensive parsing and fixture-driven behavior.
- [ ] PocketBook errors should not prevent Google Sheets sync when the remaining source can still run.
- [ ] Google Sheets rows are parsed into drafts only; sheet metadata writeback, if added, must not rewrite user-owned columns (`word`, `translations`, `contexts`, `note`, `tags`).
- [ ] Source sync order remains PocketBook first, Google Sheets second.
- [ ] Full sync stays idempotent: reruns do not duplicate forms, translations, contexts, notes, tags, or anchors.

---

## 5. Config and secrets review

- [ ] New runtime config is loaded through `internal/config`.
- [ ] `.env.example` is updated when env vars change.
- [ ] No production secrets, Telegram tokens, Google service account JSON, PocketBook credentials, refresh tokens, cookies, or raw credential-bearing payloads are committed.
- [ ] Tests, fixtures, docs, and committed examples use obviously artificial credential placeholders (`FAKE_*_FOR_TEST_ONLY`, `example.test`, `old-access`), not values that resemble real JWTs, Telegram bot tokens, MongoDB passwords, or other production-like secrets.
- [ ] Scan the tracked repo for realistic-looking credential placeholders before staging:

```bash
rg -P -n 'eyJhbGci|SecretPass|\b\d{8,10}:(?!FAKE_)[A-Za-z0-9_-]{20,}\b|mongodb\+srv://[^:]+:(?!FAKE_)[^@]+@|Bearer eyJ' \
  --glob '!.env' --glob '!.env.*' --glob '!*.log' .
```

Expected result: no matches. If matches appear, replace them with obviously artificial placeholders such as `FAKE_*_FOR_TEST_ONLY` before staging.

- [ ] Logs do not include secrets or full raw provider responses that may contain credentials.
- [ ] `.gitignore` keeps local `.env`, logs, build output, coverage output, and temp files out of commits.

---

## 6. Logging and operations review

- [ ] Logs are written in English.
- [ ] File logging still defaults to `$TMPDIR/vocabulary-bot/logs/vocabulary.log`.
- [ ] `/start`, `/info`, `/health`, `/list_words`, `/turn_off`, `/turn_on`, `/sync`, `/push`, and `/logs` remain the intended Telegram command surface.
- [ ] Weekly log delivery/rotation behavior is preserved if touched: clear or rotate only after successful Telegram delivery.
- [ ] Operational errors include enough context to debug without leaking secrets.

---

## 7. Docs and agents alignment

- [ ] Update `README.md` when commands, structure, setup, or runtime behavior changes.
- [ ] Update `AGENTS.md` when package boundaries, Go conventions, validation commands, or agent gates change.
- [ ] Update `docs/README.md` when adding, moving, or removing docs.
- [ ] Re-read relevant files under `docs/product/` when changing product behavior:
  - `docs/product/DEVELOPMENT_PLAN.md`;
  - `docs/product/POCKETBOOK_GOALS.md`;
  - `docs/product/SPREADSHEET_GOALS.md`.
- [ ] Keep this skill updated when finalization steps change.

---

## 8. Idiomaticity pass

- [ ] Touched code is conventional Go: small packages, clear exported names, minimal interfaces at consumer boundaries, explicit errors, no needless abstractions.
- [ ] Comments are concise and explain non-obvious intent, trade-offs, or constraints only.
- [ ] Avoid catch-all `utils`/`helpers` packages; place pure functions next to their domain.
- [ ] Keep dependencies minimal; prefer the standard library until an external package has a clear payoff.
- [ ] Avoid unrelated refactors and generated churn in the same change set.

---

## 9. Full check run

- [ ] Run the whole module:

```bash
go test ./...
go vet ./...
go run ./cmd/vocabulary-bot
```

- [ ] Or use the Makefile wrappers:

```bash
make test
make vet
make run
```

- [ ] If the default Go build cache is not writable in the current sandbox, rerun with a temp cache:

```bash
GOCACHE=/tmp/vocabulary-bot-go-cache go test ./...
GOCACHE=/tmp/vocabulary-bot-go-cache go vet ./...
GOCACHE=/tmp/vocabulary-bot-go-cache go run ./cmd/vocabulary-bot
```

- [ ] For risky domain changes, also run coverage:

```bash
GOCACHE=/tmp/vocabulary-bot-go-cache go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

- [ ] Fix failures before considering the change set done.

### Final formatting

- [ ] Format Go code after all intended edits, review changes, and checks are complete:

```bash
make fmt
```

- [ ] If `make` is unavailable, run the underlying formatter directly:

```bash
gofmt -w cmd internal
```

- [ ] If formatting changed files, rerun the relevant checks.

---

## 10. Commit preparation — stage, but never commit yourself

The expected end state of finalization is a staged change set and a drafted commit message. Do not run `git commit` or `git push` unless the user explicitly asks in the current message.

- [ ] Review `git status --short`.
- [ ] Stage relevant files for the completed change set; leave unrelated scratch files unstaged.
- [ ] Draft a Conventional Commit message:

```text
<type>(<scope>): <subject>
```

- [ ] Use `type` from `feat`, `fix`, `refactor`, `perf`, `test`, `docs`, `chore`, `style`, `ci`.
- [ ] Use a scope matching the touched area, for example `vocabulary`, `spreadsheet`, `pocketbook`, `telegram`, `sync`, `config`, `docs`, `scaffold`, or `finalization`.
- [ ] Present the drafted message to the user.
- [ ] Do not commit or push for the user.
