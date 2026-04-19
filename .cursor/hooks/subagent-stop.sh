#!/usr/bin/env bash
# subagentStop: keep instant — do not run a full module build here (use afterFileEdit for that).
set -euo pipefail
cat >/dev/null
printf '{"followup_message":""}\n'
