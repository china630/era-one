# ERA Office — Stage O-0 (PRD-P0 sync)

**Wave:** O-0  
**Версия:** 1.0  
**Дата:** 7 июля 2026 г.  
**Продукт:** ERA Office P0  
**PRD:** [`PRD-Office-P0.md`](products/PRD-Office-P0.md)  
**Предусловие:** Wave **O-GOV** gate = PASS

---

## 1. Цель этапа

> Синхронизировать техническую документацию P0: PRD-P0, AC-O0 table, ADR cross-links. **Не дублировать** O-GOV acceptance scaffold.

Этап закрыт, когда §7 Stage Gate = PASS.

## 2. Scope

### Входит

- [`PRD-Office-P0.md`](products/PRD-Office-P0.md) — финальный AC-O0 table
- Ссылки из [`PRD-Office-MVP.md`](products/PRD-Office-MVP.md) на P0/P1 split
- Gap-list mapping GAP-O-P0-* → волны O-1…O-6

### НЕ входит

- Дублирование O-GOV docs (уже созданы)
- Production код Drive

## 3. Backlog (OM0-*)

| ID | Задача | Артефакт | Статус |
|----|--------|----------|--------|
| OM0-1 | PRD-Office-P0 Accepted scope | `PRD-Office-P0.md` | [x] |
| OM0-2 | AC-O0 в Implementation-Matrix | `Office-Implementation-Matrix.md` | [x] |
| OM0-3 | MVP-Spec wave O-0 row | `Office-MVP-Spec.md` | [x] |

## 4. E2E-сценарий приёмки

1. O-GOV gate PASS.
2. PRD-Office-P0 содержит AC-O0-1…O0-7.
3. Gap-list §4 ссылается на волны O-1…O-6.

## 5. Критерии приёмки

| ID | Критерий | Доказательство | Статус |
|----|----------|----------------|--------|
| F-O-0-1 | PRD-P0 published | file exists | [x] |
| F-O-0-2 | Matrix AC-O0 rows | matrix §P0 | [x] |
| F-O-0-3 | No false `[x]` on O-GA | gap-list §P0-7 | [x] |

## 6. Stage Gate (обязательно перед закрытием)

| # | Проверка | Доказательство | Статус |
|---|----------|----------------|--------|
| G1 | `run-office-stage-gate.ps1 -Stage O-0` | PASS | [x] |
| G2 | E2E §4 | PRD-P0 + gap-list | [x] |
| G3 | Implementation-Matrix обновлена | PR diff | [x] |
| G4 | Office-MVP-Spec Wave O-0 → [x] | PR diff | [x] |
| G5 | editions — N/A | — | — |
| G6 | Signoff | `reports/office-stage-O-0-signoff.md` | [ ] |

## 7. Связано

- [`Office-Sprint-Index.md`](Office-Sprint-Index.md)
- [`Office-Pilot-Gap-List.md`](Office-Pilot-Gap-List.md)
