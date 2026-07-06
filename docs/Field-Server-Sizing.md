# ERA XDR — Field Server Sizing

Канонический документ для **field-прогона** (AC2 loadgen, pilot sign-off, soak).
Для быстрого lab-deploy см. также [`deploy/prod/README.md`](../deploy/prod/README.md).

## Назначение

| Контур | Зачем отдельный сервер |
|--------|------------------------|
| **AC2 loadgen** | Цель ≥10 000 ev/s × 5 мин × 3 прогона; dev-ноут с Docker Desktop даёт ~200–700 ev/s — недостаточно |
| **Pilot sign-off** | Полный prod compose (`scale` + `pg`), mTLS, Postgres CP, chaos smoke |
| **Soak 7×24** | Кластерный прогон — [gate: field], см. [`Hardening-Scale-Spec.md`](Hardening-Scale-Spec.md) |

## Профили железа

- **Минимум пилот (1 нода):** 8+ vCPU, 16+ GiB RAM, ≥600 GiB SSD (Kafka + ClickHouse + MinIO)
- **Sizing для loadgen AC2:** 16 vCPU, 32 GiB RAM, NVMe SSD; profile `scale` (2× event-writer, 2× detection-engine)
- **Средний контур (500–2000 узлов):** 16 vCPU, 32 GiB RAM, Kafka RF=3 — [`docker-compose.prod-ha.yml`](../deploy/docker-compose.prod-ha.yml)
- **HA опция:** 3× broker Kafka, 2× ingest за L4 LB

Компоненты (ориентир): Kafka 2 vCPU/4 GiB; ClickHouse 4 vCPU/8 GiB; Go services 2 vCPU/2 GiB.

## Откуда взять сервер

- **Lab VM** — Hyper-V, Proxmox, VMware на выделенном хосте
- **Выделенный хост в ЦОД** организации
- **Сервер заказчика** до пилота (типичный air-gap сценарий)

**Не использовать:** облачный SaaS/CDN как runtime продукта (инвариант air-gap, ADR).

## ОС хоста

- **Предпочтительно:** Linux x86_64 + Docker Engine 24+
- **Windows Server:** допустимо для lab; Docker Desktop/WSL2 — ниже throughput loadgen
- **macOS:** только dev-smoke, не для AC2

## Сеть и порты

| Порт | Сервис |
|------|--------|
| 50051 | ingest-gateway gRPC (mTLS) |
| 8082 | ingest HTTP (mTLS) |
| 8090 | control-plane (mTLS) |
| 8089 | event-writer |
| 8091 | ai-core |
| 8092 | soar |
| 9092 | Kafka (внутри контура) |

mTLS dev-CA: `scripts/gen-dev-tls.ps1` → `deploy/tls/`. Production — офлайн PKI заказчика.

## Команды прогона

```powershell
# из корня репо
./scripts/gen-dev-tls.ps1

$env:ERA_STORE_DRIVER = "postgres"
$env:ERA_STORE_DSN = "postgres://era:era_cp_pw@postgres:5432/era_cp?sslmode=disable"

docker compose -f deploy/docker-compose.prod.yml --profile scale --profile pg up -d --build

# AC2 (на sizing-сервере)
./scripts/run-loadgen-prod.ps1 -Rate 10000 -DurationSec 300 -Runs 3 -MinEvPerSec 10000

# Pilot + chaos
./scripts/run-pilot-local.ps1
./scripts/run-chaos-smoke.ps1
```

## Критерий PASS (AC2)

- `run-loadgen-prod.ps1`: ≥10 000 ev/s, duration 300s, 3 runs, **fail=0**
- Лог: `reports/loadgen-prod.log`
- Smoke на dev-машине (пониженный порог) — не закрывает AC2; см. `reports/prefield-proof-*.log`

## Шаблон фиксации факта

Скопируйте в `reports/field-server-YYYYMMDD.log`:

```
# Field server proof — YYYY-MM-DD

## Hardware
- Host: _______________
- CPU: __ vCPU (model: ________)
- RAM: __ GiB
- Disk: __ GiB (__ SSD/NVMe)
- OS: _______________

## Stack
docker compose -f deploy/docker-compose.prod.yml --profile scale --profile pg up -d --build
Result: PASS / FAIL

## Loadgen AC2
Command: run-loadgen-prod.ps1 -Rate 10000 -DurationSec 300 -Runs 3 -MinEvPerSec 10000
Run 1: ____ ev/s  Run 2: ____ ev/s  Run 3: ____ ev/s
fail=0: YES / NO

## Pilot / chaos
run-pilot-local.ps1: PASS / FAIL
run-chaos-smoke.ps1: PASS / FAIL

Signed: _______________
```

## Связанные документы

- [`Field-Server-Setup.md`](Field-Server-Setup.md) — пошаговая установка на field-хост
- [`Production-Readiness-Assessment.md`](Production-Readiness-Assessment.md) — матрица готовности
- [`Install-Guide-GA.md`](Install-Guide-GA.md)
- [`Pilot-Readiness-Checklist.md`](Pilot-Readiness-Checklist.md)
- [`Pre-Field-Code-Backlog.md`](Pre-Field-Code-Backlog.md)
- [`MVP-Sprint-1-Spec.md`](MVP-Sprint-1-Spec.md) — AC2
- [`Hardening-Scale-Spec.md`](Hardening-Scale-Spec.md)
