-- ERA Mail Moderation (Stage C-MM)
CREATE TABLE IF NOT EXISTS moderation_rules (
    id TEXT PRIMARY KEY,
    priority INT NOT NULL DEFAULT 100,
    yaml_body TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS moderation_curators (
    sender_email TEXT PRIMARY KEY,
    curator_email TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS moderation_holds (
    id TEXT PRIMARY KEY,
    status TEXT NOT NULL DEFAULT 'pending',
    rule_id TEXT NOT NULL DEFAULT '',
    sender TEXT NOT NULL,
    recipients TEXT NOT NULL,
    subject TEXT NOT NULL DEFAULT '',
    moderators TEXT NOT NULL,
    blob_path TEXT NOT NULL DEFAULT '',
    raw_bytes BYTEA,
    comment TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS moderation_holds_status_idx ON moderation_holds (status, expires_at);
