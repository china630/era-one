# Office Stage O-STUB — Stub → Lite

**Дата:** 2026-08-02  
**Статус:** `[x]` Implemented

## Цель

Поднятие Stub-контролов с audit canvas до честного **Lite** (не Word/Excel Full).

## Docs P0

| ID | Lite bar | Evidence |
|----|----------|----------|
| DOC-SECTION | `section_break` via `insert_block` + sync | `era_plus.js` → `insertBlockAfter` |
| DOC-TEXTBOX | Bordered block, optional width%, sendOp | `later.js` |
| DOC-COLUMNS | Dialog 1–3, persist `page.columns` | `columnsDlg` + `applyPageChrome` |
| DOC-MANAGE-STYLES | CRUD named styles, gallery `custom:`, apply marks | `stylesDlg` + `eraSyncCustomStylesGallery` |
| DOC-FOOTNOTE | Insert `[fnN]` + footnote block, click jump | `era_plus.js` + `wireBlocksSurface` |

## Tables P2

| ID | Lite bar | Evidence |
|----|----------|----------|
| TBL-SPARKLINE | SVG + persist `charts[{chart_type:sparkline}]` | `renderLiteSparkline` / `restoreCharts` |
| TBL-WHATIF | Preview / Apply Goal Seek | existing dialog (kept) |
| TBL-SCENARIOS | Persist on tab via `set_scenarios` | `SheetTab.scenarios` + sync op |
| TBL-CONSOLIDATE | Sum from active / name / index sheet | `cellsMapForSheetRef` |
| TBL-SUBTOTAL | Group-by left column; SUM rows below data | `insertSubtotalLite` |

## Вне scope

- Word section page-setup, floating drawings, Excel SUBTOTAL API / Scenario Manager.
- DOC-REVIEW / COMPARE / MERGE remain Stub on canvas (not in this wave).
