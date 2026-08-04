# ERA Office — Stage O-FMT-0 (UI Controls Catalog canon)

**Wave:** O-FMT-0  
**Дата:** 31 июля 2026 г.  
**Статус:** `[x]` (gate O-FMT-0 PASS)  
**Связано:** [Office-UI-Controls-Catalog.md](Office-UI-Controls-Catalog.md) · [Office-Sprint-Index.md](Office-Sprint-Index.md) §2

## Цель

Зафиксировать канон UI-контролов и программу MS-class enrichment (O-FMT-1…3) без изменения runtime-кода.

## Backlog

| ID | Критерий | Статус |
|----|----------|--------|
| OM-FMT0-1 | Controls Catalog существует | [x] |
| OM-FMT0-2 | Menu-Map / Feature-Inventory / Baseline / OSS-Delta обновлены | [x] |
| OM-FMT0-3 | Sprint-Index §2 + Roadmap A4-FMT | [x] |
| OM-FMT0-4 | Stage specs OFMT1–3 + gate ValidateSet | [x] |
| OM-FMT0-5 | Gate O-FMT-0 PASS | [x] |

## Gate

```powershell
.\scripts\run-office-stage-gate.ps1 -Stage O-FMT-0 -WriteSignoff
```

Проверки: Catalog, Inventory IDs (`DOC-FORMAT-PAINTER`, `TBL-AVG-MIN-MAX-ROUND`, `PRE-DUP-SLIDE`), Sprint-Index `O-FMT-0`, specs OFMT1–3.
