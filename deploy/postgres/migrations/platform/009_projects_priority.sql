-- PRJ-LITE: task priority + in-column sort_key
ALTER TABLE era_platform.projects_tasks
    ADD COLUMN IF NOT EXISTS priority TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS sort_key DOUBLE PRECISION NOT NULL DEFAULT 0;
