# Project Review

Date: 2026-06-14

Scope: local review of the Go application, docs, deploy files, and current validation path. Existing code was not changed.

## Recheck: 2026-06-15

Status: all findings from the latest pass are resolved. No current high, medium, or low findings were confirmed after finalization.

Validation on the current tree:

- `gofmt -l cmd internal` returned no files.
- `bash deploy/render-env_test.sh` passed.
- `GOCACHE=/tmp/vocabulary-bot-go-cache go vet ./...` passed.
- `GOCACHE=/tmp/vocabulary-bot-go-cache go test ./...` passed outside the sandbox. Inside the sandbox it failed because `httptest.NewServer` cannot listen on local ports.
- `GOCACHE=/tmp/vocabulary-bot-go-cache go test -race ./...` passed outside the sandbox.
- `GOCACHE=/tmp/vocabulary-bot-go-cache go run ./cmd/vocabulary-bot` passed.
- Finalization secret scan returned no matches.

Resolved since the previous recheck:

- Spreadsheet row reassignment to an existing item now returns the target item outcome, deletes an emptied source item through `Repository.Delete`, and has a regression test in `TestMergeDraftReassignsSheetRowToExistingItem`.
- Both storage adapters now expose `Delete`, and Mongo storage has delete coverage.
- Deploy env rendering now has a shared Python parser/formatter and a shell regression test for quoted secrets and MongoDB URI validation.
- Config loading now unwraps quoted env values consistently, which matches `.env.example` and deploy rendering.
- Opt-in Necromancer FB2 investigation tests are gated by `NECROMANCER_FB2_PATH` and skip without a local fixture.
- In-memory `Replace` now matches Mongo's stricter repository contract: missing previous keys fail, collisions fail, and focused memory repository tests cover rename, missing-key, and collision paths.

Current findings: none.

## Recheck: 2026-06-14

Status: mostly fixed. The original findings below are kept as historical context; current status is summarized here.

Validation on the current tree:

- `gofmt -l cmd internal` returned no files.
- `GOCACHE=/tmp/vocabulary-bot-go-cache go vet ./...` passed.
- `GOCACHE=/tmp/vocabulary-bot-go-cache go test ./...` passed outside the sandbox. Inside the sandbox it still fails because `httptest.NewServer` cannot listen on local ports.
- `GOCACHE=/tmp/vocabulary-bot-go-cache go test -race ./...` passed outside the sandbox.

Resolved or no longer confirmed:

- Telegram review callback data now uses a short hash token instead of embedding the normalized key.
- `/push` now sends the Telegram message before marking the item as pushed.
- Runtime sync/notification flags now use a mutex, and sync execution is serialized.
- PocketBook and spreadsheet partial failures now appear in source details.
- Typed environment variables now fail fast on invalid values instead of silently falling back.
- Deploy env rendering now includes the previously missing runtime variables.
- `GOOGLE_SHEET_RANGE=Vocabulary!A:F` matches the current input-column format (`word` through `enabled`); metadata columns are intentionally outside the read range.
- Mongo storage no longer imports `internal/source/pocketbook`; session state moved to `internal/source/session`.
- `slog.Logger.With` attributes are now sanitized.
- The PocketBook discovery command now mentions `POCKETBOOK_SYNC_ENABLED`.
- `internal/config` no longer imports `internal/logging`.
- Book download temp-file ownership is now centralized in the client.
- EPUB text normalization no longer calls `builder.String()` inside the rune loop.
- Telegram HTTP non-2xx errors now include sanitized Bot API descriptions when available.

Remaining finding:

### High: spreadsheet row reassignment to an existing item can leave an orphan source item

Files:

- `internal/vocabulary/sheet_row.go:43`
- `internal/vocabulary/sheet_row.go:49`
- `internal/vocabulary/sheet_row.go:87`
- `internal/vocabulary/sheet_row.go:90`
- `internal/vocabulary/sheet_row.go:94`
- `internal/storage/memory/repository.go:130`
- `internal/storage/mongo/repository.go:149`

The new spreadsheet row update path handles a simple row rename, but the branch where the edited row now matches another existing item is still risky. `applySheetRowUpdate` calls `reassignSheetRow`, which removes the row contribution and anchor from the source item, saves that source item, then merges the row into the target item. The caller then returns an outcome built from the source item, not the target item.

For a sheet-only source item, removing the only sheet contribution can leave an enabled item with no anchors, no forms, and no lookup keys, while the target item correctly receives the row anchor. In memory storage this empty source object remains in the map; in Mongo storage the source document remains unless another explicit delete path exists. The current tests cover `carve -> gauge` when `gauge` does not already exist, but do not cover `carve -> gauge` when `gauge` already exists.

Practical effect:

- `/list_words` and review selection can still see an orphan item after a sheet row is moved to an already existing word.
- Sync summaries can report the old/source normalized key for this update path.
- Mongo can retain a stale source document because repositories expose `Replace` but not a delete/merge-removal operation.

Recommendation: add an explicit repository deletion or deactivation path for emptied source items, make `reassignSheetRow` return the updated target item/outcome, and add a regression test for a row move onto an existing vocabulary item.

## Validation

- `gofmt -l cmd internal` returned no files.
- `GOCACHE=/tmp/vocabulary-bot-go-cache go vet ./...` passed.
- `GOCACHE=/tmp/vocabulary-bot-go-cache go test ./...` failed inside the sandbox because `httptest.NewServer` could not listen on local ports.
- The same `go test ./...` command passed when rerun outside the sandbox.

## Findings

### High: Google Sheets row edits are append-only, not real edits

Files:

- `internal/vocabulary/merge.go:61`
- `internal/vocabulary/merge.go:75`
- `internal/vocabulary/merge.go:159`
- `internal/vocabulary/merge.go:166`
- `internal/vocabulary/merge.go:211`
- `internal/sync/service_test.go:121`

When a Google Sheets row is found by `(sheetName, rowNumber)`, the code calls `MergeIntoItem` on the existing item. That merge only appends forms, lookup keys, translations, contexts, notes, tags, and anchors. It does not replace the data previously contributed by that spreadsheet row.

Practical effect:

- If row 5 changes from `carve` to `gauge`, the stored item can keep `carve` as `displayWord`, keep old `carve` lookup keys, and keep old translations.
- If the edited word already matches another vocabulary item, the row-anchor branch can update the old item anyway, producing conflicting lookup keys across items.
- The existing test `TestMergeDraftUpdatesExistingSheetRowWhenWordChanges` asserts that an update happened, but does not assert that stale word data was removed or that `displayWord` changed.

Recommendation: define spreadsheet row semantics explicitly. If a sheet row is editable input, row changes should either replace that row's previous contribution or move the row anchor to the item matched by the new lookup keys. Avoid mixing old and new row meanings in one entity by default.

### High: Telegram review callback data can exceed Telegram's limit

Files:

- `internal/telegram/review.go:25`
- `internal/telegram/review.go:56`
- `internal/telegram/handler.go:276`
- `internal/vocabulary/merge.go:129`

`reviewCallbackData` embeds `item.NormalizedKey` directly into `callback_data`. Sentence-like spreadsheet rows can produce long normalized keys. Telegram documents `callback_data` for inline buttons as `1-64 bytes`: https://core.telegram.org/bots/api#inlinekeyboardbutton

Practical effect:

- `/push` can fail for long sentence rows or long phrases when the keyboard is sent.
- The failure happens at delivery time, not at import time, so the problematic vocabulary item is hard to diagnose.

Recommendation: store a short review token in the callback payload, for example a database ID, a compact hash, or a server-side pending callback ID. Keep the normalized key in storage, not in Telegram callback data.

### High: `/push` marks a word as pushed before delivery succeeds

Files:

- `internal/telegram/handler.go:268`
- `internal/telegram/handler.go:271`
- `internal/telegram/handler.go:276`

`pushReviewWord` calls `review.MarkPushed`, updates the repository, and only then sends the Telegram message. If `SendHTMLMessageWithKeyboard` fails, the item has already had `LastPushedAt` and `PushCount` updated even though the user never received the card.

Practical effect:

- A transient Telegram failure can suppress or deprioritize the word.
- Metrics say the word was pushed when it was not delivered.

Recommendation: send first and persist the push state only after successful delivery, or store an explicit delivery state that can distinguish attempted, delivered, and failed pushes.

### High: command state is mutated without synchronization

Files:

- `internal/telegram/handler.go:31`
- `internal/telegram/handler.go:123`
- `internal/telegram/handler.go:235`
- `internal/telegram/handler.go:348`
- `internal/telegram/handler.go:442`
- `internal/app/app.go:201`
- `internal/app/app.go:239`

`syncEnabled` and `notificationsEnabled` are plain bool fields on `CommandHandler`. Telegram polling handles user commands while the scheduler runs in a separate goroutine. `/turn_off` and `/turn_on` write those fields, while scheduled sync, push, logs, and health reads them.

Practical effect:

- This is a data race under normal runtime shape.
- Scheduled jobs may observe stale or inconsistent state.
- The same `syncService` and source adapters can also be entered by a manual `/sync` and scheduled sync at the same time.

Recommendation: protect command runtime state with a mutex or atomics, and serialize sync execution with a lock/single-flight guard.

### Medium: source error reporting hides partial failures

Files:

- `internal/source/pocketbook/adapter.go:127`
- `internal/source/pocketbook/book_context.go:268`
- `internal/source/spreadsheet/adapter.go:79`
- `internal/source/details.go:4`
- `internal/sync/service.go:62`

PocketBook per-book and per-note failures are logged and skipped, but `Adapter.Sync` still returns `nil` error. Spreadsheet row parse errors are also only logged. `sync.Summary.SourceErrors` only counts adapter-level errors, and `source.Details` has no counters for skipped books, failed notes, or row parse errors.

Practical effect:

- `/sync` can report a clean source run even when every PocketBook book failed internally.
- Invalid spreadsheet rows are invisible in the Telegram summary unless the user checks logs.

Recommendation: extend `source.Details` with row parse errors, skipped books, failed notes, and parse skips. Surface those counters in `formatSyncSummary`.

### Medium: invalid env values silently fall back to defaults

Files:

- `internal/config/config.go:162`
- `internal/config/config.go:176`
- `internal/config/config.go:190`

`getenvBool`, `getenvInt`, and `getenvFloat` return the fallback when parsing fails. That makes misconfiguration hard to detect.

Practical effect:

- `TELEGRAM_POLLING_ENABLED=flase` silently becomes `true`.
- `POCKETBOOK_BOOK_CACHE_MAX=oops` silently becomes `2`.
- `REVIEW_BOOK_PUSH_FACTOR=0` silently becomes `1`, which may be intended in code but surprising in config.

Recommendation: make invalid configured values return an error from `config.Load`, the same way `optionalInt64` does for invalid Telegram IDs.

### Medium: deploy workflow does not pass all supported runtime variables

Files:

- `.github/workflows/deploy.yml:49`
- `.github/workflows/deploy.yml:69`
- `deploy/render-env.sh:39`
- `deploy/render-env.sh:69`

`deploy/render-env.sh` supports more variables than the deploy workflow passes into the render step. Missing examples include `TELEGRAM_POLLING_ENABLED`, `TELEGRAM_API_BASE_URL`, `TELEGRAM_REVIEW_SPOILER_TRANSLATIONS`, `SCHEDULE_TIMEZONE`, `POCKETBOOK_REFRESH_TOKEN`, `POCKETBOOK_SHOP_NAME`, `POCKETBOOK_API_BASE_URL`, `POCKETBOOK_TOKEN_PATH`, `POCKETBOOK_BOOK_CONTEXT_ENABLED`, `POCKETBOOK_BOOK_CACHE_DIR`, `POCKETBOOK_BOOK_CACHE_MAX`, and `LOG_PATH`.

Practical effect:

- Setting those GitHub variables has no effect in production deploy.
- Production behavior can differ from `.env.example` and deploy docs in hard-to-see ways.

Recommendation: either pass every supported render variable from the workflow, or remove unsupported variables from `render-env.sh` and docs.

### Medium: default Google Sheets range conflicts with the documented full sheet layout

Files:

- `internal/config/config.go:127`
- `.env.example:116`
- `docs/product/SPREADSHEET_GOALS.md:117`

The default range is `Vocabulary!A:F`. The full documented sheet layout puts `source` in column F and `enabled` in column G, with metadata columns after that. With the default range, `enabled` is not fetched when the user follows the full layout.

Practical effect:

- Disabled rows can still be imported if the sheet uses the documented A-J structure.
- Future metadata columns are invisible to the adapter by default.

Recommendation: use `Vocabulary!A:Z` as the default range, or make the recommended full layout match the default `A:F` manual-import layout.

### Medium: Mongo storage depends on the PocketBook source package

Files:

- `internal/storage/mongo/repository.go:10`
- `internal/storage/mongo/repository.go:160`
- `internal/source/pocketbook/session_store.go:13`

`internal/storage/mongo` imports `internal/source/pocketbook` to implement `PocketBookSessionStore`. This is a sideways dependency from storage into a source adapter package, which cuts against the repository's dependency-direction rules.

Practical effect:

- Storage is now coupled to a concrete source package.
- A future source/session store abstraction will be harder to move without touching Mongo storage.

Recommendation: move the session type and store interface to a neutral boundary, for example `internal/source/session`, `internal/storage/session`, or an app-level auth/session package.

### Medium: sanitizing slog handler does not sanitize attrs added through `Logger.With`

Files:

- `internal/logging/handler.go:34`
- `internal/logging/handler.go:42`

`Handle` sanitizes attrs attached to each log record, but `WithAttrs` passes attrs directly to the wrapped handler. If future code uses `logger.With(slog.String("token", "..."))`, those attrs can be preformatted by the inner handler without passing through `sanitizeAttr`.

Practical effect:

- Current code does not appear to use `logger.With`, so this is latent.
- The logging boundary is incomplete for secret-bearing attrs.

Recommendation: sanitize attrs inside `WithAttrs` before calling `h.inner.WithAttrs`.

### Low: discovery command mentions a non-existent env var

Files:

- `cmd/pocketbook-dump-note/main.go:29`
- `internal/config/config.go:111`

`pocketbook-dump-note` exits with `POCKETBOOK_ENABLED is false`, but the actual config key is `POCKETBOOK_SYNC_ENABLED`.

Practical effect:

- Debug output points the user to the wrong env var.

Recommendation: change the message to `POCKETBOOK_SYNC_ENABLED is false`. Consider making `pocketbook-list-books` check the same switch for consistency.

### Low: config package imports logging for a default path

Files:

- `internal/config/config.go:9`
- `internal/config/config.go:85`
- `internal/logging/logger.go:12`

`internal/config` imports `internal/logging` only to get `DefaultLogPath`. This is small, but it couples config loading to the logging package.

Practical effect:

- The dependency direction is less clean than the app composition model suggests.

Recommendation: keep the default path value in config, or pass it from `internal/app` when wiring logging.

### Low: book-cache download uses two nested temp-file protocols

Files:

- `internal/source/pocketbook/book_cache.go:195`
- `internal/source/pocketbook/book_cache.go:202`
- `internal/source/pocketbook/client.go:368`
- `internal/source/pocketbook/client.go:385`

`BookCache.download` asks the client to download to `destination + ".partial"`. `Client.downloadToFile` then writes to `destination + ".part"` and renames it to the requested destination. The happy path works, but the naming and rename chain are unnecessarily hard to reason about.

Practical effect:

- Failure cleanup is split across two layers.
- Future changes can easily leave `.partial` or `.part` files behind.

Recommendation: make exactly one layer responsible for atomic temp-file handling.

### Low: EPUB text normalization has avoidable O(n^2) behavior

Files:

- `internal/source/pocketbook/epub_text.go:128`
- `internal/source/pocketbook/epub_text.go:131`

`normalizeBookText` calls `builder.String()` inside the rune loop to test whether the builder currently ends with a newline. That allocates a full string repeatedly on large books.

Practical effect:

- Large EPUB flattening can use unnecessary CPU and memory.

Recommendation: track the previous emitted rune or a `previousNewline` bool instead of converting the builder to a string inside the loop.

### Low: Telegram HTTP non-2xx errors discard useful API descriptions

Files:

- `internal/telegram/client.go:298`
- `internal/telegram/client.go:300`

For non-2xx responses, the Telegram client drains and discards up to 4096 bytes, then returns only `telegram http error <status>`. Telegram often includes a JSON `description` that would explain the actual request problem.

Practical effect:

- Debugging Bot API failures is harder, especially for formatting, callback data, and permission errors.

Recommendation: decode the standard Telegram error body when possible, sanitize it, and include the description in the returned error.

## Notes

- The package layout is mostly coherent and tests cover the core vocabulary merge, source parsing, Telegram formatting, scheduling, and storage paths.
- The highest-risk area is no longer basic compilation; it is runtime semantics around editable spreadsheet rows, Telegram delivery state, and concurrency between manual commands and scheduled jobs.
