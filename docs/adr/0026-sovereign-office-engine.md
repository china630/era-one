# ADR-0026: Sovereign Office Engine (native-first, zero GPL runtime)

**Статус:** Accepted  
**Дата:** 5 июля 2026 г.  
**Контекст:** ERA Office требует суверенный офисный движок без лицензий OnlyOffice/Collabora
и **без GPL-компонентов в runtime** (в т.ч. LibreOffice headless).

**Связано:** [`ADR-0003`](0003-repository-structure-and-donor-strategy.md) · [`ADR-0025`](0025-era-one-shared-platform.md) ·
[`editions-office.yaml`](../../editions-office.yaml)

---

## 1. Решение

**Свой office engine:** native-форматы + CRDT co-editing + **Rust-парсеры** import/export
OOXML (docx/xlsx/pptx). **OnlyOffice, Collabora, LibreOffice headless — out of scope.**

| Слой | Реализация |
|------|------------|
| **Native truth** | `.era-doc`, `.era-sheet`, `.era-deck` (proto + golden tests) |
| **Co-editing** | CRDT (Yjs/Automerge **паттерн**; своя интеграция) + Rust sync в `platform/docs-engine` |
| **Documents UI** | Block editor (ProseMirror **паттерн**) в ERA Workspace |
| **Tables** | Rust calc engine (grid, formulas; ironcalc **паттерн**) |
| **Presentations** | Phase 3 (после Documents + Tables) |
| **Import/export** | Rust OOXML parsers; **zero GPL** в runtime |
| **Storage** | Файлы только в **ERA Drive**; engine не хранит authoritative copy |

---

## 2. Отклонённые альтернативы

| Вариант | Причина отклонения |
|---------|-------------------|
| OnlyOffice Document Server | Лицензия / AGPL; платежи третьим сторонам |
| OnlyOffice WASM | Тяжёлый R&D; не «лёгкий» движок |
| Collabora Online | Лицензия + слабее OOXML fidelity |
| LibreOffice headless convert | GPL в runtime — **запрещено** решением продукта |
| Свой полный OOXML editor с нуля без этапов | Нереалистичный MVP |

---

## 3. MVP fidelity (критерий приёмки)

Import docx/xlsx для **типовых гос/корпоративных шаблонов AZ** без критичных сдвигов
layout; явный disclaimer для legacy/macro-heavy файлов. Критерий закрывается **golden-тестами**
(`testdata/`): фикс. вход → canonical native → export → compare с golden.

---

## 4. Фазы

| Фаза | Срок (ориентир) | Deliverable |
|------|-----------------|-------------|
| **P0** | 6–9 мес | Shared platform P0: identity, Drive, Workspace shell |
| **P1** | +9–12 мес | **ERA Documents**: CRDT, basic formatting, co-edit, docx I/O |
| **P2** | +6 мес | **ERA Tables**: formulas, co-edit, xlsx I/O |
| **P3** | +12 мес | **ERA Presentations** (минимум) |
| **P4** | ongoing | Fidelity, ERA Sign, ERA Office AI |

**ERA Projects** — roadmap издание (4-е в Office), не в MVP.

---

## 5. Издания ERA Office

См. [`editions-office.yaml`](../../editions-office.yaml):

- **ERA Documents**, **ERA Tables**, **ERA Presentations**, **ERA Projects** (roadmap)
- Bundle **ERA Office Suite** = Drive + Documents + Tables + Presentations + Projects
  (**Drive всегда включён** в suite)

---

## 6. Donor strategy (идеи, не код)

- **Zitadel** — OIDC, multi-tenant (identity; ADR-0025)
- **oCIS** — microservices file platform (Drive)
- **ProseMirror / CRDT** — editing & sync patterns
- **ironcalc** — spreadsheet engine patterns

---

## 7. Последствия

**Плюсы:** полный суверенитет, единый Rust/Go stack, co-editing в platform.

**Минусы:** более длинный time-to-market vs OnlyOffice; команда engine; ongoing OOXML support.

**Обязательства:** golden + fuzz на парсеры; license gate `office-documents`, `office-tables`, …
