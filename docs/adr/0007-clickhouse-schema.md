# ADR-0007: Схема хранения ClickHouse

**Статус:** Accepted
**Дата:** 8 июня 2026 г.
**Артефакты:** [`deploy/clickhouse/`](../../deploy/clickhouse/) (`001_schema.sql`,
`002_storage_policy.xml`, `003_rollups.sql`)

---

## Контекст

ADR-0004 описал стратегию хранения (tiering, rollup), но без конкретных таблиц.
Здесь — реализуемая схема под `Envelope` (ADR-0001) и нагрузку ~300 ТБ/год.

## Решение

### Таблица `events` (сырьё, горячий слой)

- **Движок `ReplacingMergeTree(ingested_at)`** — идемпотентность при at-least-once
  доставке (дедуп по ключу сортировки, последняя версия по `ingested_at`).
- **`ORDER BY (tenant_id, node_id, observed_at, event_id)`** — совпадает с Kafka
  partition key → корректная сборка process-tree/storyline, быстрый поиск по хосту
  во времени. `event_id` в ключе обеспечивает дедуп.
- **`PARTITION BY toYYYYMMDD(observed_at)`** — посуточные партиции для эффективного
  TTL-перемещения и удаления.
- **TTL:** `+7 дней TO VOLUME 'cold'`, `+365 дней DELETE` — реализует tiering.
- **Skip-индексы** (`bloom_filter`) по `mitre_techniques` и `detection_rule_id` —
  быстрый threat hunting без полного скана.
- **`LowCardinality`** на повторяющихся строках (tenant, platform, severity) —
  кратная экономия места и ускорение.
- **`payload String`** — на этапе MVP тело события хранится сериализованным
  (JSON/CBOR); типизированную раскладку в колонки делает `detection-engine`
  (отдельные доменные таблицы — Фаза 2).

### Таблица `detections` (алерты, долгое хранение)

Малый объём (~0.1%), retention 3 года, отдельный жизненный цикл (`status` для
case management). Не подпадает под агрессивный TTL сырья.

### Rollup `events_hourly` (downsampling)

`SummingMergeTree` + Materialized View инкрементально агрегируют шумные события в
почасовые счётчики (~1% объёма). Сырьё шума удаляется по TTL, тренды и baseline
сохраняются.

### Storage policy `tiered`

`002_storage_policy.xml` определяет тома `hot` (NVMe) и `cold` (в проде — `s3`/MinIO
для air-gap). Монтируется в конфиг сервера ДО создания таблиц.

## Порядок применения

```
1. Положить 002_storage_policy.xml в /etc/clickhouse-server/config.d/, рестарт.
2. clickhouse-client --multiquery < 001_schema.sql
3. clickhouse-client --multiquery < 003_rollups.sql
```

## Последствия

**Плюсы:** идемпотентность; экономичный tiering; готовый threat-hunting индекс;
сохранение трендов при удалении шума.

**Минусы / обязательства:** в проде нужен cold-диск на MinIO/S3; типизированная
раскладка payload в доменные таблицы — отдельная задача Фазы 2; мониторинг
эффективности дедупа `ReplacingMergeTree`.

## Связано

- [`ADR-0001`](./0001-unified-event-envelope.md) — структура конверта
- [`ADR-0004`](./0004-storage-and-retention.md) — стратегия retention/tiering
- [`ADR-0008`](./0008-ingest-grpc-contract.md) — источник данных (ingest)
