# ERA Office — Stage O-LITE (Drive + Documents Lite → Full MVP)

**Wave:** O-LITE  
**Дата:** 1 августа 2026 г.  
**Prerequisite:** O-FMT-1/2, Drive floor  
**Статус:** `[x]` (2026-08-01)  
**Inventory:** [Office-UI-Feature-Inventory.md](Office-UI-Feature-Inventory.md)  
**Catalog:** [Office-UI-Controls-Catalog.md](Office-UI-Controls-Catalog.md)

## Цель

Поднятие Lite-контролов **Drive** и **Documents** до **Full** в рамках Office MVP (не MS/Word parity).

## Уровни

| Уровень | Значение |
|---------|----------|
| Full | E2E в Office MVP + sync/export где уместно + тест/доказательство |
| Lite | тонкая реализация (до волны) |

## Backlog

### P0 — Honesty + collab + Drive

| ID | Критерий | Inventory | Статус |
|----|----------|-----------|--------|
| OLITE-P0-1 | DRV-PREVIEW: side pane, 8MB cap, image by CT\|ext, e2e text/image/unsupported | DRV-PREVIEW | [x] |
| OLITE-P0-2 | DRV-LOCK: Unlock only locker/owner; Open-with warn; e2e lock-by-other | DRV-LOCK | [x] |
| OLITE-P0-3 | DOC-PDF-PRINT: menu «Print / Save as PDF» (browser only) | DOC-PDF-PRINT | [x] |
| OLITE-P0-4 | DOC-SHARE: Copy link · ACL in Drive + deep-link | DOC-SHARE | [x] |
| OLITE-P0-5 | DOC-PEER-CARET: server relays `presence_caret` | DOC-PEER-CARET | [x] |
| OLITE-P0-6 | DOC-PRESENCE: peer colors on roster | DOC-PRESENCE | [x] |
| OLITE-P0-7 | Format Painter / Symbol / Color → `sendOp` | DOC-FORMAT-PAINTER, DOC-SYMBOL, DOC-COLOR | [x] |
| OLITE-P0-8 | Clipboard: plain paste into selection spans | DOC-CLIPBOARD | [x] |

### P1 — Insert + export

| ID | Критерий | Inventory | Статус |
|----|----------|-----------|--------|
| OLITE-P1-1 | Image: Drive picker / upload → `image_url` or object ref | DOC-IMAGE | [x] |
| OLITE-P1-2 | Table cells model + row/col insert lite | DOC-TABLE, DOC-TABLE-DIALOG | [x] |
| OLITE-P1-3 | TOC refresh + jump; bookmark navigate; selection links | DOC-TOC, DOC-BOOKMARK, DOC-LINK | [x] |
| OLITE-P1-4 | Comments: `start`/`end` offset + quote in rail | DOC-COMMENTS | [x] |
| OLITE-P1-5 | HR block_type + H/F on-page strip | DOC-HR, DOC-HEADER-FOOTER | [x] |
| OLITE-P1-6 | ODT/RTF thicken + golden | DOC-ODT, DOC-RTF | [x] |

### P2 — Chrome / spell / suggest

| ID | Критерий | Inventory | Статус |
|----|----------|-----------|--------|
| OLITE-P2-1 | Ruler margins + first-line; show-marks soft-break; line-num honesty | DOC-RULER, DOC-SHOW-MARKS, DOC-LINE-NUM | [x] |
| OLITE-P2-2 | List marker gallery + restart visible | DOC-LIST-MARKER, DOC-LIST-RESTART | [x] |
| OLITE-P2-3 | Style gallery named map | DOC-STYLES, DOC-STYLE-GALLERY | [x] |
| OLITE-P2-4 | Lang → spell wordlist (air-gap) | DOC-LANG, DOC-SPELL | [x] |
| OLITE-P2-5 | Suggesting: shared revisions accept/reject ops | DOC-SUGGEST | [x] |

## Proof

- `cargo test -p era-docs-engine --lib convert::` — 10 PASS (incl. ODT/RTF O-LITE golden) — 2026-08-01
- `cargo test -p era-docs-engine --test ws_coedit ws_presence_caret_relay` — PASS
- Playwright additions: `ui/office/e2e/drive.spec.ts` (preview side pane, image/unsupported, lock-by-other)
- Spec + Inventory + Menu-Map + canvas audit bumped

## Gate

```powershell
cargo test -p era-docs-engine
# Playwright: ui/office/e2e/drive.spec.ts (+ docs smoke if present)
```

## Вне scope

Server PDF, Word Track Changes parity, HTML paste from Office, PDF Drive preview, Stub DOC-* (section/footnote/textbox/columns/review/compare/merge), DRV-COPY-LINK / DRV-SORT.
