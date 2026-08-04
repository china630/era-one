#!/usr/bin/env bash
# ERA Office — zero GPL/AGPL runtime SBOM gate (MVP: Cargo + Go module scan).
# LGPL / "Lesser" dual-license options are allowed (ADR-0026).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FAIL=0

DENY_FILE="$ROOT/deploy/sbom/office-deny-licenses.txt"
DENY_PATTERNS=()
if [[ -f "$DENY_FILE" ]]; then
  while IFS= read -r line || [[ -n "$line" ]]; do
    line="$(echo "$line" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
    [[ -z "$line" || "$line" == \#* ]] && continue
    DENY_PATTERNS+=("$line")
  done < "$DENY_FILE"
fi
if [[ ${#DENY_PATTERNS[@]} -eq 0 ]]; then
  DENY_PATTERNS=("AGPL" "GNU Affero" "GPL-2.0" "GPL-3.0" "GNU General Public License")
fi

denied_license() {
  local lic="$1"
  [[ -z "$lic" ]] && return 1
  echo "$lic" | grep -qiE 'LGPL|Lesser' && return 1
  echo "$lic" | grep -qiE 'AGPL|GNU Affero' && return 0
  # Avoid matching LGPL-2.1 as GPL-2
  echo "$lic" | grep -qiE '(^|[^A-Za-z])GPL-[23]' && return 0
  echo "$lic" | grep -qi 'GNU General Public License' && return 0
  return 1
}

echo "==> office SBOM gate (Cargo metadata licenses)"
if ! command -v jq >/dev/null 2>&1; then
  echo "FAIL: jq required for cargo license scan (AC-O5)"
  exit 1
fi
META="$(cargo metadata --format-version 1 --manifest-path "$ROOT/services/platform/docs-engine/Cargo.toml" 2>/dev/null || true)"
if [[ -z "$META" ]]; then
  echo "FAIL: cargo metadata unavailable (required for AC-O5)"
  exit 1
fi
while IFS= read -r lic; do
  if denied_license "$lic"; then
    echo "FAIL: denied runtime license: $lic"
    FAIL=1
  fi
done < <(echo "$META" | jq -r '.packages[].license // empty' 2>/dev/null | sort -u)

echo "==> office SBOM gate (Go modules - known GPL importers)"
GO_GPL_MARKERS=(
  "github.com/hashicorp/go-plugin"
)
for mod in "${GO_GPL_MARKERS[@]}"; do
  if rg -q "$mod" "$ROOT/services/platform/go.sum" 2>/dev/null; then
    echo "WARN: known module present (review): $mod"
  fi
done

if [[ $FAIL -ne 0 ]]; then
  echo "SBOM GATE FAIL"
  exit 1
fi
echo "SBOM GATE PASS (MVP scan)"
