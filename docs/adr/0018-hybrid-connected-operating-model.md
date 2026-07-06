# ADR-0018: Sovereign Hybrid — connected operating model и лицензирование

**Статус:** Implemented  
**Дата:** 1 июля 2026 г.
**Контекст:** После лобовых сравнений с Trend Micro Vision One, ManageEngine и Ivanti
([`head-to-head/`](../distributor/head-to-head/)) внутри команды возник обоснованный
протест: конкуренты «вынуждают» клиента в облако (Cortex Cloud / Vision One / Ivanti
Neurons) не качеством детекции, а **операционной моделью** — непрерывные обновления,
threat intel, managed-сопровождение, единый портал, usage-лицензии. Без ответа на это
суверенный on-prem рискует восприниматься как «офлайн из прошлого».

**Решение принято** после разбора с тремя моделями и явного согласования продукт-оунером
(режимы, health-уровни, TI-scope, Managed View как наш портал, lease без хардкода, Relay
как модуль control-plane, контекст Азербайджана).

**Связано:** [`ADR-0005`](0005-module-independence-and-packaging.md) (издания) ·
[`ADR-0009`](0009-pii-redaction-and-agent-budget.md) (PII на агенте, KMS в контуре) ·
[`ADR-0010`](0010-licensing-and-activation.md) (офлайн-лицензия, CRL, sealed clock) ·
[`ADR-0016`](0016-uem-scope-vs-ivanti.md) (VPN/ZTNA — вне core) ·
[`ADR-0017`](0017-vision-one-onprem-patterns.md) (BYO-EDR Hub, Federated) ·
[`ERA-Platform-Vision.md §11`](../ERA-Platform-Vision.md) · `editions-control.yaml`

---

## 1. Решение (одной фразой)

**ERA One остаётся on-prem / air-gap first, но получает Hybrid operating model:**
**суверенный data plane у заказчика + cloud control plane у вендора** (лицензии,
обновления контента, health, сопровождение). Мы **не строим** multi-tenant SaaS-клон
Cortex Cloud как ближайший шаг — это отдельная платформа эксплуатации (см. §11).

Разделение плоскостей — базовый инвариант этого ADR:

| Плоскость | Что входит | Где живёт |
|---|---|---|
| **Data plane** (суверенный) | сырые события, ClickHouse/MinIO lake, LLM-инференс, кейсы, PII, расследования | **всегда в контуре заказчика** |
| **Control plane** (гибридный) | лицензии/lease, пакеты обновлений (Sigma/CVE/коннекторы/AI), CRL, health, usage, opt-in TI | on-prem, при `connected` — обмен с облаком вендора через Relay |

> **Инвариант:** сырой event stream, содержимое кейсов и PII **никогда** не покидают
> контур — ни в одном режиме. Наружу идут только метаданные, артефакты обновлений,
> лицензии и (opt-in) обезличенные индикаторы.

---

## 1.1. Компоненты Sovereign Hybrid (именованные)

Четыре именованных компонента. Два — на стороне вендора (control plane), один — в контуре
заказчика, один — представление для партнёра.

| Компонент | Где живёт | Деплой | Роль | Раздел |
|---|---|---|---|---|
| **ERA Cloud Portal** | вендор (control plane) | ядро-сервис | зонтичный сервис вендора: лицензии/контракты, выпуск **lease**, CRL, приём health, точка входа Update Service и Managed View | §7 |
| **ERA Update Service** | вендор (за Portal) | **отдельный сервис** | конвейер сборки/подписи + доставка **подписанного контента**: Sigma, CVE-feed, коннекторы, AI-паки, секрет-сканеры → в локальное зеркало (pull **или** носитель) | §1.1.1, §3.2, §12 |
| **ERA Hybrid Relay** | **контур заказчика** | модуль `control-plane` | единственный **outbound-only** канал: тянет lease/CRL/updates, шлёт health/opt-in TI; egress allowlist + audit | §3 |
| **ERA Managed View** | вендор (представление Portal) | **модуль Portal (RBAC)** | мульти-клиентский пульт для вендора/MSSP: health, лицензии, версии — **без доступа к сырому lake/кейсам** клиента | §1.1.1, §7 |

**Как они соотносятся:**

```
        ВЕНДОР (control plane, cloud)                    КОНТУР ЗАКАЗЧИКА (data plane)
┌────────────────────────────────────────┐        ┌───────────────────────────────────┐
│  ERA Cloud Portal                        │        │  control-plane                     │
│   ├── lease / контракты / CRL            │◀──mTLS─┤   └── ERA Hybrid Relay (модуль)    │
│   ├── ERA Update Service ─(signed)──────►│───────►│        → локальное зеркало (MinIO)  │
│   └── ERA Managed View (для MSSP)        │        │  ingest · lake · LLM · cases (НИКОГДА│
│                                          │        │        не выходят наружу)          │
└────────────────────────────────────────┘        └───────────────────────────────────┘
```

- **Portal** — «мы и есть cloud» для эксплуатации; **Update Service** — его контент-канал;
  **Managed View** — его представление для партнёра; всё это control plane.
- **Relay** — единственная точка контура, которая инициирует исходящие соединения.
- Границы Portal/Managed View по данным (что видно, что нет) — §7; поведение Relay — §3.

### 1.1.1. Гранулярность деплоя: что отдельный сервис, что модуль Portal

Компоненты **не симметричны** по жизненному циклу и границе доверия. Решение по деплою
(критерии — как в [`ADR-0005`](0005-module-independence-and-packaging.md): разный
lifecycle, граница безопасности, профиль нагрузки, владелец данных):

| Компонент | Гранулярность | Обоснование |
|---|---|---|
| **ERA Cloud Portal** | ядро control plane | источник истины: lease, контракты, CRL, health |
| **ERA Update Service** | **отдельный сервис** | свой конвейер **сборки и подписи** бандлов; **dual-use** — тот же подписанный контент едет и pull (connected), и носителем (air-gap); отдельная **граница подписи** (ключ в HSM/KMS, [`ADR-0010`](0010-licensing-and-activation.md)) — не в одном юните с партнёрским web-UI; другой ритм (контент — ежедневно) и профиль (статические артефакты → объектное хранилище/зеркало) |
| **ERA Managed View** | **модуль Portal (RBAC)** | **read-модель** поверх данных Portal, не владеет данными; граница «партнёр не видит сырьё/кейсы» = **ролевая модель и API-скоупы**, а не отдельный процесс; общий с Portal жизненный цикл |
| **ERA Hybrid Relay** | модуль `control-plane` в контуре | единственный outbound; живёт у заказчика |

**Формула:** *Portal + Managed View — один сервис с ролевой моделью; Update Service —
отдельный конвейер подписи и раздачи контента (работает и в pull, и в offline).*

> **Важно (air-gap):** Update Service нужен **даже без гибрида** — как «фабрика
> offline-бандлов». В чистом air-gap Portal/Managed View не участвуют, а подписанный
> контент Update Service доставляется носителем. Поэтому его конвейер подписи —
> самостоятельный, не производный от Portal.

**Триггеры вынести Managed View в отдельный сервис** (пересмотреть решение при наступлении):
- **white-label** для дистрибьюторов (свой брендинг/домен/SLA);
- **мультитенантность партнёров** (изоляция «партнёр A не видит клиентов партнёра B»,
  100+ клиентов) — соответствует ступени 3–4 (§11);
- резко иной профиль нагрузки (много одновременных партнёрских сессий).

До наступления триггеров — Managed View остаётся вкладкой/модулем Portal с отдельным RBAC.

---

## 2. Модель развёртывания: режимы и уровни opt-in

Три режима — это **профили одной платформы** (feature-flags + build tags), не разные
продукты (согласовано с [`ERA-Platform-Vision.md §11`](../ERA-Platform-Vision.md)).

### 2.1. Режимы (`deployment_mode`)

| Режим | Связь с вендором | Лицензия | Обновления | Health наружу |
|---|---|---|---|---|
| **`air-gap`** *(default)* | нет | файл Ed25519 ([ADR-0010](0010-licensing-and-activation.md)) | offline-бандл (носитель) | нет |
| **`connected` / hybrid** | через Relay (outbound-only) | lease + локальная подпись | pull с Portal | по policy |
| *(cloud — вне scope этого ADR, см. §11)* | | | | |

- **По умолчанию — `air-gap`.** `connected` включается **явно** администратором при
  установке или позже (осознанное действие, не «тихий» phone-home).
- Смена режима — операция control-plane с записью в audit-журнал.

### 2.2. `connected` vs `opt-in` — это разные оси

- **`connected`** = инсталляция технически **умеет** ходить к вендору (есть Relay + URL).
- **`opt-in`** = какие **классы данных** при этом разрешено передавать. Это отдельные
  флаги в tenant policy, не «всё или ничего».

Уровни opt-in внутри `connected` (складываются):

| Уровень opt-in | Что разрешает наружу | Кому обычно подходит |
|---|---|---|
| **hybrid-base** *(минимум)* | lease + updates (Sigma/CVE/коннекторы) + CRL | почти все connected-клиенты |
| **+ health** | эксплуатационные метрики на Portal (см. §4) | клиенты с нашим/партнёрским сопровождением |
| **+ TI-share** | обезличенные IoC / detection-metadata / FP-feedback (см. §5) | зрелые SOC, готовые делиться индикаторами |

По умолчанию при включении `connected`: **hybrid-base ON, health = уровень A, TI-share OFF**.

---

## 3. ERA Hybrid Relay — модуль control-plane

**Решение: Relay — это модуль внутри `control-plane`, не отдельный продукт/контейнер**
(согласовано; соответствует ADR-0005 — feature-flag/модуль, а не новый деплой-зоопарк).

### 3.1. Назначение

Единственная точка контура, которая инициирует **исходящие** соединения к вендору.
Firewall банка/госа обычно разрешает outbound HTTPS на 1–2 адреса вендора, но не
входящие подключения. Relay концентрирует весь egress в одном месте с политикой и
аудитом.

```
[ Агенты ] → [ ingest / lake / AI / cases ]         ── всё внутри контура, без интернета
                        ↑
[ control-plane ] ─┬─ [ hybrid_relay (модуль) ] ──outbound HTTPS──▶ [ ERA Cloud Portal ]
                   │
                   └─ применяет telemetry policy ДО отправки + пишет audit «что ушло»
```

### 3.2. Что Relay делает и чего не делает

**Делает (outbound-only):**
- тянет и обновляет **lease** (§6) и **CRL** ([ADR-0010 §6](0010-licensing-and-activation.md));
- скачивает **подписанные пакеты обновлений** из **ERA Update Service** (Sigma, CVE-feed,
  коннекторы, AI-паки, секрет-сканеры) и кладёт в локальное зеркало (MinIO, ADR-0009/Vision §9);
- отправляет **health/usage** согласно policy (§4);
- (opt-in) отправляет **обезличенные** IoC/metadata (§5);
- верифицирует подпись **всего** входящего контента перед применением.

**Не делает:**
- не хранит и не проксирует сырой lake / event stream;
- не открывает входящих портов из интернета;
- не заменяет `ingest-gateway` (агенты по-прежнему шлют в локальный ingest);
- ничего не отправляет наружу сверх того, что разрешено policy.

### 3.2.1. Типы контента и каналы доставки

Не весь обновляемый контент идёт одним механизмом. Явная таблица — чтобы не путать
bundle kinds Update Service с file-based IoC.

| Тип контента | Механизм доставки | Bundle kind | Air-gap (носитель) | Connected (Relay pull) | Статус |
|---|---|---|---|---|---|
| **Sigma-правила** | подписанный bundle → локальное зеркало → detection-engine | `sigma-corpus` | ✅ | ✅ | ✅ |
| **CVE-feed** | подписанный bundle → policy ref / VM | `cve-feed` | ✅ | ✅ | 🟡 пайплайн контента |
| **AI-паки** (weights, prompts) | подписанный bundle | `ai-pack` | ✅ | ✅ | ✅ |
| **Коннекторы** | подписанный bundle | `connector` | ✅ | ✅ | ✅ |
| **Секрет-сканеры** | подписанный bundle | *(в editions-control.yaml)* | ✅ | ✅ | 🟡 |
| **IoC (STIX)** | file path `ERA_STIX_BUNDLE` → tip ingest | — *(не bundle kind)* | ✅ replace file | ✅ + inbound TI pack | ✅ MVP |
| **National IoC** | file `data/national-iocs/patterns.json` | — | ✅ | ✅ | ✅ MVP |
| **TAXII collections** | `national-hub` inbound (внутри контура) | — | ✅ | ✅ | 🟡 MVP |
| **OTA агентов** | signed artifact → `era-agent-core/ota` | отдельный канал | ✅ | ✅ | ✅ |
| **CRL / lease** | Relay → control-plane | — | носитель опционально | ✅ | ✅ |

**Инварианты:**
- Всё, что приходит через Relay или носитель — **верифицируется Ed25519** перед применением.
- IoC **не смешивается** с `sigma-corpus` bundle: отдельные пути загрузки в detection-engine.
- Сырой event stream и содержимое кейсов **никогда** не входят в исходящие пакеты (§5).

**Связано:** детальный процесс governance контента — [`ADR-0022`](0022-detection-content-governance.md).

### 3.3. Требования к реализации

- Подмодуль `control-plane` (напр. `connected_mode` / `hybrid_relay`): очередь
  исходящих задач, ретраи, backoff, устойчивость к обрыву связи.
- **Egress allowlist** — фиксированный список хостов/эндпойнтов вендора (конфиг).
- **Audit-журнал egress**: что, когда, какой уровень policy, размер, хэш пакета.
- Вся связь — mTLS + подпись полезной нагрузки (доверяем контенту, не только каналу).
- Отдельный контейнер-relay в DMZ — **возможная фаза 2** (клиент с требованием
  «relay только в DMZ»); в MVP не требуется.

---

## 4. Health / telemetry policy (3 уровня)

Что уходит наружу — **определяется policy tenant, а не хардкодом в коде** (тот же
принцип, что PII-редакция в [ADR-0009](0009-pii-redaction-and-agent-budget.md)).
Три уровня, задаются в policy bundle; уровень по умолчанию для `connected` — **A**.

### Уровень A — Minimal *(default)*
Только эксплуатация продукта, без бизнес-данных:
- `deployment_id`, `tenant_id` (псевдоним);
- версии компонентов (agent, control-plane, модули, rule packs);
- статус лицензии (`VALID`/`GRACE`/`EXPIRED`), дата истечения;
- `active_nodes` / `max_nodes`;
- uptime, дата последнего успешного обновления;
- критические эксплуатационные алерты (диск lake ≥ N%, Kafka lag, «lease истекает через N дней»).

### Уровень B — Operational *(для MSSP / Managed View)*
Всё из A плюс:
- ingestion rate (агрегаты, **не** сырые события);
- ошибки сервисов (коды, без stack trace с путями пользователей);
- версии/время применения rule packs;
- top-N кодов отказа агентов (коды, без hostname/пользователей).

### Уровень C — Support *(break-glass, временный)*
Всё из B плюс — **только по явному тикету и с TTL 24–72 ч**:
- расширенные логи control-plane (redacted);
- снимок конфигурации (**без секретов**).

### Жёсткий запрет (инвариант, любой уровень)
Никогда не уходит наружу: сырой event stream; command line; имена пользователей / email;
IP с привязкой к личности; содержимое кейсов; любая PII **до** редакции.

Реализация: поле в policy bundle + UI «Telemetry policy» + запись фактически отправленного
в audit-журнал Relay (§3.3).

---

## 5. Threat Intelligence sharing (opt-in)

**Не шерим:** сырые логи, телеметрию, кейсы, файлы, письма.

**Можем шерить (только при `TI-share` opt-in, только ПОСЛЕ PII-редакции на агенте):**

| Тип | Пример | Ценность |
|---|---|---|
| **IoC** | hash, domain, IP (без привязки к клиенту) | обогащение общего TIP |
| **Detection metadata** | `rule_id`, MITRE technique, confidence | улучшение корпуса Sigma |
| **Campaign fingerprint** | паттерн цепочки (без hostnames) | раннее предупреждение другим контурам |
| **False-positive feedback** | «rule X — FP на профиле Y» (обобщённо) | калибровка правил |

**Направления обмена:**
- **Inbound** (доступно всем `connected`, часть hybrid-base): Portal отдаёт **подписанные**
  пакеты TI внутрь контура — как обычные updates.
- **Outbound** (только `TI-share` opt-in): Relay отправляет **обезличенный** батч
  IoC/metadata; `tenant` — только как one-way псевдоним, без раскрытия заказчика.

Родственно [`ERA Federated / National`](0017-vision-one-onprem-patterns.md) и
[`ADR-0002`](0002-learning-topology.md): это обмен **индикаторами**, а не cloud data lake.

---

## 6. Лицензирование: lease поверх ADR-0010

База [ADR-0010](0010-licensing-and-activation.md) (Ed25519, `modules`, `max_nodes`,
`deployment`, grace, CRL, sealed clock) **не ломается** — остаётся ядром проверки во всех
режимах. Гибрид добавляет **online entitlement lease** как верхний слой.

### 6.1. Механика lease (`connected`)

1. Вендорский **Portal** выпускает подписку (контракт → editions/modules → квоты).
2. `control-plane` через Relay периодически запрашивает **подписанный lease** (короткий
   срок: дни).
3. Проверка lease — **локальная**, по подписи (как основной токен). Канал не является
   доверенным сам по себе.
4. **Нет связи → работаем на последнем валидном lease**, затем grace, затем деградация —
   в точности как поведение air-gap (никакого kill-switch).
5. Активация модулей — через тот же `modules` (источник истины ADR-0010 §5).

### 6.2. Параметры — в настройках, без хардкода (согласовано)

Значения ниже — **стартовые рекомендации**; финально задаются в лицензии / контракте /
tenant policy (не константы в коде).

| Параметр | Стартовое значение | Где задаётся |
|---|---|---|
| `lease_period_days` | 30 | лицензия / Portal / контракт |
| `lease_renewal_interval` | каждые 24 ч (если `connected`) | tenant policy |
| `grace_days` | 30 (совпадает с ADR-0010) | лицензия |
| `offline_max_days` | 90 без успешного renew → деградация | tenant policy |
| `degradation_mode` | `no_new_nodes` + `no_updates`, детекция работает | политика |

Профили по сегментам (пример): банк — lease 90 / renew раз в неделю / grace 60;
коммерция — lease 30 / renew 24 ч / grace 30. Всё — поля в JSON, конфигурируется.

### 6.3. Поведение при истечении (инвариант из ADR-0010)

Security-продукт **не глушим жёстко**. Деградация: стоп онбординга новых узлов, стоп
обновлений, настойчивые предупреждения; **ядро детекции и расследования продолжают
работать**. Конкретика — по политике (банк ≠ госструктура).

### 6.4. Cloud/SaaS лицензирование (вне scope, на будущее)

Для полноценного SaaS понадобится отдельный слой над токеном: `tenant_id`,
subscription/contract id, partner/distributor id, usage **metering**, overage policy,
billing period, централизованный Entitlement Service вместо офлайн-выпуска. Это **не
входит** в данный ADR и не привязывается к hybrid-lease на первом этапе.

---

## 7. ERA Cloud Portal и Managed View

**Решение: Portal — наш control plane («мы и есть cloud»), но только для управления,
не для данных клиента.**

**На Portal вендор/партнёр (MSSP) видит:**
- список инсталляций (`deployment_id`, заказчик, партнёр);
- health уровня A/B (§4);
- лицензии/lease: сроки, модули, квоты, статус;
- статус и версии обновлений контента;
- алерты сопровождения («контур отвалился», `GRACE`, диск lake полон).

**Portal НЕ видит** (без отдельного break-glass уровня C с TTL и аудитом):
- события, кейсы, lake, расследования, PII.

Позиционирование: **«мы cloud для эксплуатации и контента, не cloud для ваших данных»**.

**Managed View — модуль Portal, а не отдельный сервис** (см. §1.1.1): это read-модель
поверх данных Portal, граница «партнёр не видит сырьё/кейсы» реализуется **ролевой моделью
(RBAC) и API-скоупами**, а не отдельным деплой-юнитом. Вынос в самостоятельный сервис —
только при триггерах white-label / мультитенантности партнёров (§1.1.1).

> **Update Service — наоборот, отдельный сервис** (§1.1.1): конвейер подписи контента с
> собственной границей безопасности (ключ в HSM/KMS) и dual-use доставкой (pull + offline).

---

## 8. Связь с BYO-EDR (это другая ось — не путать)

Частая путаница: BYO-EDR и Hybrid — **ортогональные** оси.

| Ось | Вопрос | Артефакт |
|---|---|---|
| **Hybrid** (этот ADR) | как **наша платформа** получает обновления/лицензии от вендора | Relay, Portal, lease |
| **BYO-EDR Hub** | как **клиент** заводит **чужой EDR** (Cortex/Defender/…) в **наш lake** | [`ADR-0017`](0017-vision-one-onprem-patterns.md), `era-collectors` |

BYO-EDR Hub — про **источники данных внутри контура клиента** (миграция/гибридный парк:
«Cortex на 5000 хостов ещё 2 года — не выкидываем сразу»). Hybrid — про **канал к
вендору ERA**. Они могут сосуществовать в одном проекте, но это разные фичи и разные
лицензионные линии. В пресейле формулировка: **«Cortex у вас на endpoint — ERA единый
on-prem SOC и путь миграции»**, а НЕ «мы заменяем Cortex Cloud облаком ERA».

---

## 9. Регуляторика (контекст: Азербайджан)

> Не юридическая консультация; для тендеров нужна локальная экспертиза. Ниже — продуктовые
> и пресейл-инварианты. Согласуется с [`Market-Positioning-AZ.md`](../Market-Positioning-AZ.md).

Наблюдение: заказчики, использующие Palo Alto / Cortex cloud, **обошли не «запрет
облака»**, а решили вопрос **моделью данных + договором** (cloud-delivered services без
хранения регулируемых ПДн в открытом виде). ERA укладывается в ту же логику, но **строже**:
lake и расследования остаются в контуре.

Что закладываем в продукт и документы под AZ:

1. **Локализация Portal endpoint** — AZ/EU-region или контракт с местным юрлицом;
   опция **self-hosted Portal** для госа (фаза 2).
2. **Cross-border transfer** — в health/TI **нет ПДн**, только псевдонимы и агрегаты
   (главный рычаг для DPA).
3. **Режим «hybrid minimal» для регуляторики** — только lease + updates + CRL,
   health = уровень A, TI-outbound = OFF. Часто достаточно, чтобы закрыть «как у PA cloud,
   но безопаснее».
4. **Пакет для ИБ/закупок** — одностраничная **схема потоков** (az/ru/en): что остаётся в
   ЦОД AZ, что идёт на `*.era-one.solutions`, протокол, шифрование, retention.
5. **Суверенный стек** — ClickHouse/MinIO/LLM в контуре: нет зависимости от US-cloud для
   SOC-данных.
6. **Партнёр/MSSP в AZ** — Managed View без доступа к данным клиента (важно для дистрибуции).

**Пич (AZ):** «ERA One — полный XDR и IT-Ops в вашем дата-центре в Азербайджане; облако
ERA — только для лицензий, обновлений защиты и сопровождения, без выноса журналов и
расследований.»

---

## 10. Совместимость с текущей архитектурой

**На hybrid ложится хорошо** — фундамент уже есть:
- `Envelope` — единый контракт событий ([ADR-0001](0001-unified-event-envelope.md));
- Kafka + topic-per-domain — модули развязаны ([ADR-0005](0005-module-independence-and-packaging.md));
- помодульные editions / feature-flags;
- ClickHouse + MinIO lake ([ADR-0004](0004-storage-and-retention.md), [ADR-0007](0007-clickhouse-schema.md));
- PII-редакция на агенте ([ADR-0009](0009-pii-redaction-and-agent-budget.md)) — поток
  уже готов к opt-in TI без утечки PII;
- лицензирование уже мыслит `connected`/`disconnected` + CRL ([ADR-0010](0010-licensing-and-activation.md));
- Federated/National — почти готовая модель межконтурного обмена ([ADR-0017](0017-vision-one-onprem-patterns.md)).

**Минимальные доработки для hybrid v1:**
- `deployment_mode`: `air-gap | connected` в control-plane;
- модуль `hybrid_relay` + egress allowlist + telemetry policy + audit;
- канал обновлений (частично заложен в CRL/бандлах правил);
- Health API уровня A в control-plane;
- lease-слой поверх license-верификатора.

**Не тащим в hybrid v1:** multi-tenancy в ClickHouse/Kafka, перенос lake в облако,
cloud-native security (CNAPP/CSPM), TI-outbound, Managed View на 100+ клиентов.

---

## 11. Лестница зрелости (фазирование)

| Ступень | Что это | Статус / приоритет |
|---|---|---|
| **1. Air-gap on-prem** | как сейчас, дифференциатор | GA / ядро |
| **2. Sovereign Hybrid** (этот ADR) | lake on-prem + Portal + Relay + Updates + lease | **следующий шаг (Platform P7-hybrid)** |
| **3. Managed private cloud** | single-tenant в ЦОД клиента / у локального провайдера | после пилотов |
| **4. Multi-tenant SaaS** | общий cloud для всех | только по подтверждённому спросу (см. Vision §11.2) |

**Сознательно НЕ делаем** (из [ADR-0017](0017-vision-one-onprem-patterns.md) + консенсус):
cloud data lake (сырьё у вендора), cloud AI-инференс у вендора, global threat cloud «как
TM» без opt-in/обезличивания, MDR-as-a-service как обязательный SKU, полный multi-tenant
SaaS из коробки.

---

## 12. MVP: Hybrid-0

Минимальный scope, чтобы закрыть протест команды и дать пресейлу аргумент:

1. `connected_mode` в control-plane (вкл/выкл + URL Portal + egress allowlist).
2. Lease-renew поверх [ADR-0010](0010-licensing-and-activation.md).
3. **ERA Update Service** v0: канал доставки контента (Sigma/CVE-бандлы — как сейчас
   offline, но pull через Relay).
4. CRL через `connected`.
5. Health **уровень A only** + policy в tenant.
6. Portal v0: инсталляции, lease, health, версии.
7. DPA-шаблон + одностраничная схема потоков **для AZ**.

**Не в MVP:** TI-outbound, Managed View для 100+ клиентов, BYO-EDR, отдельный
relay-контейнер в DMZ.

---

## 13. Открытые вопросы (к следующей итерации)

Зафиксированные решения уже учтены в §2–§9. Осталось решить на реализации/пилоте:

1. Portal hosted у вендора (EU/AZ) **vs** опция on-prem Portal для госа — включать ли
   on-prem Portal в roadmap сразу или после пилота.
2. Первый пилот hybrid: коммерческий клиент **vs** внутренний стенд.
3. Точная схема псевдонимизации `tenant` для outbound TI (one-way, ротация ключа).
4. Формат DPA/схемы потоков под конкретного AZ-регулятора (юр. ревью).

---

## 14. Последствия

**Плюсы:**
- снимаем реальную причину протеста (операционная отсталость) **без** отказа от
  суверенного УТП;
- эволюция текущей архитектуры, а не новая платформа;
- лицензирование остаётся единой моделью (offline + lease), безопасная деградация;
- готовый нарратив для AZ и для клиентов на Palo Alto / Cortex cloud.

**Минусы / обязательства:**
- нужен вендорский Portal + инфраструктура доставки контента (эксплуатация вендора);
- egress-политика и audit-журнал Relay — критичны для доверия ИБ;
- дисциплина инварианта «сырьё/PII/кейсы не выходят наружу никогда»;
- DPA/юридическая обвязка под каждую юрисдикцию (начиная с AZ).

## Связано (реализация — при старте Platform phase)

- [`ERA-Platform-Vision.md §11`](../ERA-Platform-Vision.md) — deployment models
- [`ADR-0010`](0010-licensing-and-activation.md) — база лицензирования (lease — надстройка)
- [`ADR-0009`](0009-pii-redaction-and-agent-budget.md) — PII-редакция, telemetry policy
- [`ADR-0017`](0017-vision-one-onprem-patterns.md) — BYO-EDR (другая ось)
- [`ADR-0022`](0022-detection-content-governance.md) — Sigma, MITRE, TI, FP
- [`ADR-0023`](0023-ai-investigation-explainability.md) — AI audit trail
- [`editions-control.yaml`](../../editions-control.yaml) — deployment modes / connected relay
