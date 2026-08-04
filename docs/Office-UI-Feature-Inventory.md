# ERA Office — UI Feature Inventory

**Дата:** 2 августа 2026 г.  
**Baseline:** [Office-UI-Baseline.md](Office-UI-Baseline.md) (Collab v2)  
**Menu map:** [Office-UI-Menu-Map.md](Office-UI-Menu-Map.md)  
**Controls catalog:** [Office-UI-Controls-Catalog.md](Office-UI-Controls-Catalog.md) (O-FMT)  
**Легенда статуса:** `✅` есть · `🟡` partial · `❌` нет · `⏸` later · `[ ]` planned (O-FMT) · `🚫` never

Колонка **Tier:** `floor` (TE) · `target` (Collab v2) · `later` · `never` · `fmt` (O-FMT wave)

---

## Drive

| ID | Контрол / действие | Tier | Status | Notes |
|----|-------------------|------|--------|-------|
| DRV-LOGIN | Sign in / out | floor | ✅ | staging token |
| DRV-FOLDER-CREATE | Create folder | floor | ✅ | |
| DRV-FOLDER-NAV | Breadcrumb / open folder | floor | ✅ | |
| DRV-FOLDER-TREE | Left folder tree | target | ✅ | expand/collapse + navigate; sync with listing |
| DRV-FLUID | Full-width layout | target | ✅ | `.era-main-fluid` + `.era-drive-layout` |
| DRV-UPLOAD | Upload into folder | floor | ✅ | |
| DRV-DOWNLOAD | Download | floor | ✅ | |
| DRV-VERSIONS | Versions panel | floor | ✅ | |
| DRV-NEW-DOC | New document | floor | ✅ | unique name |
| DRV-NEW-SHEET | New sheet | floor | ✅ | |
| DRV-NEW-DECK | New presentation | floor | ✅ | |
| DRV-NEW-MENU | New dropdown (product picker) | target | ✅ | O-SHELL: ribbon-left New (Document/Sheet/Pres/Project/Folder) |
| DRV-VIEW | List / grid view toggle | target | ✅ | O-MS: persist `era_drive_view`; O-SHELL grid polish |
| DRV-SORT | Sort listing | target | ✅ | O-SHELL: name / modified / type |
| DRV-MULTI | Multi-select | target | ✅ | O-SHELL: checkbox + Shift-range + Ctrl; selection bar |
| DRV-TRASH | Trash / Restore | target | ✅ | O-SHELL: soft-delete API + Trash view |
| DRV-OPEN-WITH | Open in Docs/Tables/Pres | floor | ✅ | content_type + ext + Open with… |
| DRV-RENAME | Rename | target | ✅ | PATCH object/folder name |
| DRV-MOVE | Move between folders | target | ✅ | PATCH folder_id / parent_id; bulk via selection bar |
| DRV-SHARE-UI | Share dialog ACL | target | ✅ | GET meta + PATCH acl; TE live sign-off open |
| DRV-SEARCH | Search files | target | ✅ | W2 `GET /api/v1/drive/search?q=` |
| DRV-PREVIEW | Preview pane | later | ✅ | O-LITE Full MVP: side pane, 8MB cap, image CT\|ext, e2e |
| DRV-LOCK | Lock / unlock file | later | ✅ | O-LITE Full MVP: Unlock locker/owner only; Open-with warn |

---

## Documents

| ID | Контрол / действие | Tier | Status | Notes |
|----|-------------------|------|--------|-------|
| DOC-NEW | New document | floor | ✅ | |
| DOC-OPEN | Open by id / Drive | floor | ✅ | |
| DOC-H1 | Heading 1 | floor | ✅ | |
| DOC-H2H3 | Heading 2/3 | target | ✅ | H2/H3 toolbar + `heading_level` |
| DOC-LIST | Bullet list | floor | ✅ | |
| DOC-BOLD | Bold | floor | ✅ | set_inline_marks |
| DOC-ITALIC | Italic | floor | ✅ | |
| DOC-UNDERLINE | Underline | target | ✅ | `set_inline_marks.underline` + docx `<w:u>` |
| DOC-LINK | Insert link | target | ✅ | `link_url` on span (block-level lite) |
| DOC-PAGE | Page canvas (not cards) | target | ✅ | `era-doc-page` desk |
| DOC-AUTOSAVE | Autosave to Drive | target | ✅ | snapshot debounce 2.5s |
| DOC-SNAPSHOT | Manual snapshot | floor | ✅ | Save now |
| DOC-IMPORT-DOCX | Import docx | floor | ✅ | |
| DOC-EXPORT-DOCX | Export docx | floor | ✅ | |
| DOC-COEDIT-WS | WS sync ops | floor | ✅ | |
| DOC-PRESENCE | Peer cursors / names | target | ✅ | O-LITE: colored peer chips + roster |
| DOC-COMMENTS | Comments on selection | target | ✅ | O-MS: header Comments toggle + floating rail; not Word threads |
| DOC-FIND | Find in doc | target | ✅ | Find next + highlight |
| DOC-WORDCOUNT | Word count | target | ✅ | toolbar |
| DOC-AI-SUM | Summarize with AI | floor | ✅ | handoff |
| DOC-VBA | Macros | never | 🚫 | |
| DOC-PAGE-SETUP | Page setup | target | ✅ | margins/size/orient; File menu |
| DOC-HEADER-FOOTER | Header/footer/page # | target | ✅ | Insert + strip |
| DOC-ALIGN | Align / indent | target | ✅ | Format menu + toolbar |
| DOC-NUMBERED | Numbered lists | target | ✅ | ordered list_type |
| DOC-STRIKE | Strikethrough | target | ✅ | |
| DOC-FONT | Font family / size | target | ✅ | |
| DOC-STYLES | Title / H4–H6 | target | ✅ | heading_level + style_name |
| DOC-PAGE-BREAK | Page break | target | ✅ | |
| DOC-UNDO | Undo / Redo | target | ✅ | local stack + Edit menu |
| DOC-PEER-CARET | Peer carets | target | ✅ | O-LITE Full: presence_caret relay in docs-engine |
| DOC-PDF-PRINT | Download PDF / Print | target | ✅ | O-LITE Full: menu «Print / Save as PDF» (browser only) |
| DOC-VERSIONS | Version history | target | ✅ | W2 Drive versions dialog |
| DOC-REPLACE | Find and replace | target | ✅ | W2 |
| DOC-IMAGE | Insert image | target | ✅ | O-LITE: Drive picker or URL; auth fetch for Drive objects |
| DOC-TABLE | Insert table | target | ✅ | O-LITE: `table_cells` + row/col; TSV compat |
| DOC-TOC | Table of contents | target | ✅ | O-LITE: refresh + jump to heading blocks |
| DOC-BOOKMARK | Bookmark | target | ✅ | O-LITE: navigate / TOC anchors |
| DOC-COLOR | Text color / highlight | target | ✅ | O-LITE: selection `set_marks_range` + sendOp |
| DOC-LANG | Language (BCP-47) | target | ✅ | O-LITE: feeds spell wordlist locale |
| DOC-SPELL | Spelling lite | target | ✅ | O-LITE: air-gap wordlists + ignore |
| DOC-RTF | Download RTF | later | ✅ | O-LITE Full: tables/strike/H-F/image stub + golden |
| DOC-LINE-NUM | Line numbers | later | ✅ | O-LITE: honest block numbers (not wrapped lines) |
| DOC-SUGGEST | Suggesting mode | target | ✅ | O-LITE: shared `revision_event` WS + accept/reject |
| DOC-FULLSCREEN | Full screen | target | ✅ | `requestFullscreen` on `.era-doc-page` |
| DOC-TEXTBOX | Text box | later | ✅ | O-STUB Lite: bordered block + width% + sendOp |
| DOC-COLUMNS | Columns 1–3 | later | ✅ | O-STUB Lite: columns dialog + `page.columns` |
| DOC-REVIEW | Track changes | later | ✅ | LATER revisions accept/reject |
| DOC-COMPARE | Compare documents | later | ✅ | LATER block-text diff |
| DOC-MERGE | Mail merge lite | later | ✅ | LATER CSV → <<fields>> |
| DOC-ODT | Download ODT | later | ✅ | O-LITE Full: table_cells, H/F, image stub, HR border + golden |
| DOC-SECTION | Section break | later | ✅ | O-STUB Lite: `section_break` + insert_block sync |
| DOC-FOOTNOTE | Footnote | later | ✅ | O-STUB Lite: `[fnN]` + jump + footnote block |
| DOC-MANAGE-STYLES | Manage styles | later | ✅ | O-STUB Lite: named styles in gallery + apply marks |
| DOC-JUSTIFY-TB | Justify on toolbar | fmt | ✅ | O-FMT-1 |
| DOC-INDENT | Increase/Decrease indent | fmt | ✅ | O-FMT-1; model `indent_mm` |
| DOC-LINE-SPACING | Line spacing presets | fmt | ✅ | O-FMT-1 |
| DOC-PARA-SPACING | Paragraph space before/after | fmt | ✅ | O-FMT-1 |
| DOC-LIST-LEVEL | List nest level ± | fmt | ✅ | O-FMT-1 |
| DOC-LIST-MARKER | Bullet/number marker set | fmt | ✅ | O-FMT-1 |
| DOC-LIST-RESTART | Restart / continue numbering | fmt | ✅ | O-FMT-1 |
| DOC-FORMAT-PAINTER | Format Painter | fmt | ✅ | O-LITE: set_block_format + set_marks_range sendOp |
| DOC-SHARE | Share dialog | target | ✅ | O-LITE: copy link · ACL in Drive deep-link |
| DOC-CLIPBOARD | Cut/Copy/Paste plain | target | ✅ | O-LITE: selection paste + multi-para plain |
| DOC-SUPER-SUB | Superscript / Subscript | fmt | ✅ | O-FMT-1 |
| DOC-STYLE-GALLERY | Style gallery (Normal/Quote/Caption…) | fmt | ✅ | O-FMT-1 |
| DOC-SHOW-MARKS | Show formatting marks (¶) | fmt | ✅ | O-FMT-1 |
| DOC-SYMBOL | Insert symbol | fmt | ✅ | O-FMT-1 |
| DOC-HR | Horizontal line | fmt | ✅ | O-FMT-1 |
| DOC-RULER | Horizontal ruler | fmt | ✅ | O-FMT-2; O-UX: + vertical ruler, A4 sheet |
| DOC-STICKY-CHROME | Sticky menubar+ribbon | target | ✅ | O-UX: scroll only below ribbon |
| DOC-SEL-KEEP | Keep selection after style | target | ✅ | O-UX: restore range + multi-block marks |
| DOC-TOOLBAR-MENUS | Grouped align/list menus | target | ✅ | O-UX: button menus + Headings-like selects |
| DOC-FONT-STEPPER | Font size − / + | target | ✅ | O-UX: stepper around size select |
| DOC-TIPS | Google-like tooltips | target | ✅ | O-UX: `toolbar-chrome.js` |
| DOC-TABLE-DIALOG | Insert table N×M | fmt | ✅ | O-FMT-2 |
| DOC-WORDCOUNT-DLG | Word count dialog | fmt | ✅ | O-FMT-2 |

---

## Tables

| ID | Контрол / действие | Tier | Status | Notes |
|----|-------------------|------|--------|-------|
| TBL-NEW | New sheet | floor | ✅ | |
| TBL-GRID | Editable grid | floor | ✅ | viewport starts A–Z×40; expand-on-scroll to WW×10K; engine 10000×621 |
| TBL-FORMULA-BAR | Formula bar | floor | ✅ | |
| TBL-SUM | SUM/AVERAGE/MIN/MAX | floor | ✅ | |
| TBL-IF-COUNT | IF, COUNT | target | ✅ | calc.rs + toolbar |
| TBL-COUNTIF | COUNTIF | target | ✅ | Wave C |
| TBL-NUM-FMT | Number formats | target | ✅ | Cell.format + UI |
| TBL-INSERT-RC | Insert/delete row/col | target | ✅ | SheetOp |
| TBL-FILL | Fill-down / fill-right | target | ✅ | |
| TBL-FIND | Find in cells | target | ✅ | |
| TBL-MULTI-SHEET | Multiple sheets | target | ✅ | O-UX: bottom tabs bar (+ sheet) like Sheets |
| TBL-SCROLL | Grid scroll / wheel | target | ✅ | O-SHELL: `.era-sheet-pane` + Shift/Ctrl wheel → X |
| TBL-NOTES | Cell notes rail | target | ✅ | O-MS: Comments toggle near Share; persist `Cell.note` |
| TBL-CHART | Chart lite | target | ✅ | T-LITE: persist `{type,range}` on tab + SVG re-render |
| TBL-RESIZE | Column resize | target | ✅ | drag th edge |
| TBL-FREEZE | Freeze header | target | ✅ | freeze-on sticky |
| TBL-FREEZE-PANES | Freeze panes (rows/cols) | target | ✅ | T-LITE: freeze at selection + clear; `SheetTab.freeze_*` |
| TBL-EXPORT-ODS | Export / import ODS | target | ✅ | T-LITE: multi-sheet + formulas/merges/wrap/border; `import_ods` |
| TBL-SUBTOTAL | Subtotal lite | later | ✅ | O-STUB Lite: group-by left col + SUM rows below data |
| TBL-SORT | Sort | target | ✅ | T-LITE: row-aware `SortRange` A↔Z; sibling cols move |
| TBL-FILTER | Filter | target | ✅ | T-LITE: AutoFilter criteria persist on tab |
| TBL-FILTER-OPTS | Filter options | target | ✅ | T-LITE: persist + optional AND second col |
| TBL-PROTECT | Protect sheet lite | target | ✅ | Protect/Unprotect toggle; blocks edits |
| TBL-CSV | Download CSV | target | ✅ | client export of used cells |
| TBL-MERGE | Merge cells lite | target | ✅ | T-LITE: selection merge + Unmerge |
| TBL-PROTECT-RANGES | Protect ranges lite | target | ✅ | list / REMOVE / CLEAR UI + engine |
| TBL-SPARKLINE | Sparkline | later | ✅ | O-STUB Lite: SVG + persist `chart_type=sparkline` |
| TBL-WHATIF | What-if / Goal Seek lite | later | ✅ | O-STUB Lite: Preview + Apply binary search |
| TBL-SCENARIOS | Scenarios | later | ✅ | O-STUB Lite: named snapshots persist on `SheetTab.scenarios` |
| TBL-CONSOLIDATE | Consolidate | later | ✅ | O-STUB Lite: sum from active/name/index sheet → target |
| TBL-IMPORT-XLSX | Import xlsx | floor | ✅ | T-LITE: multi-sheet + shared strings + formulas progressive |
| TBL-EXPORT-XLSX | Export xlsx | floor | ✅ | T-LITE: multi-sheet + formulas/merges lite |
| TBL-COEDIT | Cell WS sync | floor | ✅ | |
| TBL-PRESENCE | Presence | target | ✅ | T-LITE: roster + `presence_cell` peer highlight |
| TBL-VBA | Macros / VBA | never | 🚫 | |
| TBL-AVG-MIN-MAX-ROUND | AVERAGE / MIN / MAX / ROUND | fmt | ✅ | O-FMT-2 |
| TBL-CELL-BOLD-ALIGN | Cell bold + align | fmt | ✅ | O-FMT-2 |
| TBL-WRAP-BORDERS | Wrap text + borders lite | fmt | ✅ | T-LITE: per-side `border_sides` + xlsx/ODS bits |
| TBL-PASTE-VALUES | Paste values | fmt | ✅ | O-FMT-2 |

---

## Presentations

| ID | Контрол / действие | Tier | Status | Notes |
|----|-------------------|------|--------|-------|
| PRE-NEW | New deck | floor | ✅ | |
| PRE-EDIT | Title/body | floor | ✅ | |
| PRE-ADD-SLIDE | Add slide | floor | ✅ | |
| PRE-NAV | Prev/Next | floor | ✅ | |
| PRE-SAVE | Save | floor | ✅ | session |
| PRE-IMPORT | Import pptx | floor | ✅ | multi-slide |
| PRE-EXPORT | Export pptx | floor | ✅ | P-LITE: multi-slide + image/notes/body2/bg |
| PRE-LAYOUTS | Layouts | target | ✅ | title_body / title_only / section / two_column |
| PRE-REORDER | Reorder slides | target | ✅ | Move up/down + filmstrip drag |
| PRE-PRESENT | Slideshow / Present | target | ✅ | P-LITE: bg + image + notes strip; keyboard |
| PRE-THEME-BG | Theme / slide background | target | ✅ | O-UX: Background dialog + Theme side panel |
| PRE-LAYOUT-UI | Layout chrome button | target | ✅ | O-UX: Layout dialog with presets |
| PRE-TRANSITION-UI | Transition chrome | target | ✅ | O-MS: Motion panel — fade/push/wipe/morph lite; persist on slide |
| PRE-SLIDE-NOTES | Comment → speaker notes | target | ✅ | O-MS: Comments toggle + rail; append to notes |
| PRE-UNDO | Undo / Redo | target | ✅ | P-LITE: dual stack + Ctrl+Y; debounced typing |
| PRE-FIND | Find in slides | target | ✅ | P-LITE: title/body/notes |
| PRE-PRINT | Print setup | target | ✅ | P-LITE: all slides, 1/page + notes strip |
| PRE-SHARE | Share | target | ✅ | P-LITE: dialog copy link + Manage ACL in Drive |
| PRE-MASTER | Edit master | later | ✅ | P-LITE: `ErapDeck` layout/placeholders persist |
| PRE-ODP | Download ODP | later | ✅ | P-LITE: marks, image, solid bg + notes/two-col |
| PRE-NOTES | Speaker notes | target | ✅ | present strip + pptx notes part |
| PRE-ANIM | Animations | target | ✅ | O-MS: appear stagger in Present; Morph = CSS lite (not Google Morph engine) |
| PRE-DUP-SLIDE | Duplicate slide | fmt | ✅ | O-FMT-3 |
| PRE-TEXT-FORMAT | Bold / Align on title·body | fmt | ✅ | O-FMT-3 |
| PRE-FONT-STEP | Increase / Decrease font | fmt | ✅ | O-FMT-3 |
| PRE-INSERT-IMAGE | Insert image (URL/Drive) | fmt | ✅ | P-LITE: Drive picker; present + pptx/odp embed |

---

## Projects

| ID | Контрол / действие | Tier | Status | Notes |
|----|-------------------|------|--------|-------|
| PRJ-BOARD | Kanban columns | floor | ✅ | |
| PRJ-ADD | Add task | floor | ✅ | |
| PRJ-MOVE | Move columns | floor | ✅ | buttons + drag |
| PRJ-DEL | Delete | floor | ✅ | |
| PRJ-DRIVE-LINK | Open in Docs | floor | ✅ | |
| PRJ-MENUBAR | Menubar chrome | target | ✅ | Wave E |
| PRJ-RENAME | Rename board | target | ✅ | `/api/v1/projects/board` |
| PRJ-DRAG | Drag cards | target | ✅ | PRJ-LITE: HTML5 DnD + `sort_key` |
| PRJ-ASSIGN | Assignee | target | ✅ | PRJ-LITE: datalist picker + free-text |
| PRJ-DUE | Due date | target | ✅ | chip overdue/soon |
| PRJ-FILTER | Filter tasks | target | ✅ | PRJ-LITE: text + facets (assignee/label/priority/overdue) |
| PRJ-DRIVE-PICKER | Drive object picker | target | ✅ | PRJ-LITE: folder browse + files |
| PRJ-LABELS | Labels / tags | target | ✅ | W2 chips + menu |
| PRJ-CHECKLIST | Card checklist | target | ✅ | W2 items + progress |
| PRJ-SHARE | Share | target | ✅ | PRJ-LITE: dialog + Manage ACL in Drive (`.eraj`) |
| PRJ-COMMENTS | Board comments rail | target | ✅ | O-MS: Comments near Share; localStorage Lite |
| PRJ-SWIMLANES | Swimlanes by assignee | later | ✅ | PRJ-LITE: persist viewMode; DnD → assignee |
| PRJ-GANTT | Gantt | later | ✅ | O-MS: Lite timeline by due date (3-day bars); View → Gantt |
| PRJ-PRIORITY | Priority chip P0–P2 | later | ✅ | PRJ-LITE: p0\|p1\|p2 field + chip + filter |

---

## Office AI

| ID | Контрол / действие | Tier | Status | Notes |
|----|-------------------|------|--------|-------|
| AI-SUM | Summarize | floor | ✅ | stub/ollama |
| AI-BANNER | Air-gap honesty | floor | ✅ | |
| AI-REWRITE | Rewrite selection | target | ✅ | `/docs-ai/rewrite` + Docs handoff |
| AI-CLOUD | Cloud LLM | never | 🚫 | |

---

## Shell / Design

| ID | Контрол / действие | Tier | Status | Notes |
|----|-------------------|------|--------|-------|
| SHL-NAV | Unified icon product nav | target | ✅ | O-SHELL: distinctive `nav*` marks + accent colors |
| SHL-MENUBAR | Google-like menubar + item icons | target | ✅ | Docs/Tables/Pres/Projects/Office AI; Menu-Map |
| SHL-ICONS | Shared SVG icon set | target | ✅ | O-SHELL: `fontInc`/`fontDec`/`listLevel*` distinct from indent |
| SHL-SELECT | Native select / flyouts | target | ✅ | O-SHELL: no preventDefault on select; overflow/z-index |
| SHL-AUTH | authFetch + JWT exp | target | ✅ | O-SHELL: 401 + exp → `/login`; session watch |
| SHL-COMMENTS | Unified Comments rail chrome | target | ✅ | O-SHELL: head + Close; semantics still per-product |
| SHL-CHROME | Editor title + Share + save pill | target | ✅ | Docs (+ Tables/Pres lite) |
| SHL-TOOLBAR | Compact grouped toolbars | target | ✅ | Wave G primary/secondary groups |
| SHL-DRIVE-LIST | Drive file rows + icon actions | target | ✅ | Wave G `.era-file-list` |
| SHL-TOKENS | Shared CSS tokens | target | ✅ | `--era-*` + layout tokens; Theme Matrix |
| SHL-SKU-THEME | SKU chrome accents via `data-sku` | target | ✅ | Phase A: body attrs + `[data-sku]` in office.css |
| SHL-SWITCHER | ERA One product switcher | target | ✅ | Phase B: `era-chrome.js` mountSwitcher |
| SHL-ACCOUNT | Shared account chip pattern | target | ✅ | Phase B: Office chip + Control role chip |
| SHL-STATUS | Toast / status line | target | ✅ | save/auth status + save pill |
| SHL-USER | User chip | target | ✅ | topbar JWT sub; expired → Sign in |
| SHL-OSS-DELTA | OSS vs Google backlog map | target | ✅ | [Office-OSS-Delta.md](Office-OSS-Delta.md) |
