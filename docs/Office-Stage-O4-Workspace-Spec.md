# ERA Office — Stage O-4 (Workspace + Drive UI)

**Wave:** O-4  
**Предусловие:** O-2 and O-3 gate = PASS  
**Gap:** GAP-O-P0-20, GAP-O-P0-22 · **PRD:** AC-O0-3

## 1. Цель

Workspace BFF (`app.customer.local`), `/drive` SPA, OIDC session, Playwright e2e.

## 2. Backlog (OM4-*)

| ID | Задача | Модуль | Статус |
|----|--------|--------|--------|
| OM4-1 | workspace BFF | `cmd/workspace` | [ ] |
| OM4-2 | OIDC middleware | workspace | [ ] |
| OM4-3 | ui/drive SPA | `ui/drive` | [ ] |
| OM4-4 | /docs stub route | workspace | [ ] |
| OM4-5 | Playwright e2e | `ui/drive/e2e` | [ ] |
| OM4-6 | Proxy /mail to ui-mail | workspace | [ ] |

## 3. E2E

1. OIDC login → redirect `/drive`.
2. Upload file → visible in list.
3. Create folder → list children.

## 4. Stage Gate

| # | Проверка | Статус |
|---|----------|--------|
| G1 | `run-office-stage-gate.ps1 -Stage O-4` | [ ] |
| G2 | `reports/office-stage-O-4-e2e.log` | [ ] |
| G3 | Matrix updated | [ ] |
| G4 | MVP-Spec O-4 → [x] | [ ] |
| G5 | editions N/A | — |
| G6 | signoff | [ ] |
