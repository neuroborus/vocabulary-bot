"""Minimal .env parsing helpers shared by deploy scripts."""

from __future__ import annotations

import sys
from pathlib import Path


def parse_value(raw: str) -> str:
    value = raw
    if len(value) >= 2 and value[0] == value[-1] and value[0] in {'"', "'"}:
        quote = value[0]
        value = value[1:-1]
        if quote == '"':
            value = (
                value.replace("\\\\", "\\")
                .replace('\\"', '"')
                .replace("\\n", "\n")
            )
    return value


def parse_dotenv(path: Path) -> dict[str, str]:
    values: dict[str, str] = {}

    for line in path.read_text(encoding="utf-8").splitlines():
        stripped = line.strip()
        if not stripped or stripped.startswith("#"):
            continue
        if "=" not in stripped:
            continue

        key, raw_value = stripped.split("=", 1)
        key = key.strip()
        if not key:
            continue

        values[key] = parse_value(raw_value)

    return values


def strip_surrounding_quotes(value: str) -> str:
    return parse_value(value.strip())


def format_dotenv_value(value: str, *, force_quote: bool = False) -> str:
    if value == "":
        return ""

    if force_quote:
        value = strip_surrounding_quotes(value)
        return format_quoted_value(value)

    if all(ch.isalnum() or ch in "._/-:" for ch in value):
        return value

    return format_quoted_value(value)


def format_quoted_value(value: str) -> str:
    if "'" not in value:
        return f"'{value}'"

    escaped = (
        value.replace("\\", "\\\\")
        .replace('"', '\\"')
        .replace("\n", "\\n")
    )
    return f'"{escaped}"'


def validate_mongodb_uri(uri: str) -> str | None:
    if uri == "":
        return None

    uri = strip_surrounding_quotes(uri)

    if uri.startswith("MONGODB_URI="):
        return "MONGODB_URI secret must contain only the connection string, not KEY=value"

    if uri.startswith("mongodb://") or uri.startswith("mongodb+srv://"):
        return None

    return "MONGODB_URI must start with mongodb:// or mongodb+srv://"


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: validate-rendered-env <path>", file=sys.stderr)
        return 2

    path = Path(sys.argv[1])
    values = parse_dotenv(path)

    if error := validate_mongodb_uri(values.get("MONGODB_URI", "")):
        print(error, file=sys.stderr)
        return 1

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
