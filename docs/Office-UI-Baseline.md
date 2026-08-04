# ERA Office — UI / Product Baseline (Collab v2)

**Статус:** Accepted (product direction)  
**Дата:** 30 июля 2026 г.  
**Связано:** [ADR-0026](adr/0026-sovereign-office-engine.md) · [Office-Tech-Eval-Checklist](Office-Tech-Eval-Checklist.md) · [Office-UI-Feature-Inventory](Office-UI-Feature-Inventory.md) · [Office-UI-Design-System](Office-UI-Design-System.md) · [Office-Product-Readiness-Matrix](Office-Product-Readiness-Matrix.md) · [Office-OSS-Delta](Office-OSS-Delta.md) · [Office-UI-Menu-Map](Office-UI-Menu-Map.md)  
**Код shell:** `ui/office-shell` → workspace `/office-assets/`

---

## 1. Решение одной строкой

> **База продукта = Collab Office** (класс Google Docs / Sheets **по ощущению работы**), в суверенном контуре.  
> **TE checklist = floor** (минимальная планка живого показа).  
> **Microsoft Office / VBA = never** (не референс и не цель).

Не копируем MS Office. Не обещаем OnlyOffice/Collabora. Свой engine (ADR-0026), файлы только в ERA Drive.

---

## 2. Два уровня (не смешивать)

| Уровень | Назначение |
|---------|------------|
| **Floor (TE / Gov Eval)** | Можно показать технарю: «не stub» — create, edit, co-edit, простой OOXML |
| **Target (Collab v2)** | Продукт, в котором **реально работают** день за днём |

Inventory и UI-доработки ведутся от **Target**; Floor остаётся gate для sign-off.

---

## 3. Never (запрещено в языке продукта и backlog P0–P2)

- VBA / macros / ActiveX / «как в Excel макросы»
- Promise pixel-perfect Word/Excel/PPT
- Cloud SaaS LLM phone-home
- GPL office runtime (LibreOffice headless и т.п.)

---

## 4. Target — Collab v2 по изданиям

### 4.1 ERA Drive (hub)

- Login, tenant, list/upload/download  
- Folders, breadcrumb, rename, move, search (search — wave 2)  
- Versions понятные человеку  
- Share UI (viewer/editor)  
- Open with Documents / Tables / Presentations  
- New document / sheet / deck  

### 4.2 ERA Documents ≈ Docs lite

**Must (target):**
- Страница-документ (не «карточки» прототипа)  
- H1–H3, lists, bold/italic/underline, links  
- Прозрачное сохранение в Drive (autosave), не «Snapshot» как главная модель  
- Co-edit: presence (имя/цвет) 2+ users  
- Comments на выделении (wave 1.5)  
- Find; word count  
- docx import/export + honest dropped-features banner  

**Must+ (O-FMT-1, MS-class capabilities — не Ribbon-паритет):**
- Абзац: justify на ленте, indent±, line/paragraph spacing  
- Списки: nest level, marker set, restart numbering  
- Стили: gallery (Normal / Quote / Caption + H1–H6)  
- Run: superscript/subscript; Format Painter; show ¶  

**Floor (уже/рядом):** create/open, H1/list/bold, WS sync, docx I/O, Drive bind.

> **MS Office** = источник *возможностей* (formatting/lists/styles). Продуктовый UX остаётся Collab/Google-menubar. VBA / pixel-perfect Word = Never.

### 4.3 ERA Tables ≈ Sheets lite

**Must (target):**
- Полноценная сетка (не демо 12×6)  
- Арифметика + SUM/AVERAGE/MIN/MAX/IF/COUNT  
- Multi-sheet (2+)  
- Fill-down, resize columns, freeze header row  
- Sort / simple filter  
- Cell co-edit + presence  
- xlsx single→multi sheet progressively  

**Never:** VBA / script macros.

**Floor:** grid, SUM family, xlsx 1 sheet, WS cells.

### 4.4 ERA Presentations ≈ Slides lite

- Title/body (+ simple layouts), reorder, notes  
- pptx subset  
- После стабилизации Docs/Tables  

### 4.5 ERA Projects / Office AI

- Projects: Collab kanban (menubar, drag, assignee/due, filter, Drive picker) — не Jira / не Gantt  
- AI: in-contour summarize/rewrite assist; stub без Ollama  
- Chrome (Wave F): Google-like title/Share/save pill/icons; ERA+ menu slots from OSS-Delta  

---

## 5. Packaging

| Bundle | Состав |
|--------|--------|
| **office-mvp (demo floor)** | Drive + Documents (+ Tables для гос. показа) |
| **office-suite (target)** | + Presentations + Projects |
| **office-suite-ai** | + Office AI |

Drive **всегда** в suite.

---

## 6. Как обновлять

1. Меняется целевой UX → этот файл + Feature-Inventory.  
2. Floor TE → Tech-Eval-Checklist (живая подпись отдельна).  
3. Запрещено отвечать «готовность Office» только Implementation-Matrix или только Floor.
