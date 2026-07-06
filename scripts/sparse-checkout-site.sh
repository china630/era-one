#!/usr/bin/env bash
# Минимальный checkout монорепо — только файлы, нужные для сборки и деплоя site/.
# На хостинге (DigitalOcean droplet / CI deploy job):
#
#   git clone --filter=blob:none --sparse git@github.com:ORG/era-one.git era-one-site
#   cd era-one-site
#   git sparse-checkout init --cone
#   bash scripts/sparse-checkout-site.sh
#   git checkout dev   # или master для production
#   ./scripts/build-site.sh /var/www/era-one
#
set -euo pipefail

git sparse-checkout set \
  site \
  scripts/build_portal.py \
  scripts/build-site.sh \
  scripts/sparse-checkout-site.sh \
  docs/distributor/pricing-data.yaml \
  docs/distributor/pricing-comms-data.yaml \
  docs/distributor/pricing-office-data.yaml \
  docs/distributor/assets

echo "OK: sparse checkout configured for public site build"
