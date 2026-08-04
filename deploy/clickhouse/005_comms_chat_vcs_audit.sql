-- ERA Communications — chat/VCS audit (Wave C-4, F-C23)
CREATE DATABASE IF NOT EXISTS era_comms;

CREATE TABLE IF NOT EXISTS era_comms.chat_vcs_audit
(
    event_id        String,
    schema_version  LowCardinality(String),
    observed_at     DateTime64(3, 'UTC'),
    tenant_id       LowCardinality(String),
    service         LowCardinality(String),
    action          LowCardinality(String),
    room_id         String DEFAULT '',
    user_id         String DEFAULT '',
    metadata        Map(String, String)
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(observed_at)
ORDER BY (tenant_id, observed_at, event_id)
TTL toDateTime(observed_at) + INTERVAL 365 DAY
SETTINGS index_granularity = 8192;
