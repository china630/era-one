# ERA Office — Tech Eval Checklist (гос. технарь)

> **Статус:** Active — главный gate до дистрибьютора  
> `[x]` только после живой демонстрации на стенде (не unit-тест в CI)

**Технарь:** ____________________  
**Дата eval:** ____________________  
**Стенд:** lab compose / customer contour  
**Версия сборки:** ____________________  
**Подпись «готов рекомендовать дистру»:** ____________________

**Стратегия:** [`Office-Tech-Eval-Strategy.md`](Office-Tech-Eval-Strategy.md)  
**Runbook:** [`Office-Pilot-Runbook.md`](Office-Pilot-Runbook.md) + TE-0 ниже

---

## TE-0 — Поднять стенд (обязательно)

- [ ] Полный стенд:
  ```powershell
  docker compose -f deploy/docker-compose.office.yml `
    --profile docs --profile office-engines up -d --wait --build
  .\scripts\office-te-connectivity.ps1 -RequireEngines
  ```
  (без `--profile docs` / `office-engines` Docs/Tables/Pres/Projects/AI не поднимутся)
- [ ] Workspace: `http://127.0.0.1:8170/login` → `alice@mail.gov.az` / `1234` (или Create account)
- [ ] Demo tenant `t-demo`; API только `Authorization: Bearer` (не голые `X-ERA-*`)
- [ ] Лог: `reports/office-te-connectivity-*.log` или свой TE log

---

## TE-D — ERA Drive (файлы — лицо продукта)

> UI features (folders/breadcrumb/upload/versions) реализованы + Playwright `drive.spec.ts`.  
> `[x]` ниже — только после **живой** демонстрации технарём на стенде.

- [ ] **TE-D01** Upload файла через UI → виден в списке
- [ ] **TE-D02** Создать папку → переместить/загрузить в папку → список корректен
- [ ] **TE-D03** Скачать файл → содержимое совпадает
- [ ] **TE-D04** Версия файла (повторный upload / версии API) — объяснимо пользователю
- [ ] **TE-D05** Второй пользователь / другой tenant — нет доступа к чужому файлу (ACL)
- [ ] **TE-D06** После `restart drive-api` — файлы на месте (RT-O07)

**Комментарий технаря:** ____________________

---

## TE-DOC — ERA Documents (текст, не Word)

> UI features (New from Drive, toolbar H1/List/Bold, import/export) реализованы + Playwright `docs.spec.ts`.  
> `[x]` ниже — только после **живой** демонстрации технарём на стенде.

- [ ] **TE-DOC01** Создать документ из Drive / Workspace → открывается редактор
- [ ] **TE-DOC02** Два браузера / два пользователя — одновременное редактирование видно
- [ ] **TE-DOC03** Import docx (простой шаблон) → редактирование → export docx  
  _(lab: повторный Import не должен давать 502 из‑за фиксированного `import.erad` — unique Drive name)_
- [ ] **TE-DOC04** Import **гос./корп. шаблона** (когда будет в corpus) — приемлемое качество
- [ ] **TE-DOC05** Документ только в Drive; после restart docs-engine — сессия/файл целы

**Ограничения (честно):** не Word — см. [`ERA-Documents-vs-Word.md`](ERA-Documents-vs-Word.md)

**Комментарий технаря:** ____________________

---

## TE-T — ERA Tables (**блокер для гос. показа**)

> UI/engine features (grid, SUM, xlsx I/O, Drive New sheet) реализованы + Playwright `tables.spec.ts`.  
> До **живой** подписи TE-T01…T05 технарём **нельзя** рекомендовать продукт как «Office для гос.»  
> `[x]` ниже — только после демонстрации на стенде.

- [ ] **TE-T01** Создать таблицу `.erat` → открывается grid-редактор (не stub)
- [ ] **TE-T02** Ввод данных в ячейки, навигация клавиатурой
- [ ] **TE-T03** Формула SUM (и минимум ещё одна) пересчитывается
- [ ] **TE-T04** Import xlsx (простой лист) → edit → export → golden match
- [ ] **TE-T05** Два пользователя — правка разных ячеек без потери данных
- [ ] **TE-T06** Import **гос./корп. xlsx** (когда будет в corpus)

**Ограничения:** см. [`ERA-Tables-vs-Excel.md`](ERA-Tables-vs-Excel.md)

**Комментарий технаря:** ____________________

---

## TE-P — ERA Presentations (не блокер гос. показа)

> UI features (New deck, slides, pptx import/export) реализованы + Playwright `presentations.spec.ts`.  
> O-FMT-3: Duplicate slide, Format bold/align/font±, Insert image (URL) — gate PASS; live TE ниже.  
> `[x]` ниже — только после **живой** демонстрации технарём. Export pptx = first-slide subset.

- [ ] **TE-P01** Создать `.erap` из Drive / Workspace → открывается редактор слайдов
- [ ] **TE-P02** Редактировать title/body, Add slide, навигация Previous/Next
- [ ] **TE-P03** Import pptx (простой) → edit → Export pptx (subset)
- [ ] **TE-P04** Честный disclaimer: не PowerPoint (анимации/макеты out of scope)
- [ ] **TE-P05** O-FMT-3: Duplicate + Format text/font± + Insert image (URL)

---

## TE-PR — ERA Projects (не блокер гос. показа)

> UI kanban (CRUD + Drive deep-link) реализован + Playwright `projects.spec.ts`.  
> `[x]` ниже — только после **живой** демонстрации. Не MS Project / Gantt.

- [ ] **TE-PR01** Открыть `/projects` → board backlog/todo/doing/done
- [ ] **TE-PR02** Создать задачу → видна в backlog
- [ ] **TE-PR03** Переместить задачу по колонкам
- [ ] **TE-PR04** Задача с `drive_object_id` → ссылка Open in Docs
- [ ] **TE-PR05** New project из Drive → файл `.eraj` → `/projects/{id}` → задача сохраняется после refresh

---

## TE-AI — ERA Office AI (не блокер гос. показа)

> Assist UI (`/office-ai`) + stub summarize реализованы + Playwright `office-ai.spec.ts`.  
> `[x]` ниже — только после **живой** демонстрации. Cloud SaaS LLM — out of scope.

- [ ] **TE-AI01** Открыть `/office-ai` → paste text → Summarize → `mode=stub` summary
- [ ] **TE-AI02** Documents «Summarize with AI» → текст переносится в Office AI
- [ ] **TE-AI03** Честный air-gap banner (no phone-home)
- [ ] **TE-AI04** (опц.) In-contour Ollama когда `ERA_OLLAMA_URL` задан

---

## TE-X — Интеграция и контур

- [ ] **TE-X01** Mail attach → ссылка на файл Drive (если Comms в стенде)
- [ ] **TE-X02** Prod license: без `ERA_OFFICE_DEV` — 403 на Drive/Docs/Tables
- [ ] **TE-X03** Нет исходящих вызовов в интернет с хоста продукта (customer verify)
- [ ] **TE-X04** SBOM: zero GPL runtime — отчёт `office-sbom-gate`

---

## Sign-off

| Вопрос | Да / Нет |
|--------|----------|
| Можно показывать **Drive** гос. заказчику? | |
| Можно показывать **Documents** (с оговоркой lite Word)? | |
| Можно показывать **Tables**? | |
| Рекомендую дистрибьютору открытое предложение (какое издание: MVP / +Tables)? | |
| Блокеры (список): | |

**Подпись технаря:** ____________________ **Дата:** __________
