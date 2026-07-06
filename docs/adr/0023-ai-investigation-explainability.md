# ADR-0023: AI Investigation Explainability & Audit Trail

**Статус:** Accepted (MVP investigate API реализован; forensic-grade trail — в развитии)
**Дата:** 2 июля 2026 г.
**Контекст:** В air-gap / госсекторе автономный вердикт ИИ без цепочки доказательств
недопустим: аудитор и SOC-аналитик должны ответить «на основании каких событий принято
решение». ADR-0002 описывает топологию обучения (Hub-and-Spoke), ADR-0006 ставит Agentic AI
как Vanga-ставку Фазы 3+, chain-of-custody есть для PAM (`platform/custody`) — но **связь
вердикта AI с неизменяемым evidence chain** не была зафиксирована. Текущий `ai-core`
investigate API возвращает storyline и verdict, однако это **triage accelerator**, не
forensic-grade reasoning.

**Связано:** [`ADR-0002`](0002-learning-topology.md) ·
[`ADR-0006`](0006-coverage-gaps-strategic-bets-and-practices.md) §2.2 ·
[`ADR-0013`](0013-era-pam-edition.md) (custody hashchain) ·
[`ADR-0017`](0017-vision-one-onprem-patterns.md) §1 (Workbench) ·
[`ADR-0022`](0022-detection-content-governance.md) ·
`services/ai-core/internal/investigate/` · `services/platform/custody/`

---

## Контекст и проблема

1. **Air-gap усиливает требование explainability.** Нет облачного «чёрного ящика» вендора —
   заказчик несёт ответственность за каждый вердикт перед регулятором.
2. **Текущий MVP — эвристика.** Verdict строится на keyword-hits в payload
   (`powershell`, `cmd.exe`, failed auth), не на ML-модели с feature attribution.
3. **Custody есть, но не привязан к AI.** Hash-chain (`platform/custody`) используется
   для PAM и forensic path, не для AI-расследований.
4. **Риск overclaim.** Маркетинг «AI SOC-аналитик» без audit trail разрушает доверие
   на пилоте быстрее, чем слабая детекция.

## Решение

### 1. Позиционирование (инвариант)

| Утверждение | Допустимо | Недопустимо |
|---|---|---|
| ИИ ускоряет triage аналитика | ✅ | — |
| ИИ даёт storyline + гипотезу | ✅ MVP | — |
| ИИ заменяет аналитика без review | ❌ | всегда |
| Вердикт юридически значим без custody | ❌ | до Фазы 2 |
| LLM narrative = доказательство | ❌ | narrative — вспомогательный текст |

**Human-on-the-loop:** каждый вердикт `malicious` / `suspicious` требует действия аналитика
(подтверждение, эскалация, закрытие как FP) — auto-case создаётся, но не закрывает инцидент
автономно.

### 2. Фазы зрелости

#### Фаза 1 — MVP (реализовано)

`POST /api/v1/investigate` (`services/ai-core/`):

| Выход | Описание |
|---|---|
| `storyline[]` | До 50 последних событий по `node_id`: event_id, category, observed_at, summary |
| `verdict` | `benign` / `suspicious` / `malicious` (эвристика по keyword-hits) |
| `confidence` | 0.55–0.92 |
| `mitre_techniques[]` | Эвристика из категорий событий (`inferMitre`) |
| `narrative` | Текст + опциональный LLM-блок (Ollama/vLLM **внутри контура**) |
| `case_id` | Авто-кейс при malicious/suspicious |

**Ограничения MVP (честно):**
- Нет immutable audit log вызова investigate;
- Нет привязки verdict → detection_id → event hash → custody seal;
- Нет визуального attack graph;
- Нет model version / prompt hash в ответе.

#### Фаза 2 — Forensic trail (target для гос/банк)

| Компонент | Решение |
|---|---|
| **Investigation record** | Неизменяемая запись: who, when, detection_id, node_id, model_version, prompt_hash |
| **Evidence chain** | `verdict` → `detection_ids[]` → `event_ids[]` → custody hash из `platform/custody` |
| **Workbench UI** | Граф цепочки: процесс → сеть → auth (reuse timeline API, ADR-0017 §1) |
| **Model pinning** | Версия LLM/weights фиксируется в air-gap; обновление — через signed `ai-pack` bundle |
| **Audit export** | PDF/JSON пакет для регулятора: вердикт + цепочка + хэши |

#### Фаза 3 — Agentic SOC (Vanga, ADR-0006)

- Автономные playbooks с human-on-the-loop;
- Обучение на подтверждённых кейсах внутри контура (ADR-0002);
- **Не раньше** закрытия Фазы 2 audit trail.

### 3. On-prem LLM (air-gap)

- Инференс **только внутри контура** заказчика (Ollama, vLLM, локальный кластер);
- LLM narrative **дополняет**, не заменяет storyline;
- При недоступности LLM — heuristic fallback (документирован в Pilot-Readiness-Checklist);
- Никаких внешних API (OpenAI, Anthropic и т.д.) в рантайме.

### 4. Связь с Chain of Custody

Переиспользовать `services/platform/custody/hashchain.go`:

```
investigate() → для каждого event_id в storyline → custody.Seal(event_hash)
→ investigation_record.custody_root_hash
```

PAM custody и AI custody — **единый механизм**, разные `record_type`.
Юридическая значимость — только после Фазы 2 + WORM-хранение (ADR-0004, ADR-0006).

### 5. Критерии приёмки

| ID | Критерий | Фаза | Доказательство |
|---|---|---|---|
| AI-1 | Investigate API возвращает storyline + verdict | 1 | `investigate_test.go`, pilot checklist |
| AI-2 | Auto-case на malicious/suspicious | 1 | `ai-core/internal/api/server.go` |
| AI-3 | LLM fallback documented | 1 | Pilot-Readiness-Checklist |
| AI-4 | Investigation immutable audit log | 2 | [ ] |
| AI-5 | Evidence chain: verdict → event custody hash | 2 | [ ] |
| AI-6 | Attack graph в Workbench UI | 2 | [ ] |
| AI-7 | Model version в investigation record | 2 | [ ] |
| AI-8 | Regulatory export pack | 2 | [ ] |

## Последствия

**Плюсы:**
- Честная граница MVP vs forensic снимает риск на пилоте в госсекторе;
- Переиспользование custody и Workbench — не «с нуля»;
- On-prem LLM остаётся дифференциатором без нарушения air-gap.

**Минусы / обязательства:**
- Фаза 2 — существенная разработка (хранилище audit, UI граф, export);
- Эвристический verdict Фазы 1 нужно явно маркировать в UI («предварительная гипотеза»);
- Agentic response (Фаза 3) блокируется до audit trail.

## Открытые вопросы

1. Минимальный набор полей для regulatory export в AZ (ЦБ, cert)?
2. Срок хранения investigation records vs retention lake (ADR-0004)?
3. Нужен ли отдельный модуль `ai-governance` или достаточно расширения `ai-core`?

## Связано

- [`ADR-0022`](0022-detection-content-governance.md) — MITRE на детекциях
- [`Pilot-Readiness-Checklist.md`](../Pilot-Readiness-Checklist.md)
- [`Production-Readiness-Assessment.md`](../Production-Readiness-Assessment.md) — AI-core field ~30%
