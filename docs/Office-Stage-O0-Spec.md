# ERA Office — Stage O-0 (Acceptance + Proto SSOT)

**Wave:** O-0  
**Версия:** 1.0  
**Дата:** 30 июля 2026 г.  
**Продукт:** ERA Office  
**PRD:** [`PRD-Office-MVP.md`](products/PRD-Office-MVP.md)  
**Предусловие:** stash audit ([`reports/office-stash-audit-20260730.md`](../reports/office-stash-audit-20260730.md))

---

## 1. Цель этапа

Зафиксировать систему приёмки (зеркало Comms) и сделать **proto SSOT** для Drive/Office в CI (`gen-proto` + golden wire).

Этап закрыт, когда §7 Stage Gate = PASS.

## 2. Scope

### Входит

- Acceptance System, Evidence Rules, MVP-Spec, Sprint-Index, Matrix, Pilot checklist/gap
- Stage specs O-0 / O-1 (+ stubs O-2…O-GA в Index)
- `DriveService` в `drive.proto`; `drive`/`office`/`comms` в `gen-proto.ps1` + `era-proto`
- Golden сериализация ключевых Drive/Office сообщений
- `run-office-stage-gate.ps1 -Stage O-0`

### НЕ входит

- Runnable Drive API / compose (→ **O-1**)
- Workspace UI, docs-engine, co-edit, docx

## 3. E2E-сценарий приёмки

1. Файлы Acceptance / Evidence / Index / Matrix / Stage O-0+O-1 существуют.
2. `.\scripts\run-office-stage-gate.ps1 -Stage O-0`
3. Proto golden / gen stubs test PASS.

## 4. Критерии приёмки

| ID | Критерий | Доказательство | Статус |
|----|----------|----------------|--------|
| F-O0-1 | Acceptance + Evidence + Index + Matrix | files | [x] |
| F-O0-2 | DriveService + request/response messages | `drive.proto` | [x] |
| F-O0-3 | gen-proto lists drive/office/comms | `scripts/gen-proto.ps1` | [x] |
| F-O0-4 | era-proto compiles drive/office | `crates/era-proto/build.rs` | [x] |
| F-O0-5 | Golden wire tests | `go test -C gen/go ./era/v1/` | [x] |
| F-O0-6 | Gate O-0 PASS | `reports/office-stage-O-0-20260730-001327.log` | [x] |

## 5. Backlog (OM0-*)

| ID | Задача | Статус |
|----|--------|--------|
| OM0-1 | Acceptance System + Evidence Rules | [x] |
| OM0-2 | MVP-Spec + Sprint-Index (plan waves) | [x] |
| OM0-3 | Implementation Matrix + Pilot docs | [x] |
| OM0-4 | Stage O-0 / O-1 specs; O-2…O-GA stubs | [x] |
| OM0-5 | DriveService + gen-proto + era-proto + golden | [x] |
| OM0-6 | Gate script O-0 checks + PASS | [x] |
| OM0-7 | Fix products README broken links | [x] |

## 6. Stage Gate

| # | Проверка | Доказательство |
|---|----------|----------------|
| G1 | `.\scripts\run-office-stage-gate.ps1 -Stage O-0` | PASS |
| G2 | (docs-only) optional | — |
| G3 | Matrix updated | O-0 rows |
| G4 | MVP-Spec / Sprint-Index O-0 | `[x]` after proof |
| G5 | — | — |
| G6 | `-WriteSignoff` | `reports/office-stage-O-0-signoff.md` |

## 7. Связано

- [`Office-Sprint-Index.md`](Office-Sprint-Index.md)
- [`Office-Evidence-Rules.md`](Office-Evidence-Rules.md)
- [`Office-Stage-O1-Spec.md`](Office-Stage-O1-Spec.md)
