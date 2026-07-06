# ADR-0022: Detection Content Governance (Sigma, MITRE, TI, FP)

**Статус:** Accepted (MVP реализован; процесс и зрелые workflow — в развитии)
**Дата:** 2 июля 2026 г.
**Контекст:** Вопросы экспертов госсектора (в т.ч. бывшие сотрудники SSPS/XDMX) выявили
разрыв между маркетинговой формулировкой «Sigma + MITRE» и фактическим состоянием:
корпус правил есть, но governance процесса, runtime-маппинг MITRE и полноценный
FP-workflow не зафиксированы в одном решении. ADR-0006 §3.1 задаёт принцип («детект-контент
— это ров»), ADR-0003 разрешает Sigma как данные, ADR-0018 описывает доставку контента —
но **кто владеет корпусом, как он версионируется и как борется с ложными срабатываниями**
— не было явным архитектурным решением.

**Связано:** [`ADR-0003`](0003-repository-structure-and-donor-strategy.md) ·
[`ADR-0006`](0006-coverage-gaps-strategic-bets-and-practices.md) §3.1 ·
[`ADR-0018`](0018-hybrid-connected-operating-model.md) §5 ·
[`ADR-0017`](0017-vision-one-onprem-patterns.md) §1 (Workbench) ·
`data/sigma-corpus/` · `services/detection-engine/internal/sigma/` ·
`services/detection-engine/internal/tip/` · `services/detection-engine/internal/risk/`

---

## Контекст и проблема

1. **Pipeline ≠ ценность.** Kafka/ClickHouse скопирует любой вендор. Продаваемая
   эффективность XDR в госсекторе определяется **качеством и актуальностью детект-контента**,
   а не скоростью ingest.
2. **Air-gap ≠ отсутствие TI.** Без облачного MISP/OpenCTI контур всё равно нуждается
   во **внутреннем** управлении IoC и обновляемых правилах.
3. **99% FP — реальность рынка.** Без suppression, risk-scoring и analyst feedback SOC
   тонет в шуме; это не «nice to have», а условие эксплуатации на 1000+ хостах.
4. **Честность перед заказчиком.** Датащит и питч не должны обещать «полную Sigma-семантику»
   или «MITRE end-to-end», если движок — MVP-подмножество.

## Решение

### 1. Владение и роли (Detection Engineering)

| Роль | Ответственность |
|---|---|
| **Detection Engineering (вендор)** | Curated-корпус, lint, golden-тесты, версионирование, release notes |
| **Региональные аналитики** (партнёр / design partner) | Локальные TTP, языковые фишинг-сценарии, FP-обратная связь с площадки |
| **SOC заказчика** | Локальный suppression, tuning per-tenant, пометка FP (будущий workflow) |
| **Update Service** (ADR-0018) | Сборка и подпись пакетов `sigma-corpus`; не редактирует правила ad-hoc |

**Инвариант:** контент (правила, IoC) версионируется **отдельно от движка** detection-engine.
Обновление правил ≠ обновление бинарника.

### 2. Sigma-корпус: состав и процесс

**Текущий состав (MVP):**
- `data/sigma-corpus/rules/` — community-правила (~500 файлов);
- `data/sigma-corpus/curated/` — curated под ERA (~100 файлов, `era-curated-*`, `era-sigma-NNNN`);
- lint при старте `detection-engine` — ошибки в YAML блокируют запуск.

**Процесс выпуска новой версии корпуса:**

```
правка YAML → lint → golden detection test → bump content_version → подписанный bundle (ERABNDL1)
→ offline / Hybrid Relay → policy ref bump → hot-reload detection-engine
```

**Ограничения движка (честно зафиксированы):**
- Парсер — **MVP-подмножество** Sigma: `logsource`, `detection.selection`, `level`;
  полная algebra `condition`, field modifiers, aggregations — **не в scope MVP**.
- Matcher — substring/contains на сериализованном payload, не полный Sigma backend
  (Chainsaw/SigmaHQ spec).
- **MITRE-теги в YAML (`tags: attack.T1003`) не маппятся автоматически на алерт** —
  поля `mitre_tactics` / `mitre_techniques` на detection заполняются из envelope агента,
  коррелятора или AI-эвристики (см. ADR-0023).

**Целевое состояние (Фаза 2):**
- Runtime-маппинг MITRE из тегов правила на каждый detection;
- MITRE ATT&CK coverage heatmap в UI;
- Регрессия через `data/mitre-eval/scenarios.json`.

### 3. Threat Intelligence в air-gap (без облачного TI)

Внешний облачный TI (MISP Cloud, VirusTotal, облачный OpenCTI) **не используется**
в рантайме — инвариант air-gap ([`security-and-air-gap`](../../.cursor/rules/security-and-air-gap.mdc)).

**Каналы обновления IoC:**

| Канал | Механизм | Статус |
|---|---|---|
| **National IoC file** | `data/national-iocs/patterns.json` → `tip.LoadFile` | ✅ MVP |
| **STIX 2.1 bundle** | `ERA_STIX_BUNDLE` file path → `tip/stix.go` (indicator-only) | ✅ MVP |
| **National TAXII hub** | `services/national-hub` inbound collection | 🟡 MVP |
| **Signed update bundle** | kind `sigma-corpus`, `cve-feed`, `connector`, `ai-pack` | ✅ (IoC — **не** bundle kind) |
| **Hybrid inbound TI** | Portal → Relay → локальное зеркало (ADR-0018 §5) | 🟡 |

**Инвариант:** IoC обогащает события **внутри контура**; сырой lake наружу не уходит.

### 4. False Positive management

**Фаза 1 (MVP, реализовано):**
- Risk-based dedup: 15-минутное окно `rule_id + node_id` (`detection-engine/internal/risk/`);
- Entity score bump + severity lift при score ≥ 50;
- Авто-кейс на high/critical с дедупликацией;
- Кросс-доменная корреляция (APT chain, Observe+endpoint) — снижает изолированный шум.

**Фаза 2 (roadmap):**
- Per-tenant suppression lists (правило, хост, профиль);
- Агрегация алертов в инциденты (case-level, не только detection-level);
- Analyst FP feedback → локальный tuning + opt-in outbound metadata (ADR-0018 §5);
- Динамические пороги risk-scoring.

**Фаза 3:**
- FP-feedback в Update Service → калибровка глобального корпуса;
- ML-assisted tuning (on-prem, без exfil).

**Позиционирование:** не заявлять «99% FP отфильтровано» без field-метрик с заказчика.

### 5. CVE-feed (связанный контент)

- Bundle kind `cve-feed` существует (`update-service/internal/bundle/`);
- Контент-пайплайн и hot-reload в VM-сканер — 🟡 в развитии;
- Версионирование CVE отдельно от Sigma (тот же механизм подписи).

### 6. Критерии приёмки контента

| ID | Критерий | Доказательство |
|---|---|---|
| DC-1 | Lint корпуса при старте detection-engine | CI + startup log |
| DC-2 | Golden: фикс. событие → ожидаемый detection | `testdata/` + `*_golden_test.go` |
| DC-3 | MITRE eval сценарии регрессия | `data/mitre-eval/`, `mitreval/scenarios_test.go` |
| DC-4 | Подписанный bundle применяется через Relay | `relay_e2e_test.go` |
| DC-5 | FP dedup: повтор rule+node за 15 мин не эмитится | `risk/golden_test.go` |
| DC-6 | Coverage heatmap в UI | [ ] Фаза 2 |
| DC-7 | Analyst suppression UI | [ ] Фаза 2 |

## Последствия

**Плюсы:**
- Единая точка правды для вопросов «кто создал корпус / как MITRE / как без облачного TI»;
- Честное разделение MVP vs target снижает риск потери доверия на пилоте;
- Процесс detection engineering масштабируется с привлечением региональных аналитиков.

**Минусы / обязательства:**
- Нужна дисциплина release контента (не только кода);
- Curated-корпус требует постоянного наполнения — это OPEX, не разовая задача;
- Полная Sigma-совместимость — отдельный крупный эпик, не «допилить парсер за спринт».

## Открытые вопросы

1. Минимальный размер curated-корпуса для пилота в госсекторе (100? 500? по MITRE coverage)?
2. Формат локального FP-feedback: только CP API или экспорт в Update Service?
3. Лицензирование community Sigma при поставке в air-gap — юридическая проверка корпуса.

## Связано

- [`ADR-0023`](0023-ai-investigation-explainability.md) — MITRE на AI-вердикте
- [`ADR-0018`](0018-hybrid-connected-operating-model.md) — доставка контента
- [`Production-Readiness-Assessment.md`](../Production-Readiness-Assessment.md) — field gates
