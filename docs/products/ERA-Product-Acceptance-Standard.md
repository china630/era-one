# ERA One — Product Acceptance Standard (канон)

**Версия:** 1.3  
**Дата:** 30 июля 2026 г.  
**Статус:** Accepted — AC Matrix ≠ Product Readiness; один экран готовности на линейку  
**Refs:** ADR-0024 · ADR-0025 · `.cursor/rules/task-acceptance.mdc`  
**Consistency check:** `.\scripts\check-acceptance-consistency.ps1`

---

## 1. Зачем

В любой момент команда и product owner должны видеть по **каждому** продукту:

1. **Что обещали** (PRD / RFQ / AC-*)
2. **Что реализовали** (код + тест)
3. **На какой стадии** (волна / edition)
4. **Насколько соответствует ожиданиям** — только с доказательством

Без «зелёного в голове». Правило: **нет лога / CI artifact — нет gate/`[x]`**.

**Инварианты:**

1. scaffold-gate ≠ pilot / field sign-off  
2. stage `gate[x]` ≠ AC Scaffold ✅  
3. **AC Scaffold Matrix ≠ Product Readiness** (UI / demo / field)  
4. шапка / PRD / signoff **не** красят продукт зелёным поверх Readiness 🟡/❌  

---

## 2. Обязательный стек (на продукт или крупное издание)

```
PRD (+ AC-*)
        ↓
Program / MVP-Spec (F-*)          ← шапка = Product Readiness rollup
        ↓
Sprint-Index (Gate | AC rollup | Pilot-ready)
        ↓
Stage specs (backlog ID, G1…G6)
        ↓
Implementation-Matrix             ← SSOT **AC / Scaffold BE**
        ↓
Product-Readiness-Matrix          ← SSOT **готовности продукта** (один экран)
        ↓
Evidence-Rules
        ↓
scripts/run-<product>-stage-gate.ps1  (+ check-acceptance-consistency.ps1)
        ↓
Pilot-Readiness-Checklist + Pilot-Gap-List + Demo/TE sources
```

| Слой | Вопрос |
|------|--------|
| PRD | *Что продаём?* |
| Program Spec | *Какими волнами?* |
| Sprint-Index | *Gate / AC / Pilot?* |
| **Implementation-Matrix** | *Закрыт ли PRD AC на API/engine?* |
| **Product-Readiness-Matrix** | *Можно ли показывать / пилотировать / продавать?* |
| Evidence-Rules | *Можно ли ✅ / `ga`?* |
| Gap / Pilot / TE | *Доказательства слоёв Readiness* |

---

## 3. Легенда статусов (единая)

### 3.1. Три уровня (не смешивать)

| Уровень | Где | Маркер | Значение |
|---------|-----|--------|----------|
| **Gate** | Sprint-Index / stage-spec | `gate[x]` / `[~]` / `[ ]` | stage-gate log PASS |
| **AC Scaffold** | Implementation-Matrix | ✅ / 🟡 / `[ ]` | PRD intent + negative path |
| **Product ready** | Matrix Pilot-ready + editions | `[x]` / `[ ]` / ⏸ · `mvp`/`ga` | staging/field / edition |

Legacy: одиночный `[x]` в Index **допустим только** как синоним `gate[x]` и **обязан** сопровождаться колонками AC rollup и Pilot-ready.  
**Запрещено:** один столбец «Статус `[x]`» без уточнения уровня.

Backlog ID: `[ ]` · `[~]` · `[x]` (задача закрыта с proof) · `[blocked]`.

### 3.2. Implementation-Matrix (колонки готовности)

| Колонка | Маркер | Значение |
|---------|--------|----------|
| **Scaffold** | ✅ / 🟡 / `[ ]` | Soft proof; не поле |
| **Pilot-ready** | `[x]` / `[ ]` / ⏸ | Staging или field PASS / подпись |

**Scaffold ✅ только если одновременно:**

1. Proof покрывает **формулировку PRD AC** (включая negative path: spoof → 401/403, deny → заявленный эффект, stub → явный `mode=stub`).  
2. **Worst-component:** нет подкомпонента того же AC со статусом 🟡 / `[ ]`.  
3. Нет открытого **Critical residual** в том же контуре AC.  
4. AC **не** является field-intent (см. ниже).

Иначе — **🟡**, даже при зелёном stage-gate.

**No soft-green on field AC:** если формулировка требует customer/prod/field (loadgen на кластере, coverage на пилоте, подпись checklist, WHQL) — Scaffold максимум **🟡** (artifact/script/template есть), **не** ✅. Pilot-ready остаётся `[blocked]` / `[ ]` до поля.

Stage-gate `gate[x]` ≠ AC Scaffold ✅.

### 3.3. Edition honesty (`editions-*.yaml` — единственный SSOT статуса издания)

| status | Когда |
|--------|--------|
| `roadmap` | Нет product-ready proof / `exists: false` |
| `scaffold` | Каркас без полного gate |
| `mvp` | Unit/gate PASS; **не** field |
| `ga` | Pilot-ready + field / partner sign-off |
| `ga-option` | Soft GA по отдельной лицензии (только Control) |
| `software-ga` | Только carve-out §3.6 (Core / AI / Response) |

Ложный `ga` без Pilot-ready **запрещён**.

**Слово `ga` вне yaml:** Partner Bundle, PRD, RFQ, signoff, HTML — либо копируют `status` из `editions-*.yaml` / `products.yaml`, либо пишут «см. editions», либо **нарушают стандарт**.  
Запрещены формулировки `ga (partner)`, `ga (greenfield)` при yaml = `mvp`.

### 3.4. Два SSOT (не смешивать)

| SSOT | Файл | Вопрос |
|------|------|--------|
| **AC / Scaffold** | `*-Implementation-Matrix.md` | Закрыт ли PRD AC (BE/API/engine)? |
| **Product Readiness** | `*-Product-Readiness-Matrix.md` | Готов ли **продукт** (UI, demo, pilot, sell)? |

```
Запрос «матрица готовности» / «готовность линейки» / «можно ли продавать/показывать»
  → только Product-Readiness-Matrix (все колонки).

Запрос «Scaffold / AC / backend matrix»
  → Implementation-Matrix.
```

**Product Readiness — обязательные колонки:**

| Издание | Gate | Scaffold BE | UI* | Demo / TE* | Pilot lab | Pilot field | Edition | Sell / show |

`*` = обязательна, если у линейки есть UX/демо; иначе `n/a` с причиной.  
**Office:** Demo/TE = Tech-Eval-Checklist.  
**Comms:** Demo = staging RT / partner smoke (отдельного TE может не быть).  
**Control:** Demo = SOC/lab walkthrough / WHQL gate; UI = portal где есть.

```
Product_edition = worst(Gate, BE, UI*, Demo*, Pilot lab, Pilot field)
  порядок: [ ] / ❌ < 🟡 < ✅
  (⏸ external не даёт «зелёный» sell)

AC_Scaffold_edition = worst(Scaffold AC) — только для BE-слоя.

Запрещено:
  называть Implementation-Matrix «матрицей готовности продукта»;
  «all ✅» / «продукт готов», если Product_edition ≠ ✅;
  отвечать на «готовность» одним столбцом Scaffold BE.
```

Шапка MVP-Spec / Index honesty / PRD «Статус:» копируют **Product Readiness** rollup (или ссылаются на него).

### 3.5. Consistency (один PR)

При смене AC Scaffold **или** UI/TE/Pilot статуса — в том же PR:

1. Implementation-Matrix (если AC)  
2. **Product-Readiness-Matrix** сводка  
3. Sprint-Index / MVP-Spec шапка  
4. Pilot-Gap / TE-Gap резюме при необходимости  
5. Partner / RFQ / Product-Line, если `ga` / «готов»

Проверка: `.\scripts\check-acceptance-consistency.ps1`.

### 3.6. Carve-out: software GA (только Control Core / AI / Response)

Исторический статус продаж: `era-core`, `era-control-ai`, `era-response` могут оставаться **software-ga** / `ga` в yaml при закрытом soft DoD.

Ограничения carve-out:

- действует **только** на эти три издания;  
- field-AC (F-GA-5/8/15 и аналоги) в Matrix = **🟡** soft artifact + Pilot **[blocked]** — **не** Scaffold ✅;  
- новые издания (Manage, Perimeter, Resolve, Comms, Office, …) — **без** carve-out; `ga` только после Pilot-ready.

---

## 4. Stage Gate (G1…G6) — шаблон

| # | Проверка | Доказательство |
|---|----------|----------------|
| G1 | Авто-тесты этапа | `run-<product>-stage-gate.ps1 -Stage <wave>` PASS |
| G2 | E2E (если есть) | `reports/<product>-stage-<wave>-e2e.log` |
| G3 | Matrix обновлена; **нет** prose «Matrix Scaffold ✅», если rollup волны 🟡 | PR diff Matrix |
| G4 | Index: **gate[x]** + AC rollup из Matrix (не `all ✅`) | PR diff Index / Program Spec |
| G5 | editions (только если license/deploy) | licensegate / manifest test |
| G6 | Signoff уровня gate | `reports/<product>-stage-<wave>-signoff.md` — имя/текст = `scaffold-gate-pass`, **не** product-green, если Pilot open |

Следующая волна не стартует, пока **gate** предыдущей ≠ PASS (параллель — по Index).

---

## 5. Рабочий процесс задачи

### Начали

1. Backlog ID → `[~]`  
2. В матрице — Scaffold 🟡  

### Закончили

1. Тест / golden  
2. Stage-gate → `gate[x]` + лог  
3. Matrix: ✅ только по §3.2; иначе 🟡; Pilot-ready только после staging/field  
4. Синхронизация §3.5 (Index / MVP / Gap / PRD)  
5. `editions-*.yaml` — только если edition/license изменился  
6. `.\scripts\check-acceptance-consistency.ps1` — PASS  

### Перед пилотом / GA

Pilot checklist → подпись; Gap P0 закрыт или `[blocked]` с владельцем; edition `ga` только при Pilot-ready.

---

## 6. Карта продуктов → документы

| Продукт | Acceptance-System | **Product Readiness (SSOT готовности)** | **AC Matrix (SSOT BE)** | Gate |
|---------|-------------------|------------------------------------------|-------------------------|------|
| **Control** | [`Control-Acceptance-System.md`](Control-Acceptance-System.md) | [`Control-Product-Readiness-Matrix.md`](../Control-Product-Readiness-Matrix.md) | [`Control-Implementation-Matrix.md`](../Control-Implementation-Matrix.md) | `ci-gates-stage10` · `run-ga-full` |
| **Communications** | [`Comms-Acceptance-System.md`](Comms-Acceptance-System.md) | [`Comms-Product-Readiness-Matrix.md`](../Comms-Product-Readiness-Matrix.md) | [`Comms-Implementation-Matrix.md`](../Comms-Implementation-Matrix.md) | `run-comms-stage-gate` |
| **Office** | [`Office-Acceptance-System.md`](Office-Acceptance-System.md) | [`Office-Product-Readiness-Matrix.md`](../Office-Product-Readiness-Matrix.md) | [`Office-Implementation-Matrix.md`](../Office-Implementation-Matrix.md) | `run-office-stage-gate` |
| **Shared** | [`Shared-Acceptance-System.md`](Shared-Acceptance-System.md) | через Office/Control Readiness (Drive UI → Office) | ADR §0025 + consumer AC | package tests |

[`ADR-Implementation-Matrix.md`](../ADR-Implementation-Matrix.md) — ADR→код; не Readiness и не AC rollup продукта.

Продажи: [`ERA-Product-Line.md`](../distributor/ERA-Product-Line.md) не противоречит §3.3–3.4.

---

## 7. Shared platform — ownership

| Компонент | Scaffold proof | Product AC owner |
|-----------|----------------|------------------|
| identity, tenant, licensegate, adminportal | ADR-0025 / Control matrix | Shared + consumers |
| drive, docs-engine, workspace, signing | Office matrix (AC-O*) | **Office** |
| Comms mail / CH hooks | Comms matrix | **Comms** |

Без двойного Pilot-ready `[x]`.

---

## 8. Cursor / агент

Обновлять Acceptance-System **того** продукта (см. `.cursor/rules/task-acceptance.mdc`).  
Запрещено закрывать задачу только через `MVP-Sprint-1-Spec.md` для Comms/Office/Shared.  
**Scaffold-Green / green signoff ≠ приёмка продукта.**  
**«Матрица готовности» → Product-Readiness-Matrix, не Implementation-Matrix.**

### 8.1. Tooling (обязательная обвязка)

| Слой | Артефакт | Назначение |
|------|----------|------------|
| Project rule | `.cursor/rules/task-acceptance.mdc` | Matrix SSOT + consistency |
| Hooks | `.cursor/hooks.json` | block force-push / hard reset / stash nudge; stop-closeout checklist |
| Skill | `.cursor/skills/acceptance-closeout/` | закрытие задачи по v1.3 |
| Skill | `.cursor/skills/quality-gates/` | lint / secrets / vuln / e2e / dep-graph |
| Local | `scripts/run-quality-gates.ps1` | один вход для агента/человека |
| Consistency | `scripts/check-acceptance-consistency.ps1` | CI job `acceptance-consistency` |
| Secrets | `.gitleaks.toml` + CI `secrets` | утечки ключей/ПД |
| Lint | `.golangci.yml` + clippy (scoped) | CI job `lint` |
| Vuln | `Deny.toml` + govulncheck | CI job `vuln` |
| E2E | `ui/office/e2e` Playwright | CI job `e2e-office-smoke` |
| Dep graph | `scripts/export-dep-graph.ps1` → `reports/deps/` | blast radius без LLM |
| Mutation (optional) | `scripts/run-mutation-authz.ps1` | ловля фейковых AuthZ-тестов |

Агент **не** заменяет CI: hooks/skills ускоряют цикл внутри fail-closed рельс.  
Human-on-loop остаётся для Pilot-ready / WHQL / field.

---

## 9. Honesty audit / history

- [`Acceptance-Honesty-Audit-20260730.md`](../Acceptance-Honesty-Audit-20260730.md)  
- v1.2: rollup SSOT, gate vs AC, ban prose `ga` / `all ✅`, field-AC max 🟡  
- v1.2+tooling: Cursor hooks/skills + CI (см. §8.1)  
- **v1.3:** Product-Readiness-Matrix SSOT на Control/Comms/Office; запрос «готовность» ≠ BE matrix  

