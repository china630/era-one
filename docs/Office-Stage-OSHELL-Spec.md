# Office Stage O-SHELL — shell polish, Drive trash, auth harden

**Дата:** 2026-08-02  
**Статус:** Implemented (Lite)  
**Предшественник:** [O-MS](Office-Stage-OMS-Spec.md)

## Цель

Исправить UX/консистентность из critique после O-UX / O-MS:

1. **Dropdowns** — native `<select>` и toolbar flyouts не ломаются (`mousedown` + overflow/z-index).
2. **Auth** — `authFetch` + JWT `exp` client check + session watch → `/login`.
3. **Comments chrome** — единый rail (head + Close) во всех редакторах.
4. **Ribbon canon** — font stepper icons, Find = field→button, Docs без мёртвых H1–H3, list-level ≠ indent icons.
5. **Tables scroll** — sheet pane row layout; Shift/Ctrl + wheel → horizontal.
6. **Product icons** — distinctive nav marks + accent colors.
7. **Drive** — New в ribbon-left; multi-select (Shift/Ctrl); Trash + Restore API; bulk Move/Trash; sort.

## Never / границы Lite

| Тема | Lite | Не входит |
|------|------|-----------|
| Trash | Soft-delete + Restore list | Permanent purge UI, retention policy |
| Multi-select | Shift range + Ctrl toggle + selection bar | Drag-marquee, keyboard grid select |
| Comments | Unified chrome; semantics per product unchanged | Unified comment server model |
| Auth | Client JWT exp + 401 redirect | Silent refresh / refresh tokens |

## Доказательства

| ID | Доказательство |
|----|----------------|
| SHELL-SELECT | `toolbar-chrome.js` skips `preventDefault` on `select`/`input` |
| SHELL-FLYOUT | `office.css` / `menubar.css` overflow visible + z-index on chrome |
| SHELL-AUTH | `EraOfficeShell.authFetch` / `isTokenExpired` / `wireSessionWatch`; apps use `officeFetch` |
| SHELL-COMMENTS | `#commentsPanel` + `#commentsCloseBtn` + `wireCommentsToggle` |
| SHELL-ICONS | `icons.js` `nav*` + `fontInc`/`fontDec`/`listLevel*` |
| TBL-SCROLL | `.era-sheet-pane` + Shift/Ctrl wheel X |
| DRV-TRASH | Migration `010_drive_trash.sql`; `GET /trash`; `POST …/trash|restore` |
| DRV-MULTI | Checkboxes + Shift-range; selection bar Move/Trash |
| DRV-NEW-MENU | New control in toolbar-left (not header-right) |

## Inventory

- SHELL-* / DRV-TRASH / DRV-MULTI / DRV-SORT → ✅ (см. Feature-Inventory)
- TBL-SCROLL notes updated (Ctrl+wheel)
