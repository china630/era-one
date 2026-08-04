#!/usr/bin/env bash
# Fuzz smoke when cargo-fuzz nightly unavailable — runs import on random bytes.
set -euo pipefail
cargo test -p era-docs-engine fuzz_docx_smoke --quiet
echo "office fuzz smoke PASS"
