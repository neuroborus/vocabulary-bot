# RHYTHM.md

Chronological log of meaningful repo decisions. **Newest sections first:** add each new `## YYYY-MM-DD` block right below this paragraph, not at the end of the file.

## 2026-06-20

- `/save` confirmation spoiler omits `<code>` tags because Telegram leaves monospace text visible inside `<tg-spoiler>`.
- `/save` confirmation now shows word, context, and translation first; sheet row status and the scheduled-sync hint are grouped in a Telegram spoiler at the bottom.

## 2026-06-16

- Empty `/save` in groups now always shows the privacy/admin workaround hint, and failed group saves log whether Telegram sent `reply_to_message` or `quote`.
- `/save` reply handling now reads Telegram `quote` text and replied message captions, which fixes group replies where `reply_to_message.text` is empty; the empty-reply prompt now mentions group privacy and admin workarounds.
- `TELEGRAM_LEAVE_DISALLOWED_CHATS` (default `true`) controls whether the bot calls `leaveChat` for groups and channels outside the allowlist; when `false`, disallowed updates are still ignored.
- `/save` confirmation now explains that local import happens on the scheduled sync and that only an admin can run `/sync` in private chat for an immediate import.
- Startup now logs whether Telegram `/save` is enabled and why it is disabled when OpenAI or Google Sheets prerequisites are missing; the handler no longer receives a typed-nil save service interface.
- `/save` is no longer admin-user-only: any user in an allowlisted chat may append to Google Sheets, while other commands and review callbacks remain admin-only. Telegram command menus are refreshed for all configured allowlisted chats.
- Telegram command menu setup now sends `/save` in both the default command list and every configured allowlisted chat scope, so stale scoped menus are refreshed.
- `/save` accepts text before or after the command token; an empty non-reply `/save` returns a usage prompt asking the user to reply to the message to save or include text around the command.
- `/save` now appends one OpenAI-structured row to Google Sheets only; it does not merge into local storage or reply with a review card. The new row reaches MongoDB through the normal `/sync` flow.

## 2026-06-15

- Added `/save` with OpenAI structuring and Google Sheets append support. Translation language is configured with `OPENAI_TRANSLATION_LANGUAGE` (default `Russian`).
- `/health` includes the current chat ID and caller user ID to simplify allowlist setup.
- `/sync` and `/logs` are accepted only from the admin private chat (`TELEGRAM_ADMIN_ID` with `chat.type=private`). Scheduled sync/log delivery is unchanged.
- Telegram env vars are now `TELEGRAM_ADMIN_ID` and `TELEGRAM_TARGET_CHANNEL_ID`. Admin and target channel chats are always allowlisted; `TELEGRAM_ALLOWED_CHAT_IDS` adds optional extra chats.
- In-memory vocabulary repository `Replace` now matches Mongo's stricter contract: missing previous keys and normalized-key collisions fail, with focused memory repository tests covering rename, missing-key, and collision behavior.

## 2026-06-14

- `config.Load` now unwraps single and double quotes for all env values, including quoted cron expressions such as `AUTO_SYNC_CRON='0 9 * * *'`.
- GitHub Actions workflows now use Node.js 24-native action majors (`actions/checkout@v6`, `actions/setup-go@v6`) instead of the temporary `FORCE_JAVASCRIPT_ACTIONS_TO_NODE24` opt-in.
- PocketBook FB2 context enrichment now unwraps `.fb2.zip` downloads, skips `notes`/`comments` bodies, joins hyphenated line breaks, matches inflected surface forms (`carved`, `gauges`, `Teasely`) when resolving dictionary headwords, and searches within ±512 bytes around `offs` before falling back to the whole book.
- CI now runs only on pull requests targeting `main`, not on every push to `dev` or `main`.
- Deploy remote steps now fall back to `sudo docker` when the SSH user cannot access `/var/run/docker.sock` directly.
- Deploy now defaults to `~/vocabulary-bot` when `DEPLOY_PATH` is unset and falls back to `sudo mkdir/chown` for fixed paths such as `/opt/vocabulary-bot`.
- Sheet row reassignment onto an existing vocabulary item now deletes the emptied source item, returns the updated target in merge outcomes, and has a regression test for `carve -> gauge` when `gauge` already exists.
- Telegram HTTP failures now decode the Bot API `description` field when present and include the sanitized text in returned errors.
- `normalizeBookText` now tracks newline state in O(1) instead of calling `builder.String()` inside the rune loop when flattening large EPUBs.
- Book cache downloads now rely on `Client.DownloadFile` for atomic temp-file handling instead of adding a second `.partial` rename layer on top of the client's `.part` file.
- `DefaultLogPath` now lives in `internal/config`, so config loading no longer imports the logging package.
- PocketBook discovery commands now report `POCKETBOOK_SYNC_ENABLED is false` and both `pocketbook-dump-note` and `pocketbook-list-books` honor the same source switch.
- Mongo PocketBook session persistence now depends on `internal/source/session` instead of `internal/source/pocketbook`, keeping storage adapters free of concrete source-package imports.
- Spreadsheet docs now match the parser layout: `enabled` lives in column F, so the default `GOOGLE_SHEET_RANGE=Vocabulary!A:F` is correct for the manual-import sheet.
- Deploy workflow now passes every variable supported by `deploy/render-env.sh`, so optional production settings such as `TELEGRAM_POLLING_ENABLED`, `SCHEDULE_TIMEZONE`, and PocketBook cache overrides take effect on deploy.
- `config.Load` now returns an error for invalid boolean, integer, and positive-float env values instead of silently falling back to defaults.
- Source sync summaries now surface partial adapter failures in `/sync`: PocketBook books skipped/failed and note fetch failures, plus spreadsheet row parse errors, through extended `source.Details`.
- Command handler runtime flags and sync execution are now mutex-protected so Telegram polling and the scheduler do not race on `/turn_off`, `/turn_on`, and overlapping manual/scheduled sync runs.
- `/push` now updates `LastPushedAt` and `PushCount` only after Telegram delivery succeeds, so transient send failures do not suppress the word from future review selection.
- Telegram review callbacks now use a short SHA-256 token in `callback_data` instead of embedding `normalizedKey`, keeping inline keyboard payloads within Telegram's 64-byte limit.
- Google Sheets sync now skips unchanged rows on repeat syncs using `sheetName + rowNumber` anchor fingerprints (`rowFingerprint` on the sheet anchor); full range is still fetched, but unchanged rows avoid Mongo writes and show `rows skipped (unchanged)` in `/sync`.
- Implemented Google Sheets source sync: service-account auth from `GOOGLE_SERVICE_ACCOUNT_JSON` (raw JSON or base64), `spreadsheets.values.get` for `GOOGLE_SHEET_RANGE`, row parsing into `vocabulary.Draft`, and row-level error logging without aborting the whole source; sentence-style rows with commas in `word` keep the whole `translations` cell intact.
- Documented the recommended manual Google Sheets import template `word | translations | contexts | note | tags | enabled`, including how optional `note`, `tags`, and `enabled` cells are parsed and when to leave them empty.
- Default PocketBook book cache dir renamed to `$TMPDIR/vocabulary-bot-cache/books` to avoid collision when `/tmp/vocabulary-bot` is a leftover Go binary; `/sync` now reports `book context skipped (already in DB)` and `book context enriched` under the pocketbook source line.
- `IsUsageExampleLine` now requires Cyrillic on the right side of `english - translation` lines so book sentences with dashes (e.g. crackpot) are not misclassified; fixes repeated enrichment on every sync for those words.
- PocketBook book-context enrichment uses an on-disk cache keyed by `fast_hash`, with `lastUsedAt` sidecar metadata and LRU eviction at `POCKETBOOK_BOOK_CACHE_MAX` (default `2`); stale versions for the same `bookId` are removed when `fast_hash` changes; words that already have a book sentence in Mongo are skipped during enrichment.
- `/push` cards now show the PocketBook source label at the bottom of the message, after context and translations.
- Quoted `AUTO_SYNC_CRON` and `AUTO_PUSH_CRON` in `.env.example` so `source .env` does not treat cron spaces as shell commands.
- Added `TELEGRAM_REVIEW_SPOILER_TRANSLATIONS` to hide `/push` translation blocks behind a Telegram spoiler by default.
- Added in-process cron scheduling via `AUTO_SYNC_CRON` (default `0 9 * * *`) and `AUTO_PUSH_CRON` (default `0 12-21/2 * * *`) with optional `SCHEDULE_TIMEZONE`; scheduled jobs respect `/turn_off` and `/turn_on`.
- Review `/push` selection is now weighted random: difficulty, due state, recency, and `REVIEW_DOCUMENT_PUSH_FACTOR` / `REVIEW_BOOK_PUSH_FACTOR` change selection weight instead of forcing a deterministic order.
- Review `/push` callbacks edit the card in place: buttons disappear and the chosen Easy/Hard result is appended to the message.
- Moved dictionary usage-example lines (`english - перевод`) from `translations` into `contexts` during PocketBook parse and merge via `vocabulary.PartitionUsageExamples`.
- Added PocketBook discovery commands `cmd/pocketbook-dump-note` and `cmd/pocketbook-list-books` for raw note/book inspection during API validation.
- Documented and implemented sequential PocketBook book-download context enrichment during sync (`POCKETBOOK_BOOK_CONTEXT_ENABLED`): download book file (EPUB/FB2), extract `pbr:/word` sentence by `offs`, delete temp file.
- Added MongoDB-backed runtime storage using the official MongoDB Go driver. When `MONGODB_URI` is configured, vocabulary items are persisted in `vocabulary_items` and PocketBook access/refresh sessions are persisted in `pocketbook_sessions`; the file-backed PocketBook session store remains the fallback when MongoDB is not configured.
- Added a minimal Telegram long-polling runtime gated by `TELEGRAM_ALLOWED_USER_ID`. Supported commands are `/start`, `/info`, `/health`, `/sync`, `/list_words`, `/logs`, `/turn_off`, `/turn_on`, and `/push`; command handlers call the existing sync and vocabulary repository boundaries instead of duplicating merge logic.
- Added centralized log sanitization for secret-bearing strings: a `slog` wrapper redacts MongoDB URIs, bearer tokens, Telegram bot tokens, OAuth query params, and JSON token fields before file logging; sync summaries and fatal startup errors use the same sanitizer. `.env.example` now notes that values with shell metacharacters such as `&` should be quoted.
- Hardened PocketBook book metadata parsing with `FlexibleString` so numeric `year` and `isbn` values from the unofficial API no longer break JSON unmarshaling.
- Implemented the first real PocketBook Cloud sync boundary using the unofficial community-observed API flow: shop discovery, password bootstrap, refresh-token renewal, automatic fallback from failed stored refresh token to password bootstrap, session persistence, book listing, note ID listing, note detail fetching, defensive note-to-`vocabulary.Draft` parsing, per-book/per-note error logging, and tests with `httptest`. `POCKETBOOK_API_BASE_URL` is available for test/discovery override.
- Clarified the PocketBook authentication decision: email/password are for bootstrap, the bootstrap flow should automatically capture and persist the returned refresh/session token, normal scheduled sync should prefer the stored token, and `POCKETBOOK_REFRESH_TOKEN` remains only an optional override for environments without a writable token store. MongoDB is an acceptable token store for this app if the collection is treated as secret-bearing storage and never stores the PocketBook password.
- Consolidated documentation to reduce early-project noise. Root `AGENTS.md` is now the canonical operational guide for AI coding agents, `docs/README.md` is the local product-doc gate/index, long product specs moved from `agents/` to `docs/product/`, and standalone `docs/ARCHITECTURE.md` / `docs/CONVENTIONS.md` were folded into `AGENTS.md`. The `agents/` directory is intentionally not used for product docs; `.agents/skills/finalization/SKILL.md` remains the single finalization checklist and is readable as plain markdown by any agent.
- Initialized the repository as a Go application module (`github.com/neuroborus/vocabulary-bot`) with the standard library only for the current foundation stage. Root tooling now includes `go.mod`, `Makefile`, `.env.example`, `.gitignore`, and a runnable `cmd/vocabulary-bot` entrypoint.
- Established the Go project layout: `cmd/vocabulary-bot` stays thin; `internal/app` is the composition root; `internal/vocabulary` owns source-agnostic domain behavior; `internal/source/*` owns source adapter boundaries; `internal/sync` orchestrates source drafts and merge; `internal/storage/*` owns persistence boundaries; `internal/telegram` owns bot command/notifier boundaries; `internal/review` owns review scheduling helpers. No `pkg/` exists because there is not yet an external Go consumer.
- Mapped the user's Nest monorepo approach to idiomatic Go boundaries instead of copying the TypeScript/Nest tree directly.
- Encoded the shared vocabulary model and source-agnostic merge path in `internal/vocabulary`: PocketBook and Google Sheets both produce `vocabulary.Draft`; `Item` preserves `forms`, `lookupKeys`, `translations`, `contexts`, `notes`, `tags`, `anchors`, and review metadata; the repository boundary is an interface consumed by the vocabulary service.
- Implemented word normalization and structural lookup keys without a hardcoded or configurable article/preposition/particle whitelist. Raw display data is never destructively trimmed for presentation; structural edge-token variants are used only for lookup and merge. Two-token phrases intentionally avoid adding both single-token edge candidates when that would over-promote a short edge token such as `to`.
- Implemented merge semantics: new drafts create an item with the full compact key; matching drafts merge by `lookupKeys`; multiple matches return an ambiguous merge error instead of auto-merging; forms/translations/contexts/notes/tags/anchors deduplicate by normalized keys while preserving visible values.
- Refined `displayWord` selection after product feedback: richer observed forms beat bare forms; leading articles `a`, `an`, and `the` are preferred among otherwise similar forms; trailing auxiliary/context words after the main word are also preserved as the display form when that is the richer observed phrase. Tests cover `apple` / `apple an` / `an apple` and `decelerate` / `to decelerate` / `decelerate to`.
- Added a Google Sheets row parser scaffold under `internal/source/spreadsheet`: strict MVP headers are supported, translations split by comma, contexts split by dot, disabled rows are skipped, invalid `enabled` values become row-level errors, and rows emit common `vocabulary.Draft` values with sheet anchors.
- Added PocketBook and Google Sheets adapter boundaries as no-network placeholders. This keeps the compile/test path green while leaving room for the required future PocketBook API discovery scripts and sanitized fixtures under `scripts/pocketbook` and `test/fixtures/pocketbook`.
- Added an in-memory vocabulary repository for tests and a MongoDB repository boundary placeholder for future driver-backed persistence. The domain package depends only on the repository interface, not on storage implementation details.
- Added the project-local finalization skill at `.agents/skills/finalization/`, adapted from the user's Go finalization checklist style. It covers tests, `go test`/`go vet`/`go run`, Go package boundaries, vocabulary merge/display rules, source adapter rules, config/secrets, logs, docs alignment, idiomaticity, and commit preparation.
- Validation status after scaffold/domain changes: `GOCACHE=/tmp/vocabulary-bot-go-cache go test ./...`, `GOCACHE=/tmp/vocabulary-bot-go-cache go vet ./...`, and `GOCACHE=/tmp/vocabulary-bot-go-cache go run ./cmd/vocabulary-bot` passed. The explicit `GOCACHE` is used because the default sandbox build cache under the home directory can be read-only in this workspace.
