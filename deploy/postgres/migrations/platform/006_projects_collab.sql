-- Wave E: Projects Collab — assignee, due date, board meta
ALTER TABLE era_platform.projects_tasks
    ADD COLUMN IF NOT EXISTS assignee TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS due_date TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS era_platform.projects_boards (
    tenant_id TEXT PRIMARY KEY,
    name TEXT NOT NULL DEFAULT 'Board',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
