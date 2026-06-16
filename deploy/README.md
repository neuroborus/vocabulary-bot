# Deploy to Ubuntu

Lightweight deployment: a static Go binary in a small Alpine image (~43 MB), `192m` memory limit in Compose.

Runtime config is rendered from GitHub **secrets** and **variables** on each deploy and written to the deploy directory on the server (default: `~/vocabulary-bot`). You do not need to maintain `.env` on the host manually after the first deploy.

## One-time server setup

```bash
sudo apt-get update
sudo apt-get install -y docker.io docker-compose-v2
sudo usermod -aG docker "$USER"
```

Log out and back in so the `docker` group applies. Deploy falls back to `sudo docker` when the SSH user is not in the `docker` group but passwordless sudo is available.

Create the app directory when using `/opt`:

```bash
sudo mkdir -p /opt/vocabulary-bot
sudo chown "$USER:$USER" /opt/vocabulary-bot
```

If `DEPLOY_PATH` is unset, deploy uses `~/vocabulary-bot` and creates it over SSH without sudo.

Add the deploy public key to `~/.ssh/authorized_keys` for the deploy user.

## GitHub configuration

### Repository secrets

| Secret | Purpose |
|--------|---------|
| `DEPLOY_SSH` | Private SSH key for deploy |
| `MONGODB_URI` | MongoDB connection string only (`mongodb://...` or `mongodb+srv://...`). Deploy writes secrets quoted in `.env` (single quotes by default, double when needed); the GitHub secret may be raw or wrapped in single/double quotes. |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token |
| `POCKETBOOK_PASSWORD` | PocketBook account password |
| `GOOGLE_SERVICE_ACCOUNT_JSON` | Google service account JSON (raw or base64) |

### Repository variables

| Variable | Example | Purpose |
|----------|---------|---------|
| `DEPLOY_HOST` | `ubuntu@150.230.145.55` | SSH target user and host |
| `DEPLOY_PATH` | empty | Remote install directory; default is `~/vocabulary-bot` |
| `APP_ENV` | `beta` | Runtime environment label |
| `LOG_PATH` | empty | Override log file path (optional) |
| `SYNC_ENABLED` | `true` | Master sync switch |
| `NOTIFICATIONS_ENABLED` | `true` | Telegram notifications |
| `MONGODB_DB_NAME` | `vocabulary_bot` | MongoDB database name |
| `TELEGRAM_ADMIN_ID` | `490734700` | Admin user allowed to run commands |
| `TELEGRAM_ALLOWED_CHAT_IDS` | empty | Optional extra allowlisted chats |
| `TELEGRAM_TARGET_CHANNEL_ID` | `-1004299028040` | Review push target channel |
| `TELEGRAM_POLLING_ENABLED` | `true` | Start Telegram long polling |
| `TELEGRAM_API_BASE_URL` | empty | Optional Bot API override |
| `TELEGRAM_REVIEW_SPOILER_TRANSLATIONS` | `true` | Hide translations behind spoiler |
| `SCHEDULE_TIMEZONE` | `Europe/Berlin` | Cron timezone (optional) |
| `AUTO_SYNC_CRON` | `0 9 * * *` | Daily sync schedule |
| `AUTO_PUSH_CRON` | `0 12-21/2 * * *` | Review push schedule |
| `AUTO_LOGS_CRON` | `0 21 * * 5` | Weekly log delivery |
| `POCKETBOOK_SYNC_ENABLED` | `true` | PocketBook source switch |
| `POCKETBOOK_EMAIL` | `reader@example.test` | PocketBook login |
| `POCKETBOOK_REFRESH_TOKEN` | empty | Optional refresh-token override |
| `POCKETBOOK_SHOP_NAME` | empty | Optional shop name |
| `POCKETBOOK_API_BASE_URL` | empty | Optional API override for tests |
| `POCKETBOOK_TOKEN_PATH` | empty | File session store path when MongoDB is off |
| `POCKETBOOK_BOOK_CONTEXT_ENABLED` | `true` | Download books for sentence context |
| `POCKETBOOK_BOOK_CACHE_DIR` | empty | Book cache directory override |
| `POCKETBOOK_BOOK_CACHE_MAX` | `2` | Cached books kept by LRU |
| `GOOGLE_SHEET_SYNC_ENABLED` | `true` | Google Sheets source switch |
| `GOOGLE_SPREADSHEET_ID` | spreadsheet id from URL | Sheet document id |
| `GOOGLE_SHEET_NAME` | `Vocabulary` | Sheet tab name |
| `GOOGLE_SHEET_RANGE` | `Vocabulary!A:F` | Range to read (`word`…`enabled`) |
| `REVIEW_DOCUMENT_PUSH_FACTOR` | `0.8` | Spreadsheet/PDF push weight |
| `REVIEW_BOOK_PUSH_FACTOR` | `1` | Book-anchor push weight |

Cron variables should contain the expression only, without shell quotes, for example `0 21 * * 5` rather than `'0 21 * * 5'`.

Recommended: create a GitHub `production` environment for the deploy job so deploy-only secrets can be scoped separately later.

## CI/CD

- `CI` — runs on pull requests into `main`: `gofmt` check, `go test`, `go vet`, build.
- `Deploy` — runs after tests on push to `main` and on manual **Run workflow**:
  1. build Docker image in GitHub Actions;
  2. render `.env` from GitHub secrets/variables;
  3. validate rendered secrets such as `MONGODB_URI` before upload;
  4. copy `docker-compose.yml`, `.env`, and image archive over SSH;
  5. `docker load` + `docker compose up -d` on the server.

## Manual operations on the server

```bash
cd ~/vocabulary-bot   # or /opt/vocabulary-bot when DEPLOY_PATH is set
docker compose logs -f
docker compose restart
docker compose down
```

Logs default to `$TMPDIR/vocabulary-bot/logs/vocabulary.log` inside the container unless `LOG_PATH` is set in the rendered `.env`.
