# ERA Office — UI Menu Map (Collab v2)

**Дата:** 31 июля 2026 г.  
**Baseline:** [Office-UI-Baseline.md](Office-UI-Baseline.md) · [Office-UI-Design-System.md](Office-UI-Design-System.md)  
**OSS delta:** [Office-OSS-Delta.md](Office-OSS-Delta.md)  
**Controls catalog (live + O-FMT):** [Office-UI-Controls-Catalog.md](Office-UI-Controls-Catalog.md)  
**Код:** `ui/office-shell/web/menubar.js` + `menubar.css` · Wave F chrome · **O-FMT** MS-class enrichment

IA копирует структуру Google (File / Edit / View / Insert / Format / Tools / Help; Sheets + Data; Slides + Slide).  
Пункт либо работает, либо `disabled` + tooltip (`Planned` / `ERA+` / `Never`) — без пустых кликов.

**ERA+** = нет (или почти нет) у Google; приоритет суверенного контура (см. OSS-Delta).

---

## Documents

| Menu | Command | Wave | Status |
|------|---------|------|--------|
| File | New, Open, Import, Share, Page setup, Version history, Print, Save | A/B/W2 | live |
| File | **Download** › docx / PDF / ODT / RTF (Google-style flyout) | W2/ERA+/LATER | live |
| Edit | Undo, Redo, Cut, Copy, Paste, Paste plain, Find, Select all | A/B | live |
| Edit | Find and replace | W2 | live |
| Edit | **Format Painter** (MS-class) | O-FMT-1 / O-LITE | live |
| View | Print layout, Word count | A/B | live |
| View | Suggesting mode, Full screen | Planned→live | live |
| View | Line numbers | LATER | live |
| View | **Show formatting marks** (¶); **Ruler** | O-FMT-1 / O-FMT-2 / O-LITE | live |
| Insert | Image, Table, TOC, Bookmark, Drawing; **Headers & footers** ›; **Break** ›; Footnote; Link; Comment | B/W2/ERA+/LATER | live |
| Insert | **Symbol**; **Horizontal line**; Table N×M dialog | O-FMT-1 / O-FMT-2 / O-LITE | live |
| Format | **Text** › · **Paragraph styles** › · **Align & indent** › · **Bullets & numbering** › · Columns · Language | B/W2/LATER/ERA+ | live |
| Format | Justify on toolbar; Indent±; Line/para spacing; List level/marker/restart; Super/Sub; Style gallery | O-FMT-1 / O-LITE | live |
| Tools | Word count, Summarize AI, Rewrite AI | A | live |
| Tools | Spelling (lite) | W2 | live |
| Tools | Word count dialog | O-FMT-2 | live |
| Tools | Review (track changes), Compare, Mail merge lite | LATER | live |
| Help | Shortcuts, About | A | live |

## Tables

| Menu | Command | Wave | Status |
|------|---------|------|--------|
| File | New, Import; **Download** › xlsx / CSV / ODS | A/Planned/ERA+ | live |
| Edit | Find, Fill down/right, Delete values | C | live |
| Edit | **Paste values** (MS-class) | O-FMT-2 | planned |
| View | Freeze, Formula bar | A/C | live |
| View | Freeze panes… | ERA+ | live |
| Insert | **Functions** ›; **Sheet & charts** › (submenu) | C | live |
| Insert | Functions › AVERAGE / MIN / MAX / ROUND | O-FMT-2 | planned |
| Insert | Sparkline | LATER | live |
| Format | Number formats, Bold, Align, Wrap | C | live |
| Format | Merge cells (lite) | Planned→live | live |
| Format | Cell bold/align + wrap/borders on toolbar | O-FMT-2 | planned |
| Data | Sort, Filter | A/C | live |
| Data | Filter options; **Protect** › sheet/ranges; **Analysis** › what-if/… | W2/LATER | live |
| Data | Subtotal | ERA+ | live |
| Help | About | A | live |

## Presentations

| Menu | Command | Wave | Status |
|------|---------|------|--------|
| File | New, Import, Share; **Download** › pptx / ODP; Print; Save | A/ERA+/W2/P-LITE | live |
| File | Print setup (all slides, 1/page) | P-LITE | live |
| Edit | Undo, Redo, Find (incl. notes) | D/W2/P-LITE | live |
| Slide | New, Layouts, Move up/down, Delete | A/D | live |
| Slide | **Duplicate slide** | O-FMT-3 | live |
| Slide | Edit master (persist on deck) | P-LITE | live |
| Slide | Speaker notes (per-slide + present + pptx) | P-LITE | live |
| Slide | Background / theme presets | P-LITE | live |
| Insert | (text via canvas) | A | — |
| Insert | **Image** (URL / Drive) | O-FMT-3 / P-LITE | live |
| Format | Bold / Align / Font ± on title·body | O-FMT-3 | live |
| View | Filmstrip, Present / Slideshow (image+bg) | D/P-LITE | live |
| Help | About | A | live |

## Projects (Wave E)

| Menu | Command | Wave | Status |
|------|---------|------|--------|
| File | New project (.eraj), Refresh, Rename board, Open Drive | E | live |
| Edit | New task, Focus filter | E | live |
| Edit | Labels, Checklist | W2 | live |
| View | Board | E | live |
| View | Swimlanes | LATER | live |
| View | Gantt | — | disabled (Never) |
| Help | About | E | live |

Toolbar: Add task · assignee · due · Drive picker · filter · drag cards.

## Drive (hub)

| Menu / action | Wave | Status |
|---------------|------|--------|
| New folder / doc / sheet / deck, upload | floor | live |
| Full-width layout + left folder tree | target | live |
| Rename, move, share ACL, open with | target | live |
| Search | W2 | live |
| Lock file | ERA+ | live |
| Preview pane | O-LITE Full (side pane, 8MB cap) | live |
| Lock / Unlock | O-LITE Full (locker/owner) | live |

## Chrome (Wave F–G)

| Element | Status |
|---------|--------|
| Icon product nav (все SPA, вкл. Office AI) | live |
| Menubar + icons on every command | live |
| Google Docs–style flyouts: no scrollbars, max-content width, icon column | live |
| Grouped toolbars (primary / secondary) | live |
| Editor title + Share + save pill + presence | live (Docs; Tables/Pres lite) |
| Drive file list + icon row actions | live |
| Shared SVG icon set (`icons.js`) | live |
| ERA+ / Planned disabled honesty | live (ERA+ cmds now mostly live with badge) |
