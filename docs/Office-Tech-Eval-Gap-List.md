# ERA Office — Tech Eval Gap List (TE-*)

**Версия:** 1.0  
**Дата:** 8 июля 2026 г.  
**Статус:** Active — **исполняемый backlog до sign-off технаря**  
**Стратегия:** [`Office-Tech-Eval-Strategy.md`](Office-Tech-Eval-Strategy.md)

> INFRA/pilot gaps (RT-O*, GAP-O-P0-*) — в [`Office-Pilot-Gap-List.md`](Office-Pilot-Gap-List.md).  
> Здесь только **продуктовая глубина** для гос. показа.

---

## 1. Резюме

| Издание | Infra (pilot) | Product (tech eval) | Блокер дистра |
|---------|---------------|---------------------|---------------|
| ERA Drive | ✅ | ✅ UI TE-D features | TE sign-off [ ] |
| ERA Documents | ✅ | ✅ UI TE-DOC features | TE sign-off [ ]; GAP-TE-DOC04 gov samples |
| ERA Tables | ✅ tables-engine | ✅ UI TE-T features | TE sign-off [ ]; GAP-TE-T gov xlsx |
| ERA Presentations | ✅ engine | ✅ UI TE-P features | не блокер гос.; TE sign-off open |
| ERA Projects | ✅ docs-projects | ✅ UI TE-PR kanban | не блокер гос.; TE sign-off open |
| ERA Office AI | ✅ docs-ai stub | ✅ UI TE-AI assist | не блокер гос.; air-gap only |

---

## 2. TE-0 — Demo stand

| ID | Задача | Модуль | Критерий |
|----|--------|--------|----------|
| GAP-TE-00 | Единый runbook «показ за 30 мин» | `Office-Tech-Eval-Runbook.md` | [x] |
| GAP-TE-01 | Demo tenant + seed data | bootstrap SQL / script | [ ] |
| GAP-TE-02 | HTTPS profile для customer-like URL | deploy TLS | [ ] |

---

## 3. TE-1 — Drive UI (P0 polish)

| ID | Задача | Модуль | Критерий |
|----|--------|--------|----------|
| GAP-TE-D01 | Create folder в UI | `ui/drive` | [x] `createFolderBtn` + API POST `/folders` |
| GAP-TE-D02 | List children по папке | `ui/drive` | [x] navigate into folder |
| GAP-TE-D03 | Upload в выбранную папку | `ui/drive` | [x] `folder_id` on upload |
| GAP-TE-D04 | Breadcrumb / root navigation | `ui/drive` | [x] `#breadcrumb` |
| GAP-TE-D05 | Playwright folder scenario | `ui/office/e2e/drive.spec.ts` | [x] `drive create folder navigate upload breadcrumb` |

---

## 4. TE-2 — Documents scenarios (P1)

| ID | Задача | Модуль | Критерий |
|----|--------|--------|----------|
| GAP-TE-DOC01 | «New document» из Drive UI | `ui/drive` + workspace | [x] `newDocBtn` → POST `/api/v1/docs` + Open in Docs |
| GAP-TE-DOC02 | Import docx из UI | `ui/docs` | [x] `#importBtn` + `/api/v1/docs/import` |
| GAP-TE-DOC03 | Toolbar: heading, list, bold | `ui/docs` + `set_inline_marks` | [x] H1 / List / B (+ Playwright) |
| GAP-TE-DOC04 | Gov docx в testdata (когда есть) | `testdata/gov/` | [ ] blocked on samples |
| GAP-TE-DOC05 | E2E против live compose (не mock) | `run-office-e2e.ps1` | [ ] mocked `docs.spec.ts` есть; live compose отдельно |

---

## 5. TE-3 — Tables MVP (P2) — **критический путь**

| ID | Задача | Модуль | Критерий |
|----|--------|--------|----------|
| GAP-TE-T01 | `office.proto` — EratSheet wire model | `proto` / JSON `.erat` | [x] model EratSheet (JSON SSOT) |
| GAP-TE-T02 | `tables-engine` | Rust | [x] `services/platform/tables-engine` |
| GAP-TE-T03 | Grid + cell store + formula SUM/AVG | engine + UI | [x] calc + toolbar SUM |
| GAP-TE-T04 | Drive bind `.erat` | drive_bind | [x] put/get `.erat` |
| GAP-TE-T05 | xlsx import/export golden | convert + HTTP | [x] golden + `/import` + `/export/xlsx` |
| GAP-TE-T06 | WS co-edit cells | sync | [x] `ws_sheet_coedit` + cells snapshot |
| GAP-TE-T07 | `ui/tables` SPA + workspace `/tables` | `ui/tables` | [x] grid SPA + Playwright `tables.spec.ts` |
| GAP-TE-T08 | `office-tables` license gate | license | [x] fail-closed prod tests |
| GAP-TE-T09 | compose: tables routes health | compose/workspace | [x] workspace `/tables` + API proxy |
| GAP-TE-T10 | fuzz xlsx import | fuzz | [x] `fuzz_xlsx_smoke` (smoke) |

PRD scope: [`products/PRD-Office-P2.md`](products/PRD-Office-P2.md).

---

## 5b. TE-P — Presentations UI (P3, не блокер гос.)

| ID | Задача | Модуль | Критерий |
|----|--------|--------|----------|
| GAP-TE-P01 | `ui/presentations` slide editor | `ui/presentations` | [x] filmstrip + title/body |
| GAP-TE-P02 | New deck from Drive | `ui/drive` | [x] `newDeckBtn` → `/presentations/` |
| GAP-TE-P03 | pptx import/export HTTP + UI | engine + UI | [x] `/import` + `/export/pptx` |
| GAP-TE-P04 | Playwright presentations scenarios | `presentations.spec.ts` | [x] 4 mocked e2e |
| GAP-TE-P05 | Multi-slide pptx fidelity | convert | [ ] export first-slide subset only |

---

## 5c. TE-PR — Projects UI (не блокер гос.)

| ID | Задача | Модуль | Критерий |
|----|--------|--------|----------|
| GAP-TE-PR01 | Kanban SPA `/projects` | `ui/projects` | [x] 4 columns + Add task |
| GAP-TE-PR02 | Move / delete via API | `docs-projects` + UI | [x] POST upsert + DELETE |
| GAP-TE-PR03 | Drive deep-link in UI | `drive_object_id` | [x] Open in Docs |
| GAP-TE-PR04 | Playwright projects scenarios | `projects.spec.ts` | [x] create/move/delete |
| GAP-TE-PR05 | Workspace `ProjectsUI` mount | `cmd/workspace` | [x] wired |

---

## 5d. TE-AI — Office AI UI (не блокер гос.)

| ID | Задача | Модуль | Критерий |
|----|--------|--------|----------|
| GAP-TE-AI01 | Assist SPA `/office-ai` | `ui/office-ai` | [x] summarize + mode badge |
| GAP-TE-AI02 | Docs → Office AI handoff | `ui/docs` | [x] Summarize with AI |
| GAP-TE-AI03 | JSON body summarize | `docs-ai` | [x] `{ "text": "…" }` |
| GAP-TE-AI04 | Workspace mount + API proxy | workspace | [x] UI + `:8146` default |
| GAP-TE-AI05 | Playwright office-ai scenarios | `office-ai.spec.ts` | [x] stub + docs handoff |

---

## 6. TE-4 — Gov golden corpus

| ID | Задача | Критерий |
|----|--------|----------|
| GAP-TE-G01 | Обезличенный docx шаблон AZ/gov | golden PASS |
| GAP-TE-G02 | Обезличенный xlsx шаблон AZ/gov | golden PASS |
| GAP-TE-G03 | Disclaimer UI для unsupported features | [x] `#teBanner` dismissible + Help → About (Docs/Tables/Pres); legacy `#banner` on import |

---

## 7. TE-sign — Дистрибьютор

| ID | Задача | Критерий |
|----|--------|----------|
| GAP-TE-S01 | [`Office-Tech-Eval-Checklist.md`](Office-Tech-Eval-Checklist.md) подписан | [ ] |
| GAP-TE-S02 | RFQ/compare синхронизированы с фактическим scope | [ ] |
| GAP-TE-S03 | `editions-office.yaml` статусы честные | [ ] |

---

## 8. Команды

```powershell
.\scripts\run-office-pilot-staging.ps1 -UseCompose   # infra smoke
.\scripts\run-office-e2e.ps1 -UseCompose             # UI smoke
# TE-T: после появления tables-engine
cargo test -p era-docs-engine --features tables      # TBD
```
