# ADR-0017: On-prem паттерны Vision One — Workbench, Exposure, BYO-EDR, Virtual Patching

**Статус:** Accepted (модули 1–4 Implemented; §4 Virtual Patching — monitor-ready, боевой enforce [gate: external])
**Дата:** 1 июля 2026 г.
**Контекст:** Сопоставление Trend Micro Vision One с ERA One выявило **не облачные**
паттерны (workflow корреляции, exposure management, гибридные сенсоры, server
hardening), которые усиливают суверенную линейку без нарушения air-gap
([`ADR-0003`](0003-repository-structure-and-donor-strategy.md), Blueprint §1).

**Связано:** [`ADR-0006`](0006-coverage-gaps-strategic-bets-and-practices.md) ·
[`ADR-0011`](0011-cmdb-itam-data-model.md) · [`ADR-0012`](0012-agent-enforcement-mode.md) ·
`editions-control.yaml`

---

## Решение

Заимствуем **идеи и модели данных** (не код TM), четыре дополнения к roadmap:

| # | Модуль | Суть (on-prem) | Издание | Фаза |
|---|---|---|---|---|
| 1 | **ERA Workbench** | Единый incident timeline: endpoint + identity + network + email-логи + алерты в одном UI расследования | **ERA Core** (UI + API) | GA-2 / Platform P1 |
| 2 | **ERA Exposure** | Локальный CREM-аналог: risk score = f(уязвимости, критичность актива, детекты, misconfig, identity) | **ERA Vuln** + Core risk engine | GA-2 / Platform P2 |
| 3 | **ERA BYO-EDR Hub** | Приём телеметрии сторонних EDR/SIEM (API/syslog) в тот же lake + корреляция | **era-collectors** + TIP | GA-2 |
| 4 | **Virtual Patching** | Mitigation известных эксплойтов на сервере до выхода патча ОС (OS API / policy hooks) | **ERA Manage** (enforcement plugin) | Platform P2b |

### Сознательно не берём

Cloud data lake, cloud AI, SaaS email gateway, ZTNA, глобальный threat cloud TM, MDR-as-a-service.

---

## 1. ERA Workbench

- **Донор-паттерн:** Vision One Workbench / Search App (идея единого timeline).
- **Реализация:** расширение SOC Portal + API control-plane: merge событий по `case_id` /
  `node_id` / `correlation_id` из ClickHouse и collectors.
- **DoD:** golden-тест timeline на синтетическом multi-source инциденте; RBAC как в Cases.

## 2. ERA Exposure

- **Донор-паттерн:** CREM/ASRM — приоритизация риска, не только список CVE.
- **Реализация:** `services/vm` + risk engine в `detection-engine`; входы: CVE, asset
  criticality (CMDB), активные детекты, posture rules.
- **DoD:** единый exposure score на актив; отчёт «топ-10 рисковых хостов» в контуре.

## 3. ERA BYO-EDR Hub

- **Донор-паттерн:** интеграция сторонних endpoint (SentinelOne → Vision One).
- **Реализация:** адаптеры в `crates/era-collectors` → Envelope → Kafka; нормализация
  в OCSF-подобный конверт; enrichment через TIP.
- **DoD:** ingest тестового feed (JSON/syslog) + появление в Workbench timeline.

## 4. Virtual Patching (ERA Manage)

- **Донор-паттерн:** Apex / Deep Security virtual patch (mitigation без reboot).
- **Реализация:** policy bundle от control-plane; enforcement через
  [`ADR-0012`](0012-agent-enforcement-mode.md) (WDAC/AppLocker, eBPF-LSM) — **блокировка
  вектора**, не замена полноценного патча.
- **DoD:** политика на тестовый CVE-vector; audit event в Envelope; security-review gate.

---

## Лицензирование

| Модуль | `license_module` |
|---|---|
| Workbench | `core` (входит в Core) |
| Exposure | `vm` + `core` (Vuln upsell усиливает) |
| BYO-EDR Hub | `core` (базовый 1 адаптер); доп. коннекторы — опция |
| Virtual Patching | `manage` |

Обновить `editions-control.yaml` и [`ERA-Product-Line.md`](../distributor/ERA-Product-Line.md).

---

## Последствия

- Blueprint: акцент «кросс-доменная корреляция» получает явные модули в roadmap.
- Пресейл: паритет с Vision One по **workflow**, не по облачным SKU.
- Риск: Virtual Patching требует driver/signing gate — не обещать до прохождения ADR-0012.

## Связано

- [`ADR-0023`](0023-ai-investigation-explainability.md) — AI evidence chain поверх Workbench timeline (Post-GA)
- [`ADR-0022`](0022-detection-content-governance.md) — детект-контент для Exposure/корреляции
