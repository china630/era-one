# ERA Communications — Stage C-MM-H (Mail Moderation Hardening)

**Wave:** C-MM-H  
**Версия:** 1.0  
**Дата:** 29 июля 2026 г.  
**Продукт:** ERA Mail Moderation  
**PRD:** [`PRD-Mail-Moderation.md`](products/PRD-Mail-Moderation.md)  
**Предусловие:** Stage **C-MM** = PASS  
**Статус волны:** [x] 2026-07-29

---

## 1. Цель

> Довести lab MVP до partner-ready: реальный SMTP upstream, SMTP notify, PG hold/curators/rules, LDAP directory adapter, IceWarp lab script.

## 2. Backlog

| ID | Задача | Статус |
|----|--------|--------|
| MM-H-1 | SMTP Upstream | [x] `internal/engine/smtp_upstream.go` + mock test |
| MM-H-2 | SMTP Mailer (notify) | [x] `internal/notify/smtp_mailer.go` + mock test |
| MM-H-3 | PG Hold store | [x] `internal/hold/pgstore.go` (DSN → PG, else memory) |
| MM-H-4 | PG curators + LDAP Directory | [x] `resolve/pg.go`, `ldap.go` (JSON snapshot) |
| MM-H-5 | Rules persist (PG) | [x] `internal/rules` + adminapi Persist |
| MM-H-6 | Compose/env + lab doc | [x] compose + partner env example + IceWarp lab doc |
| MM-H-7 | IceWarp lab script (SKIP without host) | [x] `scripts/run-comms-mm-icewarp-lab.ps1` |
| MM-H-8 | Stage gate C-MM-H + matrix update | [x] gate PASS · Scaffold ✅ · **Pilot-ready [ ]** (IceWarp field open) |

## 3. Stage Gate

| # | Проверка | Доказательство |
|---|----------|----------------|
| G1 | `.\scripts\run-comms-stage-gate.ps1 -Stage C-MM-H` | PASS |
| G2 | e2e log | `reports/comms-stage-C-MM-H-e2e.log` |
| G3 | Matrix AC-MM 🟡 (engine unit; admin AuthZ open); Pilot-ready open | docs |
| G4 | IceWarp lab script | SKIP or PASS log |

Edition остаётся **mvp** (ga — после field sign-off).

## 4. Связано

- [`Comms-Stage-CMM-Spec.md`](Comms-Stage-CMM-Spec.md)
- [`Comms-Mail-Moderation-IceWarp-Lab.md`](Comms-Mail-Moderation-IceWarp-Lab.md)
- [`Comms-Stage-CMM-P1-Spec.md`](Comms-Stage-CMM-P1-Spec.md)
