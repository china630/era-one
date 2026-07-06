# ERA XDR — Production GA Specification

**Версия:** 1.1  
**Дата:** 9 июня 2026 г.  
**Статус:** Активный — **софтверный беклог закрыт**; остаются полевые тесты и чек-листы (пилот, soak, pen-test, loadgen proof на prod).  
**Контекст:** переход от Functional MVP (Фазы 0–4) к **зрелому продукту** для банка/госсектора.

## Целевая поставка v1.0 GA

| Параметр | Значение |
|---|---|
| **Платформы агента** | Windows + Linux + macOS (unified log export) |
| **Издания в GA v1.0** | **ERA Core** + **ERA Control AI** + **ERA Response** |
| **Не в GA v1.0 (опции позже)** | ERA Vuln (полный GA), Federated, National, Perimeter — Wave GA-3+ |
| **Контур** | On-prem, air-gap friendly (ADR security-and-air-gap) |

---

## Статус программы

| Волна | Софт | Остаётся (не код) |
|---|---|---|
| **GA-1 + GA-1.1 (S5, S8)** | [x] | Loadgen proof на prod (`run-loadgen-prod.ps1`), pilot sign-off |
| **GA-2 (S6)** | [x] | Soak 7×24, pen-test, pilot feedback |
| **GA-3 (S7)** | [x] | Legal audit doc (S7-13), pilot 2-org national |

---

## Definition of Done — Production GA (Wave GA-1)

| ID | Критерий | Софт | Доказательство (чек-лист / пилот) |
|---|---|---|---|
| F-GA-1 | Win capture | [x] | Pilot Win hosts |
| F-GA-2 | Linux capture | [x] | Pilot Linux hosts |
| F-GA-3 | Prod deploy | [x] | compose up |
| F-GA-4 | Persistent CP | [x] | restart test |
| F-GA-5 | ≥10k ev/s 5 мин | [x] script | `run-loadgen-prod.ps1` PASS |
| F-GA-6 | mTLS agent→gateway | [x] | TLS env on pilot |
| F-GA-7 | Case lifecycle + UI | [x] | `/ui/portal/` |
| F-GA-8 | Asset coverage ≥90% | [x] API | метрика на пилоте |
| F-GA-9 | Sigma ≥100 | [x] | corpus lint |
| F-GA-10 … F-GA-14 | AI, SOAR, license, docs | [x] | smoke scripts |
| F-GA-15 | Pilot checklist signed | [x] template | **подпись на пилоте** |

---

## Sprint-5 Backlog (Wave GA-1) — детально

| ID | Задача | F-GA | Статус |
|---|---|---|---|
| S5-1 | Production-GA-Spec + Development-Plan Phase GA | — | [x] |
| S5-2 | SQLite store control-plane | F-GA-4 | [x] |
| S5-3 | `docker-compose.prod.yml` + prod README | F-GA-3 | [x] |
| S5-4 | Linux auditd capture | F-GA-2 | [x] |
| S5-5 | Windows Sysmon/EVTX capture | F-GA-1 | [x] |
| S5-6 | `ERA_PRODUCTION=1` без stub | F-GA-1/2 | [x] |
| S5-7 | Agent installer | F-GA-14 | [x] |
| S5-8 | mTLS ingest-gateway | F-GA-6 | [x] |
| S5-9 | Loadgen 10k + gate | F-GA-5 | [x] (proof — checklist) |
| S5-10 | Curated Sigma 100 | F-GA-9 | [x] |
| S5-11 | Case API notes/timeline | F-GA-7 | [x] |
| S5-12 | UI + portal | F-GA-7 | [x] |
| S5-13 … S5-27 | AI, SOAR, license, RBAC | F-GA-* | [x] |
| S5-21 | Backpressure integration | F1-7 | [x] |
| S5-28 | E2E demo script | F-GA-15 | [x] |

---

## Sprint-5 Backlog (Wave GA-1) — сводка

| ID | Задача | F-GA | Статус |
|---|---|---|---|
| S5-* | все пункты Sprint-5 | | [x] (см. таблицу выше) |

---

## Sprint-6 Backlog (Wave GA-2)

| ID | Задача | Статус |
|---|---|---|
| S6-1 … S6-10 | ITDR, TIP, custody, tamper, NDR, risk, chaos | [x] |
| S6-4 | Compliance CH + PDF/ZIP | [x] |
| S6-8 | SOC portal + SSO | [x] → `ui/portal/` |
| S6-9 | mTLS mesh (CP TLS + agent↔gateway) | [x] |
| S6-11 | Soak 7×24 | **[checklist]** — полевой тест |
| S6-12 | Pen-test remediation | **[checklist]** — внешний аудит |
| S6-13 … S6-18 | audit, MITRE eval | [x] |
| S6-14 | Postgres store | [x] (`ERA_STORE_DRIVER=postgres`) |
| S6-15 | Backup/restore scripts | [x] (`scripts/backup-prod.ps1`) |
| S6-16 | Kafka RF=3 HA profile | [x] (`deploy/docker-compose.prod-ha.yml`) |
| S6-17 | PII golden CI gate | [x] (`scripts/run-pii-gate.ps1`) |
| S6-19 | Pilot feedback | **[checklist]** |
| S6-20 | GA-2 sign-off | **[checklist]** — `docs/GA-2-Signoff.md` |

---

## Sprint-7 Backlog (Wave GA-3)

| ID | Задача | Статус |
|---|---|---|
| S7-1 … S7-12 | VM, federated, national SQLite, perimeter profiles, OT collector | [x] |
| S7-13 | Federated DP legal audit | **[checklist]** — юр. документ |
| S7-14 … S7-17 | regulatory ZIP, edition matrix tests, helm upgrade, loadgen -agents | [x] |
| S7-18 | GA-3 sign-off | **[checklist]** — `docs/GA-3-Signoff.md` |

---

## Sprint-8 (GA-1.1)

| ID | Задача | Статус |
|---|---|---|
| S8-1 | macOS production capture | [x] (`macos_unified.rs`, `ERA_MACOS_UNIFIED_JSONL`) |
| S8-2 | SOC portal + SSO | [x] (`ui/portal/`) |
| S8-3 | Loadgen prod script | [x] (proof — чек-лист) |
| S8-4 | EVTX parser | [x] (`evtx_parser.rs`, golden test) |
| S8-5 | Install-Guide полный | [x] |
| S8-6 | E2E prod demo | [x] (`run-ga-e2e-prod.ps1`) |

---

## Приёмка софта (автоматизировано)

```powershell
powershell -File scripts/run-ga-full.ps1
# опционально на поднятом prod:
powershell -File scripts/run-loadgen-prod.ps1
```

**Локальный софт-proof (1 июля 2026):** `run-ga-full.ps1` PASS — `reports/ga-full-20260701-151724.log`
(go test всех сервисов + cargo test era-agent/era-collectors + PII-gate; без prod-стека).
Попутно исправлен `crates/era-proto/src/lib.rs` (модуль `era::v1`).

| Критерий | Локально | Перенос |
|---|---|---|
| F-GA-5 loadgen ≥10k×5мин | [blocked: field] | Этап 10 (sizing + кластер) |
| F-GA-8 coverage ≥90% на пилоте | [blocked: field] | реальный пилот |
| F-GA-15 подпись checklist | [blocked: field] | реальный заказчик |

Полный список F-GA-* и детальный Sprint-5 — см. git history v1.0 spec; критерии F-GA-5/8/15 закрываются **доказательством на пилоте/кластере**, не коммитом.

---

## Связь с лицензиями (Core + AI + Response)

```powershell
cd services/license
go run ./cmd/era-keygen issue -modules ai,response -nodes 5000 ...
$env:ERA_LICENSE_MODULES="ai,response"
```

Модули `vm`, `federated`, `national` включаются лицензией и Wave GA-3 compose profiles.
