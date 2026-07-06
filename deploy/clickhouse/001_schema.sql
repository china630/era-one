-- ============================================================================
-- ERA XDR — ClickHouse schema (Phase 1 / MVP)
-- Реализует ADR-0001 (Unified Event Envelope) и ADR-0004 (retention/tiering).
--
-- Применение:
--   clickhouse-client --multiquery < deploy/clickhouse/001_schema.sql
--
-- Storage policy 'tiered' (hot/cold) задаётся в 002_storage_policy.xml и должна
-- быть смонтирована в конфиг сервера ДО создания таблиц с TTL TO VOLUME.
-- ============================================================================

CREATE DATABASE IF NOT EXISTS era_xdr;

-- ── Сырые события (горячий слой, дедуп по event_id) ───────────────────────────
-- ReplacingMergeTree обеспечивает идемпотентность (at-least-once доставка).
CREATE TABLE IF NOT EXISTS era_xdr.events
(
    -- идентификация / время
    event_id        String,                         -- ULID (26 симв. Crockford base32)
    correlation_id  String DEFAULT '',
    schema_version  LowCardinality(String),
    observed_at     DateTime64(9, 'UTC'),
    ingested_at     DateTime64(9, 'UTC') DEFAULT now64(9),

    -- источник (иерархия tenant -> ... -> agent)
    tenant_id       LowCardinality(String),
    environment     LowCardinality(String) DEFAULT '',
    cluster_id      LowCardinality(String) DEFAULT '',
    node_id         String,
    hostname        String DEFAULT '',
    agent_id        String DEFAULT '',
    agent_version   LowCardinality(String) DEFAULT '',
    platform        Enum8('unspecified'=0,'windows'=1,'linux'=2,'macos'=3) DEFAULT 'unspecified',
    src_ip          Array(String) DEFAULT [],

    -- классификация
    severity        Enum8('unspecified'=0,'info'=1,'low'=2,'medium'=3,'high'=4,'critical'=5) DEFAULT 'unspecified',
    category        Enum8('unspecified'=0,'process'=1,'network'=2,'file'=3,'registry'=4,'auth'=5,'dns'=6,'module'=7) DEFAULT 'unspecified',

    -- нормализация (OCSF / MITRE)
    ocsf_class_uid     UInt32 DEFAULT 0,
    ocsf_category_uid  UInt32 DEFAULT 0,
    ocsf_activity_id   UInt32 DEFAULT 0,
    mitre_tactics      Array(LowCardinality(String)) DEFAULT [],
    mitre_techniques   Array(LowCardinality(String)) DEFAULT [],

    -- on-agent детекция
    detection_rule_id     String DEFAULT '',
    detection_engine      LowCardinality(String) DEFAULT '',
    detection_confidence  Float32 DEFAULT 0,

    -- контроль
    pii_sanitized   UInt8 DEFAULT 0,

    -- payload: типизированные поля разворачиваются в processors; здесь — гибко.
    payload         String DEFAULT '',              -- JSON/CBOR-сериализованное тело

    -- индекс пропуска для быстрого поиска по технике ATT&CK
    INDEX idx_technique mitre_techniques TYPE bloom_filter GRANULARITY 4,
    INDEX idx_rule detection_rule_id TYPE bloom_filter GRANULARITY 4
)
ENGINE = ReplacingMergeTree(ingested_at)
PARTITION BY toYYYYMMDD(observed_at)
ORDER BY (tenant_id, node_id, observed_at, event_id)
TTL
    -- ADR-0004: hot -> cold; полное удаление через 365 дней.
    toDateTime(observed_at) + INTERVAL 7 DAY  TO VOLUME 'cold',
    toDateTime(observed_at) + INTERVAL 365 DAY DELETE
SETTINGS
    storage_policy = 'tiered',
    index_granularity = 8192;

-- ── Детекции/алерты (долгое хранение, малый объём) ───────────────────────────
CREATE TABLE IF NOT EXISTS era_xdr.detections
(
    detection_id    String,                         -- ULID
    event_id        String,
    correlation_id  String DEFAULT '',
    observed_at     DateTime64(9, 'UTC'),
    tenant_id       LowCardinality(String),
    node_id         String,
    rule_id         String,
    rule_name       String,
    severity        Enum8('unspecified'=0,'info'=1,'low'=2,'medium'=3,'high'=4,'critical'=5),
    engine          LowCardinality(String),
    confidence      Float32 DEFAULT 0,
    mitre_techniques Array(LowCardinality(String)) DEFAULT [],
    status          Enum8('new'=0,'triaged'=1,'investigating'=2,'closed'=3,'false_positive'=4) DEFAULT 'new'
)
ENGINE = ReplacingMergeTree()
PARTITION BY toYYYYMM(observed_at)
ORDER BY (tenant_id, observed_at, detection_id)
TTL toDateTime(observed_at) + INTERVAL 3 YEAR DELETE
SETTINGS storage_policy = 'tiered', index_granularity = 8192;

-- ── История инвентаря (typed, ADR-0011) ───────────────────────────────────────
CREATE TABLE IF NOT EXISTS era_xdr.inventory_history
(
    event_id        String,
    tenant_id       LowCardinality(String),
    node_id         String,
    hostname        String DEFAULT '',
    agent_id        String DEFAULT '',
    agent_version   LowCardinality(String) DEFAULT '',
    platform        LowCardinality(String) DEFAULT '',
    os_name         String DEFAULT '',
    os_version      String DEFAULT '',
    kernel          String DEFAULT '',
    cpu_cores       UInt32 DEFAULT 0,
    ram_mb          UInt64 DEFAULT 0,
    software        String DEFAULT '',              -- JSON array
    observed_at     DateTime64(9, 'UTC'),
    ingested_at     DateTime64(9, 'UTC') DEFAULT now64(9),

    INDEX idx_node node_id TYPE bloom_filter GRANULARITY 4
)
ENGINE = ReplacingMergeTree(ingested_at)
PARTITION BY toYYYYMM(observed_at)
ORDER BY (tenant_id, node_id, observed_at, event_id)
TTL toDateTime(observed_at) + INTERVAL 2 YEAR DELETE
SETTINGS storage_policy = 'tiered', index_granularity = 8192;
