# ERA Comms — Pilot Runbook (lab / staging)

**Версия:** 1.0  
**Дата:** 4 августа 2026 г.  
**Статус:** lab ops — не customer field sign-off  
**Refs:** GAP-P0-34 · Deepen D1 · [`deploy/profiles/comms.yaml`](../deploy/profiles/comms.yaml)

## Install (air-gap lab)

1. Load images from offline bundle (`docker load`) when applicable.
2. Secrets via env (not in git): `ERA_IDENTITY_JWT_SECRET`, Postgres/CH passwords, TLS material.
3. **Lab:**  
   `docker compose -f deploy/docker-compose.comms.yml -f deploy/docker-compose.comms.dev.yml up -d --wait`
4. **Prod honesty (RT-08 path):**  
   `pwsh deploy/comms-tls/gen-dev-certs.ps1`  
   then  
   `docker compose -f deploy/docker-compose.comms.yml -f deploy/docker-compose.comms.prod.yml up -d --wait`  
   (`ERA_*_DEV=0` explicit in prod overlay).
5. Profile SSOT: `deploy/profiles/comms.yaml` (`compose.file` + overlays).

## Health

| Service | Default | Probe |
|---------|---------|-------|
| era-mail-api | :8150 | `GET /healthz`, `GET /readyz` |
| era-mail-core | SMTP/IMAP | compose healthcheck |
| identity-api | :8160 | `GET /healthz` |
| ui-mail | :8180 | `GET /mail/healthz` |
| clickhouse | :8123 | compose healthcheck |
| postgres | :5432 | compose healthcheck |

## Backup (lab)

- Postgres: `pg_dump` of Comms schema / DSN used by mail-api.
- ClickHouse: snapshot volumes or `BACKUP` for audit tables `004…008`.
- Volumes named in compose project `era-one-comms`.

## Staging gate

```powershell
.\scripts\run-comms-pilot-staging.ps1 -UseCompose
.\scripts\run-comms-pilot-staging.ps1 -UseCompose -ProdProfile   # DEV=0 + TLS
```

Log: `reports/comms-pilot-staging.log` (RT-01…08 lab).

## Rollback

1. `docker compose -f deploy/docker-compose.comms.yml down` (add same `-f` overlays used at up).
2. Restore Postgres + ClickHouse volumes from backup.
3. Re-load previous image tags from offline bundle.

## Incidents (типовые)

| Symptom | Check |
|---------|--------|
| readyz 503 | ClickHouse up; `ERA_MAIL_AUDIT_REQUIRE=1` needs CH |
| SMTP/IMAP auth fail | prod TLS certs mounted; not mixing DEV and prod overlays |
| license deny | `ERA_*_DEV=0` + `ERA_LICENSE_MODULES` on service |
| Autodiscover SSL off | set `ERA_MAIL_TLS=1` via prod overlay |

## Field

Customer RT-09 / SignOff — [`reports/comms-rt09-skip.md`](../reports/comms-rt09-skip.md) until partner. Checklist: [`Comms-Pilot-Readiness-Checklist.md`](Comms-Pilot-Readiness-Checklist.md).
