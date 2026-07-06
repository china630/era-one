# ERA Office — шаблон RFQ / спецификации

**Версия:** 1.0  
**Дата:** 5 июля 2026 г.  
**Pricing:** [`pricing-office-data.yaml`](./pricing-office-data.yaml)

---

## 1. Преамбула (RU)

```
Поставщик: ERA One (www.era-one.solutions)
Продуктовое семейство: ERA Office
Описание: суверенные офисные приложения и co-editing в контуре.
         Свой office engine (без OnlyOffice). Standalone — не требует ERA Control.

Модель: подписка per-user, срок ___ месяцев.
Workspace: https://app.customer.local
```

---

## 2. Спецификация — MVP

| № | Издание | Описание | Ед. | Кол-во | Срок |
|---|---------|----------|-----|--------|------|
| 1 | **ERA Drive** | Файлы, sync, ACL (включён в Suite) | user | | 12 мес |
| 2 | **ERA Documents** | Текст, co-editing, docx I/O | user | | 12 мес |

---

## 3. Bundle — ERA Office Suite

**Состав:** ERA Drive + ERA Documents + ERA Tables + ERA Presentations  
(Drive **всегда включён**)

| Пакет | EU list / user / год | СНГ / user / год |
|-------|----------------------|------------------|
| ERA Office MVP (Drive + Documents) | €10.8 | ~€5.4 |
| **ERA Office Suite** | **€17.25** | **~€8.6** |
| Suite + Office AI | ~€20.9 | ~€10.5 |

Цены — [`ERA-Pricing-Office-Client.md`](./ERA-Pricing-Office-Client.md).

---

## 4. Roadmap издания (отдельные строки)

| Издание | Фаза |
|---------|------|
| ERA Tables | P2 |
| ERA Presentations | P3 |
| ERA Projects | post-MVP |
| ERA Sign | post-MVP |
| ERA Office AI | post-MVP |

---

## 5. Интеграция с Communications

При наличии **ERA Mail Server** — вложения через ERA Drive; co-editing по лицензии Office.

---

*Цены — pricing-office-data.yaml*
