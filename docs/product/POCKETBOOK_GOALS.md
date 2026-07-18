# POCKETBOOK_GOALS.md

Last reviewed: 2026-06-14

## 1. Goal

Build a PocketBook integration that periodically imports saved dictionary translation notes from PocketBook Cloud into the vocabulary bot database.

The integration must support this workflow:

```text
PocketBook Era Color
  -> PocketBook Cloud sync
  -> PocketBook sync job
  -> parse translation notes
  -> normalize and merge words
  -> MongoDB vocabulary collection
  -> Telegram review notifications
```

The PocketBook part must be implemented as a source adapter. It must not contain Telegram-specific, scheduler-specific, or MongoDB-specific business logic beyond returning normalized import drafts.

## 2. Current state of PocketBook access

### 2.1 What is officially supported

PocketBook officially supports saving a selected word's translation and context as a note from the Dictionary panel. The saved translation becomes available in the book's Notes / table of contents. The dictionary panel also allows configuring how translation and context are saved as a note.

PocketBook officially supports exporting notes from the Notes app. Exported notes can be written into a chosen folder; by default, exported notes appear in the `Notes` folder.

PocketBook Reader / PocketBook Cloud is presented as a free sync service for books, reading positions, notes, and bookmarks across devices.

### 2.2 What is not officially guaranteed

As of this review date, there is no stable, publicly documented, official PocketBook Cloud API for third-party automation that we should treat as a guaranteed contract.

However, third-party open-source projects demonstrate that it is technically possible to log into PocketBook Cloud and fetch books, highlights, and notes:

- `PocketBook2Capacities` can fetch all books from a PocketBook Cloud library, retrieve highlights and notes per book, and supports CLI automation and incremental sync.
- `obsidian-pocketbook-cloud-highlight-importer` imports highlights and notes from a PocketBook Cloud account into Obsidian. Its setup notes say it works only with login by password and stores a refresh token rather than the password.

Therefore, the integration should be designed as an unofficial PocketBook Cloud adapter with a fallback to manual exported Notes files.

## 3. Cost assumption

The PocketBook Cloud sync path should be free for this use case in the sense that no paid PocketBook Cloud plan is currently required for syncing books, reading positions, notes, and bookmarks.

The risk is not price. The risk is API stability: the automation relies on non-public behavior used by community tools. PocketBook may change authentication, endpoints, response format, rate limits, or account flow.

## 4. Supported source modes

### 4.1 Primary mode: PocketBook Cloud sync

This is the target mode.

```text
PocketBook device syncs notes to PocketBook Cloud
  -> sync worker authenticates to PocketBook Cloud
  -> sync worker fetches books
  -> sync worker fetches notes/highlights for each book
  -> sync worker extracts translation notes
  -> sync worker returns PocketBookVocabularyDraft[]
```

### 4.2 Fallback mode: exported Notes files

This is not the main goal, but it must remain possible because it is the official/manual fallback.

```text
PocketBook device exports notes to Notes folder
  -> user uploads/copies exported HTML/text files
  -> parser reads exported notes
  -> parser extracts translation notes
  -> parser returns PocketBookVocabularyDraft[]
```

The fallback parser can be implemented later, but the core model must not depend on Cloud-only fields.

## 5. Authentication principle

### 5.1 Preferred authentication

Use normal PocketBook account login by email/password for bootstrap, then persist the refresh token or equivalent long-lived session token returned by the unofficial PocketBook Cloud login flow.

The user should not have to manually extract or paste a refresh token during normal setup. The bootstrap command should authenticate with email/password, capture the token from the login response, save it through the application's token store, and verify that the stored token can be used for a follow-up authenticated request.

After bootstrap, scheduled sync should prefer the stored refresh/session token and should use the password only when an explicit re-bootstrap is requested or when no valid stored token exists.

Do not store the PocketBook password in the application database.

Required bootstrap secrets:

```text
POCKETBOOK_EMAIL
POCKETBOOK_PASSWORD             # bootstrap-only secret; do not persist in the app database
POCKETBOOK_SHOP_NAME            # optional, only if account exposes multiple shops/stores
```

Optional override secrets:

```text
POCKETBOOK_REFRESH_TOKEN        # manual/CI override when a writable token store is unavailable
```

### 5.2 Social login is out of scope

Do not rely on Google/Facebook/social login for automation. Community tooling notes that the Obsidian importer works only with password login. Password login is the only practical target for an automated cron job.

### 5.3 Token storage

Refresh tokens, access tokens, cookies, and equivalent session values are secrets. Store only the minimum token material needed to refresh the PocketBook session.

Preferred application behavior:

```text
1. Bootstrap with POCKETBOOK_EMAIL and POCKETBOOK_PASSWORD.
2. Save the returned refresh/session token in the application's persistent token store.
3. During normal sync, load the stored token and refresh the short-lived access token if needed.
4. If refresh fails because the token is expired or revoked, report a clear re-bootstrap-required error.
```

For the current Go implementation, MongoDB is the preferred token store when `MONGODB_URI` is configured. PocketBook sessions are stored in the `pocketbook_sessions` collection with a stable `_id` for the current account. This collection is secret-bearing storage and must never store the PocketBook password.

When MongoDB is not configured, the app falls back to a file-backed secret store. By default it uses the user's config directory, or the explicit `POCKETBOOK_TOKEN_PATH` when configured. The file is written with owner-only permissions.

For GitHub Actions:

```text
POCKETBOOK_EMAIL        -> GitHub Actions secret
POCKETBOOK_PASSWORD     -> GitHub Actions secret, bootstrap-only if the workflow performs bootstrap
POCKETBOOK_REFRESH_TOKEN -> optional GitHub Actions secret override if DB token storage is not used
MONGODB_URI             -> GitHub Actions secret
TELEGRAM_BOT_TOKEN      -> GitHub Actions secret
TELEGRAM_ADMIN_ID -> GitHub Actions secret
```

If GitHub Actions uses MongoDB as the persistent application database, it also uses MongoDB as the token store. That avoids manually rotating `POCKETBOOK_REFRESH_TOKEN` in repository secrets after the initial bootstrap flow.

For a local runner or server:

```text
.env
credentials.json with chmod 600
system keychain / secret manager if available
MongoDB token store if the app already uses MongoDB
POCKETBOOK_TOKEN_PATH for an explicit file-backed token store location
```

No credentials, refresh tokens, raw API responses containing tokens, or cookies may be committed.

## 6. Data fetching principle

The PocketBook adapter must expose a small interface:

```ts
interface PocketBookSourceAdapter {
  authenticate(): Promise<PocketBookSession>;
  listBooks(session: PocketBookSession): Promise<PocketBookBook[]>;
  listBookNotes(session: PocketBookSession, book: PocketBookBook): Promise<PocketBookRawNote[]>;
  sync(): Promise<PocketBookVocabularyDraft[]>;
}
```

The adapter should support two sync modes:

```text
full sync        -> fetch all books and all notes
incremental sync -> fetch only books/notes changed since the last successful sync, if PocketBook data allows it
```

If the unofficial API does not provide reliable timestamps or cursors, incremental sync should still be safe by doing a full fetch and relying on database-level idempotent merge.

Current endpoint assumptions are based on public community clients and must be fixture-validated against the user's account:

```text
GET  /auth/login?username=...&client_id=...&client_secret=...
POST /auth/login/{shopAlias}
POST /auth/renew-token
GET  /books?limit=500
GET  /notes?fast_hash={bookFastHash}
GET  /notes/{uuid}?fast_hash={bookFastHash}
```

## 7. Raw data that should be preserved

Each imported note should keep enough PocketBook metadata to debug and prevent duplicates:

```ts
type PocketBookRawNote = {
  bookId?: string;
  bookTitle?: string;
  author?: string;
  noteId?: string;
  page?: number | string;
  position?: string;
  type?: string;
  text?: string;
  comment?: string;
  createdAt?: string;
  updatedAt?: string;
  raw: unknown;
};
```

Do not depend on all fields being present. The parser must be defensive.

## 8. Translation-note parsing model

PocketBook dictionary notes are not modeled as strict translation-context pairs in our database.

Important rule:

```text
Translations and contexts are independent arrays.
```

Reason:

PocketBook may show many translations for the same word. When the same word is saved in different places, the saved contexts may be different, but there is no reliable automated way to know which specific translation belongs to which specific context.

Therefore, a parsed PocketBook note should produce this draft shape:

```ts
type PocketBookVocabularyDraft = {
  source: 'pocketbook';

  word: string;
  translations: string[];
  contexts: string[];

  sourceMeta: {
    bookId?: string;
    bookTitle?: string;
    author?: string;
    sourceLabel?: string;
    noteId?: string;
    page?: number | string;
    position?: string;
  };

  raw: unknown;
};
```

The parser must extract:

```text
word          -> the selected dictionary word
translations  -> all translations available in the note
contexts      -> the saved context text, if present
sourceMeta    -> book/note/page/position information, if present
sourceLabel   -> `Title — Author` for books, `Document` for PDF library items
```

### 8.1 Dictionary-note context sources (validated on Era Color)

Real PocketBook Cloud payloads from a PocketBook Era Color account show that dictionary translation notes and book-text highlights are stored differently. This distinction matters for what the current parser can import without downloading the book file.

Validated example (`lean`, book `Necromancer`, dictionary note UUID `0314B80B-F5F6-5BC0-90E7-639DE5C1CA65`):

```text
mark.anchor     -> pbr:/word?page=11&offs=518
quotation.text  -> lean
note.text       -> full dictionary entry, including usage example lines such as
                   "to lean on a friend's advice - полагаться на совет друга"
draft.contexts  -> dictionary usage example after parser normalization
```

Interpretation:

```text
pbr:/word       -> dictionary lookup anchored to a word position in the book
pbr:/page       -> text highlight / quote saved directly from book text
quotation.text  -> for pbr:/word notes, usually only the headword
note.text       -> dictionary translation block; may also contain dictionary examples
```

The Era Color dictionary panel setting **Save translation as note** controls how translation and dictionary-side context are written into `note.text`. It does not, by itself, make PocketBook Cloud send a full book sentence in `quotation.text` for `pbr:/word` notes.

Current parser behavior for these notes:

```text
1. Parse translations and inline dictionary examples from note.text.
2. Move english-russian usage-example lines from translations[] into contexts[].
3. Add quotation.text to contexts[] only when it looks like a real sentence
   (longer than the headword; not just the selected word again).
```

Therefore, for dictionary notes the bot can reliably import dictionary examples today. It cannot import the surrounding book sentence from `quotation.text` alone when Cloud only stores the headword.

### 8.2 Book sentence backfill via book download (optional future path)

PocketBook Cloud already exposes enough metadata to attempt a second enrichment step: download the synced book file and derive the sentence around a saved word using the note anchor.

Observed API surface relevant to this path:

```text
GET /books?limit=500
  -> book id, title, mime_type, fast_hash, path

GET /notes?fast_hash={bookFastHash}
GET /notes/{uuid}?fast_hash={bookFastHash}
  -> mark.anchor, quotation.begin/end, quotation.text, note.text
```

Community PocketBook Cloud clients also demonstrate that book files can be downloaded from the user's cloud library after authentication. The exact download endpoint and response shape are unofficial and must be fixture-validated per account, but the capability is technically plausible because the book list already carries a `path` (or equivalent download locator).

Likely enrichment pipeline:

```text
1. Run the normal note sync and parse dictionary drafts as today.
2. For each pocketbook anchor with pbr:/word?page=X&offs=Y and no book-sentence context:
   a. ensure the book file is available locally (download once, cache by bookId/fast_hash)
   b. open the book in the same logical text stream PocketBook uses for offs
   c. locate character offset offs (or the range begin/end if more reliable)
   d. extract the surrounding sentence or paragraph
   e. append the extracted sentence to contexts[] as an independent context string
3. Merge into VocabularyItem using the same context dedupe rules as other sources.
```

Anchor fields to preserve for this work:

```text
bookId / fast_hash   -> choose and cache the correct book file
page                 -> debugging and fallback heuristics only; not sufficient alone
offs / begin / end   -> primary locator inside rendered book text
position raw string  -> keep unchanged in anchors[] for replay/debug
```

Important constraints:

```text
- offs is not a public, documented PocketBook contract. Treat it as an observed
  device/cloud coordinate that must be validated against real EPUB/PDF samples.
- PocketBook rendering may not equal raw EPUB text order. Extraction may require
  reproducing PocketBook's text normalization, HTML cleanup, or chapter assembly.
- Book format matters: EPUB is the likely first target; PDF/reflowable formats may
  need separate extractors or may be unsupported.
- Downloading every book on each sync is expensive. Cache book files on disk with
  invalidation by fast_hash or updated timestamp.
- Cloud sync may lag until the book is closed or the device sync completes.
- This enrichment must remain optional. Dictionary-note import must still work when
  book download or sentence extraction fails for a title.
```

Preferred source order for book sentence context:

```text
1. quotation.text when it already contains a full sentence (typical for pbr:/page highlights)
2. manually edited context saved on the device and present in note.text
3. dictionary usage examples parsed from note.text
4. optional future EPUB/offs sentence backfill for pbr:/word dictionary notes
```

This EPUB/offs backfill is implemented behind `POCKETBOOK_BOOK_CONTEXT_ENABLED`. During PocketBook sync the adapter processes each book sequentially: acquire the book file from the local cache (`$TMPDIR/vocabulary-bot-cache/books` by default, override with `POCKETBOOK_BOOK_CACHE_DIR`), enrich dictionary-word drafts that still need a book sentence, update `lastUsedAt`, evict entries older than `POCKETBOOK_BOOK_CACHE_MAX` (default `2`), and remove stale cached versions when the same `bookId` gets a new `fast_hash`. Words that already have a book-like sentence in Mongo are skipped during enrichment so reruns do not re-download or re-parse the book for them. The `/sync` Telegram summary reports `book context skipped (already in DB)` and `book context enriched` counters per source. Usage-example detection (`english - перевод`) requires Cyrillic on the translation side so book sentences with dashes are not misclassified.

Discovery tooling for this area:

```text
cmd/pocketbook-dump-note --word <word>
cmd/pocketbook-dump-note --uuid <note-uuid>
future: scripts/pocketbook-download-book --book <id>
future: scripts/pocketbook-extract-sentence --book <id> --offs <n>
```

## 9. Shared vocabulary entity

PocketBook and Google Sheet must write into the same database entity.

```ts
type VocabularyItem = {
  normalizedKey: string;

  displayWord: string;
  forms: VocabularyForm[];

  translations: string[];
  contexts: string[];

  anchors: SourceAnchor[];

  review: ReviewState;

  createdAt: Date;
  updatedAt: Date;
};

type VocabularyForm = {
  value: string;
  normalizedValue: string;
  count: number;
  firstSeenAt: Date;
  lastSeenAt: Date;
};

type SourceAnchor = {
  source: 'pocketbook' | 'google-sheet';
  externalId?: string;
  rowNumber?: number;
  bookId?: string;
  bookTitle?: string;
  author?: string;
  sourceLabel?: string;
  page?: number | string;
  position?: string;
};
```

## 10. Word normalization and matching

### 10.1 No hardcoded article/preposition filter

Do not maintain a configurable or hardcoded list like `to`, `a`, `an`, prepositions, particles, articles, etc.

There must be no linguistic whitelist/blacklist for edge tokens.

### 10.2 Keep display forms intact

Never cut the displayed word/form.

Examples of forms that must be preserved:

```text
to decelerate
decelerate to
an apple
apple an
in charge of
charge
```

The application may match them as related lookup candidates, but it must store the exact real form in `forms[]`.

### 10.3 Structural lookup keys

Generate several structural lookup keys from the raw form:

```text
full normalized compact form
form without the first token
form without the last token
form without first and last token, only if there are at least 3 tokens
```

This is not a filter by specific words. It is a structural matching strategy.

Example:

```text
raw: "to decelerate"
lookup candidates:
- todecelerate
- decelerate

raw: "decelerate to"
lookup candidates:
- decelerateto
- decelerate

raw: "in charge of"
lookup candidates:
- inchargeof
- chargeof
- incharge
- charge
```

When a new raw word arrives:

1. Build its lookup candidates.
2. Try to find an existing `VocabularyItem` by any candidate.
3. If found, merge into that item.
4. If not found, create a new item using the strongest/base candidate as `normalizedKey`.
5. Always add the exact raw form to `forms[]`.

The exact priority of candidates must be covered by tests, because aggressive matching can accidentally merge unrelated expressions.

## 11. Merge rules

For each `PocketBookVocabularyDraft`:

```text
1. Build lookup candidates from draft.word.
2. Find or create VocabularyItem.
3. Add exact draft.word to forms[].
4. Add all draft.translations to translations[] without duplicates.
5. Add all draft.contexts to contexts[] without duplicates.
6. Add source anchor without duplicates.
7. Update review metadata only if the word is new, unless explicitly configured otherwise.
```

Deduplication rules:

```text
word/form dedupe       -> compact lowercase normalized value
translation dedupe     -> trimmed lowercase value
context dedupe         -> trimmed normalized sentence/text
anchor dedupe          -> source + externalId if available, otherwise source + bookId + page + position + word
```

## 12. PocketBook sync scheduling

The PocketBook sync job should run before Google Sheet sync.

Recommended order:

```text
1. check global enabled flag
2. run PocketBook sync
3. run Google Sheet sync
4. merge all drafts into MongoDB
5. schedule due words
6. send Telegram notifications
7. write logs
```

Manual Telegram command `/sync` must trigger the same source-sync pipeline without waiting for the timer.

## 13. GitHub Actions feasibility

GitHub Actions can run this integration without a permanent server if we use scheduled workflows and external storage such as MongoDB Atlas free tier.

The workflow would be:

```text
GitHub Actions cron
  -> install dependencies
  -> authenticate to PocketBook Cloud
  -> fetch notes
  -> fetch Google Sheet rows
  -> merge into MongoDB
  -> send Telegram messages
  -> upload/commit logs if needed
```

Constraints:

```text
- GitHub-hosted runners are ephemeral.
- All state must live in MongoDB, GitHub artifacts, or committed state files.
- PocketBook Cloud refresh/session tokens must be stored in a secret-bearing persistent store, preferably MongoDB for this app; GitHub Secrets remain a valid override when DB token storage is unavailable.
- If the unofficial PocketBook auth flow requires browser interaction, a one-time local bootstrap may be needed.
- If PocketBook changes the private API, the workflow may break.
```

## 14. Logging requirements

All application logs must be saved to a log file.

A weekly job must:

```text
1. send the current LOG_PATH file to TELEGRAM_ADMIN_ID
2. only after successful Telegram delivery, truncate the active log file
3. on delivery failure, keep the active log and retry on the next AUTO_LOGS_CRON run
```

Scheduler env:

```text
AUTO_LOGS_CRON='0 21 * * 5'
```

The job respects `/turn_off` and `/turn_on` through the shared notifications flag, same as auto push.
Manual `/logs` still sends the current file on demand without clearing it.

The log file must not contain secrets, refresh tokens, passwords, cookies, or full raw API responses if they may contain credentials.

## 15. Telegram bot commands related to PocketBook

The bot must support these commands:

```text
/start
  Welcome message and command reference.

/info
  Health, schedule, and command reference.

/health
  Report health status, word count, and sync/notification flags.

/list_words
  Send a JSON file with all vocabulary items currently stored in the database.

/turn_off
  Disable automatic sync and notifications. Manual commands still work.

/turn_on
  Enable automatic sync and notifications.

/sync
  Manually synchronize all sources now, starting with PocketBook.

/push
  Manually send one review word with Easy/Hard buttons.

/logs
  Send the current log file without clearing it.
```

Admin commands and review buttons require the configured admin Telegram user ID. `/save` may be executed by any user in an allowlisted chat.

## 16. Error handling

PocketBook sync errors must not crash the whole bot if Google Sheet sync can still run.

Required behavior:

```text
- authentication failure -> log error, notify user in Telegram
- token expired -> try refresh once, then fail clearly
- book fetch failure -> retry with backoff
- note fetch failure for one book -> log and continue other books
- parse failure for one note -> save parse error summary and continue
- merge failure -> stop before notifications and notify user
```

## 17. Validation / proof-of-work checklist

Before considering the PocketBook integration done, verify:

```text
[ ] Device can save dictionary translation notes with translation and context.
[ ] Device syncs notes to PocketBook Cloud.
[ ] Adapter can authenticate using password login during bootstrap.
[ ] Bootstrap stores the returned refresh/session token without requiring manual token copy-paste.
[ ] Scheduled sync can authenticate using the stored refresh/session token.
[ ] Adapter can list books from the cloud library.
[ ] Adapter can fetch notes/highlights for at least one book.
[ ] Parser can identify translation notes.
[ ] Parser extracts word, translations, and context from real notes.
[ ] Same word saved in different contexts merges into one VocabularyItem.
[ ] Multiple translations are stored as independent translations[].
[ ] Multiple contexts are stored as independent contexts[].
[ ] Exact raw forms are preserved in forms[].
[ ] Structural matching works without an article/preposition whitelist.
[ ] Full sync is idempotent.
[ ] Manual /sync command works.
[ ] Logs are written and /logs sends them to Telegram.
```

## 18. Implementation phases

### Phase 1: PocketBook API discovery

Implement a small isolated script:

```text
scripts/pocketbook-auth-check
scripts/pocketbook-list-books
cmd/pocketbook-dump-note --word <word>
cmd/pocketbook-dump-note --uuid <note-uuid>
scripts/pocketbook-dump-notes --book <id>
scripts/pocketbook-download-book --book <id>   # future, for sentence backfill discovery
```

Goal: obtain real sanitized JSON fixtures from the user's account and validate whether book download plus `offs` anchors can recover book sentences for `pbr:/word` dictionary notes.

No parser logic should be finalized before looking at real PocketBook translation-note payloads.

### Phase 2: Parser and fixtures

Create test fixtures from sanitized real notes:

```text
test/fixtures/pocketbook/translation-note-single.json
test/fixtures/pocketbook/translation-note-multiple-translations.json
test/fixtures/pocketbook/non-translation-highlight.json
test/fixtures/pocketbook/comment-note.json
```

Add parser tests before connecting to MongoDB.

### Phase 3: Merge into vocabulary model

Implement source-agnostic merge logic shared by PocketBook and Google Sheet.

PocketBook and Google Sheet must both produce the same internal draft type.

### Phase 4: Scheduled sync

Add scheduled execution:

```text
PocketBook first
Google Sheet second
review push after successful merge
```

### Phase 5: Operational commands and logs

Add Telegram commands:

```text
/start
/info
/health
/list_words
/turn_off
/turn_on
/sync
/push
/logs
```

Weekly log delivery and rotation are implemented via `AUTO_LOGS_CRON` (default Friday 21:00).

## 19. Known risks

```text
- PocketBook Cloud API is unofficial for third-party automation.
- Endpoint shape may change without notice.
- Login flow may change and break cron automation.
- Social login is not a good target for automation.
- Some notes may be stored differently depending on book format, firmware, dictionary, or save-note setting.
- Dictionary notes (`pbr:/word`) often store only the headword in `quotation.text`; book sentences may require highlight notes (`pbr:/page`) or a future book-download/offs enrichment step.
- `offs` anchors are observed coordinates, not a documented public contract; sentence extraction may break across firmware or book formats.
- PocketBook sync may lag until the book is closed or manual sync is triggered.
- Translation notes may not have clean word/translation/context fields; parsing may require heuristics.
```

## 20. Decision

Use PocketBook Cloud as the primary automated source, but treat it as an unofficial adapter.

The implementation must be defensive, fixture-driven, and idempotent. It must preserve raw forms, store translations and contexts independently, and must not use any hardcoded list of articles, prepositions, or particles for word matching.
