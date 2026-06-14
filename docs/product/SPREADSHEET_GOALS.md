# SPREADSHEET_GOALS.md

Updated: 2026-06-14

## Purpose

The spreadsheet integration is an additional vocabulary source for the PocketBook vocabulary bot.

It must allow the user to manually add, edit, and inspect words through a simple Google Sheet, while the application keeps the same internal vocabulary model for all sources.

The spreadsheet is not a separate vocabulary database. It is one of the input adapters that feeds the common vocabulary pipeline.

```text
PocketBook notes
Google Sheet rows
      ↓
Vocabulary drafts
      ↓
Normalization + merge
      ↓
MongoDB VocabularyItem
      ↓
Telegram review bot
```

## Main goals

1. Use Google Sheets as a free, human-editable vocabulary source.
2. Read manually added words from the sheet.
3. Parse translations and contexts from plain text cells.
4. Merge spreadsheet words into the same MongoDB entity used for PocketBook words.
5. Preserve all real display forms entered by the user.
6. Avoid any hardcoded or configurable list of articles, prepositions, particles, or similar words.
7. Support manual sync through the Telegram `/sync` command.
8. Keep the implementation simple enough to run from GitHub Actions or a lightweight scheduled process.

## Non-goals

1. Do not make Google Sheets the primary database.
2. Do not build a complex two-way editor in the first version.
3. Do not try to automatically bind a translation to a specific context.
4. Do not use filters like `to / a / an` or any predefined list of articles/prepositions.
5. Do not depend on paid Google Workspace features.
6. Do not require a permanent public server.

## Free usage expectation

Google Sheets API is suitable for this project without paid infrastructure, as long as the project stays within normal API quotas.

The implementation still needs a Google Cloud project with the Google Sheets API enabled and valid credentials. This does not mean the application needs a permanent server.

The expected usage is very small:

```text
read full vocabulary sheet once per scheduled sync
optionally update sync metadata/status cells
optionally append imported/exported rows later
```

This is far below the normal per-minute limits for the Google Sheets API.

## Recommended connection approach

### Preferred approach: Service Account

Use a Google Cloud service account for the bot.

The spreadsheet owner shares the target Google Sheet with the service account email, for example:

```text
vocabulary-bot@<project-id>.iam.gserviceaccount.com
```

Recommended permission:

```text
Editor
```

Editor is useful because the bot may later write sync metadata, status columns, or normalized keys back to the sheet.

For a read-only MVP, Viewer is enough, but Editor is more practical for long-term automation.

### Required secrets

Store credentials in GitHub Actions Secrets or local `.env` variables.

```text
GOOGLE_SERVICE_ACCOUNT_JSON=<full service account JSON or base64-encoded JSON>
GOOGLE_SPREADSHEET_ID=<spreadsheet id>
GOOGLE_SHEET_NAME=Vocabulary
```

Optional:

```text
GOOGLE_SHEET_RANGE=Vocabulary!A:Z
```

### Why not OAuth user login for MVP

OAuth user login is useful for desktop or multi-user apps, but it is less convenient for scheduled jobs.

For this project, a service account is simpler because:

1. It works in GitHub Actions.
2. It does not require browser login during every deployment.
3. It does not require refresh-token handling in the MVP.
4. Access can be revoked by removing the service account from the sheet.

## Spreadsheet structure

Recommended sheet name:

```text
Vocabulary
```

Recommended columns:

| Column | Name | Required | Purpose |
|---|---|---:|---|
| A | word | Yes | Word or expression as entered by the user |
| B | translations | No | Comma-separated translations |
| C | contexts | No | Dot-separated contexts/sentences |
| D | note | No | Optional free-form note |
| E | tags | No | Optional comma-separated tags |
| F | source | No | Optional manual source label |
| G | enabled | No | Optional row-level switch |
| H | normalizedKey | No | Optional value written by the app later |
| I | lastSyncedAt | No | Optional sync metadata written by the app later |
| J | syncStatus | No | Optional sync status written by the app later |

Minimal MVP columns:

```text
word | translations | contexts
```

Recommended manual-import CSV columns:

```text
word | translations | contexts | note | tags | enabled
```

Only `word` is required by the parser. `translations` and `contexts` are optional technically, but a useful vocabulary row should normally provide both. `note`, `tags`, and `enabled` are optional convenience columns for manual Google Sheets maintenance and can be left empty.

## Cell parsing rules

### Word

The `word` cell is treated as a real display form.

Examples:

```text
deceleration
to slow down
look after
a matter of time
matter of time in
```

The displayed form must not be destructively normalized.

### Translations

Translations are split by comma for dictionary-style headwords.

Example:

```text
замедление, снижение скорости, торможение
```

When the `word` cell itself contains a comma, the row is treated as a sentence-style entry and the whole `translations` cell is kept as one value. This avoids splitting Russian or English sentence punctuation into fake glosses.

Parsed as:

```json
[
  "замедление",
  "снижение скорости",
  "торможение"
]
```

Rules:

1. Trim spaces around each translation.
2. Drop empty entries.
3. Deduplicate translations case-insensitively inside the same word entity.
4. Do not link translations to contexts.

### Contexts

Contexts are split by dot.

Example:

```text
The car began to slow down. He had to slow down before the turn.
```

Parsed as:

```json
[
  "The car began to slow down",
  "He had to slow down before the turn"
]
```

Rules:

1. Trim spaces around each context.
2. Drop empty entries.
3. Deduplicate contexts by normalized text.
4. Do not link contexts to translations.
5. For MVP, one context should normally be one sentence.

Important limitation:

Using a dot as a context separator means abbreviations and multi-sentence examples may be split incorrectly. This is acceptable for MVP because the user explicitly chose a simple dot-based rule. If this becomes painful later, the project can migrate to a stronger separator such as `|||`.

### Note

`note` is an optional free-form note cell.

Use it for information that is useful but is not a translation or a usage context:

```text
Pronunciation: квэйнт
Used mostly as an idiom
Formal legal term
```

Notes are split by newline.

Rules:

1. Trim spaces around each note.
2. Drop empty entries.
3. Deduplicate notes by normalized text.
4. Do not put source dates or import bookkeeping here unless they are useful to the user during review.

### Tags

`tags` is an optional comma-separated label list.

Examples:

```text
idiom, legal, business
```

Parsed as:

```json
[
  "idiom",
  "legal",
  "business"
]
```

Rules:

1. Trim spaces around each tag.
2. Drop empty entries.
3. Deduplicate tags case-insensitively inside the same word entity.
4. Keep tags short and user-facing.

### Enabled

`enabled` is an optional row-level import switch.

Empty `enabled` values are treated as enabled. The parser accepts these enabled values:

```text
true, yes, 1, on
```

The parser accepts these disabled values:

```text
false, no, 0, off
```

Disabled rows are skipped during spreadsheet sync. Invalid values produce a row-level parser error and the row is skipped.

## Common internal model

Spreadsheet rows and PocketBook notes must be converted into the same draft format before merging.

```ts
type VocabularyDraft = {
  rawWord: string;
  translations: string[];
  contexts: string[];
  note?: string;
  tags?: string[];
  anchor: SourceAnchor;
};

type SourceAnchor = {
  source: 'pocketbook' | 'google-sheet';
  externalId?: string;
  rowNumber?: number;
  bookTitle?: string;
  sheetName?: string;
};
```

The stored entity must keep translations and contexts independent.

```ts
type VocabularyItem = {
  normalizedKey: string;
  displayWord: string;
  forms: VocabularyForm[];

  translations: string[];
  contexts: string[];

  tags: string[];
  notes: string[];
  anchors: SourceAnchor[];

  enabled: boolean;
  review: ReviewState;

  createdAt: string;
  updatedAt: string;
};

type VocabularyForm = {
  value: string;
  normalizedValue: string;
  count: number;
  firstSeenAt: string;
  lastSeenAt: string;
};

type ReviewState = {
  dueAt?: string;
  lastPushedAt?: string;
  intervalDays?: number;
  ease?: number;
  pushCount: number;
};
```

## Normalization and matching

### Core principle

The app must not use a predefined list of articles, prepositions, particles, or similar words.

There must be no hardcoded or configurable filter like:

```text
to, a, an, the, in, on, at, of, for, with, ...
```

The app should instead generate structural lookup candidates from the entered phrase.

### Display preservation

The entered word must always be preserved in `forms[]`.

If the user enters:

```text
to slow down
```

then `to slow down` remains visible as a real form.

If the user later enters:

```text
slow down
```

both forms are preserved.

### Base normalization

Base text normalization:

```ts
function normalizeText(value: string): string {
  return value
    .toLowerCase()
    .normalize('NFKC')
    .replace(/[’‘]/g, "'")
    .replace(/[‐‑‒–—]/g, '-')
    .replace(/\s+/g, ' ')
    .trim();
}

function compactKey(value: string): string {
  return normalizeText(value)
    .replace(/[\s-]+/g, '');
}
```

Examples:

```text
Ice cream      → icecream
ice-cream      → icecream
Ice‑cream      → icecream
TO SLOW DOWN   → toslowdown
```

### Structural lookup candidates

For a phrase, generate lookup candidates without knowing which tokens are articles or prepositions.

```ts
function getLookupCandidates(rawWord: string): string[] {
  const normalized = normalizeText(rawWord);
  const tokens = normalized.split(' ').filter(Boolean);

  const candidates = new Set<string>();

  candidates.add(compactKey(normalized));

  if (tokens.length > 1) {
    candidates.add(compactKey(tokens.slice(1).join(' ')));       // remove first token
    candidates.add(compactKey(tokens.slice(0, -1).join(' ')));   // remove last token
  }

  return [...candidates];
}
```

Examples:

```text
to decelerate
  candidates: todecelerate, decelerate

decelerate to
  candidates: decelerateto, decelerate

a matter of time
  candidates: amatteroftime, matteroftime, amatterof

matter of time in
  candidates: matteroftimein, oftimein, matteroftime
```

### Matching rule

When importing a draft:

1. Generate lookup candidates from `rawWord`.
2. Search for an existing `VocabularyItem` whose `normalizedKey` matches any candidate.
3. Also search by any indexed `lookupKeys` if implemented.
4. If found, merge into that item.
5. If not found, create a new item with the full compact key as `normalizedKey`.

Recommended stored lookup fields:

```ts
type VocabularyItem = {
  normalizedKey: string;
  lookupKeys: string[];
  // ...
};
```

When an item is created or updated, add all structural lookup candidates from all known forms to `lookupKeys`.

This allows future entries with leading or trailing extra tokens to match the same entity without knowing whether the token is an article, preposition, particle, or something else.

### Conflict risk

This structural approach can sometimes merge words that should remain separate.

Example:

```text
turn on
turn
```

The phrase `turn on` may generate `turn` as a candidate after removing the last token. This is not always semantically correct.

MVP decision:

1. Accept this risk initially.
2. Preserve all forms in `forms[]` so the user can see what happened.
3. Add a future manual command or admin tool to split incorrectly merged items if needed.

## Merge rules

For every parsed spreadsheet row:

```text
1. Skip row if word is empty.
2. Skip row if enabled is explicitly false/no/0.
3. Build VocabularyDraft.
4. Generate lookup candidates.
5. Find existing VocabularyItem.
6. Merge or create.
7. Preserve row anchor.
8. Save DB changes.
9. Optionally update metadata columns in the sheet.
```

### Translation merge

```text
add new translations to translations[]
do not create duplicates
do not link translations to contexts
```

### Context merge

```text
add new contexts to contexts[]
do not create duplicates
do not link contexts to translations
```

### Form merge

```text
add rawWord to forms[] if it does not exist
increment count if it already exists
update firstSeenAt / lastSeenAt
```

### Display word selection

`displayWord` should prefer the clearest or most complete form.

Simple MVP rule:

```text
if new form has more tokens than current displayWord, use the new form
otherwise keep existing displayWord
```

This means if the app first sees:

```text
decelerate
```

and later sees:

```text
to decelerate
```

then `displayWord` can become `to decelerate`, while both forms remain preserved.

## Google Sheets read strategy

Use `spreadsheets.values.get` or `spreadsheets.values.batchGet` for reading the vocabulary range.

For this project, a single full-range read is enough:

```text
Vocabulary!A:F
```

Recommended sync flow:

```text
1. Fetch header row.
2. Map column names to indices.
3. Read all non-empty rows.
4. Convert rows to VocabularyDraft objects.
5. Merge into MongoDB.
6. Optionally write sync status back.
```

Column names should be case-insensitive:

```text
word, Word, WORD → word
translations, translation, meaning, meanings → translations
contexts, context, examples, example → contexts
```

The MVP can start with strict names only:

```text
word | translations | contexts
```

## Google Sheets write strategy

Writing back to the sheet is optional for MVP.

If enabled, write only metadata columns:

```text
normalizedKey
lastSyncedAt
syncStatus
```

Do not rewrite user-owned columns automatically:

```text
word
translations
contexts
note
tags
```

This prevents accidental data loss.

Use `spreadsheets.values.batchUpdate` to update metadata for multiple rows in one request.

## Error handling

### Row-level errors

Bad rows should not break the whole sync.

Examples:

```text
missing word
invalid enabled value
oversized cell
unsupported data type
```

Each row-level error should be logged with:

```text
sheet name
row number
problem
raw row snapshot if safe
```

### API errors

Handle Google API errors explicitly:

```text
401 / 403: credentials or sharing problem
404: spreadsheet id or sheet name problem
429: rate limit, retry with exponential backoff
5xx: retry with exponential backoff
```

### Sync safety

The sync should be idempotent.

Running `/sync` multiple times should not duplicate translations, contexts, forms, or anchors.

## Telegram integration

The spreadsheet source participates in existing bot commands.

### `/start`

Shows a welcome message and the full command reference.

Does not run sync or change runtime settings.

### `/info`

Shows the current health snapshot plus the full command reference.

Useful when you want status and help in one message.

### `/sync`

Manually runs source synchronization.

Expected behavior:

```text
1. Check whether sync is enabled.
2. Run PocketBook sync.
3. Run Google Sheet sync.
4. Merge all drafts into MongoDB.
5. Return short Telegram summary.
```

Example response:

```text
Sync completed.
PocketBook: 12 notes processed, 3 new words.
Google Sheet: 18 rows processed, 5 new words, 7 updated words.
```

### `/list_words`

Sends a JSON file with all words from MongoDB.

Spreadsheet-origin words are not separated from PocketBook-origin words, but every item includes anchors.

### `/health`

Should include spreadsheet adapter status:

```json
{
  "spreadsheet": {
    "enabled": true,
    "lastSyncAt": "2026-06-14T08:00:00.000Z",
    "lastStatus": "ok",
    "spreadsheetIdConfigured": true,
    "sheetName": "Vocabulary"
  }
}
```

### `/turn_off`

Turns off both:

```text
word notifications
source synchronization
```

### `/turn_on`

Turns both back on.

### `/push`

Placeholder in the current MVP.

Expected future behavior:

```text
1. Pick the next due word.
2. If no words are due, optionally pick the oldest enabled word.
3. Send a normal reminder message with buttons.
4. Update lastPushedAt.
```

### `/logs`

Sends the current log file to the user in Telegram.

Spreadsheet sync errors must be present in this file.

## Scheduled behavior

If the bot uses GitHub Actions:

```text
scheduled sync job
  → read PocketBook
  → read Google Sheet
  → merge
  → send due word if enabled
```

If the bot runs as a local/server process:

```text
cron / scheduler
  → run same sync pipeline
```

The source adapter must not care whether it is called from GitHub Actions, local cron, or a long-running process.

## Logging requirements

All logs must be saved to a file.

Recommended path:

```text
logs/vocabulary.log
```

Weekly behavior:

```text
1. Send log file to the user in Telegram.
2. If sending succeeds, rotate or clear the log file.
3. If sending fails, keep the file and try again later.
```

Spreadsheet adapter logs should include:

```text
sync started
rows read
rows skipped
words created
words updated
metadata writes
Google API errors
row parsing errors
sync completed
```

## Suggested implementation modules

```text
src/sources/spreadsheet/
  spreadsheet.config.ts
  spreadsheet.client.ts
  spreadsheet-auth.ts
  spreadsheet-row.parser.ts
  spreadsheet-source.adapter.ts
  spreadsheet-sync.service.ts
```

Responsibilities:

### `spreadsheet.config.ts`

Loads and validates:

```text
GOOGLE_SERVICE_ACCOUNT_JSON
GOOGLE_SPREADSHEET_ID
GOOGLE_SHEET_NAME
GOOGLE_SHEET_RANGE
```

### `spreadsheet.client.ts`

Thin wrapper over Google Sheets API.

Methods:

```ts
readRows(): Promise<SpreadsheetRow[]>;
writeSyncMetadata(rows: SpreadsheetSyncMetadata[]): Promise<void>;
```

### `spreadsheet-row.parser.ts`

Converts raw sheet rows into `VocabularyDraft[]`.

### `spreadsheet-source.adapter.ts`

Implements common source interface:

```ts
interface VocabularySourceAdapter {
  name: string;
  sync(): Promise<VocabularyDraft[]>;
}
```

### `spreadsheet-sync.service.ts`

Coordinates:

```text
read rows
parse rows
return drafts
optionally update metadata
```

The actual merge into MongoDB should happen in the common vocabulary service, not inside the spreadsheet adapter.

## Acceptance criteria

1. The app can read vocabulary rows from Google Sheets.
2. The app can parse comma-separated translations.
3. The app can parse dot-separated contexts.
4. The app stores translations and contexts as independent arrays.
5. Spreadsheet words and PocketBook words merge into one `VocabularyItem` collection.
6. No hardcoded or configurable list of articles/prepositions/particles exists.
7. Leading and trailing extra tokens are handled through structural lookup candidates.
8. Original forms are preserved in `forms[]`.
9. `/sync` runs spreadsheet synchronization manually.
10. `/list_words` exports all stored words as JSON.
11. `/health` reports spreadsheet adapter status.
12. Spreadsheet sync logs are written to file.
13. `/logs` sends the current log file to Telegram.
14. Weekly log sending includes spreadsheet logs and clears the file only after successful delivery.

## Future improvements

1. Add manual split/merge tools for incorrectly merged vocabulary items.
2. Add sheet metadata writeback.
3. Add validation warnings directly in the sheet.
4. Add stronger separators for contexts, for example `|||`.
5. Add multiple sheet support.
6. Add import conflict report.
7. Add a `disabled` status per word in MongoDB and optionally sync it from the sheet.
8. Add bidirectional export from MongoDB to Google Sheets for review.

## References

- Google Sheets API overview: https://developers.google.com/workspace/sheets/api/guides/concepts
- Google Sheets API usage limits: https://developers.google.com/workspace/sheets/api/limits
- Google Sheets API values guide: https://developers.google.com/workspace/sheets/api/guides/values
- Google Sheets API `values.batchGet`: https://developers.google.com/workspace/sheets/api/reference/rest/v4/spreadsheets.values/batchGet
- Google Sheets API `values.batchUpdate`: https://developers.google.com/workspace/sheets/api/reference/rest/v4/spreadsheets.values/batchUpdate
- GitHub Actions secrets: https://docs.github.com/actions/security-guides/using-secrets-in-github-actions
