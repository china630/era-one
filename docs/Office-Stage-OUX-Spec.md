# Office Stage O-UX — Google Docs/Sheets/Slides chrome

**Дата:** 2026-08-02  
**Статус:** `[x]` Implemented (lab)

## Цель

Подтянуть UX chrome Docs / Tables / Presentations к паттернам Google Docs / Sheets / Slides.  
Transition honesty panel (OUX-8) **superseded by** [O-MS](Office-Stage-OMS-Spec.md) live Motion Lite.

## AC

| ID | Критерий | Evidence |
|----|----------|----------|
| OUX-1 | После применения mark/font выделение текста сохраняется | `selectRangeInBlock` / `restoreMultiSelection`; toolbar `mousedown` preventDefault |
| OUX-2 | Можно выделить несколько абзацев подряд | `#blocks` single contenteditable surface + `collectFormatTargets` |
| OUX-3 | Sticky chrome: скролл только ниже ribbon | `body.era-docs-app` flex + canvas `overflow:auto` |
| OUX-4 | Страница A4 + горизонтальная и вертикальная линейка | `.era-doc-sheet-frame`, `applyPageChrome` sizes |
| OUX-5 | Align/lists — button menus; selects как Headings; tooltips; font −/+ | `toolbar-chrome.js` + docs/tables HTML |
| OUX-6 | Tables: tabs/`+` внизу; горизонтальный скролл + wheel | bottom `#sheetTabsBar`; `#gridWrap` wheel |
| OUX-7 | Tables/Pres comments работают (notes) | `Cell.note` + Notes rail; Pres comment → speaker notes rail |
| OUX-8 | Pres: Background / Layout / Theme / Transition chrome | dialogs + Theme panel; Transition → O-MS Lite |

## Вне scope

- Full Morph / path animations (O-MS = CSS Lite only)
- Word-class comment threads
