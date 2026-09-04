# PocketBook Vocabulary Bot — Implementation Plan

## 1. Goal

Build a free vocabulary reminder workflow for words saved while reading on PocketBook Era Color.

The system should:

- synchronize saved dictionary notes from PocketBook;
- optionally synchronize manually maintained words from Google Sheets;
- store all words from all sources in one shared MongoDB entity;
- deduplicate words by normalized lookup keys;
- preserve all visible word forms exactly as they appeared, including articles, prepositions, particles, and other surrounding words;
- keep translations and contexts as separate independent arrays;
- send vocabulary reminders through a Telegram bot;
- support manual commands for health checks, synchronization, exports, logs, and notification control;
- write logs to a file and send them to the user weekly.

`ntfy`, Anki, and paid flashcard services are intentionally not included at this stage.

---

## 2. High-level workflow

```text
PocketBook notes
   ↓
PocketBook sync adapter
   ↓
Vocabulary draft items
   ↓
Normalizer + merger
   ↓
MongoDB vocabulary collection
   ↑
Google Sheets sync adapter
   ↓
Review scheduler
   ↓
Telegram bot notifications + buttons
```

Synchronization order:

```text
1. Sync PocketBook notes.
2. Sync Google Sheet rows.
3. Merge all new data into the same MongoDB vocabulary entities.
4. Schedule or send due reminders.
```

PocketBook and Google Sheets must not create separate word models. They are only different input sources for the same vocabulary entity.

---

## 3. Main data model

Translations and contexts are not paired.

Reason: PocketBook shows all available translations for a word every time. When the same word is saved again in a different place, it is not possible to automatically know which specific translation belongs to which specific context.

Therefore the model is:

```ts
type VocabularyItem = {
  // Stable primary key chosen when the entity is created.
  // It is not necessarily the shortest form. Matching is done through lookupKeys.
  normalizedKey: string;

  // All keys by which this item can be found.
  // Includes the full normalized form and edge-stripped variants.
  lookupKeys: string[];

  // Preferred visible representation. This can be updated when a richer form appears,
  // for example "decelerate" → "to decelerate" or "apple" → "an apple".
  displayWord: string;

  // All observed visible forms, exactly as they appeared in sources.
  forms: VocabularyForm[];

  // Independent arrays. No strict relation between translations and contexts.
  translations: string[];
  contexts: string[];

  // Optional manual notes, mostly from Google Sheets.
  notes: string[];

  anchors: SourceAnchor[];

  review: ReviewState;

  createdAt: Date;
  updatedAt: Date;
};

type VocabularyForm = {
  value: string;
  normalizedValue: string;
  lookupKeys: string[];
  count: number;
  firstSeenAt: Date;
  lastSeenAt: Date;
};

type SourceAnchor = {
  source: 'pocketbook' | 'google-sheet';

  // PocketBook-specific anchor if available.
  externalId?: string;
  bookTitle?: string;

  // Google Sheet-specific anchor.
  rowNumber?: number;

  firstSeenAt: Date;
  lastSeenAt: Date;
};

type ReviewState = {
  enabled: boolean;
  lastPushedAt?: Date;
  nextPushAt?: Date;
  easyCount: number;
  hardCount: number;
};
```

Example:

```json
{
  "normalizedKey": "decelerate",
  "lookupKeys": ["decelerate", "todecelerate", "decelerateto"],
  "displayWord": "to decelerate",
  "forms": [
    {
      "value": "decelerate",
      "normalizedValue": "decelerate",
      "lookupKeys": ["decelerate"],
      "count": 1
    },
    {
      "value": "to decelerate",
      "normalizedValue": "to decelerate",
      "lookupKeys": ["todecelerate", "decelerate"],
      "count": 2
    }
  ],
  "translations": [
    "замедляться",
    "снижать скорость"
  ],
  "contexts": [
    "The car began to decelerate rapidly",
    "The train started to decelerate before the station"
  ],
  "anchors": [
    {
      "source": "pocketbook",
      "externalId": "note-123",
      "bookTitle": "Some Book"
    },
    {
      "source": "google-sheet",
      "rowNumber": 42
    }
  ]
}
```

---

## 4. Word normalization rules

### 4.1. Base text normalization

Before building keys:

```text
1. Convert to lowercase.
2. Normalize Unicode.
3. Trim spaces.
4. Collapse multiple spaces into one.
5. Normalize apostrophes and dashes.
6. Remove wrapping punctuation where safe.
7. Keep the original visible form separately in forms[].
```

Examples:

```text
"Ice cream"  → normalized text: "ice cream"
"ice-cream"  → normalized text: "ice-cream"
"ICE CREAM"  → normalized text: "ice cream"
```

### 4.2. Compact key

A compact key is built from normalized text by removing spaces and dashes.

Examples:

```text
"ice cream"  → "icecream"
"ice-cream"  → "icecream"
"to decelerate" → "todecelerate"
```

### 4.3. Edge-token matching without any article/preposition filter

There must be no configurable list of articles, prepositions, particles, or other markers.

No list like this should exist:

```ts
// Do not do this.
const EDGE_MARKERS = new Set(['to', 'a', 'an']);
```

The system must not decide whether a surrounding token is an article, preposition, particle, or anything else.

Instead, matching should be structural:

```text
- preserve the full visible form;
- build the exact compact key for the full form;
- also build lookup keys by removing edge tokens from the beginning and/or the end;
- do not filter these edge tokens by value.
```

This means the same logic works for `to`, `a`, `an`, `the`, `of`, `in`, `on`, `for`, `up`, or any other surrounding token without adding it to a list.

Examples:

```text
"decelerate"       → lookup keys: "decelerate"
"to decelerate"    → lookup keys: "todecelerate", "decelerate"
"decelerate to"    → lookup keys: "decelerateto", "decelerate"
"an apple"         → lookup keys: "anapple", "apple"
"apple an"         → lookup keys: "applean", "apple"
"the same thing"   → lookup keys: "thesamething", "samething", "thesame", "same"
```

Important:

```text
- Edge tokens are removed only from lookup keys.
- They are never removed from display data.
- Raw forms stay in forms[].
- displayWord may be updated to the richer visible form.
```

### 4.4. Suggested implementation

```ts
function normalizeText(value: string): string {
  return value
    .toLowerCase()
    .normalize('NFKC')
    .replace(/[’‘]/g, "'")
    .replace(/[‐‑‒–—]/g, '-')
    .replace(/^[\s"'“”‘’()\[\]{}]+|[\s"'“”‘’()\[\]{}]+$/g, '')
    .replace(/\s+/g, ' ')
    .trim();
}

function compactKey(value: string): string {
  return value
    .replace(/[\s-]+/g, '')
    .trim();
}

function tokenize(value: string): string[] {
  return normalizeText(value).split(' ').filter(Boolean);
}

function buildLookupKeys(value: string): string[] {
  const tokens = tokenize(value);
  const keys = new Set<string>();

  if (tokens.length === 0) {
    return [];
  }

  // Full form.
  keys.add(compactKey(tokens.join(' ')));

  if (tokens.length > 1) {
    // Remove one leading edge token.
    keys.add(compactKey(tokens.slice(1).join(' ')));

    // Remove one trailing edge token.
    keys.add(compactKey(tokens.slice(0, -1).join(' ')));
  }

  if (tokens.length > 2) {
    // Remove one leading and one trailing edge token.
    keys.add(compactKey(tokens.slice(1, -1).join(' ')));
  }

  return [...keys].filter(Boolean);
}
```

MVP note: the structural edge rule removes one token from the beginning and/or one token from the end. If later needed, this can be expanded to repeated edge trimming. It still must not use a configured list of allowed articles or prepositions.

---

## 5. Lookup and merge behavior

The merge should use `lookupKeys`, not only `normalizedKey`.

For every incoming word:

```text
1. Build lookup keys from the incoming visible form.
2. Find an existing VocabularyItem where any incoming lookup key is already present in item.lookupKeys.
3. If exactly one item is found, merge into it.
4. If no item is found, create a new item.
5. If multiple items are found, do not auto-merge. Log an ambiguity and report it in /health or /logs.
```

When creating a new item:

```text
- normalizedKey = the best compact key for the current form;
- lookupKeys = all lookup keys generated from the current form;
- displayWord = the current visible form;
- forms[] = the current visible form.
```

When merging into an existing item:

```text
- add the raw visible form to forms[];
- add all generated lookup keys to item.lookupKeys;
- add translations without duplicates;
- add contexts without duplicates;
- add notes without duplicates;
- add or update source anchors;
- update displayWord if the new visible form is richer.
```

This supports both directions:

```text
Existing item: "decelerate"
New source value: "to decelerate"
Result:
- same VocabularyItem
- "to decelerate" is added to forms[]
- "todecelerate" is added to lookupKeys[]
- displayWord may become "to decelerate"
```

And:

```text
Existing item: "to decelerate"
New source value: "decelerate"
Result:
- same VocabularyItem
- "decelerate" is added to forms[]
- displayWord should not be downgraded if the richer form is already known
```

---

## 6. Display word update rule

The system should preserve the user's real observed forms and avoid losing useful grammatical information.

Recommended rule:

```text
1. Never remove raw forms from forms[].
2. If displayWord is empty, set it from the first observed source value.
3. Prefer a richer form over a bare form when they resolve to the same item.
4. Do not downgrade a richer displayWord to a plain form.
5. If multiple rich forms exist, keep the latest or most frequent one as displayWord.
```

Examples:

```text
"apple" → later "an apple"
Result displayWord: "an apple"

"to decelerate" → later "decelerate"
Result displayWord: "to decelerate"
```

A simple richness heuristic:

```text
- more tokens usually means richer;
- if token count is equal, prefer the most frequent form;
- if still equal, keep the existing displayWord.
```

---

## 7. Translations and contexts

### 7.1. PocketBook

PocketBook dictionary notes can provide:

```text
word
translations
context
book title / note metadata, if available
```

When the same word appears multiple times:

```text
- merge by lookupKeys;
- add all new translations to translations[];
- add all new contexts to contexts[];
- do not try to pair a specific translation with a specific context.
```

### 7.2. Google Sheet

Google Sheet uses the same model as PocketBook.

Minimal columns:

```text
word | translations | contexts | note
```

`source` is not a sheet column. It is automatically assigned by the Google Sheets adapter.

Suggested parsing rules:

```text
translations column:
- comma separates different translations

contexts column:
- period separates different contexts
```

Example row:

```text
word: to decelerate
translations: замедляться, снижать скорость, замедлять движение
contexts: The car began to decelerate rapidly. The train started to decelerate before the station.
note: optional teacher note
```

Parsed result:

```json
{
  "word": "to decelerate",
  "translations": [
    "замедляться",
    "снижать скорость",
    "замедлять движение"
  ],
  "contexts": [
    "The car began to decelerate rapidly",
    "The train started to decelerate before the station"
  ],
  "notes": [
    "optional teacher note"
  ]
}
```

MVP limitation: one context should normally be one sentence. If multi-sentence contexts become necessary later, replace the period separator with a stronger separator such as `|||`.

---

## 8. Merge algorithm

For every draft item from PocketBook or Google Sheets:

```text
1. Read raw word.
2. Build incoming lookupKeys.
3. Find an existing VocabularyItem by lookupKeys.
4. If not found, create a new VocabularyItem.
5. If exactly one item is found, merge into that item.
6. If several items are found, log ambiguity and skip automatic merge for this item.
7. Add or update forms[].
8. Update displayWord according to display rules.
9. Add translations without duplicates.
10. Add contexts without duplicates.
11. Add notes without duplicates.
12. Add or update source anchors.
13. Update timestamps.
```

Duplicates should be checked after trimming and normalizing whitespace.

---

## 9. Telegram bot

The Telegram bot is the main user interface.

The bot should only accept updates from allowlisted chat IDs. Admin commands and review button presses require the configured admin Telegram user ID. `/save` is intentionally less restrictive: any user in an allowlisted chat may append vocabulary to Google Sheets.

Required secret/config value:

```text
TELEGRAM_ADMIN_ID
```

Messages can be sent either directly to the user or to a configured channel/chat.

Optional config value:

```text
TELEGRAM_TARGET_CHANNEL_ID
```

---

## 10. Telegram reminder message

A reminder message should show:

```text
word / known forms
translations
one or more contexts
source label at the bottom (PocketBook book title with author, or Document for PDFs)
```

Example:

```text
to decelerate

Forms:
- decelerate
- to decelerate

Translations:
- замедляться
- снижать скорость

Contexts:
- The car began to decelerate rapidly.
- The train started to decelerate before the station.

Necromancer — Fred Saberhagen
```

Suggested buttons:

```text
Easy | Hard
```

Button behavior:

```text
Easy:
- mark the word as easier;
- schedule it further in the future;
- edit the push card in place, remove buttons, and append the chosen result.

Hard:
- mark the word as harder;
- schedule it sooner;
- edit the push card in place, remove buttons, and append the chosen result.
```

Button presses should be accepted only from `TELEGRAM_ADMIN_ID`. Other users see a popup alert that they are not authorized to vote; the card is left unchanged.

---

## 11. Bot commands

### `/health`

Checks whether the bot and core dependencies are working.

Should return:

```text
status
MongoDB connection status
PocketBook sync enabled/disabled
Google Sheets sync enabled/disabled
notifications enabled/disabled
last sync time
last push time
current word count
ambiguous merge count, if any
```

---

### `/start`

Shows a welcome message and the full command reference.

Current response includes:

```text
bot title and short purpose
full command list with one-line descriptions
```

Does not run sync or change runtime settings.

---

### `/info`

Shows the current health snapshot, configured schedule, and the full command reference.

Current response includes:

```text
health status
stored word count
sync enabled/disabled
notifications enabled/disabled
schedule timezone and auto sync/push/logs cron
note that /save rows import on the next scheduled sync
full command list with one-line descriptions
```

Useful when you want status, schedule, and help in one message.

---

### `/list_words`

Exports all words from MongoDB as a JSON file and sends it to the user in Telegram.

The file should contain all vocabulary items, including:

```text
normalizedKey
lookupKeys
displayWord
forms
translations
contexts
notes
anchors
review state
createdAt / updatedAt
```

---

### `/turn_off`

Disables both:

```text
- automatic source synchronization;
- automatic word notifications.
```

Manual commands should still work unless explicitly blocked.

---

### `/turn_on`

Enables:

```text
- automatic source synchronization;
- automatic word notifications.
```

---

### `/sync`

Runs source synchronization manually without waiting for the scheduled timer.

Order:

```text
1. PocketBook sync.
2. Google Sheets sync.
3. Merge into MongoDB.
4. Report summary to Telegram.
```

Summary example:

```text
Sync completed.
PocketBook notes processed: 120
Google Sheet rows processed: 40
New words: 8
Updated words: 17
Skipped duplicates: 135
Ambiguous merges: 0
```

---

### `/push`

Sends one due word manually without waiting for the scheduled timer.

Behavior:

```text
1. Pick the next due word.
2. If no words are due, optionally pick the oldest enabled word.
3. Send a normal reminder message with buttons.
4. Update lastPushedAt.
```

---

### `/save`

Appends vocabulary from a Telegram message or reply to Google Sheets via OpenAI.

Behavior:

```text
1. Extract text from the command message or replied-to message.
2. Structure it into word, translation, and context fields.
3. Append one row to Google Sheets.
4. Do not merge into MongoDB immediately.
5. Import the new row into local storage during the normal /sync flow.
```

Accepted input forms:

```text
/save text to save
text to save /save
reply to a message with /save
```

If `/save` is sent without text and without a text reply, the bot asks the user to reply to the message that should be saved or write text before/after the command.

Access:

```text
allowed chat required
admin user not required
```

---

### `/logs`

Sends the current log file to the user in Telegram.

Should not clear the log file unless the command is explicitly extended later, for example `/logs clear`.

---

## 12. Logging

All logs should be written to a file.

Suggested path:

```text
$TMPDIR/vocabulary-bot/logs/vocabulary.log
```

Override with `LOG_PATH` when needed.

Logs should include:

```text
sync start / finish
source errors
parse errors
merge results
ambiguous merge candidates
Telegram send results
button press results
command execution results
scheduler decisions
unexpected errors
```

All log messages should be in English.

---

## 13. Weekly log delivery and cleanup

Once per week, the system should:

```text
1. Send the current log file to the configured Telegram user.
2. Clear or rotate the log file after successful delivery.
```

Recommended behavior:

```text
- If sending logs succeeds: truncate the active `LOG_PATH` file to free disk space.
- If sending logs fails: keep the file and retry on the next scheduled run.
- Do not keep a second local copy; Telegram is the off-site archive.
```

Weekly job:

```text
1. Send the active `LOG_PATH` file to TELEGRAM_ADMIN_ID.
2. If Telegram delivery succeeds, truncate `LOG_PATH`.
3. If Telegram delivery fails, keep the active log and retry on the next scheduled run.
```

Scheduler env:

```text
AUTO_LOGS_CRON='0 21 * * 5'
SCHEDULE_TIMEZONE=
```

Leave `AUTO_LOGS_CRON` empty to disable the job. Times use `SCHEDULE_TIMEZONE` or local time.
The job respects `/turn_off` and `/turn_on` through the shared `notificationsEnabled` flag, same as auto push.
Manual `/logs` still sends the current file on demand and does not clear it.

---

## 14. Application settings

Store global runtime settings in MongoDB or a config file.

Suggested model:

```ts
type AppSettings = {
  syncEnabled: boolean;
  notificationsEnabled: boolean;
  pocketBookSyncEnabled: boolean;
  googleSheetSyncEnabled: boolean;
  lastPocketBookSyncAt?: Date;
  lastGoogleSheetSyncAt?: Date;
  lastPushAt?: Date;
  lastWeeklyLogsSentAt?: Date;
};
```

`/turn_off` updates:

```json
{
  "syncEnabled": false,
  "notificationsEnabled": false
}
```

`/turn_on` updates:

```json
{
  "syncEnabled": true,
  "notificationsEnabled": true
}
```

---

## 15. Scheduling

Suggested scheduled jobs:

```text
Daily source sync:
- sync PocketBook;
- sync Google Sheets;
- merge into MongoDB.

Regular word push:
- send due words if notifications are enabled.

Telegram updates polling:
- process button presses and commands.

Weekly logs:
- send logs to Telegram;
- clear logs after successful delivery.
```

If using GitHub Actions instead of a permanent server:

```text
- Scheduled workflows can run sync and push jobs.
- Telegram commands and button presses can be processed through polling during scheduled workflow runs.
- Commands will not be real-time unless a permanent bot process is running somewhere.
```

If using a small always-on process later:

```text
- Telegram commands and buttons can be handled immediately.
- Scheduled jobs can run inside the application process.
```

---

## 16. Suggested project structure

```text
src/
  adapters/
    pocketbook.adapter.ts
    google-sheets.adapter.ts
  bot/
    telegram-bot.ts
    commands/
      health.command.ts
      list_words.command.ts
      turn_off.command.ts
      turn_on.command.ts
      sync.command.ts
      push.command.ts
      logs.command.ts
  vocabulary/
    vocabulary.model.ts
    vocabulary.service.ts
    normalize-word.ts
    merge-vocabulary.ts
  scheduler/
    sync.job.ts
    push.job.ts
    weekly-logs.job.ts
  logging/
    logger.ts
    log-delivery.service.ts
  config/
    env.ts
```

---

## 17. Environment variables

```text
MONGODB_URI=
MONGODB_DB_NAME=

TELEGRAM_BOT_TOKEN=
TELEGRAM_ADMIN_ID=
TELEGRAM_TARGET_CHANNEL_ID=
TELEGRAM_POLLING_ENABLED=true
TELEGRAM_API_BASE_URL=

POCKETBOOK_EMAIL=
POCKETBOOK_PASSWORD=      # bootstrap only
POCKETBOOK_REFRESH_TOKEN= # optional override; normal bootstrap persists token automatically
POCKETBOOK_TOKEN_PATH=    # fallback only when MongoDB is not configured
POCKETBOOK_API_BASE_URL=  # optional test/discovery override

GOOGLE_SHEET_ID=
GOOGLE_SERVICE_ACCOUNT_JSON=

SYNC_ENABLED=true
NOTIFICATIONS_ENABLED=true
```

---

## 18. MVP scope

MVP should include:

```text
1. MongoDB vocabulary model.
2. Word normalization with structural edge-token lookup keys.
3. No article/preposition/particle whitelist or configurable marker filter.
4. Preservation of all raw display forms.
5. PocketBook sync adapter.
6. Google Sheets sync adapter.
7. Merge pipeline.
8. Telegram reminders with Easy / Hard buttons that edit the card after selection.
9. Commands:
   - /start
   - /info
   - /health
   - /list_words
   - /turn_off
   - /turn_on
   - /sync
   - /push
   - /logs
10. File logging.
11. Weekly log delivery to Telegram with cleanup after successful delivery.
```

Not included in MVP:

```text
- ntfy notifications;
- Anki integration;
- paid flashcard services;
- configurable article/preposition filters;
- complex automatic translation-context matching;
- multi-sentence context parsing from Google Sheets using advanced separators.
```
