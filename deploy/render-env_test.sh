#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
target="$(mktemp)"
cleanup() {
	rm -f "$target"
}
trap cleanup EXIT

want_uri='mongodb+srv://user:FAKE_PASSWORD_FOR_TEST_ONLY@cluster0.example.test/vocabulary_bot?retryWrites=true&w=majority'

assert_mongodb_uri() {
	python3 - <<'PY' "$target" "$root/deploy"
import sys
from pathlib import Path

sys.path.insert(0, sys.argv[2])
from envfile import parse_dotenv, validate_mongodb_uri

path = Path(sys.argv[1])
values = parse_dotenv(path)
uri = values.get("MONGODB_URI", "")
if error := validate_mongodb_uri(uri):
    raise SystemExit(error)
want = "mongodb+srv://user:FAKE_PASSWORD_FOR_TEST_ONLY@cluster0.example.test/vocabulary_bot?retryWrites=true&w=majority"
if uri != want:
    raise SystemExit(f"unexpected MONGODB_URI: {uri!r}")
if values.get("POCKETBOOK_PASSWORD") != "p@ss&word":
    raise SystemExit("POCKETBOOK_PASSWORD was not preserved")
if values.get("AUTO_SYNC_CRON") != "0 9 * * *":
    raise SystemExit(f"unexpected AUTO_SYNC_CRON: {values.get('AUTO_SYNC_CRON')!r}")
PY
}

assert_quoted_mongodb_uri() {
	local rendered="$1"
	case "$rendered" in
	"MONGODB_URI='"*"'") return 0 ;;
	"MONGODB_URI=\""*"\"") return 0 ;;
	*)
		echo "expected quoted MONGODB_URI, got: $rendered"
		return 1
		;;
	esac
}

export MONGODB_URI="$want_uri"
export TELEGRAM_BOT_TOKEN='1234567890:FAKE_TELEGRAM_BOT_TOKEN_FOR_TEST_ONLY'
export POCKETBOOK_PASSWORD='p@ss&word'
export GOOGLE_SERVICE_ACCOUNT_JSON='{"type":"service_account","project_id":"example"}'
export APP_ENV='beta'
export SYNC_ENABLED='true'
export AUTO_SYNC_CRON='0 9 * * *'
export AUTO_PUSH_CRON='0 12-21/2 * * *'

bash "$root/deploy/render-env.sh" "$target"
python3 "$root/deploy/envfile.py" "$target"
assert_quoted_mongodb_uri "$(grep '^MONGODB_URI=' "$target")"
assert_mongodb_uri

export MONGODB_URI="'$want_uri'"
bash "$root/deploy/render-env.sh" "$target"
python3 "$root/deploy/envfile.py" "$target"
assert_quoted_mongodb_uri "$(grep '^MONGODB_URI=' "$target")"
assert_mongodb_uri

export MONGODB_URI="\"$want_uri\""
bash "$root/deploy/render-env.sh" "$target"
python3 "$root/deploy/envfile.py" "$target"
assert_quoted_mongodb_uri "$(grep '^MONGODB_URI=' "$target")"
assert_mongodb_uri

export MONGODB_URI="$want_uri"
export POCKETBOOK_PASSWORD="pa'ss&word"
bash "$root/deploy/render-env.sh" "$target"
python3 "$root/deploy/envfile.py" "$target"
rendered_password="$(grep '^POCKETBOOK_PASSWORD=' "$target")"
python3 - <<'PY' "$rendered_password"
import sys

line = sys.argv[1]
if not (line.startswith('POCKETBOOK_PASSWORD="') and line.endswith('"')):
    raise SystemExit(f"expected double-quoted POCKETBOOK_PASSWORD, got: {line}")
PY

python3 - <<'PY' "$target" "$root/deploy"
import sys
from pathlib import Path

sys.path.insert(0, sys.argv[2])
from envfile import parse_dotenv

values = parse_dotenv(Path(sys.argv[1]))
if values.get("POCKETBOOK_PASSWORD") != "pa'ss&word":
    raise SystemExit("apostrophe password was not preserved")
PY

echo "render-env test passed"
