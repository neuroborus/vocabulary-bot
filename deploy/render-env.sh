#!/usr/bin/env bash
set -euo pipefail

target="${1:-.env}"

dotenv_value() {
	python3 - "$1" <<'PY'
import sys

value = sys.argv[1]
if value == "":
    sys.exit(0)

if all(ch.isalnum() or ch in "._/-:" for ch in value):
    sys.stdout.write(value)
else:
    escaped = (
        value.replace("\\", "\\\\")
        .replace('"', '\\"')
        .replace("\n", "\\n")
    )
    sys.stdout.write(f'"{escaped}"')
PY
}

write_kv() {
	local key="$1"
	local value="${2:-}"

	if [ -z "$value" ]; then
		return 0
	fi

	printf '%s=%s\n' "$key" "$(dotenv_value "$value")" >>"$target"
}

: >"$target"

keys=(
	APP_ENV
	SYNC_ENABLED
	NOTIFICATIONS_ENABLED
	MONGODB_DB_NAME
	TELEGRAM_ALLOWED_USER_ID
	TELEGRAM_TARGET_CHAT_ID
	TELEGRAM_POLLING_ENABLED
	TELEGRAM_API_BASE_URL
	TELEGRAM_REVIEW_SPOILER_TRANSLATIONS
	REVIEW_DOCUMENT_PUSH_FACTOR
	REVIEW_BOOK_PUSH_FACTOR
	SCHEDULE_TIMEZONE
	AUTO_SYNC_CRON
	AUTO_PUSH_CRON
	AUTO_LOGS_CRON
	POCKETBOOK_SYNC_ENABLED
	POCKETBOOK_EMAIL
	POCKETBOOK_REFRESH_TOKEN
	POCKETBOOK_SHOP_NAME
	POCKETBOOK_API_BASE_URL
	POCKETBOOK_TOKEN_PATH
	POCKETBOOK_BOOK_CONTEXT_ENABLED
	POCKETBOOK_BOOK_CACHE_DIR
	POCKETBOOK_BOOK_CACHE_MAX
	GOOGLE_SHEET_SYNC_ENABLED
	GOOGLE_SPREADSHEET_ID
	GOOGLE_SHEET_NAME
	GOOGLE_SHEET_RANGE
	LOG_PATH
)

secret_keys=(
	MONGODB_URI
	TELEGRAM_BOT_TOKEN
	POCKETBOOK_PASSWORD
	GOOGLE_SERVICE_ACCOUNT_JSON
)

for key in "${keys[@]}"; do
	value="${!key:-}"
	write_kv "$key" "$value"
done

for key in "${secret_keys[@]}"; do
	value="${!key:-}"
	write_kv "$key" "$value"
done

chmod 600 "$target"
