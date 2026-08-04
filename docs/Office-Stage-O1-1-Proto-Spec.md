# ERA Office — Stage O1-1 (office.proto + erad wire)

**Wave:** O1-1  
**Предусловие:** O-GA gate = PASS  
**Gap:** GAP-O-P1-01 · **PRD:** AC-O2 prep

## 1. Цель

`office.proto`: `DocumentFormat.ERAD/ERAT/ERAP`, `EradDocument`, wire golden. Extend `drive.proto` with `content_format`.

## 2. Backlog (OM11-*)

| ID | Задача | Статус |
|----|--------|--------|
| OM11-1 | office.proto | [ ] |
| OM11-2 | codegen Go + Rust era-proto | [ ] |
| OM11-3 | erad_minimal.golden | [ ] |
| OM11-4 | drive content_format field | [ ] |

## 4. Stage Gate

| G1 | `run-office-stage-gate.ps1 -Stage O1-1` | [ ] |
| G2 | e2e log | [ ] |
| G3 | Matrix | [ ] |
| G4 | MVP-Spec O1-1 | [ ] |
| G5 | N/A | — |
| G6 | signoff | [ ] |
