# ERA Office — Stage O-3 (Identity + license)

**Wave:** O-3  
**Предусловие:** O-1 gate = PASS (parallel with O-2 after O-1)  
**Gap:** GAP-O-P0-10, GAP-O-P0-11 · **PRD:** AC-O0-4

## 1. Цель

`platform-drive` license module; tenant Postgres store; admin-portal in compose.

## 2. Backlog (OM3-*)

| ID | Задача | Модуль | Статус |
|----|--------|--------|--------|
| OM3-1 | ModulePlatformDrive | `licensegate/gate.go` | [x] |
| OM3-2 | edition_matrix_test | licensegate | [x] |
| OM3-3 | tenant Postgres store | `platform/tenant/pgstore` | [x] |
| OM3-4 | admin-portal compose + tenants API | deploy + adminportal | [x] |
| OM3-5 | editions-shared exists:true | `editions-shared.yaml` | [x] |

## 3. E2E

1. Request drive-api without `platform-drive` → 403.
2. Dev license with module → 200.

## 4. Stage Gate

| # | Проверка | Статус |
|---|----------|--------|
| G1 | `run-office-stage-gate.ps1 -Stage O-3` | [ ] |
| G2 | `reports/office-stage-O-3-e2e.log` | [ ] |
| G3 | Matrix updated | [ ] |
| G4 | MVP-Spec O-3 → [x] | [ ] |
| G5 | editions-shared updated | [ ] |
| G6 | signoff | [ ] |
