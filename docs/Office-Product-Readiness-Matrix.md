# ERA Office — Product Readiness Matrix (один экран)

**Дата:** 3 августа 2026 г.  
**Канон:** [`ERA-Product-Acceptance-Standard.md`](products/ERA-Product-Acceptance-Standard.md) **v1.3** §3.4  
**Назначение:** ответ на «матрица готовности / можно ли показывать / продавать».  
**Не путать с** [`Office-Implementation-Matrix.md`](Office-Implementation-Matrix.md) (= только Scaffold BE / AC).

**Источники слоёв:** Sprint-Index · Implementation-Matrix · Tech-Eval-* · Pilot-Readiness / Gap · `editions-office.yaml` / `editions-shared.yaml` · [`Office-UI-Menu-Map.md`](Office-UI-Menu-Map.md) · [`Office-Roadmap.md`](Office-Roadmap.md) §0–§3 · Solo specs (Docs-FullUI / Pres / Projects / SKU-Distro)

**Легенда:** ✅ · 🟡 · ❌/`[ ]` · ⏸ external · `n/a`  
**Rollup строки** = worst(Gate, BE, UI, Demo/TE, Pilot lab, Pilot field) — не worst только BE.  
**Solo / Corporate** — отдельные колонки ниже; Platform-строка = Browser (фаза A).

---

## Сводка линейки (SSOT готовности) — Platform Browser

| Издание | Gate | Scaffold BE | UI | Demo / Tech Eval | Pilot lab | Pilot field | Edition | Sell / show |
|---------|------|-------------|----|------------------|-----------|-------------|---------|-------------|
| **ERA Drive** | ✅ | ✅ | ✅ + W2 search + LATER preview + **ERA+ lock** + e2e | 🟡 TE-D* UI ready; живая подпись технаря открыта | ✅ RT lab | [ ] RT-O09 | `mvp` | файловый Drive **с оговорками TE sign-off** |
| **ERA Documents** | ✅ | ✅ PutVersion + snapshot reopen lab (RT-O09 field open) | ✅ A–B+F+W2+LATER+**ERA+** thicker ODT + suggesting UX + **submenus** + e2e | 🟡 TE-DOC* UI ready; живая подпись технаря открыта | ✅ Docs RT lab | [ ] RT-O09 | `mvp` | текст **не Word**, но толще lite; TE sign-off открыт |
| **ERA Tables** | ✅ | ✅ PutVersion flush/reopen lab (T4 co-edit lite; field open) | ✅ A+C+F+W2+LATER+**ERA+** thicker ODS + what-if preview + **submenus** + e2e | 🟡 TE-T* UI ready; живая подпись технаря открыта | 🟡 | [ ] | `mvp` | **не** Excel, но толще lite; TE sign-off открыт |
| **ERA Presentations** | ✅ | ✅ PutVersion put_deck/reopen lab (field open) | ✅ A+D+F+W2+LATER+**ERA+** thicker ODP (notes/cols) + e2e | 🟡 TE-P UI ready; живая подпись открыта | 🟡 | [ ] | `mvp` | thin deck **не PowerPoint**; TE sign-off открыт |
| **ERA Projects** | ✅ | 🟡 (PR4) | ✅ E+F+W2+**LATER**: swimlanes + e2e + `.eraj` New/Open | 🟡 TE-PR + TE-PR05 `.eraj`; живая подпись открыта | 🟡 | [ ] | `mvp` | `.eraj` Drive board; не MS Project / Gantt |
| **ERA Office AI** | ✅ | ✅ stub | ✅ summarize + rewrite + Docs handoff + e2e | 🟡 TE-AI UI ready; живая подпись открыта | 🟡 stub/API | [ ] | `mvp` | air-gap stub; не cloud LLM |
| **Bundle office-mvp** | ✅ | 🟡 | 🟡 | 🟡 | ✅ lab | [ ] | — | пилот с оговорками |
| **Bundle office-suite** | ✅ | 🟡 | 🟡 | 🟡 Tables UI ready; TE sign-off open | 🟡 | [ ] | — | **не** full Office дистру без TE |
| **Product `era-office`** | ✅ | mixed | 🟡 | ❌ | lab ✅ | [ ] | `mvp` not `ga` | honesty: mvp |

---

## Desktop / Solo / Corporate (фаза B) — SSOT

Один бинарь `era-office-desktop`. Solo ≠ Platform Drive; Corporate v1 = WebView → tenant Workspace (содержимое = зрелость строк Platform выше).

### Deployment × продукт

| Продукт | Solo Desktop | Corporate Desktop | Store SKU (`--sku`) | NSIS / identifier scaffold | Store publish / EV |
|---------|--------------|-------------------|---------------------|----------------------------|--------------------|
| **Documents** | ✅ bridge + `.erad`/docx + `office-docs-solo` | ✅ shell → tenant | `docs` · `az.era.office.documents` | ✅ overlay + script | [ ] |
| **Tables** | ✅ `.erat`/xlsx/ods + `office-tables-solo` | ✅ | `tables` · `az.era.office.tables` | ✅ | [ ] |
| **Presentations** | ✅ `.erap` + frame-op + pptx + `office-pres-solo` | ✅ | `presentations` · `az.era.office.presentations` | ✅ | [ ] |
| **Projects** | ✅ `.eraj` persist + `office-projects-solo` | ✅ | `projects` · `az.era.office.projects` | ✅ | [ ] |
| **Suite hub** | ✅ `/` four products | n/a (corp = tenant Drive) | `suite` · `az.era.office.desktop` | ✅ | [ ] |
| **Drive** | ❌ local hub only (no tenant Drive) | ✅ tenant `/drive/` | n/a | n/a | n/a |
| **Office AI** | ❌ hidden (out of scope) | ✅ tenant SPA | ❌ | ❌ | ❌ |
| **Mail / Comms** | ❌ hidden | ✅ tenant + AC-O8 | ❌ | ❌ | ❌ |

### Solo readiness (lab)

| SKU / mode | Code scaffold | UI (shared SPA via bridge) | Demo / TE Solo | Pilot Solo | Sell / show Solo |
|------------|---------------|----------------------------|----------------|------------|------------------|
| **docs** | ✅ S3 + FullUI | ✅ `ui/docs` | 🟡 [Lab-Demo](Office-Stage-Solo-Lab-Demo.md) checklist; TE-DOC honesty | [ ] | lab / PR demo; **не** Store listing |
| **tables** | ✅ S5 B2 | ✅ `ui/tables` | 🟡 Lab-Demo + TE-T | [ ] | lab |
| **presentations** | ✅ B4 | ✅ `ui/presentations` | 🟡 Lab-Demo + TE-P | [ ] | lab |
| **projects** | ✅ B4 | ✅ `ui/projects` | 🟡 Lab-Demo + TE-PR | [ ] | lab |
| **suite** | ✅ hub | ✅ product cards | 🟡 portable pack `dist/office-solo-lab/` | [ ] | lab portable exe+assets |
| **Corporate shell** | ✅ S4 | = Platform UI in WebView | 🟡 [Corp-Lab-Demo](Office-Stage-Corp-Lab-Demo.md); TE = Platform | [ ] | shell demo; depends on tenant |

**Evidence:** `cargo test -p era-office-desktop --lib` — 35 PASS; cores PASS; headless `scripts/smoke-office-solo-lab.ps1` → `reports/office-solo-lab-smoke-*.log`; pack `scripts/pack-office-solo-lab.ps1` → `dist/office-solo-lab/` (+ zip). Specs: [Solo-Docs-FullUI](Office-Stage-Solo-Docs-FullUI.md) · [Solo-Pres](Office-Stage-Solo-Pres-Spec.md) · [Solo-Projects](Office-Stage-Solo-Projects-Spec.md) · [SKU-Distro](Office-Stage-Solo-SKU-Distro.md) · [Lab-Demo](Office-Stage-Solo-Lab-Demo.md) · [Corp-Lab-Demo](Office-Stage-Corp-Lab-Demo.md). Build SKU: `scripts/build-office-sku.ps1`.

**Protocol / argv:** `era-office://open?path=/…` + file path → product; single-instance handoff ✅ scaffold. Branded Store icons / Partner Center / EV — ops `[ ]` in SKU-Distro checklist.

---

## UI (кратко) — Platform packages

| Продукт | Пакет | Уровень |
|---------|-------|---------|
| Drive | `ui/drive` + shell | ✅ fluid + folder tree + W2 search + LATER preview + ERA+ lock |
| Documents | `ui/docs` + shell | ✅ A–B+F+W2+LATER+ERA+ (ODT/section/…) |
| Tables | `ui/tables` + shell | ✅ A+C+F+W2+LATER+ERA+ (ODS/freeze/…) |
| Presentations | `ui/presentations` + shell | ✅ A+D+F+W2+LATER+ERA+ ODP |
| Projects | `ui/projects` + shell | ✅ E+F+W2+LATER swimlanes |
| Office AI | `ui/office-ai` + shell | ✅ summarize + rewrite (TE-AI) |
| Solo shell | `apps/era-office-desktop` + bridge assets | ✅ SKU entry + File open/save hooks; Mail/AI nav blocked |

**Есть SPA ≠ UI готов.** Demo/TE остаётся 🟡 до живой подписи. Solo переиспользует те же SPA через loopback bridge.

---

## Вердикт одной строкой

| Вопрос | Ответ |
|--------|-------|
| Backend/gates (Platform) | В основном да (с 🟡 residual) |
| UI линейки Platform | Drive/Docs/Tables/Pres/Projects/Office AI ✅ (Collab A–G + W2 + LATER + **ERA+** live; Never remain slots) |
| Кодить / демо **lab** Platform + Solo 4 SKU | **Да** — pack [`Office-Stage-Solo-Lab-Demo.md`](Office-Stage-Solo-Lab-Demo.md) (`dist/office-solo-lab/`) |
| Solo = четыре Store-продукта из одного exe | **Да в коде**; NSIS overlays; listing/EV — [`SKU-Distro`](Office-Stage-Solo-SKU-Distro.md) `[ ]` |
| Corporate desktop | **Shell готов** — [`Corp-Lab-Demo`](Office-Stage-Corp-Lab-Demo.md); содержимое/TE = Platform A |
| Показ как Office (Tech Eval) | **Нет** — TE-T/TE-D/TE-DOC sign-off ещё живой; gov xlsx corpus open |
| Pilot-ready / `ga` | **Нет** (RT-O09); Solo Store publish тоже `[ ]` |
| Mail / Office AI в Solo Store | **Нет** (намеренно out of scope) |

---

## Как обновлять

1. Меняется BE AC → Implementation-Matrix **и** колонка Scaffold BE здесь (Platform).  
2. Меняется UI / TE → Tech-Eval-* **и** колонки UI / Demo здесь.  
3. Field → Pilot field + editions.  
4. Меняется Solo / SKU / Corporate shell → этот файл § Desktop **и** Lab-Demo / Corp-Lab-Demo / SKU-Distro / Roadmap §0.  
5. Запрещено отвечать на «готовность Office» только Implementation-Matrix или только Roadmap без этой матрицы.
