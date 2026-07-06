# ERA XDR — Production deploy (Wave GA-1)

Профиль **Core + AI + Response** для on-prem / air-gap пилота.  
Win+Linux агенты → ingest → Kafka → ClickHouse; control-plane с persistent SQLite.

## Быстрый старт

```powershell
# из корня репозитория
# Dev mTLS (P0-6): один раз сгенерировать CA/сертификаты
./scripts/gen-dev-tls.ps1

docker compose -f deploy/docker-compose.prod.yml up -d --build
```

Postgres-профиль control-plane:

```powershell
$env:ERA_STORE_DRIVER = "postgres"
$env:ERA_STORE_DSN = "postgres://era:era_cp_pw@postgres:5432/era_cp?sslmode=disable"
docker compose -f deploy/docker-compose.prod.yml --profile pg up -d postgres control-plane
```

### mTLS (internal services)

Сертификаты dev-CA: `scripts/gen-dev-tls.ps1` → `deploy/tls/`.  
Compose монтирует `./tls` и задаёт `ERA_TLS_CERT`, `ERA_TLS_KEY`, `ERA_TLS_CA` для ingest/control-plane; detection использует client cert (`agent.crt`).

```powershell
./scripts/gen-dev-tls.ps1
# убедитесь, что deploy/tls/server.crt существует перед up
```

Горизонтальное масштабирование writer/detection (Stage 10):

```powershell
docker compose -f deploy/docker-compose.prod.yml --profile scale up -d event-writer-b2 detection-engine-b2
```

Проверка health:

```powershell
curl http://localhost:8090/api/v1/assets
curl http://localhost:8089/healthz
curl http://localhost:8091/healthz
curl http://localhost:8092/healthz
```

## Sizing (минимальный пилот, 1 нода)

**Канон field-прогона (AC2 10k, шаблон лога):** [`docs/Field-Server-Sizing.md`](../docs/Field-Server-Sizing.md).

| Компонент | vCPU | RAM | Disk |
|---|---:|---:|---:|
| Kafka | 2 | 4 GiB | 100 GiB |
| ClickHouse | 4 | 8 GiB | 500 GiB SSD |
| MinIO (cold) | 1 | 2 GiB | 1 TiB (опц.) |
| Go services (core 6) | 2 | 2 GiB | — |
| **Итого сервер (Core)** | **8+** | **16+ GiB** | **≥600 GiB** |

### Полный набор изданий (stages 7–9, профили compose)

| Профиль | Доп. сервисы | +vCPU | +RAM |
|---|---|---:|---:|
| `itops` | service-desk, provision | +1 | +1 GiB |
| `pam` | pam (+ dlp) | +1 | +2 GiB |
| `observe` | observe | +1 | +512 MiB |

**Средний контур (500–2000 узлов):** 16 vCPU, 32 GiB RAM, Kafka RF=3 (`docker-compose.prod-ha.yml`), 2× ingest-gateway + 2× detection-engine (consumer groups).

### Горизонтальность (scale-proof)

| Сервис | Масштабирование | Kafka topic |
|---|---|---|
| ingest-gateway | N инстансов за L4 LB | producer shared |
| event-writer | consumer group `era-writer` | все `xdr.*` |
| detection-engine | consumer group `era-detect` | `xdr.process`, `xdr.network`, … |

Экстраполяция ev/s: `scripts/run-loadgen-prod.ps1` + `services/ingest-gateway/cmd/loadgen`. Целевой AC2 — 10k ev/s на ноде (dev ~7k; кластерный прогон → [gate: field]).

Партиции: `kafka-init` — 6 партиций на hot topics; при >3 writer-инстансов увеличить до N.

Агенты — на каждом хосте: CPU < 2%, RAM < 150 МБ (ADR-0009).

## Переменные окружения

| Переменная | Назначение | Default prod compose |
|---|---|---|
| `ERA_CH_PASSWORD` | пароль ClickHouse | `era_prod_pw` |
| `ERA_MINIO_USER` / `ERA_MINIO_PASSWORD` | MinIO | `minioadmin` |
| `ERA_STORE_PATH` | SQLite control-plane | `/data/control-plane.db` |
| `ERA_STORE_DRIVER` | `postgres` для profile `pg` | пусто (SQLite) |
| `ERA_STORE_DSN` | DSN Postgres CP | — |
| `ERA_TLS_*` | mTLS dev/prod | см. `scripts/gen-dev-tls.ps1` |

**Смените пароли** перед пилотом заказчика.

## Агент (production capture)

### Linux (auditd)

```bash
export ERA_PRODUCTION=1
export ERA_AUDIT_LOG=/var/log/audit/audit.log
export ERA_GATEWAY_ADDR=http://ingest-host:50051
export ERA_CONTROL_PLANE_URL=http://control-host:8090
./era-agent
```

Требуется правило auditd на `execve` (см. Install-Guide-GA).

### Windows (Sysmon)

```powershell
$env:ERA_PRODUCTION="1"
# или ERA_SYSMON_JSONL для sidecar-export
$env:ERA_GATEWAY_ADDR="http://ingest-host:50051"
cargo run -p era-agent
```

Sysmon Operational channel — через `wevtutil` (встроено) или `ERA_SYSMON_JSONL`.

### macOS (unified log export)

```bash
export ERA_PRODUCTION=1
export ERA_MACOS_UNIFIED_JSONL=/var/log/era/unified.ndjson
export ERA_GATEWAY_ADDR=http://ingest-host:50051
export ERA_CONTROL_PLANE_URL=http://control-host:8090
./era-agent
```

## Control-plane: SQLite vs Postgres

| Контур | Хранилище | Когда |
|---|---|---|
| **Пилот / lab / одна нода** | SQLite (`ERA_STORE_PATH=/data/control-plane.db`, volume) | до ~500 активов, 1–3 аналитика, backup через `backup-prod.ps1` |
| **Production банк** | Postgres (`ERA_STORE_DRIVER=postgres`, `ERA_STORE_DSN=...`) | HA control-plane, >3 одновременных writers, audit/compliance требует RDBMS, multi-tenant SOC |

**Переход на Postgres:** после успешного пилота, до production rollout — не блокер для первого deploy.

```yaml
# docker-compose override example
environment:
  ERA_STORE_DRIVER: postgres
  ERA_STORE_DSN: postgres://era:secret@postgres:5432/era_cp?sslmode=disable
```

## SOAR pilot connectors

Compose включает `deploy/soar/connectors/isolate-host.sh` и `ticket-stub` (замените `ERA_SOAR_TICKET_WEBHOOK` на ITSM заказчика).

## SSO (lab)

```powershell
docker compose -f deploy/docker-compose.prod.yml --profile sso up -d
# Portal: http://localhost:8443/ui/portal/
```

См. [`SSO-Setup-GA.md`](../docs/SSO-Setup-GA.md).

## Что не входит в prod compose

Perimeter (WAF/NGFW/DLP), Federated hub, National hub — Wave GA-3, включаются отдельным compose/Helm.

## Обновление

```powershell
docker compose -f deploy/docker-compose.prod.yml pull
docker compose -f deploy/docker-compose.prod.yml up -d --build
```

Volumes (`control-plane-data`, `clickhouse-prod-data`, …) сохраняют состояние между перезапусками.

## Связанные документы

- [`Production-GA-Spec.md`](../docs/Production-GA-Spec.md)
- [`Install-Guide-GA.md`](../docs/Install-Guide-GA.md)
- [`Pilot-Readiness-Checklist.md`](../docs/Pilot-Readiness-Checklist.md)
