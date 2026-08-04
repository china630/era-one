-- Wave W2: Projects labels + checklist on tasks
ALTER TABLE era_platform.projects_tasks
    ADD COLUMN IF NOT EXISTS labels_json TEXT NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS checklist_json TEXT NOT NULL DEFAULT '[]';
