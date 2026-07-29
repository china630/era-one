-- ERA Communications — migration job audit (AC-MIG-4)
CREATE DATABASE IF NOT EXISTS era_comms;

CREATE TABLE IF NOT EXISTS era_comms.migration_job
(
    event_id     String,
    observed_at  DateTime64(3, 'UTC'),
    job_id       String,
    action       LowCardinality(String),
    source_uid   String DEFAULT '',
    mailbox      String DEFAULT '',
    metadata     Map(String, String)
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(observed_at)
ORDER BY (job_id, observed_at, event_id)
TTL toDateTime(observed_at) + INTERVAL 365 DAY
SETTINGS index_granularity = 8192;
