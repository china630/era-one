# ERA Office — OSS capability delta (vs Google Workspace)

**Дата:** 31 июля 2026 г.  
**Статус:** Working draft (Wave F)  
**Связано:** [Office-UI-Menu-Map.md](Office-UI-Menu-Map.md) · [Office-UI-Baseline.md](Office-UI-Baseline.md) · [Office-UI-Feature-Inventory.md](Office-UI-Feature-Inventory.md)

## Назначение

Зафиксировать **функции open-source офисов** (LibreOffice/Collabora, OnlyOffice *как UX*, CryptPad, Nextcloud Files/Deck), которых **нет или почти нет у Google Docs/Sheets/Slides/Drive**, чтобы:

1. заранее зарезервировать пункты меню (`disabled` + честный label);
2. пометить дифференциатор суверенного контура как **`ERA+`** (не «догоняем Google»);
3. не тащить GPL runtime (запрет baseline) — только **идеи capabilities**.

Легенда статуса в ERA: `live` · `W2` · `LATER` · `NEVER` · `ERA+` (= приоритетный W2/LATER, которого нет у Google).

---

## Documents

| Capability | Google Docs | OSS source | ERA menu slot | ERA |
|------------|-------------|------------|---------------|-----|
| Named style set / manage styles | weak | LO Writer | Format → Styles → Manage… | ERA+ live |
| Section breaks | no | LO | Insert → Break → Section | ERA+ live |
| Footnotes / endnotes | no (approx via add-ons) | LO | Insert → Footnote | ERA+ live |
| Bookmarks / cross-ref | weak | LO | Insert → Bookmark | W2 |
| Track changes accept/reject | Suggesting (cloud) | LO | Tools → Review | LATER live |
| Compare documents | no | LO | Tools → Compare | LATER live |
| Mail merge / fields (no VBA) | no | LO | Tools → Mail merge lite | LATER live |
| Export ODT | no | LO | File → Download → ODT | ERA+ thicker lite (marks/tables/breaks) |
| Export RTF | no | LO | File → Download → RTF | LATER live |
| Hyphenation / lang per para | weak | LO | Format → Language | W2 |
| Line numbering | no | LO | View → Line numbers | LATER live |
| Text frames / boxes | Drawing weak | LO | Insert → Text box | LATER live |
| Columns (section) | limited | LO | Format → Columns | LATER live |

---

## Tables

| Capability | Google Sheets | OSS source | ERA menu slot | ERA |
|------------|---------------|------------|---------------|-----|
| Export ODS | no | LO Calc | File → Download → ODS | ERA+ thicker lite (sheets/formulas/merges) |
| Freeze arbitrary panes | limited | LO | View → Freeze panes… | ERA+ live (per-tab rows/cols) |
| Subtotals | no | LO | Data → Subtotal | ERA+ live (Subtotal lite =SUM) |
| Advanced AutoFilter | weak | LO | Data → Filter options | W2 live (lite) |
| Protect sheet lite | limited | LO | Data → Protect | W2 live |
| Protect ranges (offline) | limited | LO | Data → Protect ranges | ERA+ |
| Goal Seek / Solver | add-on | LO | Data → What-if | LATER live (preview + seek) |
| Scenarios | no | LO | Data → Scenarios | LATER live |
| Consolidate | no | LO | Data → Consolidate | LATER live |
| Sparklines | no | OO/Excel-like | Insert → Sparkline | LATER live |
| Database range | no | LO | Data → Define range | LATER |

**Не перенимать из Google:** IMPORTRANGE / QUERY cloud / Apps Script → `NEVER`.

---

## Presentations

| Capability | Google Slides | OSS source | ERA menu slot | ERA |
|------------|---------------|------------|---------------|-----|
| Editable master layouts | weak | LO Impress | Slide → Edit master | LATER live |
| Export ODP | no | LO | File → Download → ODP | ERA+ thicker lite (notes/cols) |
| Handout / notes print | weak | LO | File → Print setup | W2 |
| Rich animation timeline | yes (cloud) | LO | — | NEVER (product choice) |

---

## Drive / Files

| Capability | Google Drive | OSS source | ERA slot | ERA |
|------------|--------------|------------|----------|-----|
| Explicit file lock | weak | NC / LO | File → Lock | ERA+ live |
| WebDAV / sync protocol | proprietary | NC | Offline client | LATER |
| Per-folder retention | limited | NC | Admin | out of SPA |
| Integrity / checksum badge | no | some DMS | Details | W2 |

---

## Projects

| Capability | Google Tasks | OSS source | ERA slot | ERA |
|------------|--------------|------------|----------|-----|
| Labels / tags | weak | NC Deck / WeKan | Edit → Labels | W2 |
| Checklist on card | weak | Deck | Edit → Checklist | W2 |
| Swimlanes | no | WeKan | View → Swimlanes | LATER live |
| Gantt | no (separate) | some | View → Gantt | NEVER |

---

## MS-class enrichment (not LO-only)

Помимо LO/NC дельты, программа **O-FMT** ([Controls Catalog](Office-UI-Controls-Catalog.md)) берёт у **MS Word/Excel/PowerPoint** реалистичный subset:

| Product | MS-class (O-FMT) | Wave |
|---------|------------------|------|
| Documents | Format Painter, indent/spacing, list level/marker/restart, super/sub, style gallery, ¶ marks, symbol/HR | O-FMT-1 |
| Documents polish | Ruler, table N×M, word-count dialog | O-FMT-2 |
| Tables | AVERAGE/MIN/MAX/ROUND, cell bold/align, wrap/borders, paste values | O-FMT-2 |
| Presentations | Duplicate slide, Format text/font±, Insert image | O-FMT-3 |

IA остаётся Google-menubar; MS = capability source, **не** Ribbon clone.

## Правило меню

- Пункт **ERA+** в UI: `disabled` + hint `ERA+` (не путать с Google Planned).
- Реализация только после engine AC + golden; до этого — только IA-резерв.
- Запрещено обещать VBA, LO headless, cloud marketplace.
