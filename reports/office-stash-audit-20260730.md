# Office stash / backup audit — 2026-07-30

**Refs:** `stash@{0}` (`temp2` on `dev`), untracked parent `stash@{0}^3`, branch `comms-stash-backup`  
**Action:** read-only inventory + targeted `git checkout stash@{0}^3 -- <paths>`  
**Not done:** `stash drop` / `stash pop` / full-tree restore

## Summary

| Location | Office/Drive/identity content |
|----------|-------------------------------|
| Working tree (before) | `platform/drive` lib + `drive.proto`/`office.proto` messages; **no** cmd/drive-api, migrations, compose.office, acceptance docs, gate |
| `stash@{0}` (tracked WIP) | mainly go.mod noise + `gen-proto.ps1` already listing drive/office/comms; profile/docs tweaks |
| `stash@{0}^3` (untracked) | **Full Office P0/P1 scaffold**: drive-api, identity-api, workspace, docs-engine, compose, Dockerfiles, SQL, Office docs/gates/scripts, ui/drive, ui/office |
| `comms-stash-backup` | marketing/pricing/office.yaml profile only — **no** platform Drive code |

## Restored into working tree (targeted)

- `deploy/docker-compose.office.yml`, `.prod.yml`, `office.env.example`
- `deploy/dockerfiles/Dockerfile.{drive-api,identity-api,workspace,docs-engine}` + go.work/Cargo helpers
- `deploy/postgres/migrations/platform/001_drive.sql`
- `deploy/sbom/office-deny-licenses.txt`
- `docs/Office-*`, `docs/products/Office-Acceptance-System.md`, `PRD-Office-P0/P1/P2.md`
- `scripts/run-office-*.ps1`, `office-*.ps1/sh`, bootstrap SQL
- `services/platform/cmd/{drive-api,identity-api,workspace}`
- `services/platform/{docs-engine,workspace}`
- `services/platform/licensegate/{office_documents,platform_drive}_test.go`
- `ui/drive`, `ui/office`
- `scripts/gen-proto.ps1` from `stash@{0}` (includes drive/office/comms)

## Intentionally not restored

- Historical `reports/office-stage-*.log` / pilot logs from stash (noise; new gates write fresh logs)
- No stash drop — keep `stash@{0}` until PO confirms

## Follow-up for plan alignment

Stash wave IDs differ from MVP plan (`O-2`=Drive in stash docs vs plan `O-1`=Drive). Phase 1 work adapts Acceptance/Sprint-Index/gates to plan waves **O-0…O-GA** without discarding recovered code.
