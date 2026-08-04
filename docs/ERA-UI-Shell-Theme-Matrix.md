# ERA One — UI Shell / Theme Matrix

**Дата:** 4 августа 2026 г.  
**Статус:** Phase A–D Implemented  
**Связано:** [ADR-0024](adr/0024-era-one-product-families.md) · [ADR-0025](adr/0025-era-one-shared-platform.md) · [Office-UI-Design-System.md](Office-UI-Design-System.md) · [`ui/shared-tokens/`](../ui/shared-tokens/)

---

## 0. Принцип одной фразой

**Одинаковая структура chrome + общие semantic tokens; разные surface/accent themes по линейке и SKU.**  
Domain UI (редактор, inbox, SOC timeline) не переписываем — только читает `--era-*`.

---

## 1. Что уже есть (as-is)

### 1.1 Shell inventory

| Артефакт | Путь | Кто потребляет | Роль |
|----------|------|----------------|------|
| **Office shell** | `ui/office-shell/web/` → `/office-assets/` | Drive, Docs, Tables, Pres, Projects, Office AI; desktop Tauri copy | User productivity chrome |
| **Control shell** | `ui/control-shell/web/` → `/control-assets/` | `ui/control/**` modules | SecOps / IT-Ops console |
| **Login** | `office-shell` `login.html` + `login.css` | Office workspace `/login` | Account page (Google-like steps) |
| **Mail UI** | `ui/mail/web/` | Comms webmail | **Нет shell** — inline styles, вне token-системы |
| **Legacy Control pages** | `ui/observe`, `ui/cases`, `ui/pam`, … | старые entrypoints | Свои локальные `:root`, не через control-shell |

### 1.2 Зоны chrome — матрица «общее / своё»

| Зона | Office shell | Control shell | Login | Comms (mail) сейчас | Цель |
|------|:------------:|:-------------:|:-----:|:-------------------:|------|
| Brand mark + product name | ✅ `.era-brand` | ✅ `.era-brand` | ✅ `.era-login-brand` | ❌ h1 only | **Shared kit** (одна разметка, theme class) |
| Product / module nav | ✅ icon nav `data-mod` | ✅ sidebar groups | — | ❌ | Shared **pattern**, разный layout (top icons vs left dense) |
| Account / user chip | ✅ `.era-user-chip` | 🟡 role meta only | — (это и есть account) | 🟡 `#user` | Shared account menu |
| Auth redirect → `/login` | ✅ `EraOfficeShell.loginUrl` | ❌ | self | ❌ | Workspace + Comms → один login; Control может свой later |
| Top bar geometry | ✅ sticky `.era-topbar` | 🟡 page title row | card | — | Align height/padding tokens |
| Menubar + toolbar | ✅ `menubar.*` + toolbar | — | — | — | **Office-only** domain chrome |
| Save / presence / status | ✅ pills | `.era-status` | form errors | — | Shared status semantics |
| Sidebar nav | Drive tree only | ✅ sticky `.era-sidebar` | — | target 3-pane | Control + Mail pattern |
| Cards / tables / buttons | ✅ `.era-btn*` | ✅ `.era-card` `.era-btn` | form btn | raw | Shared component classes over time |
| Domain canvas | Docs page / grid / stage | KPI / workbench | — | inbox list | **Never unify** |

Легенда: ✅ есть · 🟡 частично · ❌ нет · — не применимо.

### 1.3 Token inventory (as-is, имена разные)

| Роль | Office (`--era-*`) | Control (короткие) | Login (`--login-*`) |
|------|--------------------|--------------------|---------------------|
| Text | `--era-ink` | `--text` | `--login-ink` |
| Muted | `--era-muted` | `--muted` | `--login-muted` |
| Line/border | `--era-line` | `--border` | `--login-line` |
| App bg | `--era-bg` | `--bg` | `--login-bg0/1` |
| Surface/card | `--era-surface` | `--card` | `--login-card` |
| Accent | `--era-accent` `#0b5fff` | `--accent` `#58a6ff` | `--login-accent` `#0b5fff` |
| Accent soft | `--era-accent-soft` | (ad-hoc `#388bfd33`) | — |
| OK / warn / err | `--era-ok` `--era-err` warn-bg | `--ok` `--warn` `--bad` | `--login-err` |
| Font | `--era-font` | `system-ui` hardcode | `--login-font` |
| Radius / shadow | `--era-radius` `--era-shadow` | hardcode `6–8px` | `--login-radius` |

**Вывод:** Office уже на semantic tokens; Control и Login — параллельные словари. Mail вне системы. Переписывать UI не нужно — **свести имена + theme overrides**.

### 1.4 SKU accents уже в Office nav (частично)

В `office.css` иконки nav уже цветные:

| `data-mod` | Accent (факт) |
|------------|---------------|
| drive | `#1a73e8` |
| docs | `#1a56db` |
| tables | `#0f9d58` |
| pres | `#e37400` |
| projects | `#a142f4` |
| ai | `#d93025` |
| mail | `#188038` |

Но **chrome accent** (`--era-accent`) везде один синий `#0b5fff` — SKU не «прокрашивает» primary button / active nav / focus. Desktop SKU — отдельные icons (`sku-*.ico`), без CSS theme.

---

## 2. Целевая модель (to-be)

```
ui/shared-tokens/          ← NEW thin layer (или секция в каждом shell)
  era-tokens-base.css      ← semantic names + spacing/type (no brand color)
  era-theme-control.css    ← dark ops surfaces + Control accent
  era-theme-comms.css      ← light mail density + Comms accent
  era-theme-office.css     ← light productivity (сегодняшний office :root)
  era-theme-sku-*.css      ← optional overrides for --era-accent only
```

На `<html>` или `<body>`:

```html
<body class="era-app" data-line="office" data-sku="docs">
<!-- или -->
<body class="era-control" data-line="control">
```

CSS:

```css
:root { /* base semantic defaults */ }
[data-line="control"] { /* dark + control accent */ }
[data-line="comms"]   { /* comms accent */ }
[data-line="office"]  { /* office defaults */ }
[data-sku="tables"]   { --era-accent: …; --era-accent-soft: …; }
```

Модули **не трогаем**, если они уже используют `var(--era-accent)` / `var(--era-line)` и т.д.

---

## 3. Tokens зафиксировать (канон)

### 3.1 Base (все линейки — одни имена)

| Token | Назначение | Не менять per-SKU? |
|-------|------------|--------------------|
| `--era-ink` | Primary text | Обычно да (кроме Control dark → светлый ink) |
| `--era-muted` | Secondary text | Theme |
| `--era-line` | Borders / dividers | Theme |
| `--era-bg` | App chrome background | Theme |
| `--era-surface` | Panels / cards / topbar | Theme |
| `--era-accent` | Primary CTA, links, focus, active | **Line + SKU** |
| `--era-accent-soft` | Chip / selection / hover wash | Следует за accent |
| `--era-ok` `--era-warn` `--era-err` | Status (семантика одна) | Можно чуть сдвинуть hue, не роль |
| `--era-font` | UI stack | Shared |
| `--era-radius` `--era-shadow` | Geometry | Shared |
| `--era-nav-w` | Sidebar width (Control/Mail) | Shared layout |
| `--era-topbar-h` | Top chrome height | Shared layout |
| `--era-space-1`…`4` | Optional spacing scale | Shared |

**Alias bridge (без rewrite Control):** в `control.css` один раз:

```css
--bg: var(--era-bg);
--card: var(--era-surface);
--text: var(--era-ink);
--muted: var(--era-muted);
--border: var(--era-line);
--accent: var(--era-accent);
--ok: var(--era-ok);
--warn: var(--era-warn);
--bad: var(--era-err);
```

То же для login: `--login-accent: var(--era-accent)` и т.д.

### 3.2 Layout tokens (структурное единство)

| Token / rule | Значение-цель | Зачем |
|--------------|---------------|-------|
| Topbar height | `3rem` (как Office сейчас) | Память мышц |
| Nav icon size | `1.25rem` | Office icon nav |
| Sidebar width | `220px` (Control `--nav-w`) | Control + future Mail folders |
| Radius | `6px` controls / `8px` cards | Уже близко |
| Focus ring | `2px solid var(--era-accent)` | A11y + brand |

---

## 4. Accents — разводка по линейкам и SKU

### 4.1 Product line (главный сигнал «где я»)

| Line | `data-line` | Surface mode | `--era-accent` | `--era-accent-soft` | Заметка |
|------|-------------|--------------|----------------|---------------------|---------|
| **Control** | `control` | Dark ops | `#58a6ff` (как сейчас) | `#388bfd33` | Холодный SOC; не светлый Office |
| **Communications** | `comms` | Light dense | `#188038` (уже nav mail) | `#e6f4ea` | Inbox green; ближе к Workspace light |
| **Office** | `office` | Light productivity | `#0b5fff` (suite default) | `#dbe8ff` | Как `office.css` сегодня |
| **Login / Identity** | `identity` | Light branded | Suite blue `#0b5fff` **или** accent активной line из `?product=` | soft blue | Один login на Workspace; опционально tint по `next` |

Control **не** обязан совпадать по lightness с Office — только по **именам tokens** и account/switcher pattern.

### 4.2 Office SKU (вторичный сигнал внутри Office)

Меняем **только** `--era-accent` + `--era-accent-soft` (+ опционально brand-mark color).  
Не трогаем ink/bg/line — иначе «разные сайты».

| SKU | `data-sku` | Accent | Soft | Источник |
|-----|------------|--------|------|----------|
| Suite / Drive hub | `suite` / `drive` | `#1a73e8` | `#e8f0fe` | nav drive |
| Documents | `docs` | `#1a56db` | `#e8f0fe` | nav docs |
| Tables | `tables` | `#0f9d58` | `#e6f4ea` | nav tables |
| Presentations | `pres` | `#e37400` | `#fef7e0` | nav pres |
| Projects | `projects` | `#7c3aed`* | `#f3e8ff` | nav projects (`#a142f4` → чуть спокойнее для CTA) |
| Office AI | `ai` | `#d93025` | `#fce8e6` | nav ai |

\*Projects purple в nav ок для иконки; для primary button лучше чуть более accessible `#7c3aed` (контраст на белом).

**Как включить без rewrite SPA:**

```html
<!-- tables/web/index.html -->
<body class="era-app era-tables-app" data-line="office" data-sku="tables">
```

```css
/* era-theme-sku.css — 15 строк */
[data-sku="tables"] {
  --era-accent: #0f9d58;
  --era-accent-soft: #e6f4ea;
}
```

Всё, что уже на `var(--era-accent)` (кнопки, active nav, focus, selection) перекрашивается само.

### 4.3 Comms SKU (когда появится shell)

| SKU | Accent |
|-----|--------|
| Mail | `#188038` |
| Chat | `#1a73e8` (или отдельный teal `#00897b`) |
| Meet/VCS | `#e37400` |

Пока mail без shell — **первый шаг:** подключить `office.css` tokens + `data-line="comms"`, не копировать Docs menubar.

### 4.4 Control modules (не SKU-цвета)

Модули Control (Observe, PAM, Vuln…) **не** получают разные primary accents — только sidebar active state.  
Иначе SOC превращается в радугу. Достаточно:

- line accent = Control blue;
- optional **severity badge** color (ok/warn/bad), не chrome.

---

## 5. Shell zones — что унифицировать по фазам

### Phase A — Token bridge (1–2 дня, zero UI rewrite) — **Implemented**

1. Ввести канон имён `--era-*` в Control через aliases (§3.1). ✅  
2. Login: `--login-*` → alias на `--era-*`; tint по `?product=` / `next`. ✅  
3. Документировать таблицу §4 как SSOT (этот файл). ✅  
4. Добавить `data-sku` на Office body + sku accent overrides (§4.2). ✅

**Не делать:** общий React DS, мерж shell.js, перенос Mail в Office menubar.

### Phase B — Shared chrome kit (структура) — **Implemented**

| Компонент | Действие |
|-----------|----------|
| Brand block | `EraChrome.mountBrand` (Control); Office keeps `.era-brand` markup |
| Account chip | Office chip + Control role via `mountAccount` |
| Product switcher | `era-chrome.js` — Office topbar + Control sidebar |
| Focus / btn / status | `era-chrome.css` + sync script |

Sync: `pwsh -File scripts/sync-era-chrome.ps1` (office-shell → control-shell).

Layout: Office остаётся **top icon nav**; Control — **left sidebar**.

### Phase C — Comms enters Workspace — **Implemented**

- Mail на `--era-*` (`ui/mail/web/mail.css`) + `data-line="comms"` / `data-sku="mail"`.  
- Topbar brand + nav (Mail / Chat planned) + account + product switcher (`era-chrome`).  
- Без Docs menubar; 3-pane-lite (folders + pane).  
- Auth: existing PKCE; login tint via `next=/mail/` when using shared `/login`.

### Phase D — Extract shared-tokens + legacy cleanup — **Implemented**

- `ui/shared-tokens/` SSOT; sync → office/control/mail `web/tokens/` via `scripts/sync-era-tokens.ps1`.  
- Shell CSS imports tokens then keeps layout/components.  
- Legacy `/ui/observe`, `/ui/pam`, … → `/ui/control/*` redirects in control-plane.  
- ADR-0025 §6 updated.

---

## 6. Что сознательно НЕ унифицируем

| Зона | Почему |
|------|--------|
| Docs page canvas / Tables grid / Pres stage | Domain metaphor |
| Control dense tables + dark KPI | Ops density |
| Mail 3-pane | Inbox IA |
| Menubar/toolbar Office | Нет в Control/Mail |
| Marketing site (`site/`) | Отдельный коммит/визуал (site-commit-isolation) |

---

## 7. Acceptance checklist (для будущей волны)

- [x] User-facing Office SPA + Control shell читают `--era-*` (или alias).  
- [x] Смена `data-sku` на Tables меняет primary CTA/focus без правок `tables/web/app.js`.  
- [x] Control остаётся dark; Office/Comms — light.  
- [x] Login один URL; optional product tint.  
- [x] Product switcher placeholder на одном месте в Office + Control (+ Mail).  
- [x] Нет секретов/CDN; air-gap.  
- [x] Обновлены Design System §2 / Control-UI-Shell-Spec / Feature-Inventory SHL-*.  
- [x] Tokens в `ui/shared-tokens` (Phase D).

---

## 8. Быстрый map «файл → тема»

| Файл | Сейчас | После Phase A |
|------|--------|---------------|
| `ui/office-shell/web/office.css` | Suite tokens + nav SKU icon colors | Base office theme + `[data-sku]` accent blocks |
| `ui/control-shell/web/control.css` | `--bg/--accent` | Same visuals; aliases → `--era-*`; `data-line="control"` |
| `ui/office-shell/web/login.css` | `--login-*` | Alias + optional product query tint |
| `ui/mail/web/index.html` | inline | Подключить tokens + `data-line="comms"` (Phase C) |
| `apps/era-office-desktop/.../office.css` | Mirror office-shell | Sync accents with web (SKU already in tauri conf icons) |

---

## 9. Рекомендуемая палитра (сводка)

```
Control:  surface dark #0f1419 / accent #58a6ff
Comms:    surface light / accent #188038
Office:   surface light / accent #0b5fff (suite)
  Drive     #1a73e8
  Docs      #1a56db
  Tables    #0f9d58
  Pres      #e37400
  Projects  #7c3aed
  AI        #d93025
```

Это ровно ваша формула: **преемственная структура, мгновенное цветовое отличие линейки и SKU**, без переписывания domain UI.
