-- ERA Communications — mail audit (PRD AC-C7, ADR-0027)
-- Применять после 001_schema.sql

CREATE DATABASE IF NOT EXISTS era_comms;

CREATE TABLE IF NOT EXISTS era_comms.mail_audit
(
    event_id        String,
    schema_version  LowCardinality(String),
    observed_at     DateTime64(3, 'UTC'),
    tenant_id       LowCardinality(String),
    mailbox         String,
    action          LowCardinality(String),
    message_id      String DEFAULT '',
    src_ip          IPv4 DEFAULT toIPv4('0.0.0.0'),
    metadata        Map(String, String)
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(observed_at)
ORDER BY (tenant_id, observed_at, event_id)
TTL toDateTime(observed_at) + INTERVAL 365 DAY
SETTINGS index_granularity = 8192;
