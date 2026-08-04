#!/usr/bin/env bash
# Apply ERA Office Postgres migrations (platform + optional comms).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DSN="${ERA_OFFICE_DATABASE_URL:-${1:-}}"
MIGRATIONS_ROOT="${MIGRATIONS_ROOT:-$ROOT/deploy/postgres/migrations}"
APPLY_COMMS="${APPLY_COMMS:-1}"

if [[ -z "$DSN" ]]; then
  echo "ERA_OFFICE_DATABASE_URL or DSN argument required" >&2
  exit 1
fi

apply_dir() {
  local dir="$1"; shift
  for name in "$@"; do
    path="$dir/$name"
    if [[ ! -f "$path" ]]; then
      echo "migration not found: $path" >&2
      exit 1
    fi
    echo "==> applying $name"
    psql "$DSN" -v ON_ERROR_STOP=1 -f "$path"
  done
}

apply_dir "$MIGRATIONS_ROOT/platform" \
  001_drive.sql \
  002_docs_sessions.sql \
  005_projects.sql \
  006_projects_collab.sql \
  007_projects_w2.sql \
  008_drive_lock.sql

if [[ "$APPLY_COMMS" == "1" ]] && [[ -d "$MIGRATIONS_ROOT/comms" ]]; then
  while IFS= read -r -d '' f; do
    echo "==> applying $(basename "$f")"
    psql "$DSN" -v ON_ERROR_STOP=1 -f "$f"
  done < <(find "$MIGRATIONS_ROOT/comms" -maxdepth 1 -name '*.sql' -print0 | sort -z)
fi

echo "==> migrations applied"
