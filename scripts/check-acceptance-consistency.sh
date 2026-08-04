#!/usr/bin/env bash
# CI wrapper for Acceptance Standard v1.2 consistency gate.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
if command -v pwsh >/dev/null 2>&1; then
  exec pwsh -NoProfile -File "$ROOT/scripts/check-acceptance-consistency.ps1"
fi
if command -v powershell >/dev/null 2>&1; then
  exec powershell -NoProfile -File "$ROOT/scripts/check-acceptance-consistency.ps1"
fi
echo "ERROR: pwsh/powershell required for check-acceptance-consistency.ps1" >&2
exit 1
