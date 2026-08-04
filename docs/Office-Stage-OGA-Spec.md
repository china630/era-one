# ERA Office — Stage O-GA (Pilot honesty)

**Wave:** O-GA  
**Версия:** 1.2  
**Дата:** 30 июля 2026 г.  
**Предусловие:** Waves **O-0…O-5** gate = PASS  
**Статус:** `[x]` — gate PASS `reports/office-stage-O-GA-20260730-004758.log`  
**PRD:** AC rollup — см. Matrix SSOT (Documents mixed ✅/🟡; service bind 🟡; post-MVP Tables/P/PR 🟡); Pilot-ready open · канон v1.2

---

## 1. Цель этапа

> Закрыть программу Office MVP **честно**: regression O-0…O-5 green; editions Drive+Documents → `mvp`; Pilot-ready / RT-O09 **не** закрывать.

**O-GA ≠ field GA.** Customer field trial = **RT-O09** (остаётся открытым).  
**`mvp` ≠ `ga`.** Edition `ga` запрещён без Pilot-ready + field sign-off ([`Office-Evidence-Rules.md`](Office-Evidence-Rules.md)).

## 2. Scope

### Входит

- Harden gate O-GA (O-0 files + MVP/Index wave `[x]` asserts)
- Checklist / Gap list honesty
- Editions `era-drive` + `era-documents` → `mvp`
- Matrix/Index/MVP-Spec O-GA closeout with proof

### НЕ входит

- RT-O09 customer signature
- Matrix Pilot-ready ✅
- Edition / product `ga`
- New product features / post-MVP waves

## 3. E2E-сценарий приёмки

1. `pwsh -NoProfile -File ./scripts/run-office-stage-gate.ps1 -Stage O-GA -WriteSignoff`
2. Verify editions `mvp` (not `ga`)
3. Verify Matrix Pilot-ready still `[ ]`; GAP-O-P0-40 / RT-O09 open

## 4. Критерии приёмки

| ID | Критерий | Доказательство | Статус |
|----|----------|----------------|--------|
| F-OGA-1 | O-0…O-5 regression green | O-GA gate log | [x] |
| F-OGA-2 | Checklist/gap/matrix honesty | docs | [x] |
| F-OGA-3 | Editions Drive+Documents `mvp` | `editions-*.yaml` | [x] |
| F-OGA-4 | Pilot-ready / RT-O09 open | Matrix + Gap list | [x] |
| F-OGA-5 | Gate O-GA PASS | `reports/office-stage-O-GA-20260730-004758.log` | [x] |

## 5. Backlog (OM-GA-*)

| ID | Задача | Статус |
|----|--------|--------|
| OM-GA-1 | Stage O-GA Spec | [x] |
| OM-GA-2 | Harden gate O-GA | [x] |
| OM-GA-3 | Fix checklist honesty | [x] |
| OM-GA-4 | Fix gap list honesty | [x] |
| OM-GA-5 | Editions → `mvp` (not `ga`) | [x] |
| OM-GA-6 | Legacy O-GA / O1-GA → non-canon pointers | [x] |
| OM-GA-7 | Gate PASS + Index/MVP/Matrix close | [x] |

## 6. Stage Gate

| # | Проверка | Доказательство |
|---|----------|----------------|
| G1 | `run-office-stage-gate.ps1 -Stage O-GA` | PASS |
| G3 | Matrix honesty (✅/🟡 per AC); Pilot-ready `[ ]` | docs |
| G4 | Sprint-Index / MVP-Spec O-GA `[x]` | docs |
| G5 | editions `mvp` | yaml |
| G6 | signoff — customer **N/A** (RT-O09 deferred) | `reports/office-stage-O-GA-signoff.md` |

## 7. Связано

- Canon only: this file. Legacy: [`Office-Stage-O-GA-Spec.md`](Office-Stage-O-GA-Spec.md), [`Office-Stage-O1-GA-Spec.md`](Office-Stage-O1-GA-Spec.md) (non-canon pointers).
- [`Office-Pilot-Readiness-Checklist.md`](Office-Pilot-Readiness-Checklist.md)
- [`Office-Pilot-Gap-List.md`](Office-Pilot-Gap-List.md)
- Comms analogue: C-GA honesty / edition `mvp` / RT-09 open
