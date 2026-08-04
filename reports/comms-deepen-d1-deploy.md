# Deepen D1 — Deploy/ops (partner-free lab) — 2026-08-04

## Scope

GAP-P0-30…35 + P0-42 honesty · profile `deploy/profiles/comms.yaml` · prod overlay RT-08 path.

## Compose / profile

| Artifact | Role |
|----------|------|
| `deploy/docker-compose.comms.yml` | Base stack (mail-core, mail-api, CH, PG, …) |
| `deploy/docker-compose.comms.dev.yml` | Lab: `ERA_*_DEV=1` |
| `deploy/docker-compose.comms.prod.yml` | Prod honesty: `ERA_*_DEV=0` + TLS + license modules |
| `deploy/profiles/comms.yaml` | SSOT: `compose.file` + `overlays` |

```powershell
# Lab
docker compose -f deploy/docker-compose.comms.yml -f deploy/docker-compose.comms.dev.yml up -d --wait

# Prod / RT-08 offline-license path (no DEV bypass)
pwsh deploy/comms-tls/gen-dev-certs.ps1
docker compose -f deploy/docker-compose.comms.yml -f deploy/docker-compose.comms.prod.yml up -d --wait
# or: .\scripts\run-comms-pilot-staging.ps1 -UseCompose -ProdProfile
```

## RT-08 / ERA_*_DEV=0

Prod overlay **must** set explicit `"0"` (compose merge does not clear base keys):

- `ERA_MAIL_DEV=0`
- `ERA_IDENTITY_DEV=0`
- `ERA_BRIDGE_DEV=0`
- `ERA_MM_DEV=0`
- `ERA_LICENSE_MODULES` includes `comms-mail-server` (and bridge/moderation modules on those services)

Evidence path: staging logs with `-ProdProfile` (Autodiscover SSL on) · `deploy/comms-tls/README.md`.

## Gap status (lab)

| ID | Lab | Note |
|----|-----|------|
| P0-30 | [x] | profile + compose + prod overlay |
| P0-31 | [x] | CH 004…008 mounted in compose |
| P0-32 | [x] | healthcheck / readyz (CH require D0) |
| P0-33 | [x] | prod overlay DEV=0 + modules |
| P0-34 | [x] | [`docs/Comms-Pilot-Runbook.md`](../docs/Comms-Pilot-Runbook.md) |
| P0-35 | [x] | [`docs/Comms-Pilot-Readiness-Checklist.md`](../docs/Comms-Pilot-Readiness-Checklist.md) — no soft field `[x]` |
| P0-42 | [x] | Index/Matrix keep **mvp**; RT-09 field open (no Comms-MVP-Spec soft-close) |

Field Pilot / `ga` remain ⏸ until partner RT-09.
