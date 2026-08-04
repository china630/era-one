# Office Stage O-MS — Motion, Gantt, chrome, Drive, auth

**Дата:** 2026-08-02  
**Статус:** Implemented (Lite)

## Цель

1. Presentations: Transitions / Animations / Morph **Lite** (не Never) — CSS в Slideshow.
2. Projects: **Gantt Lite** по due date.
3. Comments: плавающий rail справа + кнопка **Comments** рядом с Share (все редакторы).
4. Sticky top (меню/ribbon) + status bar (Docs/Tables/Pres/Projects/Drive).
5. Drive: New-меню продуктов, folder tools, list/grid view, общий chrome.
6. Потеря токена (HTTP 401) → `/login` во всех приложениях.

## Never / границы Lite

| Тема | Lite | Не входит |
|------|------|-----------|
| Morph | CSS scale/blur crossfade | Google Morph engine / shape matching |
| Animations | Appear stagger on Present | Path animations, by-paragraph builds |
| Gantt | Bars from due (−2d…due), no deps | Critical path, resource leveling, MS Project |
| Comments (Projects) | localStorage board notes | Server-synced threads |

## Доказательства

| ID | Доказательство |
|----|----------------|
| PRE-TRANSITION-UI / PRE-ANIM | Motion panel + `ErapSlide.transition/animation`; Present CSS classes |
| PRJ-GANTT | View → Gantt; bars for tasks with due |
| Comments chrome | `#commentsToggleBtn` + `.era-comments-open` |
| Sticky | `era-projects-app` / `era-drive-app` in `office.css` |
| DRV-NEW-MENU / DRV-VIEW | Drive header New menu + list/grid |
| Auth | `EraOfficeShell.handleUnauthorized` → clear token + `/login` |

## Inventory

- PRE-TRANSITION-UI, PRE-ANIM → ✅  
- PRJ-GANTT → ✅ Lite  
- DRV-NEW-MENU, DRV-VIEW → ✅  

## Follow-up

Shell/Drive polish (dropdowns, trash, multi-select, auth harden) → [Office-Stage-OSHELL-Spec.md](Office-Stage-OSHELL-Spec.md).
