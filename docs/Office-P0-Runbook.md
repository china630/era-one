# ERA Office P0 — operator runbook
# Refs: Office-Pilot-Runbook.md, deploy/docker-compose.office.yml

## Quick start

```powershell
docker compose -f deploy/docker-compose.office.yml up -d
```

Health: identity `:8160`, drive-api `:8175`, workspace `:8170`.

## Environment

| Variable | Service | Default |
|----------|---------|---------|
| `ERA_OFFICE_DEV` | drive-api | `1` in compose — enables `platform-drive` |
| `ERA_DRIVE_BUCKET` | drive-api | `era-drive` |
| `ERA_MINIO_*` | drive-api | MinIO connection |
| `ERA_IDENTITY_JWT_SECRET` | identity, drive, mail | shared dev secret |
| `ERA_WORKSPACE_BASE_URL` | drive-api | deep links for Mail hook |

## License module

- Module id: `platform-drive`
- Without module: Drive API returns **403**
- Dev: set `ERA_OFFICE_DEV=1`

## Smoke

```powershell
.\scripts\office-p0-smoke.ps1
```
