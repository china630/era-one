# ERA Mail Moderation — IceWarp lab (AC-MM-9 / MM-H-6/7)

**Wave:** C-MM-H  
**Дата:** 29 июля 2026 г.  
**PRD:** [`PRD-Mail-Moderation.md`](products/PRD-Mail-Moderation.md)

## Topology

```
MUA (Outlook/Thunderbird)
   → SMTP host:2535 → era-mail-moderation (:2525)
        → hold (memory or PG via ERA_MM_POSTGRES_DSN)
        → notify moderator (ERA_MM_NOTIFY_SMTP / Recorder)
        → on Approve: SMTP Upstream → IceWarp (ERA_MM_UPSTREAM)
```

## Env (compose)

| Variable | Role |
|----------|------|
| `ERA_MM_UPSTREAM` | MTA after Approve (`host:port`); empty = in-memory |
| `ERA_MM_NOTIFY_SMTP` | Moderator notify SMTP; falls back to upstream |
| `ERA_MM_POSTGRES_DSN` | Hold/curators/rules persist; empty = memory |
| `ERA_MM_LDAP_JSON` | Directory snapshot JSON path |
| `ERA_MM_ICEWARP_HOST` | Lab script live check (`host` or `host:port`) |

Partner overlay: `deploy/docker-compose.comms.partner.yml` + `deploy/comms-partner.env.example`.

## Lab steps

1. `docker compose -f deploy/docker-compose.comms.yml up -d era-mail-moderation`
2. Load rules (`ERA_MM_RULES_YAML` or `POST /v1/moderation/rules/import`). Admin UI: `http://127.0.0.1:8360/ui/`
3. Point MUA submission to `host:2535`.
4. Set `ERA_MM_UPSTREAM=<icewarp>:25` for real IceWarp.
5. Approve via action link (`:8360/v1/moderation/action?token=…`) or UI holds list.
6. Confirm message in IceWarp mailbox.

## Checklist (field)

| # | Check | Result |
|---|-------|--------|
| 1 | Moderation SMTP :2535 accepts MAIL | |
| 2 | Hold created; moderator notified | |
| 3 | Approve → upstream IceWarp | |
| 4 | Reject → sender notify + no delivery | |
| 5 | Restart with PG DSN → pending holds survive | |

## CI evidence (no IceWarp required)

```powershell
go test -C services/comms/mail-moderation ./... -count=1
.\scripts\run-comms-mm-icewarp-lab.ps1          # SKIP without ERA_MM_ICEWARP_HOST
.\scripts\run-comms-stage-cmm-h-e2e.ps1
.\scripts\run-comms-stage-gate.ps1 -Stage C-MM-H -WriteSignoff
```

Logs: `reports/comms-stage-C-MM-H-e2e.log`, `reports/comms-mm-icewarp-lab.log`
