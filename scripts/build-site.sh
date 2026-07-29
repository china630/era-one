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

echo "==> RU datasheet UTF-8 gate"
python3 "$ROOT/site/test/check_datasheet_encoding.py"

echo "==> copy site/ → $OUT"
rm -rf "$OUT"
mkdir -p "$OUT"
if command -v rsync >/dev/null 2>&1; then
  rsync -a --delete \
    --exclude='test/' \
    "$ROOT/site/" "$OUT/"
else
  # GitHub runners / minimal images may lack rsync
  cp -a "$ROOT/site/." "$OUT/"
  rm -rf "$OUT/test"
fi

echo "==> SEO enrich (robots, sitemap, prerender, schema)"
python3 "$ROOT/scripts/site_seo_enrich.py" "$OUT"

echo "==> SEO artifacts gate"
python3 "$ROOT/site/test/check_seo_artifacts.py" "$OUT"

echo "OK: site artifact at $OUT ($(find "$OUT" -type f | wc -l | tr -d ' ') files)"
