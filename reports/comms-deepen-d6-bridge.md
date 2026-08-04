# Deepen D6 — Outlook Bridge (lab) — 2026-08-04

**Status:** `[x] lab` · **not `ga`** · Outlook field open.

## Honesty

| Check | Result |
|-------|--------|
| `/healthz` `upstream_mode` | Present: `stub` (default), `synthetic` (`ERA_BRIDGE_SYNTHETIC=1`), or `ERA_BRIDGE_UPSTREAM` |
| Prod DEV fail-closed | `deploy/docker-compose.comms.prod.yml` sets `ERA_BRIDGE_DEV: "0"` |
| **synthetic ≠ field** | Synthetic/lab echo is **not** Exchange/Outlook field evidence; do not mark Pilot field on synthetic |

## Proof

- `go test -C services/comms/mail-bridge ./internal/api/ -run 'Healthz|Unauthorized'`
  - `upstream_mode=stub` + `upstream_mode=synthetic`
- Compose: `ERA_BRIDGE_DEV: "0"` under `era-mail-bridge` in prod overlay

## Out of scope

- Partner Outlook Autodiscover/EWS field RT
