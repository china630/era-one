-- ERA Projects board/tasks (O-PR-H)
CREATE TABLE IF NOT EXISTS era_platform.projects_tasks (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    board TEXT NOT NULL DEFAULT 'backlog',
    drive_object_id TEXT NOT NULL DEFAULT '',
    tenant_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_projects_tasks_tenant ON era_platform.projects_tasks (tenant_id);
