-- ERA Communications — Comms AI inference audit (Wave C-5, F-C32)

CREATE DATABASE IF NOT EXISTS era_comms;



CREATE TABLE IF NOT EXISTS era_comms.ai_inference_audit

(

    event_id         String,

    schema_version   LowCardinality(String),

    observed_at      DateTime64(3, 'UTC'),

    tenant_id        LowCardinality(String),

    mailbox_id       String DEFAULT '',

    inference_type   LowCardinality(String),

    model            LowCardinality(String),

    risk_score       UInt8 DEFAULT 0,

    latency_ms       UInt32 DEFAULT 0,

    request_id       String DEFAULT '',

    body_hash        String DEFAULT '',

    metadata         Map(String, String)

)

ENGINE = MergeTree()

PARTITION BY toYYYYMM(observed_at)

ORDER BY (tenant_id, observed_at, event_id)

TTL toDateTime(observed_at) + INTERVAL 365 DAY

SETTINGS index_granularity = 8192;

