# ERA One — именование в RFQ, RFP и тендерах

**Версия:** 1.0  
**Дата:** 4 июля 2026 г.  
**Аудитория:** продажи, пресейл, дистрибьюторы, юристы  
**Статус:** SSOT — единый источник правил именования для RFQ/RFP, договоров, datasheets и сайта  
**Связано:** [`products.yaml`](../../products.yaml) · [`editions-control.yaml`](../../editions-control.yaml) ·
[`ERA-Product-Line.md`](./ERA-Product-Line.md) · [`ERA-RFQ-Template.md`](./ERA-RFQ-Template.md) ·
ADR-0024

> **Правило:** три уровня имён — **бренд → продуктовое семейство → издание (лицензия)**.
> В строках спецификации всегда **короткое имя издания** (`ERA Manage`), полная цепочка — **один раз в преамбуле**.

---

## 1. Иерархия имён

```
ERA One                    ← бренд / вендор / экосистема
│
├── ERA Control            ← продуктовое семейство (Security + IT-Ops)
│   ├── ERA Core           ← издание (база XDR, фундамент)
│   ├── ERA Manage         ← издание (IT-Ops / UEM)
│   ├── ERA Control AI
│   ├── ERA Response
│   ├── ERA PAM
│   └── …
│
├── ERA Communications     ← отдельное продуктовое семейство
└── ERA Office             ← отдельное продуктовое семейство
```

| Уровень | Пример | Где используется |
|---------|--------|------------------|
| **Бренд** | ERA One | Шапка договора, footer, сайт, «Поставщик» |
| **Продукт** | ERA Control | Описание платформы, bundle, сравнения с конкурентами |
| **Издание** | ERA Manage | Строки RFQ, лицензия, прайс, SKU |

---

## 2. Как писать ERA Manage (и другие издания Control)

### ✅ Рекомендуется

| Контекст | Формулировка |
|----------|--------------|
| Строка RFQ / спецификация | **ERA Manage** |
| Первое упоминание в договоре | **ERA Manage** (издание платформы **ERA Control**, экосистемы **ERA One**) |
| Datasheet — `<title>` | `ERA One — ERA Manage (UEM)` |
| Datasheet — `<h1>` и лицензия | **ERA Manage (UEM)** |
| Прайс-лист | **ERA Manage** |
| Сравнение с Ivanti/ManageEngine | **ERA Manage (UEM)** |

### ❌ Не использовать

| Формулировка | Почему |
|--------------|--------|
| ERA One Manage | Похоже на отдельное продуктовое семейство (как ERA Office) |
| ERA One Control Manage | Три уровня в одном имени; нечитаемо в таблицах |
| ERA Control Manage | Дублирование; Manage уже подразумевает Control |
| Manage / UEM без ERA | Потеря бренда; путаница с ManageEngine |

---

## 3. Правила для всех изданий ERA Control

Единый паттерн для **ERA Core**, **ERA Control AI**, **ERA Manage**, **ERA PAM** и т.д.:

1. **Line item (RFQ):** только имя издания — `ERA <Name>`.
2. **Преамбула:** полная цепочка один раз.
3. **Зависимости:** если издание не standalone — явно указать в примечании (см. §4).
4. **Bundle:** имя пакета на уровне продукта — `ERA Control — пакет IT-Ops`, внутри — перечень изданий.
5. **Footer / copyright:** всегда **ERA One**, не ERA Control.

### Таблица канонических имён (Control)

| Издание | RFQ / лицензия | Datasheet H1 | Категория |
|---------|----------------|--------------|-----------|
| ERA Core | ERA Core | ERA Core (XDR) | Фундамент |
| ERA Control AI | ERA Control AI | ERA Control AI | Апсейл |
| ERA Response | ERA Response | ERA Response (SOAR) | Апсейл |
| ERA Vuln | ERA Vuln | ERA Vuln | Опция |
| ERA Manage | **ERA Manage** | **ERA Manage (UEM)** | IT-Ops |
| ERA Service | ERA Service | ERA Service (ITSM) | IT-Ops |
| ERA Provision | ERA Provision | ERA Provision | IT-Ops |
| ERA PAM | ERA PAM | ERA PAM | PAM |
| ERA Observe | ERA Observe | ERA Observe | Сеть |

---

## 4. Зависимости в спецификации

Большинство изданий Control **не заменяют** ERA Core:

| Ситуация | Примечание в RFQ |
|----------|------------------|
| Только ERA Manage | «Требует **ERA Core** (базовая платформа ERA Control)» |
| Bundle IT-Ops | «**ERA Control — пакет IT-Ops**: ERA Core + ERA Manage + ERA Service + ERA Provision» |
| ERA Control AI / Response | «Требует ERA Core» |
| ERA PAM | «Требует ERA Core; рекомендуется ERA Manage для CMDB» |

---

## 5. Продуктовые семейства Communications / Office / Shared

| Продукт / слой | RFQ | Издания (канон) |
|----------------|-----|-----------------|
| ERA Communications | ERA Communications | ERA Mail Server, ERA Mail Client, **ERA Mail Connect**, ERA Conference, ERA Chat, ERA Comms AI |
| ERA Office | ERA Office | **ERA Drive**, ERA Documents, ERA Tables, ERA Presentations, ERA Projects |
| Shared platform | *(не отдельный продукт)* | Identity — **включена**; **ERA Drive**, **ERA Sign** — отдельные строки |

**Standalone:** Communications и Office **не требуют** ERA Core.

**ERA Office Suite** всегда включает **ERA Drive**.

Co-editing — лицензия ERA Office; из Comms — интеграция (deep link).

Паттерн: в line items — **издание**; в преамбуле — «издание **ERA Mail Server** продукта **ERA Communications**, экосистемы **ERA One**».

ADR: [`0025`](../../adr/0025-era-one-shared-platform.md) · [`0026`](../../adr/0026-sovereign-office-engine.md) · [`0027`](../../adr/0027-era-communications-architecture.md).

---

## 6. Шаблон преамбулы (RU)

> **Поставщик программного обеспечения:** ERA One (www.era-one.solutions)  
> **Продуктовая платформа:** ERA Control — суверенная платформа Security & IT-Ops для изолированного контура  
> **Предмет поставки:** лицензии на программные издания ERA Control, перечисленные в спецификации  
> **Модель лицензирования:** подписка на 12 месяцев; учёт по endpoint (рабочая станция / сервер ×3)

---

## 7. Шаблон строки спецификации

| № | Наименование ПО | Тип лицензии | Ед. изм. | Кол-во | Срок |
|---|-----------------|--------------|----------|--------|------|
| 1 | **ERA Core** — базовая платформа XDR | подписка | endpoint (ws) | ___ | 12 мес. |
| 2 | **ERA Manage** — IT-Ops / UEM | подписка | endpoint (ws) | ___ | 12 мес. |
| 3 | **ERA Manage** — IT-Ops / UEM | подписка | endpoint (server, ×3) | ___ | 12 мес. |

**Примечание к п. 2–3:** ERA Manage требует активной лицензии ERA Core.

Готовый шаблон: [`ERA-RFQ-Template.md`](./ERA-RFQ-Template.md).

---

## 8. Datasheets и сайт (TODO — применить позже)

При обновлении клиентских материалов:

| Элемент | Правило |
|---------|---------|
| HTML `<title>` | `ERA One — ERA Manage (UEM)` |
| H1 | `ERA Manage (UEM)` + подзаголовок |
| Блок «Лицензирование» | **ERA Manage** (без «ERA One Manage») |
| Footer | © ERA One |
| Сайт `/secure` | Продукт: **ERA Control**; карточки модулей: **ERA Core**, **ERA Manage**, … |
| Главная | Бренд **ERA One**; три плитки: Control / Communications / Office |

---

## 9. Связанные документы

- [`ERA-RFQ-Template.md`](./ERA-RFQ-Template.md) — copy-paste блоки для тендеров
- [`ERA-Product-Line.md`](./ERA-Product-Line.md) §7 — краткая выжимка для presales
- [`ERA-Pricing.md`](./ERA-Pricing.md) · [`ERA-Pricing-Client.md`](./ERA-Pricing-Client.md)
- [`products.yaml`](../../products.yaml) · [`editions-control.yaml`](../../editions-control.yaml)

---

*Внутренний SSOT. При расхождении с datasheet — приоритет у этого документа до синхронизации материалов.*
