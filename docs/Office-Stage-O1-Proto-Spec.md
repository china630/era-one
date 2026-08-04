# ERA Office — Stage O-1 (Proto + schema)

**Wave:** O-1  
**Предусловие:** O-0 gate = PASS  
**Gap:** GAP-O-P0-01 · **PRD:** AC-O0-1 prep

## 1. Цель

Contract-first: `drive.proto`, Postgres migration, ACL golden wire test.

## 2. Backlog (OM1-*)

| ID | Задача | Модуль | Статус |
|----|--------|--------|--------|
| OM1-1 | drive.proto | `proto/era/v1/drive.proto` | [ ] |
| OM1-2 | Go codegen | `gen/go/era/v1/` | [ ] |
| OM1-3 | Migration 001_drive | `deploy/postgres/migrations/platform/` | [ ] |
| OM1-4 | ACL golden | `testdata/drive_acl.golden.json` | [ ] |

## 3. E2E

1. `protoc` / gen-proto → `drive.pb.go` exists.
2. Golden ACL serialize roundtrip PASS.

## 4. Stage Gate

| # | Проверка | Статус |
|---|----------|--------|
| G1 | `run-office-stage-gate.ps1 -Stage O-1` | [ ] |
| G2 | `reports/office-stage-O-1-e2e.log` | [ ] |
| G3 | Matrix updated | [ ] |
| G4 | MVP-Spec O-1 → [x] | [ ] |
| G5 | editions N/A | — |
| G6 | signoff | [ ] |
