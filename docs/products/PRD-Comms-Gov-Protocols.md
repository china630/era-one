# PRD: ERA Communications — Gov Protocol Stack (Outlook / Mobile Parity)

**Статус:** Accepted — gov pilot requirement  
**Дата:** 7 июля 2026 г.  
**Продукт:** ERA Mail Server (+ Client)  
**Связано:** [`PRD-Comms-MVP.md`](PRD-Comms-MVP.md) · [`Comms-Pilot-Gap-List.md`](../Comms-Pilot-Gap-List.md) · ADR-0027

---

## 1. Проблема

Госсектор и регулируемый enterprise **не рассматривают** почтовый продукт без:

- **Outlook desktop** — календарь, контакты, заметки/задачи без клиентского коннектора (как IceWarp Outlook Connector);
- **Мобильные** — синхронизация «как Exchange» (ActiveSync subset);
- **Серверная** совместимость — Autodiscover видит Exchange (EWS), не только IMAP.

Решение — **на сервере**, в контуре заказчика, air-gap.

---

## 2. Решение (одной фразой)

**ERA Exchange façade:** integrated mail stack (Rust + Go) с Autodiscover EXCH, EWS subset, CalDAV/CardDAV и ActiveSync — **без** Postfix+Dovecot bundle и **без** проприетарного Outlook-плагина.

**Донор patterns:** Stalwart (integrated protocols), Z-Push (ActiveSync layout) — **свой код**, ADR-0003.

---

## 3. Архитектура (не Postfix + Dovecot)

| Слой | Технология | Роль |
|------|------------|------|
| Mail delivery | `era-mail-core` (Rust) | SMTP/IMAP, integrated store |
| Protocol adapters | `mail-api` (Go) | Autodiscover, EWS, CalDAV, CardDAV, ActiveSync |
| Calendar/contacts | `calendar/` + contacts module | CalDAV / CardDAV |
| Audit | ClickHouse | Обязательный SIEM trail |

**Не используем:** Postfix, Dovecot, Haraka как production bundle.  
**Не используем:** fork Stalwart, полный MAPI/RPC, cloud LLM.

---

## 4. Protocol matrix (gov-ready target)

| Протокол | Назначение | Клиенты | Gov scope | Out of scope |
|----------|------------|---------|-----------|--------------|
| **Autodiscover + EXCH** | Outlook видит Exchange | Outlook | R-GOV-1 | — |
| **EWS subset** | Почта + календарь + контакты + notes/tasks | Outlook desktop | R-GOV-1…4 | Полный Exchange API |
| **CalDAV** | Календарь open standard | Apple, Thunderbird | R-GOV-2 | — |
| **CardDAV** | Контакты | Apple, Thunderbird, EWS path | R-GOV-3 | Полный GAL/ACL Exchange |
| **SMTP/IMAP + TLS** | Стандартная почта | Все | P0 | — |
| **ActiveSync subset** | Mobile mail/cal/contacts | iOS/Android | R-GOV-5 | Все EAS policies |
| **MAPI** | — | — | ❌ | Полный MAPI |

---

## 5. Критерии приёмки (field, после реализации)

| ID | Критерий | Доказательство |
|----|----------|----------------|
| AC-GOV-1 | Outlook autodiscover → EXCH profile | Field log + screenshot |
| AC-GOV-2 | Outlook calendar create/edit/invite | EWS + field smoke |
| AC-GOV-3 | Outlook contacts sync | CardDAV and/or EWS Contacts |
| AC-GOV-4 | Mobile ActiveSync mail+calendar | iOS/Android field smoke |
| AC-GOV-5 | Notes/Tasks subset (if in tender) | EWS field matrix |
| AC-GOV-6 | No outbound internet during test | Air-gap firewall log |

Текущие scaffold gates (C-2) **не закрывают** AC-GOV-* — см. [`Comms-Pilot-Gap-List.md`](../Comms-Pilot-Gap-List.md).

---

## 6. Исполняемые волны

| Wave | Backlog | Фокус |
|------|---------|-------|
| **R-GOV-1** | CM-GOV-1…3 | EWS Exchange façade + Autodiscover TLS |
| **R-GOV-2** | CM-GOV-4…5 | CalDAV production (persistent, invitations) |
| **R-GOV-3** | CM-GOV-6…7 | CardDAV + EWS Contacts |
| **R-GOV-4** | CM-GOV-8 | EWS Notes/Tasks subset |
| **R-GOV-5** | CM-GOV-9…10 | ActiveSync mobile subset |

Предусловие: P0 persistence из [`Comms-Pilot-Gap-List.md`](../Comms-Pilot-Gap-List.md).

---

## 7. Позиционирование vs IceWarp

| | IceWarp | ERA Mail Server |
|---|---------|-----------------|
| Outlook UX | Часто Outlook Connector (клиент) | **Server EWS façade** (без плагина) |
| Стек | Proprietary integrated | **Native Rust+Go integrated** |
| Air-gap | On-prem | On-prem + ClickHouse audit |
| AI | Опционально cloud | Comms AI air-gap (отдельное издание) |

---

## 8. Связано

- [`ERA-Communications-Donors.md`](ERA-Communications-Donors.md) §3.1 Stalwart patterns
- [`Comms-Sprint-Index.md`](../Comms-Sprint-Index.md) — R-GOV waves
- [`ERA-RFQ-Comms-Template.md`](../distributor/ERA-RFQ-Comms-Template.md)
