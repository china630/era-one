# ERA Office — Stage O-FMT-3 (Presentations PPT-class)

**Wave:** O-FMT-3  
**Дата:** 31 июля 2026 г.  
**Prerequisite:** O-FMT-2  
**Статус:** `[x]` (gate PASS — `reports/office-stage-O-FMT-3-*`; rich-text follow-on `[~]` below)

## Цель

Presentations: duplicate slide; Format bold/align/font± on title·body; Insert image.

## Backlog

| ID | Критерий | Inventory | Статус |
|----|----------|-----------|--------|
| OM-FMT3-1 | Duplicate slide (menu + toolbar) | PRE-DUP-SLIDE | [x] |
| OM-FMT3-2 | Format menu: bold / align / font step | PRE-TEXT-FORMAT, PRE-FONT-STEP | [x] |
| OM-FMT3-3 | Insert image (URL) | PRE-INSERT-IMAGE | [x] |
| OM-FMT3-4 | Tests + Inventory ✅ | — | [x] |
| OM-PRES-RT-1 | Shared `era-office-richtext` + `TextFrame` on slides; legacy string serde | PRE-TEXT-FORMAT | [x] |
| OM-PRES-RT-2 | `POST …/frame-op` Split/Merge/SetBlockFormat; Enter/Backspace UI | — | [x] |
| OM-PRES-RT-3 | Selection `set_marks_range` + typing style; no field-level chrome | PRE-TEXT-FORMAT | [x] |
| OM-PRES-RT-4 | Proto TextFrame; pptx runs/paragraphs; `richtext-editor.js`; version/409 | — | [x] |

## Proof (rich-text)

- `cargo test -p era-office-richtext` — PASS
- `cargo test -p era-presentations-engine` — PASS (incl. `legacy_string_title_body_deserializes`, pptx/odp)
- Shared UI: `ui/office-shell/web/richtext-editor.js`

## Gate

```powershell
.\scripts\run-office-stage-gate.ps1 -Stage O-FMT-3 -WriteSignoff
```
