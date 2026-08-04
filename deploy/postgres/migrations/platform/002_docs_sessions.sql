-- ERA Platform — Documents session ops (Office P1)
CREATE TABLE IF NOT EXISTS era_platform.doc_sessions (
    tenant_id        TEXT NOT NULL,
    drive_object_id  TEXT NOT NULL,
    version          BIGINT NOT NULL DEFAULT 0,
    ops_json         JSONB NOT NULL DEFAULT '[]',
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, drive_object_id)
);

CREATE INDEX IF NOT EXISTS idx_doc_sessions_object ON era_platform.doc_sessions (drive_object_id);
