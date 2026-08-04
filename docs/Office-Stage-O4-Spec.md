# ERA Office — Stage O-4 (Co-edit)

**Wave:** O-4  
**Версия:** 1.1  
**Дата:** 30 июля 2026 г.  
**Продукт:** ERA Documents  
**PRD:** [`PRD-Office-MVP.md`](products/PRD-Office-MVP.md) · AC-O1  
**Предусловие:** Wave **O-3** gate = PASS  
**Статус:** `[x]` — gate PASS `reports/office-stage-O-4-20260730-003207.log`

---

## 1. Цель этапа

> Live co-edit ≥2 WebSocket clients на одном `.erad`: server OpLog + fan-out ops peers.

Docx golden / SBOM → **O-5**.

## 2. Scope

### Входит

- Sync room hub + broadcast (кроме отправителя)
- `ws_coedit` peer-delivery + merge asserts
- ui/docs apply remote ops
- Gate O-4 Required

### НЕ входит

- Yjs/Automerge crate
- Playwright hard gate
- docx/SBOM

## 3. E2E-сценарий приёмки

1. `cargo test -p era-docs-engine --test ws_coedit --quiet`
2. `cargo test -p era-docs-engine sync --quiet`
3. `go test -C ui/docs ./... -count=1`
4. `.\scripts\run-office-stage-gate.ps1 -Stage O-4 -WriteSignoff`

## 4. Критерии приёмки

| ID | Критерий | PRD | Доказательство | Статус |
|----|----------|-----|----------------|--------|
| F-O4-1 | 2 WS clients merge | AC-O1 | ws_coedit | [x] |
| F-O4-2 | Peer live fan-out | AC-O1 | ws_coedit peer assert | [x] |
| F-O4-3 | sync unit | — | cargo sync | [x] |
| F-O4-4 | UI remote apply | AC-O1 | go test ui/docs | [x] |
| F-O4-5 | Gate O-4 PASS | — | `reports/office-stage-O-4-20260730-003207.log` | [x] |

## 5. Backlog (OM4-*)

| ID | Задача | Статус |
|----|--------|--------|
| OM4-1 | Stage O-4 Spec | [x] |
| OM4-2 | WS room hub + fan-out | [x] |
| OM4-3 | Strengthen ws_coedit | [x] |
| OM4-4 | sync unit tests | [x] |
| OM4-5 | ui/docs remote ops | [x] |
| OM4-6 | Workspace WS proxy check | [x] |
| OM4-7 | Gate Required | [x] |
| OM4-8 | Matrix / Index / MVP-Spec | [x] |

## 6. Stage Gate

| # | Проверка | Доказательство |
|---|----------|----------------|
| G1 | `run-office-stage-gate.ps1 -Stage O-4` | PASS |
| G3 | Matrix AC-O1 Scaffold | ✅ |
| G4 | Sprint-Index O-4 `[x]` | docs |
| G6 | signoff | `reports/office-stage-O-4-signoff.md` |

## 7. Связано

- Legacy: [`Office-Stage-O1-3-Sync-Spec.md`](Office-Stage-O1-3-Sync-Spec.md)
- [`Office-Sprint-Index.md`](Office-Sprint-Index.md)
