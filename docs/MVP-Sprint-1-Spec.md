# ERA XDR — MVP Sprint-1 Specification

**Версия:** 1.0
**Дата:** 8 июня 2026 г.
**Фаза:** 1 (MVP). Цель — доказать **сквозной конвейер телеметрии** на реальных
данных (не моках), end-to-end, на одной ноде.

---

## 1. Цель спринта (Definition of Done на уровне спринта)

> Событие, порождённое `era-agent`, проходит весь путь и видно в дашборде:
> **agent → ingest-gateway → Kafka → ClickHouse → query**.

Спринт закрыт, когда выполнен сквозной сценарий §4 и пройдены критерии приёмки §5.

## 2. Scope

### Входит
- `era-agent`: генерация `ProcessEvent` (сначала заглушка capture, затем ETW/Sysmon
  на Windows), OCSF-маппинг, PII-sanitize, disk-buffer, отправка батчей.
- `ingest-gateway`: приём (REST-фолбэк → затем gRPC `PushEvents`/`StreamEvents`),
  валидация `schema_version`, простановка `ingested_at`, публикация в Kafka.
- Kafka: топик `xdr.process` (topic-per-domain).
- ClickHouse: таблица `events` (DDL из `deploy/clickhouse/001_schema.sql`).
- Минимальный query: SQL-выборка последних событий по хосту (через `/play` или CLI).
- Dev-окружение: `deploy/docker-compose.dev.yml` поднимает фундамент.

### НЕ входит (следующие спринты)
- AI Core, корреляция, SOAR, коллекторы доменов (email/identity/cloud).
- gRPC mTLS/PKI хардненинг (в Sprint-1 — токен в metadata, TLS опционально).
- Типизированная раскладка payload в доменные таблицы (Фаза 2).
- UI-дашборд богатый (Sprint-1: достаточно SQL/`/play` + простая таблица).

## 3. Архитектура потока (Sprint-1)

```
era-agent (Rust)
  capture(stub) -> ocsf map -> sanitize(PII) -> ring buffer
      -> [REST /v1/ingest | gRPC PushEvents]
ingest-gateway (Go)
  validate schema_version -> set ingested_at -> route by tenant
      -> Kafka produce (topic xdr.process, key = tenant_id|node_id)
ClickHouse
  Kafka engine / consumer -> events (ReplacingMergeTree)
query
  SELECT ... FROM era_xdr.events WHERE node_id = ? ORDER BY observed_at DESC
```

## 4. Сквозной сценарий приёмки (E2E)

1. `docker compose -f deploy/docker-compose.dev.yml up -d` — фундамент поднят,
   топики `xdr.*` созданы, таблицы `era_xdr.events` существуют.
2. Запустить `ingest-gateway` (`go run ./cmd/ingest-gateway`).
3. Запустить `era-agent` (`cargo run -p era-agent`).
4. Агент шлёт ≥ 1 батч `ProcessEvent`; в логах gateway — `ACCEPTED`.
5. В ClickHouse: `SELECT count() FROM era_xdr.events` > 0; запись содержит
   `pii_sanitized = 1`, `ocsf_class_uid = 1007`, замаскированный `command_line`.
6. PII-проверка: исходный `user`/секрет в `command_line` **отсутствуют** в открытом
   виде в ClickHouse.

## 5. Критерии приёмки (измеримые)

| # | Критерий | Цель | Статус |
|---|---|---|---|
| AC1 | E2E-сценарий §4 проходит полностью | 100% шагов | **PASS** |
| AC2 | Пропускная способность (1 нода dev) | ≥ 10 000 ev/s | **smoke PASS** (~233 ev/s dev); **10k [gate: sizing-server]** — [`Field-Server-Sizing.md`](Field-Server-Sizing.md) |
| AC3 | Потеря событий при штатной работе | 0 | **PASS** |
| AC4 | Дедуп по `event_id` | 0 дублей | **PASS** — ReplacingMergeTree |
| AC5 | PII в ClickHouse | 0 утечек | **PASS** — golden-тест |
| AC6 | Бюджет агента (ADR-0009) | CPU < 2%, RAM < 150 МБ | **PASS** — `budget_guard::check_process_memory` в CI (`ci-gates-stage10.ps1`) |
| AC7 | Backpressure | буфер + retry | **PASS** |
| AC8 | Unit-тесты agent + gateway | pass | **PASS** |

## 6. Задачи (backlog спринта)

| ID | Задача | Модуль | Статус | Зависит от |
|---|---|---|---|---|
| S1-1 | Сгенерировать Go/Rust стабы из `proto/era/v1/*.proto` (protoc + Apicurio) | contracts | [x] `gen/go/`, `crates/era-proto`, `scripts/gen-proto.ps1`; go/cargo test PASS | — |
| S1-2 | Подключить prost/tonic в `era-agent` (заменить serde-скелет) | era-agent | [x] era-proto + gRPC sender | S1-1 |
| S1-3 | gRPC-сервер `IngestService` в gateway (PushEvents) | ingest-gateway | [x] :50051 PushEvents | S1-1 |
| S1-4 | Kafka-продюсер в gateway (zstd, key=tenant\|node) | ingest-gateway | [x] segmentio/kafka-go zstd | S1-3 |
| S1-5 | Consumer Kafka → ClickHouse → `events` | event-writer | [x] `services/event-writer` | S1-4 |
| S1-6 | ETW/Sysmon capture (Windows) | era-agent | [x] sysinfo process watcher + stub | S1-2 |
| S1-7 | Golden-тест PII-редакции (CI-gate) | era-agent | [x] `tests/golden_pii.rs` | S1-2 |
| S1-8 | Бенчмарк бюджета агента (CPU/RAM) | era-agent | [x] `benches/agent_budget.rs` | S1-2 |
| S1-9 | Нагрузочный тест 10k ev/s (AC2) | loadgen | [x] 7232 ev/s dev; `cmd/loadgen` | S1-5 |
| S1-10 | UI «последние события» | ui | [x] `ui/events/` + `:8089/ui/` | S1-5 |

## 7. Риски спринта

- **protoc/тулчейн в air-gap** → зафиксировать версии, положить в внутренний реестр.
- **ClickHouse Kafka-engine vs внешний consumer** → выбрать на S1-5 (рекоменд.:
  внешний consumer на Go для контроля дедупа и enrich).
- **ETW/Sysmon на Windows** → начать со stub-capture (уже в скелете), ETW параллельно.

## 8. Связано

- [`Development-Plan.md`](Development-Plan.md) — фазы и DoD
- [`ERA-XDR-Architecture-Blueprint.md`](../reports/ERA-XDR-Architecture-Blueprint.md)
- ADR: [0001](adr/0001-unified-event-envelope.md), [0004](adr/0004-storage-and-retention.md),
  [0007](adr/0007-clickhouse-schema.md), [0008](adr/0008-ingest-grpc-contract.md),
  [0009](adr/0009-pii-redaction-and-agent-budget.md)
