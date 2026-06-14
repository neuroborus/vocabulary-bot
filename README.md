# Vocabulary Bot

Vocabulary Bot imports words from PocketBook notes and Google Sheets, merges them into one vocabulary model, and sends Telegram review reminders.

The current repository is a Go foundation scaffold. It keeps runtime adapters separate from source-agnostic vocabulary rules.

## Structure

```text
cmd/vocabulary-bot/          executable entrypoint
internal/app/                composition root
internal/config/             environment configuration
internal/vocabulary/         domain model, normalization, merge rules
internal/source/             common source adapter contract
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

`AGENTS.md` is the operational guide for AI coding agents.
`docs/README.md` is the gate/index for product docs.
`RHYTHM.md` is the chronological log of meaningful repository decisions.

## Commands

```bash
make fmt
make test
make vet
make run
```

Runtime code uses the official MongoDB Go driver when `MONGODB_URI` is configured. Most tests use fake HTTP servers and in-memory storage. MongoDB storage tests connect to `MONGODB_URI` or `mongodb://127.0.0.1:27017` and skip when MongoDB is unavailable.

## Storage

If `MONGODB_URI` is set, the app uses MongoDB:

- `vocabulary_items` stores merged vocabulary entities;
- `pocketbook_sessions` stores the current PocketBook access/refresh session.

If `MONGODB_URI` is empty, the app falls back to an in-memory vocabulary repository and a file-backed PocketBook session store.

## PocketBook Sync

The PocketBook adapter uses the unofficial PocketBook Cloud API flow observed in community clients:

- discover shops by email;
- bootstrap with `POCKETBOOK_EMAIL` and `POCKETBOOK_PASSWORD`;
- store the returned access/refresh session in MongoDB `pocketbook_sessions`, or in an owner-only file when MongoDB is not configured;
- renew the session with the stored refresh token;
- fall back to password bootstrap and replace the stored session when the old refresh token is rejected;
- fetch books, note IDs, note details, and emit common `vocabulary.Draft` values.

Useful configuration:

```bash
POCKETBOOK_EMAIL=
POCKETBOOK_PASSWORD=
POCKETBOOK_SHOP_NAME=
POCKETBOOK_TOKEN_PATH=
POCKETBOOK_REFRESH_TOKEN=
POCKETBOOK_BOOK_CONTEXT_ENABLED=true
```

When `POCKETBOOK_BOOK_CONTEXT_ENABLED=true`, sync downloads each book file once per book (EPUB or FB2), extracts dictionary-word sentences by `offs`, appends them to `contexts`, and deletes the temp file before the next book.

Discovery helpers:

```bash
go run ./cmd/pocketbook-list-books
go run ./cmd/pocketbook-dump-note -word lean -limit 2
go run ./cmd/pocketbook-dump-note -uuid <note-uuid>
```

`POCKETBOOK_REFRESH_TOKEN` is only an override. Normal setup should let the app capture and persist the token automatically. Because the API is unofficial, real account payloads still need sanitized fixtures before tightening dictionary-note parsing.

## Telegram

When `TELEGRAM_BOT_TOKEN`, `TELEGRAM_ALLOWED_USER_ID`, and `TELEGRAM_POLLING_ENABLED=true` are configured, the app starts Telegram long polling after startup sync.

Minimal commands:

```text
/start      Welcome message and command reference
/info       Health snapshot plus command reference
/health     Health status, word count, and enabled flags
/sync       Sync PocketBook and Google Sheets into storage now
/list_words Send all vocabulary items as a JSON file
/logs       Send the current log file without clearing it
/turn_off   Disable automatic sync and notifications
/turn_on    Enable automatic sync and notifications
/push       Manually send one review word with Easy/Hard/Remove buttons
```

Only the configured `TELEGRAM_ALLOWED_USER_ID` may execute commands. Service notifications, including the startup message, are always sent to that user ID. Command responses are sent back to the chat where the command was sent. Review push cards go to `TELEGRAM_TARGET_CHAT_ID` when it is configured. Set `TELEGRAM_REVIEW_SPOILER_TRANSLATIONS=false` to show translations openly on `/push` cards. Scheduled jobs use `AUTO_SYNC_CRON` and `AUTO_PUSH_CRON` (5-field cron); they respect `/turn_off` and `/turn_on` for sync and notification flags. `/turn_off` blocks `/sync` until `/turn_on` is used again; other manual commands still work.

## Main Rules Already Encoded

- PocketBook and Google Sheets produce the same `vocabulary.Draft`.
- `VocabularyItem` stores translations and contexts as independent arrays.
- Raw visible forms are preserved in `forms`.
- Lookup matching uses structural edge-token candidates, not hardcoded article/preposition lists.
- Source adapters do not write to storage directly; common merge logic owns vocabulary updates.
