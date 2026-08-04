# ERA Communications — Stage C-MM (ERA Mail Moderation)

**Wave:** C-MM  
**Версия:** 1.0  
**Дата:** 29 июля 2026 г.  
**Продукт:** ERA Mail Moderation  
**PRD:** [`PRD-Mail-Moderation.md`](products/PRD-Mail-Moderation.md)  
**Предусловие:** Waves **C-1** / **C-2** gate = PASS (SMTP upstream для lab); **не** блокирует C-GA MVP

---

## 1. Цель этапа

> Реализовать SMTP edge outbound message moderation: policy → hold → manager/curator Approve/Reject,
> с YAML rules, action links, ClickHouse audit и lab-путём перед IceWarp/любым SMTP.

Этап закрыт, когда §7 Stage Gate = PASS.

## 2. Scope

### Входит

- `services/comms/mail-moderation/` — API, SMTP proxy, policy, resolve, hold, notify, audit.
- License `comms-mail-moderation` (`exists: true`).
- PG DDL `003_mail_moderation.sql`, CH `008_comms_mail_moderation_audit.sql`.
- IceWarp lab runbook + e2e script (mock upstream в CI).

### НЕ входит

- P1: moderated DL, HR API, rich Admin UI wizard.
- P2: DLP PII, multi-level, Outlook native buttons.
- Полный milter production (только stub + unit test).

## 3. Архитектура

```
mail-moderation (Go)
  ├── :2525 smtpproxy     → submit → policy → hold|forward
  ├── :8360 adminapi      → rules CRUD, YAML, force-release, action links
  ├── policy / resolve / hold / notify / audit
  └── milter stub (optional adapter)
upstream: ERA_MM_UPSTREAM (lab = mock / IceWarp)
```

## 4. E2E-сценарий приёмки

1. `go test ./services/comms/mail-moderation/...` (policy/hold/SMTP/action links).
2. `.\scripts\run-comms-stage-cmm-e2e.ps1`
3. `.\scripts\run-comms-stage-gate.ps1 -Stage C-MM -WriteSignoff`

## 5. Критерии приёмки

| ID | Критерий | PRD | Доказательство | Статус |
|----|----------|-----|----------------|--------|
| F-MM-1 | Novices + external → hold | AC-MM-1 | `go test .../policy` | [x] |
| F-MM-2 | Approve/Reject + comment | AC-MM-2 | smtpproxy + engine tests | [x] |
| F-MM-3 | Static/attr override > manager | AC-MM-3 | resolve/policy tests | [x] |
| F-MM-4 | Keywords + VIP domain | AC-MM-4 | policy golden | [x] |
| F-MM-5 | TTL auto-reject | AC-MM-5 | hold TTL test | [x] |
| F-MM-6 | Bypass group | AC-MM-6 | policy golden | [x] |
| F-MM-7 | Action-link approve | AC-MM-7 | notify token test | [x] |
| F-MM-8 | Audit events | AC-MM-8 | audit recorder test | [x] |
| F-MM-9 | SMTP → Moderation → upstream | AC-MM-9 | smtpproxy e2e + lab runbook | [x] |
| F-MM-10 | YAML import/export | AC-MM-10 | adminapi round-trip | [x] |

## 6. Backlog (CM-MM-*)

| ID | Задача | Модуль | Статус |
|----|--------|--------|--------|
| CM-MM-1 | Stage spec + index + matrix | docs | [x] |
| CM-MM-2 | Scaffold + go.work + main | mail-moderation | [x] |
| CM-MM-3 | policy + golden | policy | [x] |
| CM-MM-4 | resolve | resolve | [x] |
| CM-MM-5 | hold + PG DDL + TTL | hold | [x] |
| CM-MM-6 | notify + action links | notify | [x] |
| CM-MM-7 | adminapi + YAML + templates | adminapi | [x] |
| CM-MM-8 | smtpproxy | smtpproxy | [x] |
| CM-MM-9 | CH audit DDL + writer | audit | [x] |
| CM-MM-10 | E2E + IceWarp lab runbook | scripts/docs | [x] |
| CM-MM-11 | milter stub | milter | [x] |
| CM-MM-12 | license + compose + stage-gate | deploy | [x] |

## 7. Stage Gate

| # | Проверка | Доказательство | Статус |
|---|----------|----------------|--------|
| G1 | Авто-тесты этапа | `.\scripts\run-comms-stage-gate.ps1 -Stage C-MM` | [x] |
| G2 | E2E §4 выполнен | `reports/comms-stage-C-MM-e2e.log` | [x] |
| G3 | Comms-Implementation-Matrix обновлена | AC-MM `[x]` | [x] |
| G4 | Sprint-Index Wave C-MM → [x] | docs | [x] |
| G5 | editions/pricing sync | `go test ./services/platform/licensegate/...` | [x] |
| G6 | Signoff-запись | `reports/comms-stage-C-MM-signoff.md` | [x] |

## 8. Связано

- [`Comms-Sprint-Index.md`](Comms-Sprint-Index.md)
- [`Comms-Implementation-Matrix.md`](Comms-Implementation-Matrix.md)
- [`Comms-Mail-Moderation-IceWarp-Lab.md`](Comms-Mail-Moderation-IceWarp-Lab.md)
- [`PRD-Mail-Moderation.md`](products/PRD-Mail-Moderation.md)
