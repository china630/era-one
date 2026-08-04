# ERA Office — Stage O-6 (Deploy + ops)

**Wave:** O-6  
**Предусловие:** O-2…O-5 gate = PASS  
**Gap:** GAP-O-P0-30…35 · **PRD:** AC-O0-6, AC-O0-7

## 1. Цель

`docker-compose.office.yml`, Dockerfiles, shared-platform profile, staging smoke, runbook.

## 2. Backlog (OM6-*)

| ID | Задача | Модуль | Статус |
|----|--------|--------|--------|
| OM6-1 | docker-compose.office.yml | deploy | [ ] |
| OM6-2 | Dockerfiles | deploy/dockerfiles | [ ] |
| OM6-3 | shared-platform.yaml profile | deploy/profiles | [ ] |
| OM6-4 | office.yaml status implemented | deploy/profiles | [ ] |
| OM6-5 | run-office-pilot-staging.ps1 | scripts | [ ] |
| OM6-6 | Runbook detail | Office-Pilot-Runbook.md | [ ] |
| OM6-7 | Health probes | compose | [ ] |

## 3. E2E

1. `docker compose config` valid.
2. Staging script RT-O01…05 PASS (when services exist).

## 4. Stage Gate

| # | Проверка | Статус |
|---|----------|--------|
| G1 | `run-office-stage-gate.ps1 -Stage O-6` | [ ] |
| G2 | `reports/office-stage-O-6-e2e.log` | [ ] |
| G3 | Matrix updated | [ ] |
| G4 | MVP-Spec O-6 → [x] | [ ] |
| G5 | office.yaml implemented | [ ] |
| G6 | signoff | [ ] |
