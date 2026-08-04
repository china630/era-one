# ERA Office — Stage T-LITE (Tables Lite → Full MVP)

**Wave:** T-LITE  
**Дата:** 1 августа 2026 г.  
**Prerequisite:** O-FMT-2, Tables floor  
**Статус:** `[x]`  
**Inventory:** [Office-UI-Feature-Inventory.md](Office-UI-Feature-Inventory.md)

## Цель

Поднятие Lite-контролов **Tables** до **Full** в рамках Office MVP Sheets lite (не Excel parity).

## Backlog

### P0 — Sort + xlsx

| ID | Критерий | Inventory | Статус |
|----|----------|-----------|--------|
| TLITE-P0-1 | Row-aware sort (whole rows) via SheetOp; A↔Z; e2e | TBL-SORT | [x] |
| TLITE-P0-2 | Multi-sheet xlsx import + formulas progressive + golden | TBL-IMPORT-XLSX | [x] |
| TLITE-P0-3 | Multi-sheet xlsx export + formulas/merges lite | TBL-EXPORT-XLSX | [x] |

### P1 — Filter / protect / merge / presence / ODS

| ID | Критерий | Inventory | Статус |
|----|----------|-----------|--------|
| TLITE-P1-1 | Filter + filter-opts persist (col criteria) | TBL-FILTER, TBL-FILTER-OPTS | [x] |
| TLITE-P1-2 | Protect / Unprotect UX; protect-ranges list/remove | TBL-PROTECT, TBL-PROTECT-RANGES | [x] |
| TLITE-P1-3 | Selection merge/unmerge | TBL-MERGE | [x] |
| TLITE-P1-4 | presence_cell relay + peer cell highlight | TBL-PRESENCE | [x] |
| TLITE-P1-5 | ODS thicken export + import_ods + golden | TBL-EXPORT-ODS | [x] |

### P2 — Viz / chrome

| ID | Критерий | Inventory | Статус |
|----|----------|-----------|--------|
| TLITE-P2-1 | Chart meta persist on tab + re-render | TBL-CHART | [x] |
| TLITE-P2-2 | Per-side borders lite + I/O | TBL-WRAP-BORDERS | [x] |
| TLITE-P2-3 | Freeze at selection + clear; Inventory dedupe | TBL-FREEZE-PANES | [x] |

## Вне scope

SPARKLINE, WHATIF, SCENARIOS, CONSOLIDATE, SUBTOTAL, VBA, Pivot.  
MVP sort: relative formula rewrite after row moves not supported (cells move as units).

## Proof

| Доказательство | Результат |
|----------------|-----------|
| `cargo test -p era-tables-engine --lib` | PASS — 36 tests (sort whole rows, multi-sheet xlsx, ODS round-trip, filter/charts persist) |
| `cargo test -p era-tables-engine --test ws_sheet_coedit ws_presence_cell_relay` | PASS |
| Playwright `tables row-aware sort moves sibling columns` | PASS (2026-08-01) |
| Inventory Notes → Full MVP; canvas Tables Lite→Full | updated |
