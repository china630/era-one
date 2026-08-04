-- Soft-delete / Trash for Drive objects and folders (O-SHELL)
ALTER TABLE era_platform.drive_objects
  ADD COLUMN IF NOT EXISTS trashed_at TIMESTAMPTZ NULL,
  ADD COLUMN IF NOT EXISTS trash_restore_folder_id TEXT NULL;

ALTER TABLE era_platform.drive_folders
  ADD COLUMN IF NOT EXISTS trashed_at TIMESTAMPTZ NULL,
  ADD COLUMN IF NOT EXISTS trash_restore_parent_id TEXT NULL;

CREATE INDEX IF NOT EXISTS idx_drive_objects_trash
  ON era_platform.drive_objects (tenant_id, trashed_at)
  WHERE trashed_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_drive_folders_trash
  ON era_platform.drive_folders (tenant_id, trashed_at)
  WHERE trashed_at IS NOT NULL;
