# ERA Office — UI Design System (shell)

**Дата:** 31 июля 2026 г.  
**Baseline:** [Office-UI-Baseline.md](Office-UI-Baseline.md)  
**Menu map:** [Office-UI-Menu-Map.md](Office-UI-Menu-Map.md)  
**Код:** `ui/office-shell` → workspace `/office-assets/`  
**Кросс-линейки (Control / Comms / Office + SKU accents):** [ERA-UI-Shell-Theme-Matrix.md](ERA-UI-Shell-Theme-Matrix.md)

---

## 1. Принципы

1. **Один продукт** — Drive / Docs / Tables / … делят chrome, не выглядят как разные сайты.  
2. **Редактор = canvas** — документ/сетка/слайд в центре; chrome тонкий.  
3. **Честность** — disclaimer без «как Word».  
4. **Air-gap calm** — без фиолетового SaaS-glow, без emoji-sticker UI.  
5. **Google-like menubar** + **floating pill toolbar** (`.era-toolbar`: `#edf2fa`, radius 24px, icon-first groups, soft separators) — не MS Ribbon / не «суп кнопок».
6. Порядок групп как Collab: Undo → Styles/Font → Marks → Insert lite → Align/Lists → Find; I/O и New — в `.era-toolbar-secondary` справа.
7. Disabled menu items = honest «Planned» (без пустых кликов).

---

## 2. Tokens

**SSOT:** [`ui/shared-tokens/`](../ui/shared-tokens/) (synced into `office-shell/web/tokens/`).  
Canon names `--era-*` (Theme Matrix). Suite defaults on `:root`; chrome accent overrides via `data-line` / `data-sku` on `<body>`.

| Token | Role |
|-------|------|
| `--era-ink` | Primary text |
| `--era-muted` | Secondary |
| `--era-line` | Borders |
| `--era-bg` | App background |
| `--era-surface` | Cards / panels |
| `--era-accent` | Primary actions / links (line + SKU) |
| `--era-accent-soft` | Selection / chip wash |
| `--era-ok` / `--era-warn` / `--era-err` | Status |
| `--era-warn-bg` | Disclaimer banner |
| `--era-font` | UI font stack |
| `--era-doc-font` | Document body stack |
| `--era-topbar-h` | Top chrome height (`3rem`) |
| `--era-nav-w` | Sidebar width (`220px`) |

**SKU accents** (`data-sku`): drive / docs / tables / pres / projects / ai / suite / mail.  
**Line:** `data-line="office"|"comms"|"control"`. Login uses `data-product` + `--login-*` aliases to `--era-*`.

Load order: base → theme-line → theme-sku → `office.css` components + `era-chrome.css`.

---

## 3. Shell layout

Editor SPA (Docs keeps a centered page; others go full width):

- topbar: brand · icon nav · user  
- menubar + compact toolbar + status  
- main canvas  

Drive (fluid):

- left: folder tree (`My Drive` + expand/collapse)  
- right: breadcrumb · actions · file list · panels  

Rules:

- **topbar** — brand mark + **icon product nav** (единый на всех SPA) + user chip  
- **menubar** — shared `menubar.js`; **иконки у пунктов** (`mountMenuIcons`); ERA+ badges  
- **editor chrome** — title · Share · save pill · presence  
- **toolbar** — группы primary / secondary; `era-icon-btn` + текст только где нужно  
- **width** — Docs: page canvas (`.era-main` / `.era-doc-page`); **Tables / Presentations / Projects / Drive**: `.era-main-fluid` (full viewport)  
- **Drive** — `.era-drive-layout` + left `.era-drive-tree` (folder structure) + `.era-file-list`  
- **status** — autosave / presence / auth  
- **canvas** — page (Docs), grid (Tables), stage (Pres), board (Projects)

---

## 4. Components (MVP)

- `.era-btn` / `.era-btn-primary` / `.era-icon-btn`  
- `.era-banner-warn`  
- `.era-status.ok|.err`  
- `.era-topbar` / `.era-user-chip`  
- `.era-doc-title` / `.era-save-pill` / `.era-editor-chrome`  
- `.era-doc-page` — white page on gray desk (Documents)  
- `.era-main-fluid` — full-width shell for Tables / Presentations / Projects / Drive  
- `.era-drive-layout` / `.era-drive-tree` — Drive folder tree + browser

---

## 5. Rollout

1. Serve `/office-assets/` (`office.css`, `shell.js`, `menubar.*`, `icons.js`) from workspace.  
2. Adopt Docs → Drive → Tables → rest.  
3. Avoid copying full CSS into each SPA long-term.  
4. OSS differentiators → [Office-OSS-Delta.md](Office-OSS-Delta.md) + Menu-Map ERA+ slots.
