# ERA Office — Stage O-2 (ERA Drive service)

**Wave:** O-2  
**Предусловие:** O-1 gate = PASS  
**Gap:** GAP-O-P0-02…04 · **PRD:** AC-O0-1, AC-O0-2

## 1. Цель

`platform/drive` + `cmd/drive-api`: upload, folders, ACL, versions, attachment-link API.

## 2. Backlog (OM2-*)

| ID | Задача | Модуль | Статус |
|----|--------|--------|--------|
| OM2-1 | Drive store + MinIO + Postgres | `services/platform/drive` | [x] |
| OM2-2 | REST API | `cmd/drive-api` | [x] |
| OM2-3 | ACL tenant isolation | drive package | [x] |
| OM2-4 | Versions API | drive package | [x] |
| OM2-5 | Integration tests | `pgstore_integration_test.go` | [x] |
| OM2-6 | Dockerfile drive-api | `deploy/dockerfiles/` | [x] |

## 3. E2E

1. Upload file → list → download — same tenant PASS.
2. Cross-tenant access → 403.
3. Restart drive-api → object still in MinIO + PG.

## 4. Stage Gate

| # | Проверка | Статус |
|---|----------|--------|
| G1 | `run-office-stage-gate.ps1 -Stage O-2` | [ ] |
| G2 | `reports/office-stage-O-2-e2e.log` | [ ] |
| G3 | Matrix updated | [ ] |
| G4 | MVP-Spec O-2 → [x] | [ ] |
| G5 | editions N/A | — |
| G6 | signoff | [ ] |
