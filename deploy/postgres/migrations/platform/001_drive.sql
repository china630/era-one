-- ERA Platform — Drive metadata (Office P0, ADR-0025)
CREATE SCHEMA IF NOT EXISTS era_platform;

CREATE TABLE IF NOT EXISTS era_platform.drive_folders (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL,
    parent_id   TEXT NOT NULL DEFAULT '',
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_drive_folders_tenant ON era_platform.drive_folders (tenant_id);

CREATE TABLE IF NOT EXISTS era_platform.drive_objects (
    id             TEXT PRIMARY KEY,
    tenant_id      TEXT NOT NULL,
    folder_id      TEXT NOT NULL DEFAULT '',
    name           TEXT NOT NULL,
    size_bytes     BIGINT NOT NULL DEFAULT 0,
    content_type   TEXT NOT NULL DEFAULT 'application/octet-stream',
    content_format INT NOT NULL DEFAULT 1,
    version        INT NOT NULL DEFAULT 1,
    blob_key       TEXT NOT NULL,
    owner_user_id  TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_drive_objects_tenant ON era_platform.drive_objects (tenant_id);
CREATE INDEX IF NOT EXISTS idx_drive_objects_folder ON era_platform.drive_objects (tenant_id, folder_id);

CREATE TABLE IF NOT EXISTS era_platform.drive_versions (
    object_id    TEXT NOT NULL REFERENCES era_platform.drive_objects(id) ON DELETE CASCADE,
    version      INT NOT NULL,
    blob_key     TEXT NOT NULL,
    size_bytes   BIGINT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (object_id, version)
);

CREATE TABLE IF NOT EXISTS era_platform.drive_acl (
    object_id   TEXT NOT NULL REFERENCES era_platform.drive_objects(id) ON DELETE CASCADE,
    principal   TEXT NOT NULL,
    role        INT NOT NULL,
    PRIMARY KEY (object_id, principal)
);

CREATE TABLE IF NOT EXISTS era_platform.tenants (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    status      TEXT NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS era_platform.tenant_domains (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL REFERENCES era_platform.tenants(id) ON DELETE CASCADE,
    fqdn        TEXT NOT NULL UNIQUE,
    is_primary  BOOLEAN NOT NULL DEFAULT false
);
