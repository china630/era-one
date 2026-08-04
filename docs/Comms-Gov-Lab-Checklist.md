# Gov protocols lab checklist (G3 / Deepen D3 — lab only)

Not field Outlook / Thunderbird / iOS until partner RT-09.  
Evidence: unit PASS + staging Autodiscover/CalDAV where noted · [`reports/comms-gov-lab-deepen-20260804.log`](../reports/comms-gov-lab-deepen-20260804.log)

## Automated (CI / local)

| ID | Check | How |
|----|-------|-----|
| GOV-01 | Autodiscover XML | `go test -C services/comms/mail ./internal/autodiscover/` |
| GOV-02 | EWS FindFolder/CreateItem unit | `go test -C services/comms/mail ./internal/ews/` |
| GOV-03 | CalDAV unit | `go test -C services/comms/calendar ./...` |
| GOV-04/05 | CardDAV / contacts | `go test -C services/comms/mail ./internal/carddav/` |
| GOV-06 | Notes/Tasks | ews notes/tasks golden if present |
| GOV-07 | ActiveSync subset | `go test -C services/comms/mail ./internal/activesync/` |
| GOV-08 | no MAPI | ADR/RFQ wording |

One-shot:

```powershell
go test -C services/comms/mail ./internal/ews/ ./internal/carddav/ ./internal/activesync/ ./internal/autodiscover/ -count=1
go test -C services/comms/calendar ./... -count=1
```

## Manual lab — Outlook (desktop)

Prerequisite: staging or lab compose with Autodiscover + EWS + mail-core TLS (prefer prod overlay).

1. DNS/hosts: map Autodiscover URL to lab mail-api (or use explicit server).
2. File → Account Settings → New → Exchange / Autodiscover to lab endpoint.
3. Confirm Autodiscover XML returns EXCH/EWS URLs (GOV-01 staging RT-08).
4. Send/receive one message; create calendar item via EWS path if account type supports it.
5. **Pass criterion (lab):** account configures without MAPI/Outlook Connector; mail visible.  
   **Field:** still open (customer AD/TLS/IdP).

## Manual lab — Thunderbird

1. IMAP: host/port from lab mail-core (`143`/`993`), AUTH LOGIN.
2. SMTP: submission port from lab (`587`/`2525` lab).
3. Optional CalDAV: subscribe to calendar URL from lab calendar service.
4. **Pass criterion (lab):** list/send one message. CardDAV optional if TB version supports it.

## Manual lab — iOS Mail (ActiveSync subset)

1. Settings → Mail → Accounts → Exchange (ActiveSync).
2. Server = lab ActiveSync endpoint; use lab credentials.
3. Exercise Provision + FolderSync + Sync mail subset (GOV-07).
4. **Pass criterion (lab):** account provisions; inbox syncs subset. Full parity = field (GAP-P2-04).

## Honesty

| Layer | Status |
|-------|--------|
| Unit / golden | [x] lab (D3 log) |
| Staging Autodiscover/CalDAV | [x] where RT-04/08 ran |
| Outlook / TB / iOS on customer | ⏸ Pilot field |

Field Outlook/iOS Pilot-ready remains open until customer clients (G7 / RT-09).
