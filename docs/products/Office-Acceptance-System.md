# ERA Office — система контроля и приёмки

**Версия:** 2.2  
**Дата:** 30 июля 2026 г.  
**Статус:** Accepted  
**Канон:** [`ERA-Product-Acceptance-Standard.md`](ERA-Product-Acceptance-Standard.md) **v1.3**  
**Evidence:** [`Office-Evidence-Rules.md`](../Office-Evidence-Rules.md)  
**Product Readiness (SSOT готовности):** [`Office-Product-Readiness-Matrix.md`](../Office-Product-Readiness-Matrix.md)  
**AC Matrix (SSOT BE):** [`Office-Implementation-Matrix.md`](../Office-Implementation-Matrix.md)  
**Канон волн:** O-0…O-GA ([`Office-Sprint-Index.md`](../Office-Sprint-Index.md))  
**Аналоги:** [`Comms-Acceptance-System.md`](Comms-Acceptance-System.md) · [`Control-Acceptance-System.md`](Control-Acceptance-System.md)

---

## 1. Зачем

В любой момент команда и product owner должны видеть:

1. **Что обещали** (PRD / RFQ)
2. **Что реализовали** (код + тест)
3. **На какой стадии** (волна O-* / edition)
4. **Насколько соответствует ожиданиям** — только с доказательством

Без «зелёного в голове» ([`task-acceptance`](../../.cursor/rules/task-acceptance.mdc)).

**Правило:** «матрица готовности» → Product-Readiness-Matrix (не только BE). RT-O09 / TE открыты.

---

## 2. Слои (сверху вниз)

```
PRD-Office-MVP / P0 / P1 / P2 / P3 / Projects / AI  ← ожидания
        ↓
Office-MVP-Spec         ← программа: волны O-0…O-GA, F-O*
        ↓
Stage specs (O-0…O-GA)  ← OM*-* backlog — Office-Sprint-Index
        ↓
Office-Implementation-Matrix       ← AC / Scaffold BE
Office-Product-Readiness-Matrix    ← **один экран готовности** (UI+TE+Pilot)
        ↓
Office-Evidence-Rules
        ↓
scripts/run-office-stage-gate.ps1
        ↓
Office-Pilot-* · Office-Tech-Eval-*   ← доказательства слоёв Readiness
```

| Слой | Файл | Вопрос |
|------|------|--------|
| **Product Readiness** | [`Office-Product-Readiness-Matrix.md`](../Office-Product-Readiness-Matrix.md) | *Готовность издания?* |
| AC / BE | [`Office-Implementation-Matrix.md`](../Office-Implementation-Matrix.md) | *PRD AC на engine?* |
| Tech Eval proof | [`Office-Tech-Eval-Checklist.md`](../Office-Tech-Eval-Checklist.md) | *Demo TE-*? |
| UI baseline (Collab v2) | [`Office-UI-Baseline.md`](../Office-UI-Baseline.md) · [`Office-UI-Feature-Inventory.md`](../Office-UI-Feature-Inventory.md) · [`Office-UI-Design-System.md`](../Office-UI-Design-System.md) · [`Office-UI-Menu-Map.md`](../Office-UI-Menu-Map.md) · [`Office-OSS-Delta.md`](../Office-OSS-Delta.md) | *Target UX / controls?* |
| Пилот | [`Office-Pilot-Readiness-Checklist.md`](../Office-Pilot-Readiness-Checklist.md) | *Compose/field?* |
| Индекс | [`Office-Sprint-Index.md`](../Office-Sprint-Index.md) | *Gate волны?* |

**ADR-0025/0026** — архитектура; **не** заменяют матрицу.  
Drive/docs-engine ownership: [`Shared-Acceptance-System.md`](Shared-Acceptance-System.md) — Office owner для product AC.

---

## 3. Легенда статусов

Как в [`ERA-Product-Acceptance-Standard.md`](ERA-Product-Acceptance-Standard.md) §3:

| Контекст | Маркеры |
|----------|---------|
| Backlog / Index | `[ ]` `[~]` `[x]` `[blocked]` |
| Matrix Scaffold | ✅ / 🟡 / [ ] |
| Matrix Pilot-ready | `[x]` / `[ ]` / ⏸ |
| Editions | `roadmap` → `scaffold`/`mvp` → `ga` |

**Правило:** `[x]` / ✅ только после доказательства. Обновлять **Office-Implementation-Matrix** и stage spec в **одном PR** с кодом.

---

## 4. Рабочий процесс

### 4.1. Начали задачу

1. Backlog ID в stage-spec → `[~]`
2. В матрице — Scaffold 🟡

### 4.2. Закончили задачу

1. Тест или golden (proto, ACL, docx — обязательно)
2. `.\scripts\run-office-stage-gate.ps1 -Stage O-X` — PASS
3. Матрица: Scaffold ✅ + команда proof; Pilot-ready — staging/field
4. [`Office-MVP-Spec.md`](../Office-MVP-Spec.md) F-O* при необходимости
5. [`editions-office.yaml`](../../editions-office.yaml) / [`editions-shared.yaml`](../../editions-shared.yaml) — только если edition/license изменился

### 4.3. Еженедельный статус

```powershell
.\scripts\run-office-acceptance.ps1
```

### 4.4. Перед пилотом

[`Office-Pilot-Readiness-Checklist.md`](../Office-Pilot-Readiness-Checklist.md) → подпись.

### 4.5. Закрытие этапа (Stage Gate G1…G6)

1. Backlog этапа → `[x]` в stage-spec.
2. `.\scripts\run-office-stage-gate.ps1 -Stage O-X` — **PASS**.
3. E2E — `reports/office-stage-OX-e2e.log` (если есть).
4. Matrix + MVP-Spec / Sprint-Index.
5. `-WriteSignoff` → `reports/office-stage-OX-signoff.md` (G6).

Следующий этап не стартует, пока gate предыдущего ≠ PASS ([`Office-Sprint-Index.md`](../Office-Sprint-Index.md) §3).

---

## 5. Связь с Control / Communications

| Control | Comms | Office |
|---------|-------|--------|
| F-GA-* | F-C* | F-O* |
| AC1–AC8 Sprint-1 | AC-C* | AC-O* / AC-T* |
| ADR-Implementation-Matrix | Comms-Implementation-Matrix | Office-Implementation-Matrix |
| ci-gates-stage10 | run-comms-stage-gate | run-office-stage-gate |
| Pilot-Readiness-Checklist | Comms-Pilot-Readiness-Checklist | Office-Pilot-Readiness-Checklist |

---

## 6. Документы

- [`ERA-Product-Acceptance-Standard.md`](ERA-Product-Acceptance-Standard.md)
- [`Office-Product-Readiness-Matrix.md`](../Office-Product-Readiness-Matrix.md)
- [`Office-Implementation-Matrix.md`](../Office-Implementation-Matrix.md)
- [`Office-MVP-Spec.md`](../Office-MVP-Spec.md)
- [`Office-Sprint-Index.md`](../Office-Sprint-Index.md)
- [`Office-Tech-Eval-Checklist.md`](../Office-Tech-Eval-Checklist.md)
- [`Office-Pilot-Gap-List.md`](../Office-Pilot-Gap-List.md)
