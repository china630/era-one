-- Mail Moderation quorum / multi-level (Summer S2-C)
ALTER TABLE moderation_holds ADD COLUMN IF NOT EXISTS require_all BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE moderation_holds ADD COLUMN IF NOT EXISTS approved_by JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE moderation_holds ADD COLUMN IF NOT EXISTS level INT NOT NULL DEFAULT 0;
