# ERA One — продуктовая линейка для продаж

> **ONE AGENT. ONE PLATFORM. ONE CONTROL.**

**Версия:** 1.6
**Дата:** 4 июля 2026 г.
**Аудитория:** отдел продаж, пресейл, дистрибьюторы
**Назначение:** внутренний справочник изданий с **честным статусом готовности** (GA / GA-опция / MVP / Roadmap),
сверенным с [`ADR-Implementation-Matrix.md`](../ADR-Implementation-Matrix.md).
Клиентские датащиты (`datasheets/`) — без этих пометок.

> **Главная мысль для клиента:** ERA — это **единая платформа** с **одним лёгким агентом**
> и **набором серверных изданий**, которые включаются по лицензии. Заказчик платит за то,
> что использует, и наращивает функциональность модулями — **без зоопарка из 5–6 агентов
> и отдельных подписок за каждую функцию** (как у ManageEngine/Ivanti). Всё работает
> **в контуре заказчика (on-prem / air-gap)**, данные не уходят наружу.

**PDF (внутренний):** [`ERA-Product-Line.pdf`](./ERA-Product-Line.pdf)

---

## 1. Карта изданий (одним взглядом)

| № | Издание | Статус | Что даёт | Тип |
|---|---|---|---|---|
| 01 | **ERA Core** | **GA** | База XDR: агент-телеметрия, приём, хранилище, детекция (Sigma), активы, кейсы, SOC-портал | Фундамент |
| 02 | **ERA Control AI** | **GA** | ИИ-аналитик: расследование, storyline, verdict, on-prem LLM | Апсейл |
| 03 | **ERA Response** | **GA** | SOAR: playbooks — изоляция хоста, блок IP, тикет в ITSM | Апсейл |
| 04 | **ERA Vuln** | **GA-опция** | Сканер уязвимостей (CVE), расписания, credentialed-скан | Опция |
| 05 | **ERA Federated / National** | **GA-опция** | Обмен IoC (STIX/TAXII) между подразделениями / организациями | Опция гос/холдинг |
| 06 | **ERA Workbench** | **MVP** | Единый timeline расследования (endpoint + identity + network + BYO-EDR) | Усиление Core |
| 07 | **ERA Exposure** | **MVP** | Локальный CREM: risk score актива, топ-10 | Усиление Vuln/Core |
| 08 | **ERA BYO-EDR Hub** | **MVP** | Приём телеметрии сторонних EDR (JSON/CEF) в единое озеро | Миграция / гибрид |
| 09 | **ERA Manage** | **MVP** | IT-Ops: CMDB, финансовый ITAM, deploy, патчи, App/USB Control, BitLocker, Virtual Patching | IT-Ops |
| 10 | **ERA Service** | **MVP** | ITSM-lite: сервис-деск, портал, SLA, CMDB | IT-Ops |
| 11 | **ERA Provision** | **MVP** | PXE/imaging bare-metal, unattended-установка | IT-Ops |
| 12 | **ERA PAM** | **MVP** | Vault, SSH/RDP-прокси, запись привилегированных сессий | PAM |
| 13 | **ERA Observe** | **MVP** | SNMP, NetFlow, discovery (agentless) | Сеть |

**Легенда:** **GA** — в продукте сегодня · **GA-опция** — GA по отдельной лицензии ·
**MVP** — код готов, автотесты/golden в CI (monitor-ready); до GA — стенд/полевые прогоны и внешние гейты (§4) ·
**Roadmap** — в дорожной карте, кода нет.

> **Правило продаж:** MVP ≠ Roadmap (код есть и покрыт тестами), но и ≠ GA. Клиенту MVP-издания
> предлагаются **как пилот с поэтапным rollout** и явными GA-гейтами (§4), не как «готовый продукт».

> **Терминология:** **ERA Core** = база *XDR*. **ERA Manage** = *IT-Ops*. Разные издания.
> Именование в RFQ/RFP — [`ERA-Naming-and-RFQ-Guide.md`](./ERA-Naming-and-RFQ-Guide.md).

---

## 2. Состав и функции каждого издания

### ERA Core — GA
- Агент Win/Linux/macOS; PII на агенте; ingest → Kafka → ClickHouse; Sigma + MITRE; SOC Portal; **ERA Workbench** (timeline UI, Этап 4).

### ERA Control AI — GA
- Storyline, verdict, on-prem LLM (Ollama / vLLM).

### ERA Response — GA
- SOAR: isolate_host, block IP, create_ticket; журнал действий.

### ERA Vuln — GA-опция
- CVE-скан, credentialed-скан, scheduler (`services/vm`).

### ERA Federated / National — GA-опция
- STIX/TAXII, federated DP; license gate off by default.

### ERA Workbench — MVP
- Единый incident timeline: `event-writer` + BFF + `ui/workbench` (ADR-0017 §1 ✅).

### ERA Exposure — MVP
- Per-asset exposure score + топ-10: `detection-engine` `/api/v1/exposure` (ADR-0017 §2 ✅).

### ERA BYO-EDR Hub — MVP
- Generic JSON/CEF → Envelope в `era-collectors` (ADR-0017 §3 ✅).

### ERA Manage — MVP
- ITAM/CMDB + финансовый ITAM (ADR-0011 ✅), deploy/patch, App/USB Control, BitLocker, EPM-lite, Virtual Patching
  (ADR-0012 — monitor-ready 🟡). **GA-гейт:** WHQL-подпись драйвера для боевого enforce/блокировки.

### ERA Service — MVP
- ITSM-lite, портал, SLA (ADR-0016 §4 ✅, server MVP). **GA-гейт:** field rollout.

### ERA Provision — MVP
- PXE/imaging, post-install enroll агента (ADR-0016 §3 ✅, server MVP). **GA-гейт:** field rollout.

### ERA PAM — MVP
- Vault (AES-GCM + Shamir), checkout RBAC/TTL, SSH-прокси с записью сессий (ADR-0013 ✅).
  **GA-гейт:** RDP-прокси (security-review), HSM-аудит.

### ERA Observe — MVP
- SNMP/NetFlow/discovery (Path A+B, ADR-0020 ✅ MVP) или интеграция PRTG/Zabbix. **GA-гейт:** боевой SNMP-poll на стенде.

---

## 3. Готовые bundle для тендера

| Bundle | Состав | Кому |
|---|---|---|
| **Старт SecOps** | ERA Core + ERA Control AI + ERA Response | Greenfield SOC, банк, гос |
| **+ Уязвимости** | + ERA Vuln | Требование VM/CTEM |
| **ERA IT-Ops** | ERA Core + Manage + Service + Provision | Замена legacy-UEM в контуре |
| **ERA Unified / Sovereign Stack** | XDR (Core+AI+Response) + Manage + Service + Observe | Единый локальный вендор IT + Security |
| **+ PAM** | + ERA PAM | Корпоративный сейф паролей админов |

---

## 4. GA-гейты MVP-изданий (что осталось до GA)

Код готов и покрыт тестами в CI (см. [`ADR-Implementation-Matrix.md`](../ADR-Implementation-Matrix.md)).
До статуса GA остаются не-кодовые гейты — стенд/поле и внешние проверки:

| Возможность | Издание | Статус кода | GA-гейт (не-код) |
|---|---|---|---|
| Incident timeline | ERA Workbench | ✅ | — (в составе Core) |
| Exposure score | ERA Exposure | ✅ | — |
| BYO-EDR adapters | era-collectors | ✅ | доп. коннекторы по спросу |
| CMDB + финансовый ITAM | ERA Manage | ✅ | field rollout |
| BitLocker mgmt | ERA Manage | 🟡 monitor | хранение ключей на пилоте |
| Application Control | ERA Manage | 🟡 monitor | **WHQL-подпись драйвера** + security-review |
| Device Control (USB) | ERA Manage | 🟡 monitor | **WHQL-подпись драйвера** + security-review |
| Virtual Patching | ERA Manage | 🟡 monitor | **WHQL-подпись драйвера** + security-review |
| Развёртывание ПО / патчи | ERA Manage | ✅ | пилот rollout |
| ITSM-lite | ERA Service | ✅ | field rollout |
| OS Provisioning | ERA Provision | ✅ | пилот rollout |
| PAM (vault/checkout/SSH) | ERA PAM | ✅ | — |
| PAM (RDP-прокси, HSM) | ERA PAM | 🟡 | RDP security-review, HSM-аудит |
| Network monitoring | ERA Observe | 🟡 MVP | боевой SNMP-poll на стенде (или PRTG-интеграция) |
| Масштаб 10k ev/s, soak 7×24 | ERA Core | 🟡 | прогон на sizing-сервере (см. `Field-Server-Sizing.md`) |

> **Вне продукта (ADR-0016):** MDM/Mobile UEM и VPN/ZTNA — зона интеграции.
> **Легенда:** ✅ реализовано и протестировано · 🟡 monitor/MVP (боевой режим за внешним гейтом).

---

## 5. Как отвечать на запрос «в стиле ManageEngine»

| Позиция запроса | Наш ответ | Статус |
|---|---|---|
| Vulnerability Management | **ERA Vuln** | GA-опция |
| Базовый UEM (CMDB/ITAM/deploy) | **ERA Manage** | MVP |
| Application Control | **ERA Manage** | MVP (monitor; боевой enforce — WHQL-гейт) |
| BitLocker | **ERA Manage** | MVP |
| Доп. SOC-аналитика | **ERA Control AI + Response** | GA |
| Password Manager Pro (PAM) | **ERA PAM** | MVP (SSH — ✅; RDP — гейт) |

---

## 6. Модель развёртывания и ответ на «а где ваш cloud?»

Возражение с рынка: «клиентов вынуждают в Cortex Cloud / Vision One — без облака вас не
возьмут». Наш ответ — **Sovereign Hybrid** (детально: [`ADR-0018`](../adr/0018-hybrid-connected-operating-model.md)).
Мы **не** строим SaaS-клон; мы отделяем **данные** от **эксплуатации**.

| Плоскость | Что | Где |
|---|---|---|
| **Data plane** (суверенный) | сырьё, ClickHouse/MinIO lake, LLM, кейсы, PII, расследования | **всегда в контуре заказчика** |
| **Control plane** (гибридный) | лицензии/lease, обновления (Sigma/CVE/коннекторы/AI), CRL, health, opt-in TI | on-prem; при `connected` — обмен с ERA Cloud Portal |

**Именованные компоненты** (детали — [`ADR-0018 §1.1`](../adr/0018-hybrid-connected-operating-model.md)):

| Компонент | Где | Деплой | Роль |
|---|---|---|---|
| **ERA Cloud Portal** | вендор | ядро-сервис | лицензии/контракты, lease, CRL, health — зонтик control plane |
| **ERA Update Service** | вендор | отдельный сервис | конвейер подписи + доставка контента (Sigma, CVE, коннекторы, AI-паки); работает и в air-gap (носитель) |
| **ERA Hybrid Relay** | контур клиента | модуль `control-plane` | outbound-only, egress allowlist + audit |
| **ERA Managed View** | вендор | модуль Portal (RBAC) | пульт для партнёра/MSSP — **без доступа к сырью и кейсам** |

**Три режима (профили одной платформы):**

| Режим | Связь | Лицензия | Обновления | Кому |
|---|---|---|---|---|
| **Air-gap** *(default)* | нет | offline Ed25519 | носитель | госсектор, КИИ, строгий банк |
| **Sovereign Hybrid** | outbound-only (Relay) | offline + lease | pull с Portal | банк «данные дома, сопровождение у вендора» |
| *Cloud (SaaS)* | — | — | — | вне текущего scope (по спросу) |

**Как продавать (не путать оси):**
- **Sovereign Hybrid** = данные у вас, а обновления/лицензии/сопровождение — «как в cloud».
  Пич: *«полный XDR в вашем ЦОД; облако ERA — только лицензии, обновления защиты и
  сопровождение, без выноса журналов и расследований»*.
- **BYO-EDR Hub** (другая ось, [`ADR-0017`](../adr/0017-vision-one-onprem-patterns.md)) =
  чужой EDR (Cortex/Defender) в наш lake. Пич: *«Cortex у вас на endpoint — ERA единый
  on-prem SOC и путь миграции»*. **Не** «мы заменяем Cortex Cloud облаком ERA».

**Инвариант для клиента:** сырьё, PII и кейсы **не покидают контур ни в одном режиме**.
Наружу — только метаданные, обновления, лицензии и (opt-in) обезличенные индикаторы.

**AZ-нюанс:** для регуляторики есть режим «hybrid minimal» (lease + updates + CRL,
health Minimal, TI off) + DPA + одностраничная схема потоков. Portal — region AZ/EU или
self-hosted для госа.

**Статус:** Sovereign Hybrid — **MVP «Hybrid-0»** (ADR-0018 ✅: relay, lease, Update Service, CRL,
egress-audit; ступени 3–4 и TI-share B/C — Roadmap). Доставка контента по типам — ADR-0018 §3.2.1.
Detection content governance — [`ADR-0022`](../adr/0022-detection-content-governance.md).
Клиенту не обещать как GA; текущий фокус — Sovereign on-prem.

---

## 7. Терминология для RFQ, RFP и тендеров

**SSOT:** [`ERA-Naming-and-RFQ-Guide.md`](./ERA-Naming-and-RFQ-Guide.md) · шаблон: [`ERA-RFQ-Template.md`](./ERA-RFQ-Template.md)

### Иерархия (три уровня)

| Уровень | Пример | В RFQ |
|---------|--------|-------|
| Бренд | **ERA One** | Поставщик, footer, сайт |
| Продукт | **ERA Control** | Описание платформы, bundle |
| Издание | **ERA Manage** | **Строки спецификации и лицензии** |

### ERA Manage — как писать

| ✅ Использовать | ❌ Не использовать |
|----------------|-------------------|
| **ERA Manage** (строка RFQ) | ERA One Manage |
| **ERA Manage (UEM)** (datasheet H1) | ERA One Control Manage |
| Первое упоминание: *ERA Manage (издание ERA Control, экосистема ERA One)* | Manage без ERA |

**Зависимость:** ERA Manage в RFQ **всегда** с примечанием «требует ERA Core».

### Паттерн для всех изданий Control

- Line item → **ERA Core**, **ERA Control AI**, **ERA Manage**, …
- Bundle → **ERA Control — пакет IT-Ops** (состав изданий перечислить)
- Datasheet title → `ERA One — ERA Manage (UEM)`; лицензия → **ERA Manage**

> Datasheets и сайт — синхронизировать по этому гайду (см. §8 Naming Guide).

---

## 8. Связанные документы

- [`ERA-Naming-and-RFQ-Guide.md`](./ERA-Naming-and-RFQ-Guide.md) — именование в RFQ/RFP, тендерах, datasheets, сайте (SSOT)
- [`ERA-RFQ-Template.md`](./ERA-RFQ-Template.md) — copy-paste шаблон спецификации
- [`ERA-Product-Line.pdf`](./ERA-Product-Line.pdf) — этот справочник (PDF, внутренний)
- [`ERA-Pricing.md`](./ERA-Pricing.md) — ценовая политика (внутр.): прайс, скидки, floor, perpetual
- [`ERA-Pricing-Client.md`](./ERA-Pricing-Client.md) — клиентский прайс (регион СНГ, без floor/маржи)
- [`ERA-One-DataSheet.pdf`](./ERA-One-DataSheet.pdf) — общий датащит для клиента
- Датащиты по изданиям: `datasheets/01-ERA-Core.pdf` … `13-ERA-Observe.pdf`
- [`head-to-head/`](./head-to-head/) — сравнения с конкурентами
- [`Demo-For-Partners.md`](./Demo-For-Partners.md) — live-демо
- ADR: [`0011`](../adr/0011-cmdb-itam-data-model.md) · [`0012`](../adr/0012-agent-enforcement-mode.md) · [`0013`](../adr/0013-era-pam-edition.md) · [`0016`](../adr/0016-uem-scope-vs-ivanti.md) · [`0017`](../adr/0017-vision-one-onprem-patterns.md) · [`0018`](../adr/0018-hybrid-connected-operating-model.md) · [`0022`](../adr/0022-detection-content-governance.md) · [`0023`](../adr/0023-ai-investigation-explainability.md)

---

*Внутренний материал. MVP-издания клиенту — только как пилот с GA-гейтами (§4), не как готовый продукт;
Roadmap не выдавать за GA. Для клиента — датащиты без пометок статуса.*
