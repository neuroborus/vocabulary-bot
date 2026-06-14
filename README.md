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

The scaffold uses only the Go standard library for now, so tests run without network access or external services.

## PocketBook Sync

The PocketBook adapter uses the unofficial PocketBook Cloud API flow observed in community clients:

- discover shops by email;
- bootstrap with `POCKETBOOK_EMAIL` and `POCKETBOOK_PASSWORD`;
- store the returned access/refresh session in an owner-only file;
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
```

`POCKETBOOK_REFRESH_TOKEN` is only an override. Normal setup should let the app capture and persist the token automatically. Because the API is unofficial, real account payloads still need sanitized fixtures before tightening dictionary-note parsing.

## Main Rules Already Encoded

- PocketBook and Google Sheets produce the same `vocabulary.Draft`.
- `VocabularyItem` stores translations and contexts as independent arrays.
- Raw visible forms are preserved in `forms`.
- Lookup matching uses structural edge-token candidates, not hardcoded article/preposition lists.
- Source adapters do not write to storage directly; common merge logic owns vocabulary updates.
