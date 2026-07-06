#!/usr/bin/env bash
# Stage 10 CI gates (Linux/GitHub Actions). Windows: scripts/ci-gates-stage10.ps1
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

run_go() {
  echo "==> go test $*"
  go test "$@"
}

TESTS=(
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

for t in "${TESTS[@]}"; do
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

if [[ -f services/control-plane/internal/store/parity_test.go ]]; then
  echo "==> postgres parity (docker)"
  pg_name="era-ci-pg-$RANDOM"
  pg_port=$((55432 + RANDOM % 1000))
  pg_started=0
  cleanup_pg() {
    if [[ "$pg_started" -eq 1 ]]; then
      docker rm -f "$pg_name" >/dev/null 2>&1 || true
    fi
    unset ERA_STORE_PG_DSN || true
  }
  trap cleanup_pg EXIT
  if docker run -d --name "$pg_name" \
    -e POSTGRES_USER=era \
    -e POSTGRES_PASSWORD=era_ci_pw \
    -e POSTGRES_DB=era_cp \
    -p "${pg_port}:5432" \
    postgres:16-alpine >/dev/null 2>&1; then
    pg_started=1
    for _ in $(seq 1 30); do
      if docker exec "$pg_name" pg_isready -U era >/dev/null 2>&1; then
        break
      fi
      sleep 2
    done
    export ERA_STORE_PG_DSN="postgres://era:era_ci_pw@127.0.0.1:${pg_port}/era_cp?sslmode=disable"
    (cd services/control-plane && go test ./internal/store/... -run TestPostgresParity -count=1)
  else
    echo "docker unavailable - postgres parity skipped"
  fi
  trap - EXIT
  cleanup_pg
fi

if [[ -f services/event-writer/internal/timeline/testdata/timeline_merged.golden.json ]]; then
  echo "==> workbench timeline golden"
  go test ./services/event-writer/internal/timeline/...
fi

echo "==> PII golden (agent)"
( cd crates/era-agent && cargo test golden_pii )

echo "==> Agent budget + tamper prod guard"
( cd crates/era-agent-core && cargo test budget_guard:: && cargo test tamper:: )

echo "==> era-plugin-vuln (L-05)"
( cd crates/era-plugin-vuln && cargo test )

if command -v helm >/dev/null 2>&1 && command -v pwsh >/dev/null 2>&1; then
  pwsh -NoProfile -File scripts/helm-template-check.ps1
fi

echo "Stage 10 CI gates PASS"
