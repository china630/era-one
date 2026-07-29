-- Phase 2 migration orchestrator (Comms MIG-P0 continuation)
CREATE TABLE IF NOT EXISTS migration_jobs (
    id TEXT PRIMARY KEY,
    source TEXT NOT NULL,
    mailbox TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    items_total INT NOT NULL DEFAULT 0,
    items_ok INT NOT NULL DEFAULT 0,
    items_fail INT NOT NULL DEFAULT 0,
    error TEXT,
    payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS migration_jobs_status_idx ON migration_jobs (status, created_at);

CREATE TABLE IF NOT EXISTS migration_seen_uids (
    uid_key TEXT PRIMARY KEY,
    job_id TEXT REFERENCES migration_jobs(id) ON DELETE CASCADE,
    seen_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS migration_seen_uids_job_idx ON migration_seen_uids (job_id);
