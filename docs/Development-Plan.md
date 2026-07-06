# ERA XDR — План разработки по фазам

**Версия:** 1.2  
**Дата:** 2 июля 2026 г.  
**Статус:** Активный (Functional MVP закрыт; **Wave GA-1** софт-proof локально PASS; полевая приёмка → Этап 10)

Документ связывает стратегическую дорожную карту
([Blueprint](../reports/ERA-XDR-Architecture-Blueprint.md)) с измеримыми
**Definition of Done (DoD)** на каждую фазу. Тактика первого спринта —
[`MVP-Sprint-1-Spec.md`](MVP-Sprint-1-Spec.md).  
**Production GA** — [`Production-GA-Spec.md`](Production-GA-Spec.md) (Wave GA-1/2/3).  
**Platform vision (endpoint + IT + Observe, не в разработке)** — [`ERA-Platform-Vision.md`](ERA-Platform-Vision.md).

---

## Текущая позиция

| Фаза | Статус | Прогресс |
|---|---|---|
| **Фаза 0** — Проектирование и фундамент | [x] | 100% |
| **Фаза 1** — MVP (ERA Core) | [x] | 100% (Sprint-1 закрыт) |
| **Фаза 2** — AI + домены + SOAR | [x] | 100% (Sprint-2 закрыт) |
| **Фаза 3** — Perimeter + Federated (опция) | [x] | 100% (Sprint-3 закрыт) |
| **Фаза 4** — National immunity (опция) | [x] | 100% (Sprint-4 закрыт) |
| **Фаза GA** — Production GA (Core+AI+Response) | [~] | Софт-proof PASS (`reports/ga-full-20260701-151724.log`); loadgen 10k + пилот → [blocked: field/Этап 10] |

**Сейчас:** Functional MVP (Фазы 0–4) закрыт. **Wave GA-1** — локальный софт-proof закрыт
(1 июля 2026); Platform phase — **Этап 10** (hardening soft-complete). Post-GA gaps
(детект-контент, AI audit trail) — [`ADR-0022`](adr/0022-detection-content-governance.md),
[`ADR-0023`](adr/0023-ai-investigation-explainability.md), [`Implementation-Roadmap.md`](Implementation-Roadmap.md) §Post-GA.

---

## Фаза GA — Production GA

**Цель:** зрелый продукт для банка/госсектора: реальный capture Win+Linux, prod stack,
on-prem AI/SOAR, пилот с подписанным checklist.

**Издания v1.0 GA:** ERA Core + ERA Control AI + ERA Response (Vuln/Federated/National → Wave GA-3).

### Программа волн

| Волна | Фокус | Спека |
|---|---|---|
| **GA-1** | Sellable Core+AI+Response, Win+Linux | Sprint-5 (`S5-*`) |
| **GA-1.1** | macOS capture, React+SSO, loadgen 10k proof | Sprint-8 (`S8-*`) |
| **GA-2** | ITDR, TIP, custody, mTLS mesh, chaos, soak | Sprint-6 (`S6-*`) |
| **GA-3** | Vuln GA, Federated/National/Perimeter, Helm | Sprint-7 (`S7-*`) |

> **Примечание:** в спеках нет оценок в человеко-днях — для нашей команды они не несут смысла; трекинг только по статусу задач и критериям приёмки (F-GA / AC).

### Definition of Done (Wave GA-1)

См. **F-GA-1 … F-GA-15** в [`Production-GA-Spec.md`](Production-GA-Spec.md).

Критический путь GA-1 (старт):

| ID | Задача | Статус |
|---|---|---|
| S5-2 | SQLite store control-plane | [x] (`store/sqlite.go`, `ERA_STORE_PATH`) |
| S5-3 | `docker-compose.prod.yml` | [x] (`deploy/docker-compose.prod.yml`) |
| S5-4 | Linux auditd capture | [x] (`linux_audit.rs`, тесты PASS) |
| S5-5 | Windows Sysmon capture | [x] (`windows_events.rs`, JSONL/wevtutil) |
| S5-6 | `ERA_PRODUCTION=1` без stub | [x] (`production.rs`, `sysinfo_cap.rs`) |
| S5-18 | `run-ga1-smoke.ps1` | [x] |
| S5-19 | Install-Guide-GA | [x] (sketch) |
| S5-20 | Pilot checklist | [x] (sketch) |

**Фаза GA-1 закрыта**, когда все F-GA-* = PASS и Pilot-Readiness-Checklist подписан.

### Следующий цикл — Sprint-8 (GA-1.1) — **закрыт (софт)**

Все S8-* реализованы. Остаётся: прогон `run-loadgen-prod.ps1` на prod и подпись Pilot-Readiness-Checklist.

---

## Фаза 0 — Проектирование и фундамент

**Цель:** зафиксировать архитектуру, контракты и dev-окружение до начала
продуктовой разработки. Без E2E-пайплайна.

**Издание:** не продаётся отдельно — инфраструктура для всех фаз.

### Scope

- ADR (0001–0010), Blueprint, MVP Sprint-1 Spec.
- Контракты `proto/era/v1/` (Envelope, IngestService).
- ClickHouse DDL + storage policy (`deploy/clickhouse/`).
- Dev-окружение (`deploy/docker-compose.dev.yml`).
- Cursor rules, лицензирование (`services/license`, ADR-0010).
- Скелеты `era-agent`, `ingest-gateway`.

### Definition of Done (DoD)

| # | Критерий | Доказательство |
|---|---|---|
| F0-1 | ≥ 10 ADR в статусе Accepted/Implemented | `docs/adr/` |
| F0-2 | Единый конверт и gRPC-ingest описаны в proto | `envelope.proto`, `ingest.proto` |
| F0-3 | ClickHouse DDL применяется при `docker compose up` | `SHOW TABLES FROM era_xdr` |
| F0-4 | Kafka-топики `xdr.*` создаются автоматически | `kafka-topics.sh --list` |
| F0-5 | Go/Rust toolchains + protoc установлены, тесты зелёные | `go test`, `cargo test` |
| F0-6 | Офлайн-лицензирование: issue/verify/revoke + anti-rollback | `services/license` тесты |
| F0-7 | Cursor rules покрывают модули, безопасность, приёмку | `.cursor/rules/` |

**Фаза 0 закрыта**, когда все F0-* = PASS и начат Sprint-1 с codegen (S1-1).

---

## Фаза 1 — MVP (ERA Core)

**Цель:** доказать **сквозной конвейер телеметрии** на одной ноде:
`agent → ingest-gateway → Kafka → ClickHouse → query` (+ минимальная детекция).

**Издание:** ERA Core (базовая поставка).

### Scope

| Модуль | Путь | Sprint-1 |
|---|---|---|
| Агент | `crates/era-agent` | capture (stub→ETW), PII, buffer, отправка |
| Шлюз приёма | `services/ingest-gateway` | gRPC PushEvents, Kafka producer |
| Озеро данных | `deploy/clickhouse` | consumer Kafka → `events` |
| Контракты | `proto/`, `gen/go/`, `crates/era-proto` | codegen, Apicurio |
| Детекция (мин.) | `services/detection-engine` | Sigma on-process (опц. конец фазы) |
| UI (мин.) | `ui/` | таблица последних событий (S1-10) |

**Не входит:** AI Core, SOAR, коллекторы email/identity/cloud, mTLS/PKI hardening,
типизированные доменные таблицы ClickHouse, federated learning.

### Definition of Done (DoD)

| # | Критерий | Измеримо | Связь |
|---|---|---|---|
| F1-1 | E2E-сценарий Sprint-1 §4 проходит полностью | 100% шагов | AC1 |
| F1-2 | Пропускная способность (1 нода dev) | ≥ 10 000 ev/s | AC2 |
| F1-3 | At-least-once без потерь при штатной работе | 0 потерь | AC3 |
| F1-4 | Дедуп по `event_id` (ReplacingMergeTree) | 0 дублей при повторе батча | AC4 |
| F1-5 | PII не попадает в ClickHouse в открытом виде | golden-тест PASS | AC5 |
| F1-6 | Бюджет агента на dev-нагрузке | CPU < 2%, RAM < 150 МБ | AC6 |
| F1-7 | Backpressure: gateway down → агент буферизует и досылает | интеграционный тест | AC7 |
| F1-8 | Unit-тесты agent + gateway зелёные | CI pass | AC8 |
| F1-9 | Codegen Go/Rust из proto в CI, Apicurio BACKWARD-check | `scripts/gen-proto` + registry | S1-1 |
| F1-10 | Документация: Sprint-1 backlog [x], Blueprint §5 обновлён | task-acceptance | — |

**Фаза 1 закрыта**, когда F1-1 … F1-10 = PASS и демо E2E проведено
на dev-окружении без моков.

### Sprint-1 backlog (тактика)

| ID | Задача | Статус |
|---|---|---|
| S1-1 | Codegen Go/Rust из proto + Apicurio | [x] (gen/go + era-proto, тесты PASS, Apicurio OK) |
| S1-2 | prost/tonic в era-agent | [ ] |
| S1-3 | gRPC IngestService в gateway | [ ] |
| S1-4 | Kafka producer в gateway | [ ] |
| S1-5 | Consumer Kafka → ClickHouse | [ ] |
| S1-6 | ETW/Sysmon capture (Windows) | [ ] |
| S1-7 | Golden-тест PII | [ ] |
| S1-8 | Бенчмарк бюджета агента в CI | [ ] |
| S1-9 | Нагрузочный тест 10k ev/s | [ ] |
| S1-10 | UI «последние события» | [ ] |

---

## Фаза 2 — Расширение платформы (ERA Control AI / Response / Vuln)

**Цель:** превратить конвейер телеметрии в полноценную XDR-платформу с
корреляцией, hunting, реагированием и покрытием дополнительных доменов.

**Издания:** ERA Control AI · ERA Response · ERA Vuln (upsell-модули, ADR-0005/0010).

### Scope

- **AI Core** (`ai-core/`): SOC Analyst, threat hunting, baseline/анomaly on-server.
- **Detection Engine** (`services/detection-engine`): Sigma + корреляция + MITRE.
- **SOAR** (`soar/`): плейбуки, изоляция, ticketing-интеграции.
- **Коллекторы** (`crates/era-collectors`): email, identity, cloud, network logs.
- **VM** (`services/vm`): интеграция findings → Envelope → Kafka.
- **Control Plane** (`services/control-plane`): политики, правила, heartbeat.
- **Операционный слой (P0 из ADR-0006):** Case/Incident Mgmt, Asset Inventory,
  TIP, Chain of Custody, Compliance reporting, Tamper Protection (агент).

### Definition of Done (DoD)

| # | Критерий | Измеримо |
|---|---|---|
| F2-1 | AI Core: автономное расследование инцидента on-prem (air-gap) | демо: alert → storyline → verdict |
| F2-2 | ≥ 3 домена телеметрии помимо process (network, auth, file) | E2E по каждому |
| F2-3 | Кросс-доменная корреляция (≥ 1 multi-stage detection) | синтетический APT-сценарий |
| F2-4 | SOAR: ≥ 3 плейбука (isolate host, block IP, create ticket) | интеграционные тесты |
| F2-5 | VM findings публикуются как Envelope в Kafka | E2E vm → ClickHouse |
| F2-6 | Case Management: alert → case → assign → close | UI + API |
| F2-7 | Asset Inventory: ≥ 90% хостов с актуальным профилем | метрика coverage |
| F2-8 | Tamper Protection: попытка kill агента → alert + self-heal | red-team тест |
| F2-9 | Лицензирование: upsell-модули включаются/выключаются офлайн | ADR-0010 e2e |
| F2-10 | Detection content: ≥ 500 Sigma-правил в production corpus | corpus + CI lint |

### Sprint-2 backlog (тактика)

| ID | Задача | Статус |
|---|---|---|
| S2-1 | control-plane: политики, assets, license gate | [x] (`services/control-plane`, тесты PASS) |
| S2-2 | detection-engine: Sigma + запись detections | [x] (Kafka consumer + CH writer) |
| S2-3 | Кросс-доменная корреляция APT | [x] (`era-apt-lateral-movement`, unit test) |
| S2-4 | ai-core: alert → storyline → verdict | [x] (`POST /api/v1/investigate`) |
| S2-5 | SOAR: 3 плейбука | [x] (isolate_host, block_ip, create_ticket) |
| S2-6 | VM findings → Envelope → Kafka | [x] (`services/vm/internal/publisher`) |
| S2-7 | Коллекторы network/auth/file | [x] (`ERA_DOMAIN_STUB`, era-agent) |
| S2-8 | Tamper protection агента | [x] (`ERA_TAMPER_SIM`, unit test) |
| S2-9 | Case Management API + UI | [x] (`ui/cases/`, PATCH lifecycle) |
| S2-10 | Sigma corpus ≥500 + lint | [x] (`data/sigma-corpus`, `scripts/gen-sigma-corpus`) |
| S2-11 | Asset Inventory UI | [x] (`ui/assets/`, coverage API) |
| S2-12 | E2E Phase-2 smoke | [x] (`scripts/run-phase2-e2e.ps1` PASS) |

### DoD proof (Фаза 2)

| # | Статус | Доказательство |
|---|---|---|
| F2-1 | PASS | ai-core investigate API + storyline из ClickHouse |
| F2-2 | PASS | network/auth/file в CH при `ERA_DOMAIN_STUB=1` |
| F2-3 | PASS | correlator `APTChain` test + rule `era-apt-lateral-movement` |
| F2-4 | PASS | `go test` soar playbooks (3 actions) |
| F2-5 | PASS | vm publisher → `xdr.raw` Kafka |
| F2-6 | PASS | control-plane cases API + `ui/cases/` |
| F2-7 | PASS | asset register heartbeat + coverage ≥0.9 dev seed |
| F2-8 | PASS | tamper alert envelope + self-heal log |
| F2-9 | PASS | `licensegate` + `/api/v1/license/modules` |
| F2-10 | PASS | 500 rules, `sigma.Lint` в detection-engine startup |

**Фаза 2 закрыта**, когда платформа продаётся как «зрелый XDR» для
банка/госсектора с операционным контуром SOC.

---

## Фаза 3 — Perimeter + Federated (опция)

**Цель:** расширить периметр (WAF, NGFW, DLP/UAM) и опционально включить
federated learning **внутри организации**.

**Издания:** + ERA Federated *(платная опция, не по умолчанию)*.

### Scope

- WAF (Coraza-паттерн), NGFW/eBPF (Cilium-паттерн), DLP/UAM (Teleport-паттерн).
- Federated learning: обмен градиентами/сигнатурами между подразделениями
  **без** выноса сырых данных.
- NDR, Deception, CTEM+BAS (P1 из ADR-0006).

### Definition of Done (DoD)

| # | Критерий | Измеримо |
|---|---|---|
| F3-1 | WAF: блокировка OWASP Top-10 сценария | pen-test PASS |
| F3-2 | NGFW/eBPF: сетевые политики + telemetry в Envelope | E2E network events |
| F3-3 | DLP/UAM: аудит привилегированных сессий | запись + alert |
| F3-4 | Federated (опция): 2+ tenant-зоны обучают общую модель без PII | DP/aggregate proof |
| F3-5 | NDR: детекция lateral movement по сети | MITRE T1021 сценарий |
| F3-6 | Модуль Federated **выключен по умолчанию**; активация только по лицензии | license gate test |

### Sprint-3 backlog (тактика)

| ID | Задача | Статус |
|---|---|---|
| S3-1 | WAF OWASP Top-10 | [x] (`services/waf`, pen-test unit) |
| S3-2 | NGFW policies + Envelope telemetry | [x] (`services/ngfw`) |
| S3-3 | DLP/UAM session audit + alert | [x] (`services/dlp`) |
| S3-4 | Federated DP hub (2+ zones) | [x] (`services/federated`) |
| S3-5 | NDR T1021 lateral movement | [x] (`detection-engine/ndr`) |
| S3-6 | Federated off by default | [x] (`licensegate.DevDefault`) |
| S3-7 | Deception honeypot | [x] (`services/deception`) |
| S3-8 | CTEM/BAS lateral sim | [x] (`services/ctem`) |
| S3-9 | platform/envelope publisher | [x] |
| S3-10 | E2E Phase-3 smoke | [x] (`run-phase3-e2e.ps1` PASS) |

### DoD proof (Фаза 3)

| # | Статус | Доказательство |
|---|---|---|
| F3-1 | PASS | WAF blocks SQLi/XSS/traversal/cmdi/ssrf (unit) |
| F3-2 | PASS | NGFW evaluate + Kafka `xdr.network` |
| F3-3 | PASS | DLP session alert on exfil command (unit) |
| F3-4 | PASS | Federated hub FedAvg+DP, 2 zones test |
| F3-5 | PASS | NDR `era-ndr-t1021-lateral-movement` test |
| F3-6 | PASS | `TestDevDefaultFederatedOff` |

**Фаза 3 закрыта**, когда perimeter-модули интегрированы и federated
работает как opt-in upsell.

---

## Фаза 4 — National immunity (опция)

**Цель:** межорганизационный обмен **знаниями** (не сырыми данными) для
формирования «общегосударственного иммунитета».

**Издание:** + ERA National *(платная опция)*.

### Scope

- Национальный STIX/TAXII hub (on-prem, air-gap friendly).
- Differential Privacy для агрегатов угроз.
- PQC-readiness для долгосрочных подписей лицензий/артефактов.

### Definition of Done (DoD)

| # | Критерий | Измеримо |
|---|---|---|
| F4-1 | STIX/TAXII hub: публикация/подписка IOC между ≥ 2 организациями | E2E exchange |
| F4-2 | Ни один Envelope с PII не покидает контур организации | audit + golden |
| F4-3 | Агрегированные сигнатуры улучшают детекцию у подписчиков | measurable Δ detection rate |
| F4-4 | National-модуль **выключен по умолчанию**; только по лицензии | license gate |
| F4-5 | Регуляторная отчётность (АЗ/ЦБ) из платформы | шаблон отчёта + demo |

### Sprint-4 backlog (тактика)

| ID | Задача | Статус |
|---|---|---|
| S4-1 | STIX/TAXII hub (2 org E2E) | [x] |
| S4-2 | PII audit export | [x] |
| S4-3 | National IOC detection delta | [x] |
| S4-4 | National off by default | [x] |
| S4-5 | AZ/CB regulatory report | [x] |
| S4-6 | DP aggregates | [x] |
| S4-7 | PQC-readiness | [x] |
| S4-8 | E2E Phase-4 smoke | [x] |

### DoD proof (Фаза 4)

| # | Статус | Доказательство |
|---|---|---|
| F4-1 | PASS | `taxii.TestE2EExchangeTwoOrgs` org-a publish / org-b poll |
| F4-2 | PASS | `sanitize.TestAuditBundleBlocksPII` + 422 on hub |
| F4-3 | PASS | `signature.TestDetectionDeltaImproves` + `tip` tests |
| F4-4 | PASS | `TestDevDefaultFederatedOff` + `TestNationalLicenseGate` |
| F4-5 | PASS | `compliance` report `era-reg-az-cb-v1` demo API |

**Фаза 4 закрыта**, когда нацхаб доказал ценность на пилоте ≥ 2 заказчиков
без нарушения air-gap инвариантов (ADR security-and-air-gap).

---

## Принципы планирования

1. **Фаза N+1 не начинается**, пока DoD фазы N не закрыт (или явно
   задокументировано отклонение в ADR).
2. **Приёмка = доказательство** (тест, лог, метрика) — см. `task-acceptance`.
3. **Federated/National — upsell**, не блокируют базовую поставку.
4. **Contract-first:** изменение proto → codegen → все потребители в одном PR.
5. **MVP-дисциплина:** stub-capture допустим до E2E; ETW — параллельно (S1-6).

---

## Связанные документы

- [ERA-XDR Architecture Blueprint](../reports/ERA-XDR-Architecture-Blueprint.md)
- [MVP Sprint-1 Spec](MVP-Sprint-1-Spec.md)
- [Production GA Spec](Production-GA-Spec.md)
- [Market Positioning AZ](Market-Positioning-AZ.md)
- [GA Master Execution Plan](GA-Master-Execution-Plan.md)
- [Install Guide GA](Install-Guide-GA.md)
- [ADR-0005 Module independence](adr/0005-module-independence-and-packaging.md)
- [ADR-0006 Coverage gaps](adr/0006-coverage-gaps-strategic-bets-and-practices.md)
