# ERA Communications — система контроля и приёмки

**Версия:** 2.1  
**Дата:** 30 июля 2026 г.  
**Статус:** Accepted  
**Канон:** [`ERA-Product-Acceptance-Standard.md`](ERA-Product-Acceptance-Standard.md) **v1.3**  
**Evidence:** [`Comms-Evidence-Rules.md`](../Comms-Evidence-Rules.md)  
**Product Readiness (SSOT готовности):** [`Comms-Product-Readiness-Matrix.md`](../Comms-Product-Readiness-Matrix.md)  
**AC Matrix (SSOT BE):** [`Comms-Implementation-Matrix.md`](../Comms-Implementation-Matrix.md)  
**Канон волн:** C-1…C-GA (+ R-*/partner) — [`Comms-Sprint-Index.md`](../Comms-Sprint-Index.md)  
**Аналоги:** [`Office-Acceptance-System.md`](Office-Acceptance-System.md) · [`Control-Acceptance-System.md`](Control-Acceptance-System.md)

---

## 1. Зачем

В любой момент команда и product owner должны видеть:

1. **Что обещали** (PRD / RFQ)
2. **Что реализовали** (код + тест)
3. **На какой стадии** (волна C-* / R-* / edition)
4. **Насколько соответствует ожиданиям** — только с доказательством

Без «зелёного в голове» ([`task-acceptance`](../../.cursor/rules/task-acceptance.mdc)).

**Правило:** «матрица готовности» → [`Comms-Product-Readiness-Matrix.md`](../Comms-Product-Readiness-Matrix.md). RT-09 open; `ga` только из yaml.

---

## 2. Слои (сверху вниз)

```
PRD-Comms-MVP / Migration / Bridge / Mail-Moderation / Gov  ← ожидания
        ↓
Comms-MVP-Spec         ← программа: волны F-C*, Definition of Done MVP/GA
        ↓
Stage specs            ← CM1-* … CM-GA-*, R-*, C-MM-* — Comms-Sprint-Index
        ↓
Comms-Implementation-Matrix        ← AC / Scaffold BE
Comms-Product-Readiness-Matrix     ← **один экран готовности** (UI+Demo/RT+Pilot)
        ↓
Comms-Evidence-Rules · stage-gate · Pilot / RT-09 / partner
```

| Слой | Файл | Вопрос |
|------|------|--------|
| **Product Readiness** | [`Comms-Product-Readiness-Matrix.md`](../Comms-Product-Readiness-Matrix.md) | *Готовность издания?* |
| AC / BE | [`Comms-Implementation-Matrix.md`](../Comms-Implementation-Matrix.md) | *PRD AC?* |
| Индекс | [`Comms-Sprint-Index.md`](../Comms-Sprint-Index.md) | *Gate волны?* |
| Пилот / RT | Pilot-Readiness · Gap · RT-09 | *Field?* |

**ADR-0027/0028** и donors — архитектура; **не** заменяют матрицу.  
Shared identity/CH: [`Shared-Acceptance-System.md`](Shared-Acceptance-System.md) — ссылаться, не дублировать Pilot-ready.

---

## 3. Легенда статусов

Как в [`ERA-Product-Acceptance-Standard.md`](ERA-Product-Acceptance-Standard.md) §3:

| Контекст | Маркеры |
|----------|---------|
| Backlog / Index | `[ ]` `[~]` `[x]` `[blocked]` |
| Matrix Scaffold | ✅ / 🟡 / [ ] |
| Matrix Pilot-ready | `[x]` / `[ ]` / ⏸ |
| Editions | `roadmap` → `scaffold`/`mvp` → `ga` |

**Правило:** `[x]` / ✅ только после доказательства. Обновлять **Comms-Implementation-Matrix** и stage spec в **одном PR** с кодом.

---

## 4. Рабочий процесс

### 4.1. Начали задачу

1. Backlog ID в stage-spec → `[~]`
2. В матрице — Scaffold 🟡

### 4.2. Закончили задачу

1. Тест или golden (парсеры, audit, autodiscover — обязательно)
2. `.\scripts\run-comms-stage-gate.ps1 -Stage C-X` — PASS (лог в `reports/`)
3. Матрица: Scaffold ✅ + команда proof; Pilot-ready — только RT/staging/field
4. [`Comms-Sprint-Index.md`](../Comms-Sprint-Index.md) / F-C* при необходимости (пока нет `Comms-MVP-Spec.md`)
5. [`editions-comms.yaml`](../../editions-comms.yaml) — только если edition/license изменился

### 4.3. Еженедельный статус

```powershell
.\scripts\run-comms-acceptance.ps1
```

Сводка + [`Comms-Implementation-Matrix.md`](../Comms-Implementation-Matrix.md) §«Сводка по изданиям».

### 4.4. Перед пилотом

[`Comms-Pilot-Readiness-Checklist.md`](../Comms-Pilot-Readiness-Checklist.md) → подпись.  
P0 из Gap-List закрыты или явно `[blocked]`.

### 4.5. Закрытие этапа (Stage Gate G1…G6)

1. Все backlog ID этапа → `[x]` в stage-spec.
2. `.\scripts\run-comms-stage-gate.ps1 -Stage C-X` — **PASS**.
3. E2E §4 — `reports/comms-stage-CX-e2e.log` (если есть).
4. Matrix + MVP-Spec / Sprint-Index.
5. `-WriteSignoff` → `reports/comms-stage-CX-signoff.md` (G6).

Следующий этап не стартует, пока gate предыдущего (или явно параллельного по Index) ≠ PASS.

---

## 5. Связь с Control / Office

| Control | Comms | Office |
|---------|-------|--------|
| F-GA-* | F-C* | F-O* |
| AC1–AC8 Sprint-1 | AC-C* / AC-MIG / AC-MM / AC-GOV | AC-O* / AC-T* |
| ADR-Implementation-Matrix | Comms-Implementation-Matrix | Office-Implementation-Matrix |
| ci-gates-stage10 | run-comms-stage-gate | run-office-stage-gate |
| Pilot-Readiness-Checklist | Comms-Pilot-Readiness-Checklist | Office-Pilot-Readiness-Checklist |

---

## 6. Документы

- [`ERA-Product-Acceptance-Standard.md`](ERA-Product-Acceptance-Standard.md)
- [`Comms-Product-Readiness-Matrix.md`](../Comms-Product-Readiness-Matrix.md)
- [`Comms-Implementation-Matrix.md`](../Comms-Implementation-Matrix.md)
- [`Comms-Sprint-Index.md`](../Comms-Sprint-Index.md)
- [`Comms-Evidence-Rules.md`](../Comms-Evidence-Rules.md)
- [`Comms-Pilot-Gap-List.md`](../Comms-Pilot-Gap-List.md)
