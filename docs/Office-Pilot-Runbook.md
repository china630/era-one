# ERA Office — Pilot Runbook

**Версия:** 1.0  
**Дата:** 7 июля 2026 г.  
**Статус:** Scaffold — детали заполняются в O-6 (P0)

## Install (air-gap)

1. Load images from offline bundle (`docker load`).
2. Set secrets via env files (not in git): `POSTGRES_PASSWORD`, `ERA_IDENTITY_JWT_SECRET`, MinIO keys, TLS certs.
3. **Prod:** `docker compose -f deploy/docker-compose.office.yml up -d`
4. Schema: `scripts/office-apply-migrations.ps1` or compose `office-migrate` service.
5. Demo tenant (Office+Comms): `scripts/office-bootstrap-tenants.sql`.
6. Create MinIO bucket `era-drive` (or auto-init in drive-api startup).

## Health

| Service | Port (default) | Probe |
|---------|----------------|-------|
| identity-api | :8160 | `GET /healthz` |
| drive-api | :8175 | `GET /healthz` |
| workspace | :8170 | `GET /healthz` |
| admin-portal | :8140 | `GET /healthz` |
| docs-engine (P1) | :8142 | `GET /healthz` |
| ui-mail (optional) | :8180 | `GET /mail/healthz` |

## License modules

| Module | Edition | Required for |
|--------|---------|--------------|
| `platform-drive` | ERA Drive | Drive API, Workspace /drive |
| `office-documents` | ERA Documents | docs-engine, /docs (P1) |

Prod: offline Ed25519 license via ADR-0010; no `ERA_OFFICE_DEV` in prod compose.

## Backup

- Postgres: `pg_dump -n era_platform "$ERA_OFFICE_DATABASE_URL" > era_platform.sql` (drive metadata, doc_sessions).
- MinIO: snapshot `era-drive` bucket.

## Tenant bootstrap (Office + Comms)

1. Apply platform migrations (`office-apply-migrations`).
2. Run `scripts/office-bootstrap-tenants.sql` (inserts `t-demo` into `era_platform` and `era_comms` when schema exists).
3. Use the same `tenant_id` in identity, drive-api, and mail provisioning.

## Staging gate

```powershell
.\scripts\run-office-pilot-staging.ps1
```

Log: `reports/office-pilot-staging.log`

## Stage gates (development)

```powershell
.\scripts\run-office-stage-gate.ps1 -Stage O-GOV
.\scripts\run-office-stage-gate.ps1 -Stage O-1   # after implementation
.\scripts\run-office-acceptance.ps1
```

## Field sign-off

1. Complete [`Office-Pilot-Readiness-Checklist.md`](Office-Pilot-Readiness-Checklist.md).
2. Close all GAP-O-P0-* in [`Office-Pilot-Gap-List.md`](Office-Pilot-Gap-List.md).
3. RT-O09 customer signature on checklist.

## Rollback

1. `docker compose -f deploy/docker-compose.office.yml down`
2. Restore Postgres + MinIO from backup.
3. Re-apply previous image tags from offline bundle.

## Связано

- [`Office-Pilot-Gap-List.md`](Office-Pilot-Gap-List.md)
- [`Office-Acceptance-System.md`](products/Office-Acceptance-System.md)
- [`deploy/profiles/office.yaml`](../deploy/profiles/office.yaml)
