# Vocabulary Bot

Personal vocabulary assistant in Go. It imports words from PocketBook notes and Google Sheets, merges them into one model in MongoDB, and sends spaced-repetition review cards to Telegram.

Example review channel: [t.me/vocabulary_list](https://t.me/vocabulary_list)

## What it does

- syncs PocketBook Cloud dictionary notes and a Google Sheets vocabulary table;
- merges both sources into shared `VocabularyItem` records without losing raw forms;
- sends `/push` review cards to a Telegram channel or group;
- runs scheduled sync, push, and weekly log delivery via cron;
- exposes admin commands over Telegram long polling.

Sync order is PocketBook first, then Google Sheets. Repeat syncs skip unchanged spreadsheet rows when the row fingerprint matches the stored anchor.

## Quick start

```bash
cp .env.example .env
# fill in MongoDB, Telegram, PocketBook, and Google Sheets values
make run
```

Useful local commands:

```bash
make fmt
make test
make vet
make run
```

Most tests use fake HTTP servers and in-memory storage. MongoDB storage tests connect to `MONGODB_URI` or `mongodb://127.0.0.1:27017` and skip when MongoDB is unavailable.

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
internal/schedule/           in-process cron jobs
docs/                        product/reference docs
test/fixtures/               sanitized external payload fixtures
```

- `AGENTS.md` — operational guide for AI coding agents
- `docs/README.md` — index for product docs
- `RHYTHM.md` — chronological decision log

## Telegram

When `TELEGRAM_BOT_TOKEN`, `TELEGRAM_ADMIN_ID`, and `TELEGRAM_POLLING_ENABLED=true` are configured, the app starts Telegram long polling after startup sync.

Commands:

```text
/start      Welcome message and command reference
/info       Health, schedule, and command reference
/health     Health status, word count, and enabled flags
/sync       Sync PocketBook and Google Sheets into storage now
/list_words Send all vocabulary items as a JSON file
/logs       Send the current log file without clearing it
/turn_off   Disable automatic sync and notifications
/turn_on    Enable automatic sync and notifications
/push       Manually send one review word with Easy/Hard buttons
/save       Append vocabulary from a message or reply to Google Sheets via OpenAI
```

Only `TELEGRAM_ADMIN_ID` may run admin commands. `/save` is accepted from any user inside allowlisted chats: `TELEGRAM_ADMIN_ID`, `TELEGRAM_TARGET_CHANNEL_ID`, plus any IDs in `TELEGRAM_ALLOWED_CHAT_IDS`. `/sync` and `/logs` work only in a private chat with the bot (`chat_id` equals `TELEGRAM_ADMIN_ID`). Review callbacks are accepted only from the admin user; other users see a popup alert that they are not authorized to vote. By default the bot leaves other groups and channels; set `TELEGRAM_LEAVE_DISALLOWED_CHATS=false` to disable auto-leave while still ignoring updates from non-allowlisted chats. Service notifications go to the admin. Review cards go to `TELEGRAM_TARGET_CHANNEL_ID` — for example, a public channel like [t.me/vocabulary_list](https://t.me/vocabulary_list).

`/save` structures the message with OpenAI and appends one row to Google Sheets only. Use it as a reply to the message you want to save, or write the text before or after `/save`. The new row is imported into local storage by the normal `/sync` path.

Push cards show source at the bottom (`Title — Author` for books, sheet name or `Document` for spreadsheet/PDF words), hide translations behind a spoiler by default, and update in place after Easy/Hard is chosen.

Priority tuning:

- `REVIEW_DOCUMENT_PUSH_FACTOR` — spreadsheet-only and PDF `Document` words
- `REVIEW_BOOK_PUSH_FACTOR` — PocketBook book anchors

Scheduled jobs use `AUTO_SYNC_CRON`, `AUTO_PUSH_CRON`, and `AUTO_LOGS_CRON`. `/turn_off` blocks automatic sync and `/sync` until `/turn_on` is used again.

## Sources

### PocketBook Cloud

Unofficial community-observed API flow:

- discover shops by email;
- bootstrap with `POCKETBOOK_EMAIL` and `POCKETBOOK_PASSWORD`;
- persist access/refresh session in MongoDB `pocketbook_sessions` or a local file;
- renew with stored refresh token, fall back to password bootstrap when needed;
- fetch books, note IDs, and note details into `vocabulary.Draft`.

When `POCKETBOOK_BOOK_CONTEXT_ENABLED=true`, sync downloads each book once (EPUB/FB2), extracts dictionary-word sentences by `offs`, and caches files under `$TMPDIR/vocabulary-bot-cache/books`.

Discovery helpers:

```bash
go run ./cmd/pocketbook-list-books
go run ./cmd/pocketbook-dump-note -word lean -limit 2
go run ./cmd/pocketbook-dump-note -uuid <note-uuid>
```

### Google Sheets

Reads rows from `GOOGLE_SHEET_RANGE` using a service account (`GOOGLE_SERVICE_ACCOUNT_JSON`). Expected columns: `word`, `translations`, `contexts`, `note`, `tags`, `enabled`. See `docs/product/SPREADSHEET_GOALS.md` for the import template.

## Storage

When `MONGODB_URI` is set:

- `vocabulary_items` — merged vocabulary entities
- `pocketbook_sessions` — current PocketBook session

Without MongoDB, vocabulary stays in memory and PocketBook session falls back to a file store.

## Deploy

Production deploy uses a small Alpine Docker image (~43 MB) and GitHub Actions over SSH. Runtime `.env` is rendered from GitHub secrets and variables on each deploy.

See [deploy/README.md](deploy/README.md) for server setup, `DEPLOY_SSH`, `DEPLOY_HOST`, and manual operations.

```bash
docker build -t vocabulary-bot:latest .
docker compose up -d
```

## Core rules

- PocketBook and Google Sheets both produce `vocabulary.Draft`.
- `translations` and `contexts` are independent arrays.
- Raw visible forms are always preserved in `forms`.
- Lookup matching uses structural edge-token candidates, not article/preposition whitelists.
- Source adapters do not write to storage; common merge logic owns updates.
- Ambiguous matches are reported, not auto-merged.
