# ERA Office — Stage P-LITE (Presentations Lite → Full MVP)

**Wave:** P-LITE  
**Дата:** 1 августа 2026 г.  
**Prerequisite:** O-FMT-3, Presentations floor  
**Статус:** `[x]`  
**Inventory:** [Office-UI-Feature-Inventory.md](Office-UI-Feature-Inventory.md)

## Цель

Поднятие Lite-контролов **Presentations** до **Full** в рамках Office MVP Slides-class (не PowerPoint).  
`PRE-ANIM` / transitions: ранее Never; **superseded by** [O-MS](Office-Stage-OMS-Spec.md) (CSS Lite).

## Backlog

### P0 — Share / Print / Image / Present

| ID | Критерий | Inventory | Статус |
|----|----------|-----------|--------|
| PLITE-P0-1 | Share dialog: copy link + Manage ACL in Drive | PRE-SHARE | [x] |
| PLITE-P0-2 | Print all slides, 1/page (+ notes strip) | PRE-PRINT | [x] |
| PLITE-P0-3 | Image in Present + pptx/odp embed; bg in present | PRE-INSERT-IMAGE, PRE-PRESENT | [x] |

### P1 — ODP / Theme / Undo / Master / Notes

| ID | Критерий | Inventory | Статус |
|----|----------|-----------|--------|
| PLITE-P1-1 | ODP thicken: marks, image, solid bg; golden | PRE-ODP | [x] |
| PLITE-P1-2 | Theme picker (solid + gradients); export solid | PRE-THEME-BG | [x] |
| PLITE-P1-3 | Undo + Redo stacks; Ctrl+Y | PRE-UNDO | [x] |
| PLITE-P1-4 | Master fields on `ErapDeck` + new-slide placeholders | PRE-MASTER | [x] |
| PLITE-P1-5 | pptx notes + body2; Drive image picker | PRE-NOTES, PRE-LAYOUTS, PRE-INSERT-IMAGE | [x] |

### P2 — Polish + docs

| ID | Критерий | Inventory | Статус |
|----|----------|-----------|--------|
| PLITE-P2-1 | Find-in-notes; Menu-Map O-FMT-3 → live | PRE-FIND | [x] |
| PLITE-P2-2 | Inventory Notes Full MVP; canvas Lite→Full | — | [x] |

## Вне scope

Animations, free shapes, video, co-edit OT, PowerPoint theme gallery.

## Proof

| Доказательство | Результат |
|----------------|-----------|
| `cargo test -p era-presentations-engine` | PASS — 28 tests (pptx image/notes/bg/body2, ODP image/bg/bold, master fields) |
| `go test ./ui/presentations/...` | PASS |
| Playwright `menubar-presentations.spec.ts` (6 tests, incl. P-LITE share/print/image) | PASS |
| Inventory + Menu-Map + canvas Presentations Lite→Full | updated |
