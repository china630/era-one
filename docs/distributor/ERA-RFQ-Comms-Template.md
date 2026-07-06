# ERA Communications — шаблон RFQ / спецификации

**Версия:** 1.0  
**Дата:** 5 июля 2026 г.  
**Правила именования:** [`ERA-Naming-and-RFQ-Guide.md`](./ERA-Naming-and-RFQ-Guide.md)  
**Pricing:** [`pricing-comms-data.yaml`](./pricing-comms-data.yaml)

---

## 1. Преамбула (RU)

```
Поставщик: ERA One (www.era-one.solutions)
Продуктовое семейство: ERA Communications
Описание: суверенные корпоративные коммуникации (почта, календарь, чат, ВКС)
         в изолированном контуре (on-prem / air-gap). Standalone — не требует ERA Control.

Предмет поставки: лицензии на издания ERA Communications, указанные ниже.
Модель: подписка per-user, срок ___ месяцев.
Развёртывание: on-premise в инфраструктуре Заказчика.
```

---

## 2. Спецификация — Mail MVP

| № | Издание | Описание | Ед. | Кол-во | Срок |
|---|---------|----------|-----|--------|------|
| 1 | **ERA Mail Server** | SMTP/IMAP, CalDAV, EWS (Outlook), Autodiscover, ClickHouse audit | user | | 12 мес |
| 2 | **ERA Mail Client** | Webmail + mobile/desktop (включён с Server) | user | | incl. |

**Примечание:** ERA Mail Client включён в ERA Mail Server. **ERA Core не требуется.**

---

## 3. Migration tier — ERA Mail Connect

| № | Издание | Описание | Ед. | Кол-во | Срок |
|---|---------|----------|-----|--------|------|
| 1 | **ERA Mail Connect** | Webmail + BFF к существующему IMAP/JMAP/EWS | user | | 12 мес |

**Примечание:** не заменяет ERA Mail Server; для поэтапной миграции.

---

## 4. Optional upsell

| Издание | Когда |
|---------|--------|
| **ERA Drive** | Единое файловое хранилище вложений |
| **ERA Office** | Co-editing вложений (deep link) |
| **ERA Conference / Chat / Comms AI** | Phase 2 (roadmap) |

---

## 5. Bundle — Full Suite (roadmap)

**ERA Communications Full Suite:** Mail Server + Conference + Chat + Comms AI  
(см. pricing-comms-data.yaml bundle `comms-full`)

---

*Цены — индикатив из pricing-comms-data.yaml; итог в КП.*
