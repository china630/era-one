# ERA XDR — MVP Sprint-3 Specification (Фаза 3)

**Версия:** 1.0  
**Дата:** 9 июня 2026 г.  
**Статус:** Закрыт  
**Цель:** perimeter-модули (WAF, NGFW, DLP/UAM), NDR, Deception, CTEM/BAS, federated opt-in.

---

## Backlog (S3-1 … S3-10)

| ID | Задача | Модуль | DoD | Статус |
|---|---|---|---|---|
| S3-1 | WAF OWASP Top-10 block | services/waf | F3-1 | [x] |
| S3-2 | NGFW policies + Kafka telemetry | services/ngfw | F3-2 | [x] |
| S3-3 | DLP/UAM privileged session audit | services/dlp | F3-3 | [x] |
| S3-4 | Federated hub DP aggregate | services/federated | F3-4 | [x] |
| S3-5 | NDR T1021 lateral movement | detection-engine/ndr | F3-5 | [x] |
| S3-6 | Federated license gate default off | platform/licensegate | F3-6 | [x] |
| S3-7 | Deception honeypot | services/deception | ADR-0006 P1 | [x] |
| S3-8 | CTEM/BAS lateral sim | services/ctem | ADR-0006 P1 | [x] |
| S3-9 | Shared envelope publisher | platform/envelope | F3-2 | [x] |
| S3-10 | E2E Phase-3 smoke | scripts/run-phase3-e2e.ps1 | F3-* | [x] |

---

## Запуск Phase-3 dev-стека

```powershell
docker compose -f deploy/docker-compose.dev.yml up -d
cd services/ingest-gateway;  go run ./cmd/ingest-gateway
cd services/event-writer;    go run ./cmd/event-writer
cd services/detection-engine; go run ./cmd/detection-engine

cd services/waf;             go run ./cmd/waf              # :8093
cd services/ngfw;            go run ./cmd/ngfw             # :8094
$env:ERA_KAFKA_BROKERS="localhost:9092"; cd services/ngfw; go run ./cmd/ngfw
cd services/dlp;             go run ./cmd/dlp              # :8095
cd services/federated;       $env:ERA_FEDERATED_DEV="1"; go run ./cmd/federated-hub  # :8096
cd services/deception;       go run ./cmd/deception        # :8097
cd services/ctem;            go run ./cmd/ctem             # :8098
```

Smoke: `.\scripts\run-phase3-e2e.ps1`

Federated включается только при `ERA_LICENSE_MODULES=federated` или `ERA_FEDERATED_DEV=1`.
