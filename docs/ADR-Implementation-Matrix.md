# ADR → Код → Тесты (матрица прослеживаемости)

**Дата:** 2 июля 2026 г.  
**Назначение:** честная сверка «что решили в ADR» vs «что есть в репозитории».  
**Легенда статусов:**

| Маркер | Значение |
|--------|----------|
| ✅ | Реализовано, есть тест/доказательство |
| 🟡 | Частично / MVP / sim / monitor-only |
| ⏸ | За гейтом (field / external) — код не обязан закрывать |
| 📋 | Стратегия / ориентир — не чеклист кода |
| ❌ | Сознательно вне scope / DECLINE |

---

## Сводка по ADR

| ADR | Статус в доке | В коде (итог) | Главный gap |
|-----|---------------|---------------|-------------|
| 0001 Envelope | Accepted | ✅ | — |
| 0002 Federated learning | Accepted | 🟡 | опция `federated-hub`, не core |
| 0003 Donor strategy | Accepted | ✅ | процесс, не runtime |
| 0004 Storage | Accepted | ✅ | — |
| 0005 Editions | Accepted | ✅ | — |
| 0006 Coverage gaps | Accepted (ориентир) | 📋 | P0/P1 дыры частично |
| 0007 ClickHouse schema | Accepted | ✅ | inventory typed columns — 🟡 |
| 0008 Ingest gRPC | Accepted | ✅ | — |
| 0009 PII + budget | Accepted | ✅ | — |
| 0010 Licensing | Accepted | ✅ | HSM prod — ⏸ |
| 0011 CMDB/ITAM | Implemented | ✅ | CH typed inventory — 🟡 |
| 0012 Enforcement | Implemented monitor | 🟡 | боевой enforce — ⏸ |
| 0013 PAM | Accepted MVP | 🟡 | RDP/HSM — ⏸ |
| 0014 Monorepo | Accepted | 🟡 | rename era-one — ❌ отложен |
| 0016 UEM scope | Accepted | 🟡 | MDM/VPN — ❌ |
| 0017 Vision One patterns | Accepted | 🟡 | vpatch enforce — ⏸ |
| 0018 Hybrid | Implemented Hybrid-0 | 🟡 | SaaS/TI B/C — ❌/⏸ |
| 0019 Agent orchestrator | Implemented | ✅ | не все плагины из vision |
| 0020 Observe | Implemented MVP | 🟡 | полный NMS — ❌ |
| 0021 Portal + калькулятор | Accepted | 🟡 | статический сайт `site/` + рабочий калькулятор из SSOT `pricing-data.yaml`; тесты `site/test/calculator.test.js` зелёные; контент в развитии |
| 0022 Detection content | Accepted | 🟡 | корпус ~600, lint ✅; MITRE runtime map, FP UI, heatmap — [ ] |
| 0023 AI explainability | Accepted | 🟡 | investigate API ✅; custody chain к AI, audit log — [ ] |
| 0024 Product families | Accepted | 🟡 | `products.yaml`, platform, deploy profiles; Comms/Office ADR ✅ |
| 0025 Shared platform | Accepted | 📋 | ADR + `editions-shared.yaml`; runtime — roadmap |
| 0026 Office engine | Accepted | 📋 | sovereign CRDT + Rust OOXML; no OnlyOffice/GPL |
| 0027 Communications | Accepted | 📋 | Mail Connect, standalone, Office boundary |

---

## ADR-0001 — Unified Event Envelope

| Решение | Код | Тест |
|---------|-----|------|
| Proto `Envelope`, категории | `proto/era/v1/envelope.proto`, `gen/go`, `crates/era-proto` | `gen/go/era/v1/envelope_test.go` |
| PII flag `pii_sanitized` | `services/ingest-gateway/internal/ingest/validate.go` | `validate_test.go` |
| Wire JSON/gRPC | ingest-gateway gRPC + REST `/v1/ingest` | `server_test.go` |

**Итог:** ✅

---

## ADR-0002 — Learning topology (Federated)

| Решение | Код | Тест |
|---------|-----|------|
| Опция federated в издании | `services/federated`, `editions-control.yaml` | `federated/internal/hub/*_test.go` |
| On-prem first, opt-in | license module `federated` | `licensegate` |

**Итог:** 🟡 — модуль есть, не в базовой поставке

---

## ADR-0003 — Donor strategy

| Решение | Код | Тест |
|---------|-----|------|
| Идеи, не копипаста кода | правило `.cursor/rules/donor-strategy.mdc` | — |
| Sigma как данные | `data/sigma-corpus/` | detection-engine sigma lint |

**Итог:** ✅ (процесс + данные)

---

## ADR-0004 — Storage and retention

| Решение | Код | Тест |
|---------|-----|------|
| Kafka hot path | `services/ingest-gateway`, `deploy/docker-compose.prod.yml` | e2e smoke |
| ClickHouse + MinIO | `services/event-writer`, `data-lake/` | consumer tests |
| CMDB не в CH | `services/control-plane/internal/store` | parity_test |

**Итог:** ✅

---

## ADR-0005 — Module independence

| Решение | Код | Тест |
|---------|-----|------|
| `licensegate.Module*` | `services/platform/licensegate/gate.go` | `edition_matrix_test.go` |
| Bundles | `editions-control.yaml` | bundle tests в matrix |
| Compose profiles | `deploy/docker-compose.prod.yml` profiles | — |

**Итог:** ✅

---

## ADR-0006 — Coverage gaps (стратегия)

| Пробел (P0 примеры) | Код | Статус |
|---------------------|-----|--------|
| ITDR | `detection-engine/internal/itdr/` | 🟡 rules |
| Tamper protection | `era-agent-core` tamper | 🟡 Фаза 1 detect |
| Risk-based alerting / FP | `risk/`, correlator | 🟡 dedup only |
| Sigma→MITRE runtime | tags в YAML, не на alert | 🟡 |
| Case management | `control-plane` cases API | ✅ |
| TIP/STIX | `detection-engine/internal/tip/` | ✅ |
| CMDB/Inventory | этап 5 | ✅ |
| Chain of custody | `platform/custody` | ✅ |
| Compliance | `services/compliance` | 🟡 |
| NDR | `detection-engine/internal/ndr/` | 🟡 |
| Deception | `services/deception` | 🟡 |
| CTEM | `services/ctem` | 🟡 |

**Итог:** 📋 — не все P0/P1 закрыты на 100%

---

## ADR-0007 — ClickHouse schema

| Решение | Код | Тест |
|---------|-----|------|
| Таблица `events` | `data-lake/`, event-writer | integration |
| Inventory history raw | `xdr.inventory` topic | ingest validate |

**Итог:** ✅; typed inventory columns — 🟡 (ADR-0011 отложено)

---

## ADR-0008 — Ingest gRPC

| Решение | Код | Тест |
|---------|-----|------|
| `PushEvents` | `services/ingest-gateway/internal/grpcserver` | `server_test.go` |
| Loadgen AC2 | `cmd/loadgen` | `run-loadgen-prod.ps1` |

**Итог:** ✅

---

## ADR-0009 — PII + agent budget

| Решение | Код | Тест |
|---------|-----|------|
| Redaction на агенте | `crates/era-agent` sanitize | `tests/golden_pii.rs` |
| Budget bench | `crates/era-agent-core/src/budget_guard.rs` `check_process_memory` | CI (`ci-gates-stage10.ps1`) |
| `pii_sanitized` gate | ingest validate | `validate_test.go` |

**Итог:** ✅

---

## ADR-0010 — Licensing

| Решение | Код | Тест |
|---------|-----|------|
| Ed25519 offline license | `services/license`, `crates/era-license` | `license/internal/license/*_test.go` |
| Lease (hybrid) | `lease.go`, `era-keygen issue-lease` | `lease_test.go` |
| Sealed clock anti-rollback | validate | golden |
| HSM в проде | KMS abstraction в pam | ⏸ external |

**Итог:** ✅ dev; HSM — ⏸

---

## ADR-0011 — CMDB / ITAM

| Решение | Код | Тест |
|---------|-----|------|
| Inventory plugin snapshot | `crates/era-plugin-inventory` | golden dpkg sample |
| Kafka `xdr.inventory` | kafka-init, ingest routing | validate_test inventory topic |
| Consumer + merge | `control-plane/internal/inventory/` | `merge_test.go`, golden |
| `asset_software` | `store/sqlite_cmdb.go` | parity_test |
| Финансовый ITAM | contracts, licenses, reconcile API | `cmdb.go` |
| CMDB UI | `ui/assets/` | manual |
| vm ← software | `services/vm` publisher | — |
| Observe network reconcile | `networkreconcile/` (ADR-0020) | `reconcile_test.go` |

**Итог:** ✅ (строка ADR «Observe reconcile отложено» — **устарела**, сделано в этапе 9)

---

## ADR-0012 — Enforcement mode

| Решение | Код | Тест |
|---------|-----|------|
| Policy engine monitor/enforce | `era-agent-core/src/enforce/` | `engine.rs` tests, fuzz |
| Fail-open, monitor before enforce | `engine.rs`, orchestrator | unit |
| Plugins app/device/bitlocker | `era-plugin-appcontrol` etc. | golden status |
| CP policy API | `control-plane/internal/api/enforcement.go` | go test api |
| UI | `ui/enforcement/` | — |
| Kernel minifilter / eBPF prod | — | ⏸ external |
| WHQL driver signing | — | ⏸ external |

**Итог:** 🟡 monitor-ready

---

## ADR-0013 — ERA PAM

| Решение | Код | Тест |
|---------|-----|------|
| Vault AES-GCM + seal | `services/pam/internal/vault/` | vault tests |
| Shamir 2-of-3 | `pam/internal/shamir/` | golden |
| Checkout RBAC+TTL | `pam/internal/checkout/` | api tests |
| SSH proxy stub | `pam/internal/api/server.go` | — |
| Session recording | `platform/privilegedsession`, `dlp` | — |
| Custody chain | `platform/custody` | custody tests |
| RDP proxy prod | — | ⏸ external |
| HSM crypto audit | `software-sealed-dev` KMS | ⏸ external |
| Kafka `xdr.privileged` | compose pam profile | — |

**Итог:** 🟡 MVP; §5 «не в MVP» — не в коде

---

## ADR-0014 — Multi-product monorepo

| Решение | Код | Тест |
|---------|-----|------|
| `editions-control.yaml` | корень репо | edition_matrix |
| Сервисы по папкам | `services/*`, `crates/*` | — |
| Rename `era-one` | репо всё ещё `era-xdr` | ❌ отложено |

**Итог:** 🟡

---

## ADR-0016 — UEM scope vs Ivanti

| Решение | Код | Тест |
|---------|-----|------|
| §4 Service ITSM | `services/service-desk` | go test |
| §3 Provision PXE | `services/provision` | go test |
| Deploy/patch | `era-plugin-deploy`, CP deploy API | cargo/go test |
| Device Control | этап 6 plugin | stub |
| MDM/Mobile UEM | — | ❌ DECLINE |
| VPN/ZTNA | — | ❌ INTEGRATE-ONLY |
| Field rollout | — | ⏸ |

**Итог:** 🟡 server IT-Ops MVP

---

## ADR-0017 — Vision One on-prem patterns

| Модуль | Код | Тест |
|--------|-----|------|
| §1 Workbench timeline | `event-writer /api/timeline`, `ui/workbench/` | control-plane workbench |
| §2 Exposure score | `detection-engine/internal/api/exposure.go` | — |
| §3 BYO-EDR | `crates/era-collectors` | cargo test |
| §4 Virtual Patching | enforcement hooks + monitor | ⏸ enforce external |

**Итог:** 🟡 1–3 ✅, §4 monitor-only

---

## ADR-0018 — Sovereign Hybrid

| Решение | Код | Тест |
|---------|-----|------|
| Hybrid relay module | `control-plane/internal/hybrid/` | `relay_e2e_test.go`, health redaction |
| Lease renew | `cloud-portal`, `license/lease` | lease_test |
| Update Service bundles | `services/update-service` | `bundle_test.go` wire golden |
| CRL pull | hybrid relay + portal | — |
| Connected OFF default | compose без profile `connected` | — |
| Egress audit | hybrid + `/api/v1/audit` | — |
| TI-outbound | — | ❌ не в MVP |
| Health B/C | — | ❌ не в MVP |
| Multi-tenant SaaS | — | ❌ ступень 4 |
| Managed private cloud K8s | Helm `deploy/helm/era-one` | helm-template-check |

**Итог:** 🟡 Hybrid-0 ✅; ступени 3–4 частично

---

## ADR-0019 — Agent orchestrator

| Решение | Код | Тест |
|---------|-----|------|
| `era-agent-core` split | `crates/era-agent-core/` | cargo test |
| `era-plugin-sdk` | `crates/era-plugin-sdk/` | — |
| Scheduler + license-gate | `orchestrator.rs`, scheduler | unit |
| OTA verify | `ota/` in core | verify tests |
| Budget-guard | bench + guard | `agent_budget.rs` |
| Plugins: inventory, enforce*, deploy | `era-plugin-*` | per-crate |

**Итог:** ✅; vuln/enforce prod hooks — по этапам

---

## ADR-0020 — Observe + CMDB reconcile

| Решение | Код | Тест |
|---------|-----|------|
| Path A PRTG/Zabbix/syslog | `services/observe/internal/adapters/` | golden prtg, syslog |
| Path B SNMP/discovery sim | `observe/internal/snmp`, `discovery` | — |
| NetFlow line | `observe/internal/netflow/` | golden |
| Ingest → `xdr.network` | `observe/internal/ingest/` | api test |
| CMDB network assets | `networkreconcile/`, CP API | `reconcile_test.go` |
| Correlation | `correlator ObserveNetworkEndpoint` | `engine_test.go` |
| Полный NMS / Nmap | — | ❌ не в MVP |
| Боевой SNMP poll | sim only | 🟡 |

**Итог:** 🟡 MVP Path A+B

---

## ADR-0022 — Detection Content Governance

| Решение | Код | Тест |
|---------|-----|------|
| Sigma corpus ~600 | `data/sigma-corpus/` | lint at startup |
| Sigma MVP matcher | `detection-engine/internal/sigma/` | golden tests |
| Risk dedup 15 min | `detection-engine/internal/risk/` | `golden_test.go` |
| Correlation chains | `detection-engine/internal/correlator/` | `engine_test.go` |
| STIX / national IoC | `detection-engine/internal/tip/` | stix tests |
| MITRE eval scenarios | `data/mitre-eval/` | `mitreval/scenarios_test.go` |
| MITRE tags → alert runtime | — | ❌ Фаза 2 |
| Analyst suppression UI | — | ❌ Фаза 2 |
| FP feedback outbound | ADR-0018 §5 | ❌ не в MVP |
| CVE content pipeline | bundle kind `cve-feed` | 🟡 нет `data/cve-feed/` |
| Coverage heatmap UI | — | ❌ Фаза 2 |

**Итог:** 🟡 — корпус и базовая детекция ✅; governance workflow и MITRE runtime — [ ]

---

## ADR-0023 — AI Investigation Explainability

| Решение | Код | Тест |
|---------|-----|------|
| Investigate API | `ai-core/internal/investigate/` | investigate tests |
| Storyline + verdict | `investigate.go` | pilot checklist |
| Heuristic MITRE | `inferMitre` | — |
| On-prem LLM narrative | optional Ollama/vLLM | — |
| Auto-case malicious/suspicious | `ai-core/internal/api/server.go` | — |
| Custody hashchain (PAM) | `platform/custody/hashchain.go` | custody tests |
| Investigation audit log | — | ❌ Фаза 2 |
| Evidence chain verdict→custody | — | ❌ Фаза 2 |
| Attack graph UI | workbench partial | ❌ Фаза 2 |
| Model version pinning | ai-pack bundle | 🟡 |

**Итог:** 🟡 — triage MVP ✅; forensic-grade trail — [ ]

---

## Темы pre-pilot (экспертный чеклист)

Сводка вопросов, типичных для госсектора / бывших SSPS — маппинг на ADR:

| Тема | Честный статус | ADR |
|------|----------------|-----|
| FP / alert fatigue | dedup + correlation 🟡; suppression UI ❌ | 0022, 0006 P1 |
| Sigma + MITRE | корпус ✅; runtime map ❌ | 0022 |
| Air-gap updates | bundles ✅; IoC отдельным каналом | 0018 §3.2.1, 0022 |
| Tamper protect | detect ✅; prevent ⏸ WHQL | 0006, 0012 |
| 10k+ scale | 10k ev/s target ⏸ field; не 10k hosts proof | AC2, Field-Server-Sizing |
| AI audit trail | storyline ✅; custody chain ❌ | 0023 |

---

## Гейты вне кода (сводка)

| Гейт | ADR / этап | Что нужно |
|------|------------|-----------|
| Реальный пилот, pen-test | 1 | заказчик, field |
| DPA / AZ data-flow ops | 2 | legal/ops |
| Подпись драйвера WHQL | 6, 12, 17 | external |
| Tamper prevent (kernel) | 6, 12 | external WHQL |
| Provision rollout | 7 | field |
| Vault HSM audit, RDP review | 8, 13 | external |
| Кластер 10k ev/s, soak 7×24 | 1, 10 | field |

---

## ADR-0024 — Product families (Control / Communications / Office)

| Решение | Код | Тест |
|---------|-----|------|
| `products.yaml` | корень репо | `platform/manifest/products_test.go` |
| Shared identity | `services/platform/identity` | `identity/store_test.go` |
| Shared tenant | `services/platform/tenant` | `tenant/store_test.go` |
| Admin portal shell | `services/platform/adminportal`, `cmd/admin-portal` | `adminportal/shell_test.go` |
| Comms vision | `docs/products/ERA-Communications-Vision.md`, `editions-comms.yaml` | — |
| Office stub | `services/docs/cmd/docs` | `docs/cmd/docs/main_test.go` |
| Deploy profiles | `deploy/profiles/*.yaml` | — |

**Итог:** 🟡 Control GA + shared platform MVP; Comms/Office roadmap

---

## Как обновлять

При закрытии ADR-пункта:
1. Добавить строку в таблицу ADR выше (код + тест).
2. Обновить статус в `docs/adr/00XX-*.md`.
3. При необходимости — `Implementation-Roadmap.md`, Blueprint §5.
4. Прогнать `scripts/ci-gates-stage10.ps1` или целевой `go test`/`cargo test`.

**Связано:** [Implementation-Roadmap.md](Implementation-Roadmap.md) · [Hardening-Scale-Spec.md](Hardening-Scale-Spec.md) · [ADR-0022](adr/0022-detection-content-governance.md) · [ADR-0023](adr/0023-ai-investigation-explainability.md)
