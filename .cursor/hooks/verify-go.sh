#!/usr/bin/env bash
# Cursor afterFileEdit hook: JSON on stdin; respond on stdout (schema varies by Cursor version).
set -euo pipefail
cat >/dev/null
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

for g in go /usr/local/go/bin/go; do
  if command -v "$g" >/dev/null 2>&1; then
    export PATH="$(dirname "$(command -v "$g")"):$PATH"
    break
  fi
done

if ! command -v go >/dev/null 2>&1; then
  printf '%s\n' '{}'
  exit 0
fi

go build -o /dev/null ./... 2>/dev/null || true
printf '%s\n' '{}'
exit 0
