-- ============================================================================
-- ERA XDR — Rollup / downsampling (ADR-0004, раздел "Решение 3")
--
-- Шумные события (70% объёма) держим сырьём 7-30 дней, а агрегаты — долго.
-- Materialized View пишет почасовую агрегацию в компактную таблицу.
-- ============================================================================

-- Целевая таблица почасовых агрегатов (≈1% от объёма сырья).
CREATE TABLE IF NOT EXISTS era_xdr.events_hourly
(
    hour          DateTime('UTC'),
    tenant_id     LowCardinality(String),
    node_id       String,
    category      Enum8('unspecified'=0,'process'=1,'network'=2,'file'=3,'registry'=4,'auth'=5,'dns'=6,'module'=7),
    severity      Enum8('unspecified'=0,'info'=1,'low'=2,'medium'=3,'high'=4,'critical'=5),
    events_count  UInt64,
    unique_hosts  AggregateFunction(uniq, String)
)
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (tenant_id, node_id, category, severity, hour)
TTL hour + INTERVAL 2 YEAR DELETE
SETTINGS storage_policy = 'tiered';

-- MV: на каждую вставку в events инкрементально обновляет почасовой агрегат.
CREATE MATERIALIZED VIEW IF NOT EXISTS era_xdr.events_hourly_mv
TO era_xdr.events_hourly
AS
SELECT
    toStartOfHour(observed_at)        AS hour,
    tenant_id,
    node_id,
    category,
    severity,
    count()                           AS events_count,
    uniqState(node_id)                AS unique_hosts
FROM era_xdr.events
GROUP BY hour, tenant_id, node_id, category, severity;
