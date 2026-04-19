#!/usr/bin/env bash
# afterFileEdit: read hook JSON on stdin; prefer building only the edited Go package.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

HOOK_JSON=$(cat)

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

TIMEOUT=(timeout 90s)
if ! command -v timeout >/dev/null 2>&1; then
  TIMEOUT=()
fi

PKG=""
if command -v python3 >/dev/null 2>&1; then
  PKG="$(printf '%s' "$HOOK_JSON" | python3 "$ROOT/.cursor/hooks/verify_go_pkg.py" "$ROOT" || true)"
fi

if [[ -n "${PKG}" ]]; then
  "${TIMEOUT[@]}" go build -o /dev/null "${PKG}" 2>/dev/null || "${TIMEOUT[@]}" go build -o /dev/null ./... 2>/dev/null || true
else
  "${TIMEOUT[@]}" go build -o /dev/null ./... 2>/dev/null || true
fi

printf '%s\n' '{}'
exit 0
