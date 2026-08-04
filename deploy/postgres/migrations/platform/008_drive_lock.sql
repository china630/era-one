-- ERA Platform — Drive object lock (ERA+ Lock file)
ALTER TABLE era_platform.drive_objects
    ADD COLUMN IF NOT EXISTS locked_by TEXT NULL,
    ADD COLUMN IF NOT EXISTS locked_at TIMESTAMPTZ NULL;
