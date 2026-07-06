#!/usr/bin/env bash
# Сборка артефакта публичного сайта ERA One (site/) для деплоя на отдельный хостинг.
# Использование:
#   ./scripts/build-site.sh              # → dist/site/
#   ./scripts/build-site.sh /var/www/era-one
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${1:-$ROOT/dist/site}"

echo "==> build portal pricing SSOT"
python3 "$ROOT/scripts/build_portal.py"

echo "==> calculator golden tests"
node "$ROOT/site/test/calculator.test.js"

echo "==> copy site/ → $OUT"
rm -rf "$OUT"
mkdir -p "$OUT"
rsync -a --delete \
  --exclude='test/' \
  "$ROOT/site/" "$OUT/"

echo "OK: site artifact at $OUT ($(find "$OUT" -type f | wc -l | tr -d ' ') files)"
