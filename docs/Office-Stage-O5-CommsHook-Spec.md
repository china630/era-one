# ERA Office — Stage O-5 (Comms → Drive hook)

**Wave:** O-5  
**Предусловие:** O-2 gate = PASS  
**Gap:** GAP-O-P0-24 · **PRD:** AC-O0-5 · **ADR:** [`0027`](adr/0027-era-communications-architecture.md) §4

## 1. Цель

Реализовать `DriveClient` в webmail; optional attach/save via Drive API when `platform-drive` licensed.

## 2. Backlog (OM5-*)

| ID | Задача | Модуль | Статус |
|----|--------|--------|--------|
| OM5-1 | DriveClient HTTP impl | `ui/mail` | [ ] |
| OM5-2 | attachment-link API call | drive-api | [ ] |
| OM5-3 | Comms mail-api optional path | `services/comms/mail` | [ ] |
| OM5-4 | Integration test | ui/mail | [ ] |

## 3. E2E

1. Licensed tenant: attach → Drive link contains `/drive/o/{id}`.
2. Unlicensed: no Drive attach UI.

## 4. Stage Gate

| # | Проверка | Статус |
|---|----------|--------|
| G1 | `run-office-stage-gate.ps1 -Stage O-5` | [ ] |
| G2 | `reports/office-stage-O-5-e2e.log` | [ ] |
| G3 | Matrix updated | [ ] |
| G4 | MVP-Spec O-5 → [x] | [ ] |
| G5 | editions N/A | — |
| G6 | signoff | [ ] |
