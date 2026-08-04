-- Canonical copy of deploy/postgres/migrations/comms/001_initial.sql (ADR-0029)
CREATE SCHEMA IF NOT EXISTS era_comms;

CREATE TABLE IF NOT EXISTS era_comms.tenants (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS era_comms.domains (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL REFERENCES era_comms.tenants(id),
    fqdn        TEXT NOT NULL UNIQUE,
    is_primary  BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS era_comms.mailboxes (
    id              TEXT PRIMARY KEY,
    tenant_id       TEXT NOT NULL REFERENCES era_comms.tenants(id),
    email           TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    quota_bytes     BIGINT NOT NULL DEFAULT 536870912,
    used_bytes      BIGINT NOT NULL DEFAULT 0,
    active          BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS era_comms.messages (
    id              BIGSERIAL PRIMARY KEY,
    mailbox_id      TEXT NOT NULL REFERENCES era_comms.mailboxes(id),
    uid             BIGINT NOT NULL,
    message_id      TEXT,
    subject         TEXT,
    from_addr       TEXT,
    flags           TEXT NOT NULL DEFAULT '',
    size_bytes      BIGINT NOT NULL DEFAULT 0,
    minio_key       TEXT,
    raw_inline      BYTEA,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (mailbox_id, uid)
);

CREATE INDEX IF NOT EXISTS idx_messages_mailbox ON era_comms.messages(mailbox_id);

CREATE TABLE IF NOT EXISTS era_comms.calendar_events (
    id              TEXT PRIMARY KEY,
    mailbox_id      TEXT NOT NULL REFERENCES era_comms.mailboxes(id),
    uid             TEXT NOT NULL,
    ical_data       TEXT NOT NULL,
    etag            TEXT NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (mailbox_id, uid)
);

CREATE TABLE IF NOT EXISTS era_comms.contacts (
    id              TEXT PRIMARY KEY,
    mailbox_id      TEXT NOT NULL REFERENCES era_comms.mailboxes(id),
    uid             TEXT NOT NULL,
    vcard_data      TEXT NOT NULL,
    etag            TEXT NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (mailbox_id, uid)
);

CREATE TABLE IF NOT EXISTS era_comms.tenant_policies (
    tenant_id                   TEXT PRIMARY KEY REFERENCES era_comms.tenants(id),
    inline_max_attachment_mb    INT NOT NULL DEFAULT 25,
    inline_quota_mb_per_user    INT NOT NULL DEFAULT 512,
    inline_retention_days       INT NOT NULL DEFAULT 365,
    inline_max_attachments      INT NOT NULL DEFAULT 50
);

CREATE TABLE IF NOT EXISTS era_comms.eas_device_sync_keys (
    device_id       TEXT NOT NULL,
    mailbox_id      TEXT NOT NULL REFERENCES era_comms.mailboxes(id),
    folder_id       TEXT NOT NULL,
    sync_key        TEXT NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (device_id, mailbox_id, folder_id)
);

CREATE TABLE IF NOT EXISTS era_comms.identity_users (
    id              TEXT PRIMARY KEY,
    tenant_id       TEXT NOT NULL REFERENCES era_comms.tenants(id),
    email           TEXT NOT NULL UNIQUE,
    display_name    TEXT NOT NULL DEFAULT '',
    password_hash   TEXT NOT NULL,
    active          BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE IF NOT EXISTS era_comms.oidc_clients (
    client_id       TEXT PRIMARY KEY,
    client_secret   TEXT NOT NULL,
    redirect_uris   TEXT[] NOT NULL
);
