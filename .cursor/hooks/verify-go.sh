#!/usr/bin/env bash
# Cursor command hook: stdin is JSON; stdout is JSON response.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

# Prefer devcontainer/PATH Go; fall back to common install locations.
for g in go /usr/local/go/bin/go; do
  if command -v "$g" >/dev/null 2>&1; then
    export PATH="$(dirname "$(command -v "$g")"):$PATH"
    break
  fi
done

if ! command -v go >/dev/null 2>&1; then
  printf '{"continue":true,"userMessage":"(go not in PATH; skipped verify-go hook)"}\n'
  exit 0
fi

# Fast feedback: compile all packages (no full test run here — keep hook snappy).
if ! go build -o /dev/null ./... 2>&1; then
  printf '{"continue":true,"userMessage":"go build ./... failed after edit; run make test"}\n'
  exit 0
fi

printf '{"continue":true}\n'
exit 0
