# ERA Office — Stage O-FMT-2 (Tables Excel-class + Docs polish)

**Wave:** O-FMT-2  
**Дата:** 31 июля 2026 г.  
**Prerequisite:** O-FMT-1  
**Статус:** `[x]` (gate PASS — `reports/office-stage-O-FMT-2-*`)

## Цель

Tables: AVERAGE/MIN/MAX/ROUND, cell bold/align, wrap/borders, paste values.  
Docs polish: ruler, table N×M dialog, word-count dialog.

## Backlog

| ID | Критерий | Inventory | Статус |
|----|----------|-----------|--------|
| OM-FMT2-1 | Calc + UI: AVERAGE, MIN, MAX, ROUND | TBL-AVG-MIN-MAX-ROUND | [x] |
| OM-FMT2-2 | Cell bold/align + wrap/borders lite | TBL-CELL-BOLD-ALIGN, TBL-WRAP-BORDERS | [x] |
| OM-FMT2-3 | Paste values | TBL-PASTE-VALUES | [x] |
| OM-FMT2-4 | Docs ruler + table dialog + wordcount dlg | DOC-RULER, DOC-TABLE-DIALOG, DOC-WORDCOUNT-DLG | [x] |
| OM-FMT2-5 | Tests + Inventory ✅ | — | [x] |

## Gate

```powershell
.\scripts\run-office-stage-gate.ps1 -Stage O-FMT-2 -WriteSignoff
```
