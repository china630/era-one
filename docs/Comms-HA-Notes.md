# ERA Comms — HA / multi-node notes (GAP-P2-06)

**Дата:** 30 июля 2026 г.  
**Статус:** design notes (not field proof)

## Scope

Mail Server + Bridge + Migration for air-gapped customer sites beyond single-node compose.

## Recommended topology (pilot → production)

| Tier | Pattern | Notes |
|------|---------|-------|
| Postgres | Primary + sync replica | `ERA_COMMS_DATABASE_URL` points at primary; failover runbook |
| ClickHouse | 2+ shards or replicated MergeTree | Audit must survive node loss |
| MinIO | erasure-coded cluster | Blob offload for large MIME |
| mail-api / mail-core | 2× behind L4 LB | Sticky not required for IMAP if mailbox state in PG |
| Kafka (XDR shared) | RF=3 when co-located | Not required for Mail-only MVP |

## Load proof

- Lab CI: `go run ./services/comms/cmd/loadgen-mailboxes -quick -mailboxes 1000`
- Field: `loadgen-mailboxes -mailboxes 60000` on sizing host (see [`Field-Server-Sizing.md`](Field-Server-Sizing.md))
- Evidence: log under `reports/comms-scale-60k*.log` — do not mark Pilot-ready without that log

## Out of scope until field

Automatic cross-region DR, ActiveSync HA sticky sessions.
