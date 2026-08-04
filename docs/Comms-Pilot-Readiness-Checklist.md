# ERA Comms — Pilot Readiness Checklist (honest)

> **Статус:** **lab / staging** — RT-01…08 PASS on our infra.  
> **Не** customer field until RT-09 SignOff.  
> `[x]` только с доказательством (тест / лог / CI). Soft field `[x]` запрещён.

**Заказчик:** ____________________  
**Дата пилота:** ____________________  
**Издание:** ERA Mail Server (`mvp`) · Connect/Migration optional  
**Подпись ERA:** ____________________ · **Подпись заказчика:** ____________________

> **Staging:** `.\scripts\run-comms-pilot-staging.ps1 -UseCompose` → `reports/comms-pilot-staging.log`  
> **Prod honesty:** `-ProdProfile` → `ERA_*_DEV=0` ([`Comms-Pilot-Runbook.md`](Comms-Pilot-Runbook.md))  
> **Gap SSOT:** [`Comms-Pilot-Gap-List.md`](Comms-Pilot-Gap-List.md) · Deepen D1

---

## 1. Инфраструктура (lab)

- [x] Compose config valid — `deploy/docker-compose.comms.yml` (+ overlays in profile)
- [x] Lab up `--wait` — staging RT-01 (dev overlay)
- [x] ClickHouse DDL 004…008 mounted — compose init
- [x] Health/readyz — RT-01 + D0 CH require
- [x] Prod overlay DEV=0 path documented — `docker-compose.comms.prod.yml` / D1 report
- [ ] Customer firewall / air-gap perimeter (field)
- [ ] Customer offline license HSM/KMS (field RT-08)

## 2. Mail Server AC (lab)

- [x] AC-C1 SMTP/IMAP / AuthZ — Matrix Scaffold ✅ · staging
- [x] AC-C3 Autodiscover — unit + staging
- [x] AC-C4 quota 413/552 — staging RT-07
- [x] AC-C5 Drive attach deny — D0
- [x] AC-C7 CH audit — D0
- [x] AC-C8/C9 CalDAV/EWS — unit + staging; Outlook field open
- [x] AC-C2 OIDC/BFF/PKCE — lab (Deepen D2); browser field optional

## 3. Webmail UI (lab)

- [x] Staging OIDC machine-flow RT-05 — `reports/comms-pilot-staging*.log`
- [x] PKCE in `ui/mail/web/app.js` — static test + D2 report
- [ ] Customer IdP browser PKCE (field)

## 4. Gov clients (lab unit ≠ field)

- [x] Unit EWS/CardDAV/ActiveSync/Autodiscover/CalDAV — [`Comms-Gov-Lab-Checklist.md`](Comms-Gov-Lab-Checklist.md)
- [ ] Outlook desktop Autodiscover + send (field / partner)
- [ ] Thunderbird IMAP/CalDAV (field)
- [ ] iOS ActiveSync subset (field)

## 5. Connect / Migration (optional)

- [x] Connect real IMAP when address set; stub ItemsOK=0 — AC-C6 lab (Deepen D4)
- [ ] Connect customer IMAP + vault TPM (field RT-10)
- [ ] Migration cutover (field)

## 6. Exit to customer

- [ ] RT-09 SignOff — [`reports/comms-rt09-skip.md`](../reports/comms-rt09-skip.md)
- [ ] Edition remains **mvp** until Pilot field (P0-42)
- [ ] PO approval

**Edition `ga`:** запрещён до Pilot field.
