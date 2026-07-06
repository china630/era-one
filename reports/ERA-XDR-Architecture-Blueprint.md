# ERA XDR — Architecture Blueprint

**Версия документа:** 2.0
**Дата:** 8 июня 2026 г.
**Статус:** Draft (Фаза 1 — проектирование контрактов и пайплайна)
**Предшествующий документ:** [`AI-Donors-Matrix-Deep-Analysis.md`](./AI-Donors-Matrix-Deep-Analysis.md) (v1.0, 3 апреля 2026)

---

## 1. Executive Summary

**ERA XDR** — суверенная, On-Premise (air-gapped) Cloud-Native платформа класса
Extended Detection and Response для крупных изолированных enterprise-сетей и
государственных структур локального рынка (Азербайджан, регион СНГ/ЦА).
Целевой масштаб: **до 150 000 хостов в одном кластере**.

Система агрегирует **подходы** мировых лидеров (CrowdStrike Falcon, SentinelOne
Singularity, **Trend Vision One**), но физически разворачивается внутри контура
заказчика, исключая отправку телеметрии во внешние облака.

### Три критерия продукта

1. **Надёжность** — гарантированная доставка событий (Kafka, idempotent producer,
   disk-backed buffer на агенте), отказоустойчивость, golden-тесты эквивалентности.
2. **Безопасность** — air-gap, очистка PII до записи, строгая идентификация,
   суверенность данных, security review критичных путей.
3. **Лёгкость клиента** — один Rust-бинарник, детерминированная on-agent детекция,
   без тяжёлого ML на эндпоинте.

### Киллер-фича: автономность самообучения контура

Самообучается **только серверный AI Core**; агент остаётся лёгким и лишь
**самоадаптируется** (локальный поведенческий baseline + Sigma). AI Core учится на
агрегированной телеметрии всего контура и **ретранслирует знания** (новые правила,
веса детекторов) обратно всем агентам — «стадный иммунитет» (herd immunity).

> Стратегический акцент после выбора **Trend Vision One** третьим донором: вес
> продукта смещён на **кросс-доменную корреляцию** (endpoint + identity + email +
> cloud + network), а не на «ещё один EDR-агент». Побеждает тот, кто лучше
> **объединяет телеметрию**, а не тот, у кого «умнее» агент.

---

## 2. Архитектурная парадигма: AI-Driven Reverse Engineering

Мы **не форкаем** доноров и **не оборачиваем** чужой код. Мы извлекаем из лучших
open-source проектов (преимущественно CNCF, Go/Rust) **архитектурные паттерны,
модели данных и алгоритмы** и реализуем их заново на нашем лёгком стеке.

| Назначение | Донор | Лицензия | Что берём |
|---|---|---|---|
| Сбор/транспорт | Vector (Rust) | MPL 2.0 | Backpressure, буферизация, топология (только идеи — код переписываем) |
| SIEM/аналитика | Matano (Rust/Go) | Apache 2.0 | detection-as-code, табличное хранение (Iceberg/Parquet) |
| EDR-оркестрация | Fleet/Osquery | MIT | API-слой управления эндпоинтами |
| Метрики | Prometheus | Apache 2.0 | Модель метрик, exposition format |
| Сканер уязвимостей | Nuclei | MIT | YAML-движок правил (модуль `/vm`, уже в работе) |
| WAF (Фаза 3) | Coraza | Apache 2.0 | Правила, фазы запроса/ответа |
| NGFW/eBPF (Фаза 3) | Cilium | Apache 2.0 | Модель сетевых политик |
| DLP/UAM (Фаза 3) | Teleport | Apache 2.0 | Access, аудит, запись сессий |

См. детали в [`ADR-0003`](../docs/adr/0003-repository-structure-and-donor-strategy.md).

---

## 3. Технологический стек

| Плоскость | Технология | Назначение |
|---|---|---|
| **Data Plane** (агенты) | **Rust** + eBPF (Linux) / ETW+Sysmon (Windows) / ESF (macOS) | Перехват событий ОС, on-agent Sigma, автономная работа |
| **Message Bus** | **Apache Kafka** | Гарантированная доставка ~4.5 ТБ/сутки, topic-per-domain |
| **Storage** | **ClickHouse** (hot/warm) + **MinIO/Iceberg** (cold) | Колоночное хранилище телеметрии, tiered retention |
| **Control Plane** | **Go** | gRPC/REST API, оркестрация, GitOps-хранение политик |
| **AI Core** | Суверенный inference-кластер (LLM) | AI SOC Analyst, threat hunting, NL→SQL/Sigma, MITRE-маппинг |
| **Инфраструктура** | Kubernetes, Helm, Terraform | Контейнеризация, IaC, профили развёртывания |
| **Контракты** | Protobuf + Apicurio Registry | Единый источник схем, governance совместимости |

---

## 4. Логическая архитектура

```
[era-agent (Rust)]  [era-collectors: email/identity/cloud/netflow]   [/vm scanner]
        │ Envelope (protobuf)        │                                    │
        │  PII sanitized on-agent    │                                    │
        └────────────┬───────────────┴────────────────────┬─────────────┘
                     ▼                                      ▼
              [ingest-gateway (Go)] ── validate, enrich, tenant routing, ingested_at
                     │
                     ▼
              [Apache Kafka]  topic-per-domain: xdr.process / xdr.network / ...
                     │
                     ▼
        [detection-engine / processors (Go)] ── Sigma + MITRE + OCSF + correlation
                     │
          ┌──────────┼───────────────────────────┐
          ▼          ▼                            ▼
   [ClickHouse]  [MinIO/Iceberg cold]      [ai-core] ── self-learning,
    hot/warm                                 raздаёт детекторы агентам (herd immunity)
          │                                        │
          ▼                                        ▼
   [query & dashboard API] ◄─── [soar / response playbooks]
          │
          ▼
   [unified app-shell: C-Level / SOC / Risk dashboards]
```

**Разделение плоскостей:** data plane (агенты, коллекторы, сканеры) / control plane
(политики, каталоги, RBAC) / analytics plane (хранилища, детекции, AI).

---

## 5. Реестр модулей

| # | Модуль | Стек | Плоскость | Фаза | Автономность |
|---|---|---|---|---|---|
| 1 | `era-agent` | Rust | Data | MVP | Высокая (офлайн-детект) |
| 1b | `era-agent-core` | Rust | Data | Platform P1 | [реализован] orchestrator |
| 1c | `era-plugin-sdk` | Rust | Data | Platform P1 | [реализован] ABI SDK |
| 1d | `era-plugin-inventory` | Rust | Data | Platform P1 | [реализован] cron plugin |
| 2 | `era-collectors` | Rust | Data | 2 | Высокая |
| 3 | `ingest-gateway` | Go | Control | MVP | Фундамент |
| 4 | `control-plane` | Go | Control | MVP→2 | [реализован] + hybrid_relay модуль |
| 5 | `contracts` + Apicurio | Proto | Control | MVP | Фундамент |
| 6 | Apache Kafka | — | Transport | MVP | Фундамент |
| 7 | `data-lake` (ClickHouse+MinIO) | — | Storage | MVP | Фундамент |
| 8 | `detection-engine` | Go | Analytics | MVP→2 | [реализован] corpus ~600; FP workflow 🟡 |
| 9 | `/vm` (сканер) | Go | Analytics | 2 | [реализован] |
| 10 | `ai-core` | Go | AI | 2 | [реализован] investigate MVP; forensic trail 🟡 |
| 11 | `soar` | Go | Response | 2 | [реализован] |
| 12 | `dashboards` (app-shell) | React/Next | UI | MVP→2 | Фундамент |
| 13 | `waf` | Go | Perimeter | 3 | [реализован] |
| 14 | `ngfw` | Go/Rust | Perimeter | 3 | [реализован] |
| 15 | `dlp-uam` | Go | Perimeter | 3 | [реализован] |
| 16 | `federated-hub` | Go | AI opt-in | 3 | [реализован] |
| 18 | `national-hub` | Go | National | 4 | [реализован] |
| 19 | `compliance` | Go | Reporting | 4 | [реализован] |
| 20 | `hybrid_relay` | Go (модуль CP) | Control | Hybrid-0 | [реализован] |
| 21 | `cloud-portal` | Go | Vendor CP | Hybrid-0 | [реализован] |
| 22 | `update-service` | Go | Vendor CP | Hybrid-0 | [реализован] |
| 23 | `era-workbench` | Go/UI | Analytics | Platform P1 | [реализован] timeline merge |
| 24 | `era-exposure` | Go | Analytics | Platform P2 | [реализован] per-asset score |
| 25 | `era-byo-edr` | Rust | Data | Platform P2 | [реализован] JSON/CEF adapter |
| 26 | `era-manage-cmdb` | Go | Control | Platform P2 | [реализован] ITAM/CMDB consumer |
| 27 | `era-enforcement` | Rust/Go | Data+Control | Platform P2 | [реализован] monitor policy engine + CP API |
| 28 | `era-plugin-appcontrol` | Rust | Data | Platform P2 | [реализован] monitor stub |
| 29 | `era-plugin-devicecontrol` | Rust | Data | Platform P2 | [реализован] USB monitor stub |
| 30 | `era-plugin-bitlocker` | Rust | Data | Platform P2 | [реализован] on-demand + escrow |
| 31 | `service-desk` | Go | IT-Ops | Platform P3 | [реализован] ITSM-lite MVP |
| 32 | `provision` | Go | IT-Ops | Platform P3 | [реализован] PXE catalog + enroll |
| 33 | `era-plugin-deploy` | Rust | Data | Platform P3 | [реализован] signed deploy stub |
| 34 | `era-pam` | Go | PAM | Platform P4 | [реализован] vault + checkout + SSH proxy |
| 35 | `observe` | Go | Network | Platform P5 | [реализован] NMS webhooks + SNMP/discovery MVP |

Подробно о независимости и упаковке в издания (editions) — [`ADR-0005`](../docs/adr/0005-module-independence-and-packaging.md).

---

## 6. Дорожная карта (фазы)

| Фаза | Содержание | Издание |
|---|---|---|
| **MVP (Фаза 1)** | Контракты + сквозной конвейер телеметрии: agent → ingest → Kafka → ClickHouse → detection (Sigma) → dashboard. Доказать единый конвейер на нагрузке. | ERA Core |
| **Фаза 2** | AI Core (SOC Analyst, threat hunting), коллекторы доменов (email/identity/cloud), кросс-доменная корреляция, SOAR. | + ERA Control AI / Response / Vuln |
| **Фаза 3** | WAF, NGFW/eBPF, DLP/UAM. **Federated learning внутри организации (опция, upsell).** | + ERA Federated *(опция)* |
| **Фаза 4** | **Межорганизационный / национальный обмен знаниями (опция, upsell)** — STIX/TAXII нацхаб, общегосударственный иммунитет. | + ERA National *(опция)* |

> **Важно (бизнес-правило):** federated-функции Фаз 3 и 4 предлагаются клиентам
> **как платная опция, не по умолчанию.** Базовая поставка работает полностью
> автономно без какого-либо обмена данными наружу контура.

---

## 7. Покрытие контура, дыры и стратегические ставки

Полная экспертная карта — в [`ADR-0006`](../docs/adr/0006-coverage-gaps-strategic-bets-and-practices.md).
Здесь — сжатая выжимка для дорожной карты.

### 7.1. Дыры покрытия (приоритет P0 — без них не продать в гос/банк)

| Пробел | Тип | Фаза |
|---|---|---|
| **ITDR** (детекция атак на AD/Entra) | домен | 2 |
| **Tamper Protection** (самозащита агента) | агент | MVP→2 | Фаза 1: detect-and-alert 🟡; Фаза 2: kernel prevent ⏸ |
| **Case / Incident Management** | операц. | MVP→2 |
| **TIP** (внутренний threat intel) | операц. | 2 |
| **Asset Inventory / CMDB** | операц. | MVP→2 |
| **Chain of Custody** (forensic integrity) | операц. | 2 |
| **Compliance & Reporting** (регулятор АЗ/ЦБ) | операц. | 2 |

P1: NDR, Deception (быстрая победа), CTEM+BAS, Risk-Based Alerting ([`ADR-0022`](../docs/adr/0022-detection-content-governance.md)), самоаудит платформы, AI explainability ([`ADR-0023`](../docs/adr/0023-ai-investigation-explainability.md)).
P2: CSPM/CNAPP.

### 7.2. Стратегические ставки (Vanga — опередить время)

**Топ-3 для нас (кандидаты в дифференциатор):**

1. **OT/ICS/IoT (SCADA)** — региональный джекпот (нефть/газ/энергетика АЗ); суверенность + OT = почти монопольная ниша.
2. **Agentic AI SOC on-prem** — автономное расследование/реагирование на нашем суверенном ИИ (облака этого в air-gap не дадут). **Предусловие:** forensic audit trail — ADR-0023 Фаза 2.
3. **MLSecOps** — защита суверенного ИИ + детекция атак, сгенерированных ИИ; двойная монетизация.

Прочие: PQC-readiness, Non-Human Identity, Confidential Computing (TEE), eBPF runtime security, Differential Privacy для нацхаба, Supply chain/SBOM.

### 7.3. Выстраданные практики (ключевое)

- **Детект-контент — это ров, а не пайплайн.** Локализованная библиотека правил = невоспроизводимая ценность. Governance — [`ADR-0022`](../docs/adr/0022-detection-content-governance.md).
- **«Лёгкость» измерять, а не декларировать** — hard-бюджет агента в CI (CPU/RAM).
- **Сертификация (ISO 27001 / регулятор / Common Criteria) — начинать сейчас** (цикл 12–18 мес.).
- **Доказывать эффективность измеримо** — MITRE ATT&CK Evaluations + BAS/purple team.
- **MVP-дисциплина** — один сквозной сценарий → референс-заказчик → расширение.
- **Операбельность air-gap = часть продукта** (self-healing, preflight, health-мониторинг).
- **MDR / SOC-as-a-Service поверх** — recurring revenue + кросс-клиентская экспертиза.
- **Референс-заказчик (design partner)** — главный канал продаж в регионе.

---

## 8. Связанные ADR

| ADR | Тема |
|---|---|
| [ADR-0001](../docs/adr/0001-unified-event-envelope.md) | Unified Event Envelope (Protobuf + OCSF + registry-гибрид) |
| [ADR-0002](../docs/adr/0002-learning-topology.md) | Топология обучения (hub-and-spoke → federated → национальный иммунитет) |
| [ADR-0003](../docs/adr/0003-repository-structure-and-donor-strategy.md) | Монорепо и стратегия доноров |
| [ADR-0004](../docs/adr/0004-storage-and-retention.md) | Хранение и retention-tiering |
| [ADR-0005](../docs/adr/0005-module-independence-and-packaging.md) | Независимость модулей и упаковка в издания |
| [ADR-0006](../docs/adr/0006-coverage-gaps-strategic-bets-and-practices.md) | Дыры покрытия, Vanga-ставки и практики |
| [ADR-0007](../docs/adr/0007-clickhouse-schema.md) | Схема хранения ClickHouse (DDL) |
| [ADR-0008](../docs/adr/0008-ingest-grpc-contract.md) | gRPC-контракт приёма (agent → gateway) |
| [ADR-0009](../docs/adr/0009-pii-redaction-and-agent-budget.md) | PII-редакция и бюджет ресурсов агента |
| [ADR-0010](../docs/adr/0010-licensing-and-activation.md) | Лицензирование и офлайн-активация (помодульно, 1/3 года) |
| [ADR-0011](../docs/adr/0011-cmdb-itam-data-model.md) | CMDB / ITAM data model |
| [ADR-0012](../docs/adr/0012-agent-enforcement-mode.md) | Enforcement-режим агента (Application Control) |
| [ADR-0017](../docs/adr/0017-vision-one-onprem-patterns.md) | Vision One on-prem patterns (Workbench, Exposure, BYO-EDR) |
| [ADR-0018](../docs/adr/0018-hybrid-connected-operating-model.md) | Sovereign Hybrid operating model |
| [ADR-0019](../docs/adr/0019-platform-agent-orchestrator.md) | Platform Agent Orchestrator (plugins, OTA) |
| [ADR-0020](../docs/adr/0020-network-observe-cmdb-reconciliation.md) | Network Observe + CMDB reconciliation |
| [ADR-0021](../docs/adr/0021-product-portal-and-pricing-calculator.md) | Публичный портал + калькулятор цен |
| [ADR-0022](../docs/adr/0022-detection-content-governance.md) | Detection Content Governance (Sigma, MITRE, TI, FP) |
| [ADR-0023](../docs/adr/0023-ai-investigation-explainability.md) | AI Investigation Explainability & Audit Trail |

**Реализационные артефакты:** [MVP Sprint-1 Spec](../docs/MVP-Sprint-1-Spec.md) ·
контракты [`proto/era/v1/`](../proto/era/v1/) ·
dev-окружение [`deploy/docker-compose.dev.yml`](../deploy/docker-compose.dev.yml) ·
DDL [`deploy/clickhouse/`](../deploy/clickhouse/) ·
скелеты [`crates/era-agent/`](../crates/era-agent/), [`services/ingest-gateway/`](../services/ingest-gateway/)

---

## 9. Ключевые риски (сводка)

Полный разбор — в разделе 5 апрельского отчёта. Топ-приоритеты для Фазы 1:

- **Качество AI-сгенерированного кода** → обязательные SAST/DAST, fuzzing парсеров, code review владельцем модуля.
- **Эквивалентность донорам** → golden-тесты (фикс. вход → ожидаемый выход), публичная матрица возможностей.
- **Лицензии** → юр-аудит каждого донора; MPL (Vector) — только реимплементация; скан лицензий корпуса правил.
- **On-prem / air-gap** → абстракции объектного хранилища/очередей/KMS с первого дня; инсталлятор с preflight-checks.
- **Операционная сложность K8s** → профили развёртывания (минимальный/полный), мониторинг здоровья платформы как часть продукта.
