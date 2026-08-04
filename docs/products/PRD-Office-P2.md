# PRD: ERA Office — P2 (ERA Tables)

**Статус:** Draft — **Tech Eval critical path**  
**Дата:** 8 июля 2026 г.  
**Продукт:** ERA Office (ERA One)  
**ADR:** [`0026`](../adr/0026-sovereign-office-engine.md) · [`0025`](../adr/0025-era-one-shared-platform.md)  
**Приёмка:** [`Office-Tech-Eval-Checklist.md`](../Office-Tech-Eval-Checklist.md) TE-T* · [`Office-Tech-Eval-Gap-List.md`](../Office-Tech-Eval-Gap-List.md)

**Предусловие:** P0 Drive + Workspace deployable; P1 Documents engine patterns (Drive bind, WS sync, license).

---

## 1. Цель P2 (Gov Tech Eval)

Дать **живые таблицы** в изолированном контуре — минимум для гос. технаря: «это уже Excel-уровень для типовых отчётов», не полный Excel.

**Native format:** `.erat` = wire type `erat` = `DocumentFormat.ERAT`.

**Не в P2 (честно):** макросы VBA, сводные таблицы, Power Query, 400+ функций Excel, Presentations (P3).

---

## 2. Scope — Gov Eval MVP

| # | Capability | Компонент |
|---|------------|-----------|
| 1 | Create / open `.erat` spreadsheet | `tables-engine` (или модуль в `docs-engine`) |
| 2 | Grid UI: ячейки, строки, столбцы (лимит MVP: напр. 256×1024) | `ui/tables` |
| 3 | Формулы: `SUM`, `AVERAGE`, `MIN`, `MAX`, ссылки `A1`, `B2:B10` | calc engine |
| 4 | Co-edit 2+ пользователей (ячейки / диапазоны) | WS sync |
| 5 | Import/export **xlsx** (один лист, без макросов) | Rust OOXML, golden |
| 6 | Authoritative storage только в **ERA Drive** | drive bind |
| 7 | License `office-tables` → 403 без модуля | `licensegate` |
| 8 | Workspace route `/tables` — не stub | `workspace` |

---

## 3. Критерии приёмки (AC-T1…T8)

| ID | Критерий | Доказательство |
|----|----------|----------------|
| AC-T1 | Создать `.erat`, сохранить в Drive, reopen | integration + TE-T01 |
| AC-T2 | SUM по диапазону пересчитывается при изменении ячейки | unit calc + TE-T03 |
| AC-T3 | xlsx из `testdata/` → native → export → golden | `golden_xlsx` |
| AC-T4 | Два клиента — правка разных ячеек без конфликта | WS test + TE-T05 |
| AC-T5 | Blob только в Drive | drive_bind pattern |
| AC-T6 | Zero GPL в P2 runtime | SBOM gate |
| AC-T7 | Без `office-tables` — create/open 403 | licensegate test |
| AC-T8 | Fuzz xlsx import не паникует | `fuzz_xlsx_import` (post-MVP ok) |

---

## 4. Волны (исполнение)

| Wave | Spec (создать) | Фокус |
|------|----------------|-------|
| **O2-GOV** | extend matrix/gap | TE-T rows |
| **O2-1** | `Office-Stage-O2-1-Proto-Spec.md` | `EratSheet` proto |
| **O2-2** | `Office-Stage-O2-2-Calc-Spec.md` | formula engine |
| **O2-3** | `Office-Stage-O2-3-Xlsx-Spec.md` | xlsx I/O golden |
| **O2-4** | `Office-Stage-O2-4-Sync-Spec.md` | WS cell ops |
| **O2-5** | `Office-Stage-O2-5-TablesUI-Spec.md` | `ui/tables` |
| **O2-6** | deploy + license | compose + gate |
| **O2-TE** | Tech Eval | TE-T checklist PASS |

Gate: `run-office-stage-gate.ps1 -Stage O2-*` (добавить по мере реализации).

---

## 5. Donor strategy (идеи, не код)

- **ironcalc** — паттерн spreadsheet engine (ADR-0026)
- **docs-engine** — Drive bind, persist, WS (повторное использование)
- OOXML xlsx — свой Rust parser (как docx в P1)

---

## 6. Риски

| Риск | Mitigation |
|------|------------|
| Scope creep «как Excel» | Жёсткий MVP AC-T1…T7; [`ERA-Tables-vs-Excel.md`](../ERA-Tables-vs-Excel.md) |
| Сроки | Параллель: TE-1 Drive UI не блокирует O2-1 proto |
| Gov xlsx fidelity | TE-4 golden corpus когда появятся шаблоны |

---

## 7. Связано

- [`ERA-Tables-vs-Excel.md`](../ERA-Tables-vs-Excel.md)
- [`Office-Tech-Eval-Strategy.md`](../Office-Tech-Eval-Strategy.md)
- [`editions-office.yaml`](../../editions-office.yaml) — `era-tables`
