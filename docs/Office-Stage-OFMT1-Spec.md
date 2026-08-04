# ERA Office — Stage O-FMT-1 (Documents MS-class formatting)

**Wave:** O-FMT-1  
**Дата:** 31 июля 2026 г.  
**Prerequisite:** O-FMT-0  
**Статус:** `[~]` (rich-text A–C: Enter/split, selection marks, span OT — proof below; OM-FMT1-5 go inventory TBD)  
**Catalog:** [Office-UI-Controls-Catalog.md](Office-UI-Controls-Catalog.md)

## Цель

Docs: абзац/стили/списки/Format Painter/super-sub — MS Word-class capabilities на Google-menubar IA.

## Backlog

| ID | Критерий | Inventory | Статус |
|----|----------|-----------|--------|
| OM-FMT1-1 | Model: `space_before/after_pt`, `list_level`, `list_marker`, restart, super/sub | DOC-PARA-SPACING, DOC-LIST-*, DOC-SUPER-SUB | [x] |
| OM-FMT1-2 | Sync ops + co-edit roundtrip | — | [x] (span-preserving insert/delete; WS JSON ops) |
| OM-FMT1-3 | docx import/export subset + golden | — | [x] (existing golden; multi-span survives edits) |
| OM-FMT1-4 | UI toolbar/menu: justify, indent, spacing, lists, painter, styles, ¶, symbol, HR | DOC-JUSTIFY-TB … DOC-HR | [x] |
| OM-FMT1-5 | `go test -C ui/docs` + Inventory ✅ | — | [ ] |
| OM-FMT1-6 | Enter = `split_block` / Backspace merge / Shift+Enter soft break | — | [x] (unit: `split_block_mid_multi_span`, `merge_blocks_coalesces_runs`) |
| OM-FMT1-7 | Selection-scoped `set_marks_range` + typing style; no flatten on input | — | [x] (unit: `set_marks_range_preserves_neighbors`, `insert_text_preserves_bold_neighbor`) |
| OM-FMT1-8 | OT transform for marks/split + proto DocOpType extended | — | [x] |

## Proof

- `cargo test -p era-docs-engine sync_` — 16 PASS; `span_` — 4 PASS; `proto_roundtrip` — PASS (2026-08-01)
- golden docx with lists/spacing (existing corpus)
- `go test -C ui/docs ./...` — PASS

## Gate

```powershell
.\scripts\run-office-stage-gate.ps1 -Stage O-FMT-1 -WriteSignoff
```
