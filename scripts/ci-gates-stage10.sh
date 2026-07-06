#!/usr/bin/env bash
# Stage 10 CI gates (Linux/GitHub Actions). Windows: scripts/ci-gates-stage10.ps1
# Usage: bash scripts/ci-gates-stage10.sh [go|rust|all]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

MODE="${1:-all}"

run_go() {
  echo "==> go test $*"
  go test "$@"
}

run_go_gates() {
  local tests=(
    "./services/platform/licensegate/..."
    "./services/platform/httpserver/..."
    "./services/observe/..."
    "./services/control-plane/internal/networkreconcile/..."
    "./services/update-service/internal/bundle/..."
    "./services/update-service/internal/api/..."
    "./services/pam/..."
    "./services/license/internal/license/..."
    "./services/ingest-gateway/internal/server/..."
    "./services/ingest-gateway/internal/pipeline/..."
    "./services/event-writer/internal/consumer/..."
    "./services/federated/..."
    "./services/national-hub/internal/taxii/..."
  )
  local t
  for t in "${tests[@]}"; do
    run_go "$t"
  done

  run_go ./services/detection-engine/internal/exposure/...

  echo "==> ADR-0006 golden (deception/ctem/compliance/risk/ndr)"
  go test \
    ./services/detection-engine/internal/deception/... \
    ./services/detection-engine/internal/ctem/... \
    ./services/detection-engine/internal/compliance/... \
    ./services/detection-engine/internal/risk/... \
    ./services/detection-engine/internal/ndr/... \
    -run Golden

  echo "==> hybrid relay + TI outbound"
  go test ./services/control-plane/internal/hybrid/... -count=1

  if [[ -f services/control-plane/internal/store/parity_test.go && -n "${ERA_STORE_PG_DSN:-}" ]]; then
    echo "==> postgres parity (service container / ERA_STORE_PG_DSN)"
    (cd services/control-plane && go test ./internal/store/... -run TestPostgresParity -count=1)
  elif [[ -f services/control-plane/internal/store/parity_test.go ]]; then
    echo "==> postgres parity skipped (ERA_STORE_PG_DSN not set)"
  fi

  if [[ -f services/event-writer/internal/timeline/testdata/timeline_merged.golden.json ]]; then
    echo "==> workbench timeline golden"
    go test ./services/event-writer/internal/timeline/...
  fi
}

run_rust_gates() {
  echo "==> PII golden (agent)"
  (cd crates/era-agent && cargo test golden_pii)

  echo "==> Agent budget + tamper prod guard"
  (cd crates/era-agent-core && cargo test budget_guard:: && cargo test tamper::)

  echo "==> era-plugin-vuln (L-05)"
  (cd crates/era-plugin-vuln && cargo test )

  if command -v helm >/dev/null 2>&1 && command -v pwsh >/dev/null 2>&1; then
    pwsh -NoProfile -File scripts/helm-template-check.ps1
  fi
}

case "$MODE" in
  go) run_go_gates ;;
  rust) run_rust_gates ;;
  all) run_go_gates; run_rust_gates ;;
  *) echo "usage: $0 [go|rust|all]" >&2; exit 2 ;;
esac

echo "Stage 10 CI gates PASS ($MODE)"
