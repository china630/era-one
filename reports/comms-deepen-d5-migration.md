# Deepen D5 — Migration (lab) — 2026-08-04

**Status:** `[x] lab` · **not `ga`** · Pilot field / cutover open (RT-11).

## Honesty

| Check | Result |
|-------|--------|
| Calendar stub vs `items_ok` | `ImportCalendar` reported as `calendar_stub_count` only; **does not** inflate `items_total` / `items_ok` |
| PST smoke | `mode=stub`, `items_ok=0` |
| CH `migration_job.mailbox` | Audit `Event.Mailbox` set on create; `CHWriter` inserts mailbox column |

## Proof

- Unit: `go test -C services/comms/migration ./internal/api/ -run TestCreateJobAndRerun`
  - asserts `items_ok:0`, `calendar_stub_count:1`, mailbox on response + audit events
- Live IMAP lab (RT-11 optional cutover drill):  
  `.\scripts\run-comms-migration-live-imap.ps1`  
  asserts CH `migration_job WHERE mailbox != ''`

## Out of scope (partner / field)

- GAP-P1-12 EWS calendar → ERA store (field)
- Partner cutover SignOff
