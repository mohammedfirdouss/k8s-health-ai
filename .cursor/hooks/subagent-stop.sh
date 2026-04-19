#!/usr/bin/env bash
# subagentStop: optional follow-up; stdin is hook JSON (see Cursor hooks docs).
set -euo pipefail
cat >/dev/null
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
for g in go /usr/local/go/bin/go; do
  command -v "$g" >/dev/null 2>&1 && export PATH="$(dirname "$(command -v "$g")"):$PATH" && break
done
if command -v go >/dev/null 2>&1; then
  go build -o /dev/null ./... 2>/dev/null || true
fi
printf '{"followup_message":""}\n'
