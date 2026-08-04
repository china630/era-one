# ERA Office — Stage O-AUTH (JWT / anti-spoof)

**Wave:** O-AUTH  
**Дата:** 30 июля 2026 г.  
**Статус:** `[x]`  
**PRD:** AC-O3 / AC-O4 AuthZ foundation

## Цель

JWT-only public Drive/docs/tables APIs; service token for engine→Drive; spoof `X-ERA-*` → 401; license fail-closed in production.

## Proof

```powershell
.\scripts\run-office-stage-gate.ps1 -Stage O-AUTH
```
