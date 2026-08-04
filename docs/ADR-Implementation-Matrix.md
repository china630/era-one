# ADR → Код → Тесты (матрица прослеживаемости)

**Дата:** 30 июля 2026 г. (honesty pass — code/logic audit)  
**Назначение:** честная сверка «что решили в ADR» vs «что есть в репозитории».  
**Приёмка Control:** [`Control-Acceptance-System.md`](products/Control-Acceptance-System.md) · [`Control-Evidence-Rules.md`](Control-Evidence-Rules.md) · [`Control-Sprint-Index.md`](Control-Sprint-Index.md)  
**Канон:** [`ERA-Product-Acceptance-Standard.md`](products/ERA-Product-Acceptance-Standard.md) · Shared ownership: [`Shared-Acceptance-System.md`](products/Shared-Acceptance-System.md)

**Легенда статусов** (совместима с каноном §3; для product AC предпочитать колонки Scaffold / Pilot-ready):

| Маркер | Значение |
|--------|----------|
| ✅ | Scaffold: реализовано, есть тест/доказательство |
| 🟡 | Частично / MVP / sim / monitor-only |
| ⏸ | Pilot-ready / внешний гейт (field / WHQL / pen-test) — код не обязан закрывать |
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
| 0009 PII + budget | Accepted | ✅ | process + BYO/DNS redact-or-false; flag after redact |
| 0010 Licensing | Accepted | ✅ | GateFromEnv fail-closed prod path; `ERA_LICENSE_DEV` lab-only; HSM ⏸ |
| 0011 CMDB/ITAM | Implemented | ✅ | CH typed inventory — 🟡 |
| 0012 Enforcement | Implemented lab decision | ✅ AC-E1…E3 | `effect=telemetry_only`; OS block ⏸ WHQL |
| 0013 PAM | Implemented Phase 2 | ✅ | Guacamole video + HSM — ⏸ |
| 0014 Monorepo | Accepted | 🟡 | rename era-one — ❌ отложен |
| 0016 UEM scope | Accepted | ✅ server | MDM/VPN — ❌; field — ⏸ |
| 0017 Vision One patterns | Accepted | 🟡 | vpatch kernel — ⏸ |
| 0018 Hybrid | Implemented Hybrid-0 | 🟡 | SaaS/TI B/C — ❌/⏸ |
| 0019 Agent orchestrator | Implemented | ✅ | не все плагины из vision |
| 0020 Observe | Implemented Path A+B | ✅ | полный NMS — ❌; field lab — ⏸ |
| 0021 Portal + калькулятор | Accepted | 🟡 | статический сайт `site/` + рабочий калькулятор из SSOT `pricing-data.yaml`; тесты `site/test/calculator.test.js` зелёные; контент в развитии |
| 0022 Detection content | Accepted | ✅ PP-1 | корпус ~600, lint ✅; Sigma→MITRE on alert; FP UI/heatmap — ⏸ |
| 0023 AI explainability | Accepted + Phase 3 lite | ✅ | human-on-loop recommend; GateFromEnv on ai-core/soar |
| 0024 Product families | Accepted | 🟡 | `products.yaml`, platform, deploy profiles; Comms/Office ADR ✅ |
| 0025 Shared platform | Accepted | 📋 | ADR + `editions-shared.yaml`; runtime — roadmap |
| 0026 Office engine | Accepted | 📋 | sovereign CRDT + Rust OOXML; no OnlyOffice/GPL |
| 0027 Communications | Accepted | 📋 | Mail Connect, standalone, Office boundary |
| 0031 Perimeter + Resolve | Implemented Phase 2 | ✅ | field/pen-test / live TI ⏸ |

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
| Sigma→MITRE runtime | tags → `mitre_techniques` on alert (`processor`, `sigma.Techniques`) | ✅ |
| Case management | `control-plane` cases API | ✅ API + AuthZ middleware (`ERA_RBAC_TRUST` proxy/api_key; Trusted-Proxy + hop/CIDR) |
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

| Решение | Код | Тест | Scaffold |
|---------|-----|------|----------|
| Redaction на агенте (Process/Auth) | `crates/era-agent-core` sanitize | `tests/golden_pii.rs` | ✅ |
| Budget bench | `crates/era-agent-core/src/budget_guard.rs` | CI (`ci-gates-stage10.ps1`) | ✅ |
| `pii_sanitized` gate | ingest validate | `validate_test.go` | ✅ |
| Collectors BYO/DNS redact-or-false | `era-collectors` byo_edr, dns | unit + golden | ✅ |

**Итог:** ✅ — process path + collectors redact-then-flag; DNS `mode=stub`

---

## ADR-0010 — Licensing

| Решение | Код | Тест | Scaffold |
|---------|-----|------|----------|
| Ed25519 offline license | `services/license`, `crates/era-license` | `license/internal/license/*_test.go` | ✅ |
| Lease (hybrid) | `lease.go`, `era-keygen issue-lease` | `lease_test.go` | ✅ |
| Sealed clock anti-rollback | validate | golden | 🟡 library; path-optional без env |
| Soft GateFromEnv / DevDefault | `platform/licensegate`, ai-core, soar, perimeter, resolve, pam | startup + 403 tests | ✅ prod path fail-closed; lab `ERA_LICENSE_DEV=1` |
| HSM в проде | KMS abstraction в pam | ⏸ external | ⏸ |

**Итог:** ✅ core verify + GateFromEnv fail-closed; HSM — ⏸

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

| Решение | Код | Тест | Scaffold |
|---------|-----|------|----------|
| Policy engine monitor/enforce | `era-agent-core/src/enforce/` | `engine.rs` tests, fuzz | ✅ decisions |
| Fail-open, monitor before enforce | `engine.rs`, orchestrator | unit | ✅ (design; not protect) |
| Plugins app/device/bitlocker | `era-plugin-appcontrol` etc. | golden status + would_block + `effect` | ✅ lab decision |
| CP policy API | `control-plane/internal/api/enforcement.go` | go test api + spoof→401/403; agent Bearer | ✅ AuthZ middleware + hop |
| UI | `ui/enforcement/` | — | 🟡 |
| Lab decision BlockResult | `enforce/engine.rs` apply_block | golden `enforce_vs_monitor` · `effect=telemetry_only` | ✅ |
| User-land LIVE gate | `enforce/user_land.rs` + `apply_block` | golden `enforce_live_user_land` · `effect=user_land_block` | ✅ (not WHQL) |
| Agent token ≠ admin | `control-plane/internal/rbac` | agent GET policy 200 · PUT/escrow 403 | ✅ |
| Kernel stub messaging | `enforce/kernel.rs` | unit + plugin goldens | ✅ Unavailable until WHQL |
| Kernel minifilter / eBPF prod | — | ⏸ WHQL — `ERA-Manage-WHQL-Program.md` | ⏸ |
| WHQL driver signing | — | ⏸ external | ⏸ |

**Итог:** ✅ AC-E1…E3 + E2b user-land LIVE; **AC-E4 / WHQL kernel ⏸**

---

## ADR-0013 — ERA PAM

| Решение | Код | Тест |
|---------|-----|------|
| Vault AES-GCM + seal | `services/pam/internal/vault/` | vault tests |
| Shamir 2-of-3 | `pam/internal/shamir/` | golden |
| Checkout RBAC+TTL | `pam/internal/checkout/` | api tests |
| SSH TCP proxy + command log | `pam/internal/proxy/ssh_proxy.go` | `ssh_proxy_test.go` |
| RDP TCP proxy (binary relay) | `pam/internal/proxy/rdp_proxy.go` | `rdp_proxy_test.go`, API `TestRDPProxySession` |
| RDP broker inject (P2) | `pam/internal/broker` | no password in JSON |
| Session idle/max (P2) | `pam/internal/sessionpolicy` | timeout custody |
| Metadata recording (P2) | `pam/internal/recording` | artifact+hash |
| Session recording | `platform/privilegedsession`, `dlp` | — |
| Custody chain | `platform/custody` | custody tests |
| RDP inject + graphical recording | — | ⏸ security-review — `PAM-RDP-Security-Review-Checklist.md` |
| HSM crypto audit | `software-sealed-dev` KMS | ⏸ external |
| Kafka `xdr.privileged` | compose pam profile | — |

**Итог:** ✅ MVP + Phase 2 broker/policy/recording code; Guacamole video / HSM ⏸

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
| §4 Service ITSM | `services/service-desk` + `ui/service-desk` | go test + `run-itops-smoke.ps1` |
| §3 Provision PXE | `services/provision` + `ui/provision` | PXE golden + smoke |
| Deploy/patch | `era-plugin-deploy`, CP deploy API | cargo/go test |
| Device Control | `era-plugin-devicecontrol` | golden USB |
| MDM/Mobile UEM | — | ❌ DECLINE |
| VPN/ZTNA | — | ❌ INTEGRATE-ONLY |
| Field rollout | — | ⏸ |

**Итог:** ✅ server IT-Ops MVP; field ⏸

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
| Path B SNMP PollReal + HOST-RESOURCES | `observe/internal/snmp` | `poll_prod_test.go`, metrics_source |
| NetFlow UDP + line | `observe/internal/netflow/` | golden + UDP listener |
| Ingest → `xdr.network` | `observe/internal/ingest/` | api test |
| CMDB network assets | `networkreconcile/`, CP API | `reconcile_test.go` |
| Correlation | `correlator ObserveNetworkEndpoint` | `engine_test.go` |
| Полный NMS / Nmap | — | ❌ не в MVP |
| Field lab SNMP | `run-observe-smoke.ps1` | ⏸ hardware |

**Итог:** ✅ Path A+B code; field / full NMS ⏸/❌

---

## ADR-0031 — Perimeter + Resolve

| Решение | Код | Тест |
|---------|-----|------|
| WAF reverse-proxy + rule pack | `services/waf` | rules golden, api proxy/block |
| NGFW policy decision API | `services/ngfw` | evaluate golden, persist |
| Session DLP (shared PAM) | `services/dlp` | session tests; pam\|perimeter gate |
| Resolve Guard/Trace/Atlas | `services/resolve` | verdict golden, DNS UDP, atlas pack |
| DoH RFC 8484 (P2) | `resolve/internal/doh` | doh golden NXDOMAIN |
| Atlas Update Service pack | `KindAtlasPack` + packs/reload | kinds_test |
| Agent DnsEvent emitter | `era-collectors` dns | emit_dns_event unit |
| Editions mvp + license | `editions-control.yaml` | licensegate KnownModules |
| Datasheets DNS vs ITSM fix | `site/datasheets/*/16-ERA-Resolve.html` | content review |
| Content-DLP inspect (P2) | `dlp/internal/content` | inspect golden |
| WAF body + CRS-lite (P2) | `waf` EvaluateWithBody | body/CRS unit |
| NGFW host apply opt-in (P2) | `ngfw/internal/apply` | noop default |
| Live commercial TI SaaS | — | ⏸ external/content |

**Итог:** ✅ MVP + Phase 2 code; field/pen-test / live TI ⏸

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
| MITRE tags → alert runtime | `sigma.Techniques` + chwriter `mitre_techniques` | processor_mitre_test |
| Analyst suppression UI | CP suppressions + DE cache + ui/cases | suppression_test |
| Coverage heatmap UI | `/api/v1/mitre/coverage` + workbench | mitre coverage golden |
| CVE content pipeline | `data/cve-feed/` + `vm/cvefeed` | feed_test |
| FP feedback outbound | ADR-0018 §5 | ❌ не в MVP |

**Итог:** ✅ DC-01…04 Post-GA code; FP outbound / heatmap CH-seen layer — optional field

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
| Investigation audit log | `investigate.AuditLog` + `/api/v1/investigate/audit` | forensic_test |
| Evidence chain verdict→custody | `SealEvidence` → `custody_root_hash` | forensic_test |
| Attack graph UI | `BuildAttackGraph` + workbench | forensic_test |
| Model version pinning | `ModelVersionHeuristic` + prompt_hash | forensic_test |
| recommended_actions (P3 lite) | `SuggestActions` | recommend golden |
| confirm/reject + custody | `POST .../confirm|reject` | recommend_test |
| SOAR draft handoff | `soar-draft` + `ERA_SOAR_DRAFT_URL` | no auto-execute |

**Итог:** ✅ Phase-2 forensic + Phase 3 lite agentic (human-on-loop); autonomous close ❌

---

## Темы pre-pilot (экспертный чеклист)

Сводка вопросов, типичных для госсектора / бывших SSPS — маппинг на ADR:

| Тема | Честный статус | ADR |
|------|----------------|-----|
| FP / alert fatigue | suppressions UI ✅ + risk dedup | 0022, 0006 P1 |
| Sigma + MITRE | корпус ✅; runtime map on alert ✅ (PP-1); FP UI/heatmap ⏸ | 0022 |
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

**Итог:** ✅ Control Scaffold-Green (AuthZ/license/enforce honesty/PII/AC matrix); Pilot-ready F-GA-5/8/15 + Manage OS-block / WHQL / HSM remain ⏸; Comms/Office **mvp** — honesty pass 2026-07-30 · Scaffold-Green 2026-07-30

---

## Как обновлять

При закрытии ADR-пункта:
1. Добавить строку в таблицу ADR выше (код + тест); для product AC — явно Scaffold vs Pilot-ready.
2. Обновить статус в `docs/adr/00XX-*.md`.
3. Обновить [`Control-Sprint-Index.md`](Control-Sprint-Index.md) при смене волны/издания.
4. При необходимости — `Implementation-Roadmap.md`, Blueprint §5.
5. Прогнать `scripts/ci-gates-stage10.ps1` или целевой `go test`/`cargo test`; лог в `reports/` или CI.
6. Соблюдать [`Control-Evidence-Rules.md`](Control-Evidence-Rules.md) (нет лога — нет `[x]`).

**Связано:** [Implementation-Roadmap.md](Implementation-Roadmap.md) · [Hardening-Scale-Spec.md](Hardening-Scale-Spec.md) · [ADR-0022](adr/0022-detection-content-governance.md) · [ADR-0023](adr/0023-ai-investigation-explainability.md)
