# ERA Office — Stage O-T-0 (Tables proto SSOT)

**Wave:** O-T-0  
**Дата:** 30 июля 2026 г.  
**PRD:** AC-T prep · [`PRD-Office-P2.md`](products/PRD-Office-P2.md)  
**Статус:** `[x]` (gate O-T-0 PASS)

## Цель

Acceptance + `EratSheet` proto SSOT + golden wire (mirror O-0).

## Критерии

| ID | Критерий | Proof |
|----|----------|-------|
| F-OT0-1 | Spec + ERA-Tables-vs-Excel | docs |
| F-OT0-2 | EratSheet in office.proto | proto |
| F-OT0-3 | Wire golden | `go test` EratSheet |
| F-OT0-4 | Gate O-T-0 PASS | reports log |

## Gate

`pwsh -NoProfile -File ./scripts/run-office-stage-gate.ps1 -Stage O-T-0 -WriteSignoff`
