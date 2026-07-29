-- ERA Communications — mail moderation audit (AC-MM-8)
CREATE DATABASE IF NOT EXISTS era_comms;

CREATE TABLE IF NOT EXISTS era_comms.mail_moderation_event
(
    event_id     String,
    observed_at  DateTime64(3, 'UTC'),
    hold_id      String,
    action       LowCardinality(String),
    sender       String DEFAULT '',
    rule_id      String DEFAULT '',
    moderator    String DEFAULT '',
    metadata     Map(String, String)
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(observed_at)
ORDER BY (hold_id, observed_at, event_id)
TTL toDateTime(observed_at) + INTERVAL 365 DAY
SETTINGS index_granularity = 8192;
