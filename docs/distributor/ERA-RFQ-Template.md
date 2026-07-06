# ERA One — шаблон RFQ / спецификации (тендер)

**Версия:** 1.0  
**Дата:** 4 июля 2026 г.  
**Назначение:** copy-paste блоки для ответа на RFQ/RFP и коммерческих предложений  
**Правила именования:** [`ERA-Naming-and-RFQ-Guide.md`](./ERA-Naming-and-RFQ-Guide.md)

---

## 1. Преамбула (RU)

```
Поставщик: ERA One (www.era-one.solutions)
Продуктовая платформа: ERA Control
Описание: суверенная платформа Security & IT-Ops для изолированного контура
         (on-prem / air-gap). Один endpoint-агент; серверные издания по лицензии.

Предмет поставки: программные лицензии на издания ERA Control, указанные в таблице ниже.
Модель: подписка (subscription), срок ___ месяцев.
Развёртывание: on-premise в инфраструктуре Заказчика.
```

---

## 2. Преамбула (EN)

```
Vendor: ERA One (www.era-one.solutions)
Product platform: ERA Control
Description: sovereign Security & IT-Ops platform for isolated/on-prem deployment.
             Single lightweight endpoint agent; server editions licensed separately.

Scope: software subscription licenses for ERA Control editions listed below.
Term: ___ months.
Deployment: on-premises at Customer site.
```

---

## 3. Спецификация — Security (GA)

| № | Software edition | Description | Unit | Qty | Term |
|---|------------------|-------------|------|-----|------|
| 1 | **ERA Core** | XDR base: agent, ingest, storage, detection (Sigma/MITRE), assets, cases, SOC portal | endpoint (workstation) | | 12 mo |
| 2 | **ERA Core** | XDR base (server license, ×3 multiplier) | endpoint (server) | | 12 mo |
| 3 | **ERA Control AI** | On-prem AI analyst: investigation, storyline, verdict | endpoint (workstation) | | 12 mo |
| 4 | **ERA Response** | SOAR: playbooks, host isolation, IP block, ITSM ticket | endpoint (workstation) | | 12 mo |

**Note:** ERA Control AI and ERA Response require an active ERA Core license.

---

## 4. Спецификация — IT-Ops (MVP / pilot)

| № | Software edition | Description | Unit | Qty | Term |
|---|------------------|-------------|------|-----|------|
| 1 | **ERA Core** | XDR base (required) | endpoint (workstation) | | 12 mo |
| 2 | **ERA Manage** | UEM / IT-Ops: CMDB, ITAM, deploy, patch, App/USB Control, BitLocker | endpoint (workstation) | | 12 mo |
| 3 | **ERA Manage** | UEM / IT-Ops (server license, ×3 multiplier) | endpoint (server) | | 12 mo |
| 4 | **ERA Service** | ITSM-lite: service desk, portal, SLA | endpoint (workstation) | | 12 mo |
| 5 | **ERA Provision** | OS provisioning: PXE/imaging, bare-metal | endpoint (workstation) | | 12 mo |

**Note:** ERA Manage, ERA Service and ERA Provision require ERA Core. Offered as phased pilot per ERA GA gates.

---

## 5. Bundle — ERA Control IT-Ops (для тендера)

**Наименование пакета:** ERA Control — пакет IT-Ops

**Состав:**
- ERA Core
- ERA Manage
- ERA Service
- ERA Provision

**Формулировка для документа (RU):**

> Лицензионный пакет **ERA Control — IT-Ops** включает издания: **ERA Core**, **ERA Manage**, **ERA Service**, **ERA Provision** — единая on-prem платформа управления парком и сервис-деском в контуре Заказчика.

---

## 6. Bundle — ERA Control Start (Security GA)

**Наименование пакета:** ERA Control — пакет Start

**Состав:** ERA Core + ERA Control AI + ERA Response

---

## 7. Строка только ERA Manage (типовой RFQ)

**RU (первая строка с полной цепочкой):**

> **ERA Manage** (издание платформы **ERA Control**, экосистемы **ERA One**) — лицензия IT-Ops/UEM на endpoint

**Далее в таблице — только:** `ERA Manage`

**Зависимость (обязательное примечание):**

> Лицензия ERA Manage не действует без активной лицензии ERA Core.

---

## 8. PAM / Observe / Vuln (кратко)

| Издание | RFQ name | Requires |
|---------|----------|----------|
| ERA Vuln | ERA Vuln | ERA Core |
| ERA PAM | ERA PAM | ERA Core |
| ERA Observe | ERA Observe | ERA Core |

---

## 9. ERA Communications / ERA Office (standalone)

Шаблоны RFQ:
- [`ERA-RFQ-Comms-Template.md`](./ERA-RFQ-Comms-Template.md)
- [`ERA-RFQ-Office-Template.md`](./ERA-RFQ-Office-Template.md)

> **ERA One Full Stack** — Control + Communications + Office в одном контуре.
> Communications и Office **не требуют** ERA Core.

---

## 10. Подпись / реквизиты (placeholder)

```
ERA One
www.era-one.solutions
sales@era-one.solutions

Коммерческое предложение действительно до: ___________
```

---

*Шаблон. Цены — из [`ERA-Pricing-Client.md`](./ERA-Pricing-Client.md). Статусы изданий — из [`ERA-Product-Line.md`](./ERA-Product-Line.md).*
