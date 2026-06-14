#!/usr/bin/env bash
set -euo pipefail

target="${1:-.env}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

dotenv_value() {
	local value="$1"
	local force_quote="$2"

	python3 - "$value" "$force_quote" "$script_dir" <<'PY'
import sys
from pathlib import Path

sys.path.insert(0, sys.argv[3])
from envfile import format_dotenv_value

print(format_dotenv_value(sys.argv[1], force_quote=sys.argv[2] == "true"), end="")
PY
}

write_kv() {
	local key="$1"
	local value="${2:-}"
	local force_quote="${3:-false}"

	if [ -z "$value" ]; then
		return 0
	fi

	printf '%s=%s\n' "$key" "$(dotenv_value "$value" "$force_quote")" >>"$target"
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
	write_kv "$key" "$value" "false"
done

for key in "${secret_keys[@]}"; do
	value="${!key:-}"
	write_kv "$key" "$value" "true"
done

chmod 600 "$target"
