# ERA Office — прайс-лист (регион СНГ)

> **ONE WORKSPACE. ONE PLATFORM. ONE TEAM.**

**Редакция:** 2026-07 · регион **СНГ** · валюта — EUR (без НДС)  
**Модель:** годовая подписка **per-user** · standalone (не требует ERA Control)

SSOT: [`pricing-office-data.yaml`](./pricing-office-data.yaml)

> **Для СНГ — региональная цена −50% от EU list.** В таблицах: EU list и цена СНГ.

Суверенные офисные приложения и co-editing **в вашем контуре**. Свой движок без OnlyOffice и облачных API.

---

## Как формируется цена

- **Единица:** пользователь (user) / год.
- **ERA Drive** — отдельная строка; нужен для Documents/Tables; **всегда включён** в bundle **ERA Office Suite**.
- **Identity** (SSO) — включена, отдельно не продаётся.
- Скидки за **объём users** и **срок** — те же правила, что у ERA Control ([`ERA-Pricing-Client.md`](./ERA-Pricing-Client.md) §2–3).

---

## 1. Прайс по изданиям

| Издание | EU list, €/user/год | СНГ (−50%) | Фаза |
|---------|---------------------|------------|------|
| **ERA Drive** — файлы, sync, ACL, API для Mail/Office | 4 | **2** | P0 |
| **ERA Documents** — текст, co-editing, docx | 8 | **4** | P1 MVP |
| **ERA Tables** — таблицы, формулы, xlsx | 6 | **3** | P2 |
| **ERA Presentations** — презентации | 5 | **2.5** | P3 |
| **ERA Projects** — проекты и задачи | 4 | **2** | post-MVP |
| **ERA Office AI** — LLM в контуре для документов | 6 | **3** | post-MVP |

> **ERA Documents** требует **ERA Drive** (или bundle, где Drive включён).

---

## 2. Готовые пакеты (bundle)

| Пакет | Состав | EU / user / год | СНГ / user / год |
|-------|--------|-----------------|------------------|
| **ERA Office MVP** | Drive + Documents | **10.8** | **~5.4** |
| **ERA Office Suite** | Drive + Documents + Tables + Presentations | **17.25** | **~8.6** |
| **ERA Office Suite + AI** | Suite + Office AI | **~20.9** | **~10.5** |

Скидка bundle применяется к сумме à la carte; затем — volume и term.

---

## 3. Сравнение (ориентир)

| Решение | Модель | Порядок цены / user / год |
|---------|--------|---------------------------|
| Microsoft 365 E3 | Cloud | ~€340–420 |
| Google Workspace | Cloud | ~€140–180 |
| OnlyOffice/Collabora enterprise | On-prem + лицензия вендору | ~€30–90 |
| **ERA Office Suite** | On-prem, свой engine | **~€8.6 (СНГ)** |

---

## 4. Связка с ERA Communications

| Сценарий | Лицензии |
|----------|----------|
| Только почта | ERA Mail Server ([`ERA-Pricing-Comms-Client.md`](./ERA-Pricing-Comms-Client.md)) |
| Почта + файлы | + **ERA Drive** |
| Co-editing вложений | + **ERA Office** (Documents или Suite) |

---

## 5. Perpetual (бессрочная лицензия)

Те же правила, что у ERA Control ([`ERA-Pricing-Client.md`](./ERA-Pricing-Client.md) §5):

- **Perpetual** = 3× годовой подписки (разово, после bundle-скидки и регионального множителя).
- **Maintenance** = 20%/год — обновления и поддержка; без него продукт **продолжает работать**.

В калькуляторе Office — переключатель **Subscription / Perpetual**.

---

## 6. Коммерческое предложение

**Калькулятор:** [www.era-one.solutions/office](https://www.era-one.solutions/office)  
**Email:** [sales@era-one.solutions](mailto:sales@era-one.solutions)

*Roadmap-продукт; пилот — по отдельному плану внедрения.*
