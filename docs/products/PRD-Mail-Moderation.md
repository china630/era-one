# PRD: ERA Mail Moderation (v1.0)

**Статус:** Accepted scope (product decisions locked)  
**Дата:** 29 июля 2026 г.  
**Продукт:** ERA Communications (отдельное издание, upsell)  
**Edition id:** `era-mail-moderation`  
**License module:** `comms-mail-moderation`  
**Pricing:** [`pricing-comms-data.yaml`](../distributor/pricing-comms-data.yaml) · `comms-mail-moderation`  
**Доноры:** [`ERA-Communications-Donors.md`](ERA-Communications-Donors.md) §3.7  
**Связано:** [`PRD-Comms-MVP.md`](PRD-Comms-MVP.md) · [`PRD-Outlook-Bridge.md`](PRD-Outlook-Bridge.md) · [`0027`](../adr/0027-era-communications-architecture.md)

---

## 1. Проблема

Корпорации (в т.ч. на **IceWarp** и смешанном ландшафте) требуют **визирования исходящих писем**
до доставки — типично для новичков, внешних адресатов, VIP-клиентов и ключевых тем
(«Договор», «Смета»). В Exchange / M365 это **Message Moderation** + transport rules.
У многих on-prem MTA (IceWarp и др.) полноценного аналога нет: либо нет продукта, либо
нужен клиентский плагин / ручные костыли.

Нужен **серверный** продукт без установки на ПК, работающий **перед любой SMTP-почтой**.

---

## 2. Решение

**ERA Mail Moderation** — SMTP edge (proxy и/или milter) + policy engine + hold store +
уведомления модератору (Approve / Reject + comment).

```
Sender MUA → SMTP submission
        → ERA Mail Moderation
              ├─ Policy match (group / external / keywords / recipient / …)
              ├─ Resolve moderator (LDAP manager | attr | static | PG map)
              ├─ Hold store (arbitration-like)
              ├─ Notify moderator (mail + signed HTTPS action links)
              └─ Release → upstream SMTP  |  Reject → notify sender
        → IceWarp / ERA Mail Server / any SMTP
        → Audit → ClickHouse
```

**Инвариант:** zero desktop plugin (как Outlook Bridge). Approve/Reject — письмо + web action links.

---

## 3. Границы продукта

| Тема | Решение (locked) |
|------|------------------|
| SKU | Отдельное upsell-издание; не «фича только ERA Mail Server» |
| IceWarp / чужой MTA | **Первый target** — standalone перед их SMTP |
| ERA Mail Server | Native path (MTA hook / встроенный pipeline) — тот же engine |
| DLP (паспорт, карты, NLP) | **Не ядро.** P2 или handoff в ERA Perimeter DLP; moderation может *вызвать* внешний trigger позже |
| Outlook native buttons | **Out of MVP** (нет своего Outlook UX); action links + optional simple web UI |
| Календарь / EWS path | **Out of MVP** — только SMTP mail |
| Desktop plugin | Запрещён |

---

## 4. Scope по фазам

### 4.1. MVP (in scope)

**Условия срабатывания (when):**

| ID | Тип условия | Примечание |
|----|-------------|------------|
| C-01 | Группа отправителей | LDAP/AD group + локальные группы в PG |
| C-02 | External-only | Default-шаблон для «новичков» |
| C-03 | Internal тоже | Опция на правиле, не default |
| C-04 | Keywords тема/тело | Список keyword / простой regex |
| C-05 | Получатель / VIP-домен | address или domain match |
| C-06 | Вложения (light) | has_attachment, size, extension allow/deny |
| C-07 | Исключения типов | Deny-list: NDR, системная почта, calendar noise |
| C-08 | Только authenticated submission | Outbound relay/submission; не «весь MX internet inbound» |

**Резолв модератора (who):**

| ID | Режим | Примечание |
|----|-------|------------|
| M-01 | LDAP/AD `manager` | Динамический «sender's manager» |
| M-02 | Static mapping | sender/group → moderator(s) |
| M-03 | Custom LDAP attribute | Настраиваемое имя attr (куратор ≠ линейный) |
| M-04 | Per-sender override | Высший priority |
| M-05 | PG / CSV map | Каталог без нормального AD (IceWarp reality) |
| M-06 | Несколько модераторов | **any-of** (первый Approve/Reject побеждает) |
| M-07 | Fallback | Если manager пуст → fallback group/admin + alert |

**Правила / hold / UX:**

| ID | Тема | Решение |
|----|------|---------|
| R-01 | Приоритет | Явный `priority`; первое match + stop-processing |
| R-02 | Bypass | Bypass groups/senders на правиле |
| R-03 | Admin force-release | API/UI |
| R-04 | Hold store | PG metadata + blob (MinIO/disk) |
| R-05 | TTL pending | Default **48–72h** → **auto-reject** + notify sender (настраиваемо) |
| R-06 | Auto-approve по TTL | Опция правила, **не default** |
| R-07 | Письмо модератору | Оригинал + signed HTTPS Approve/Reject links |
| R-08 | Reject + comment | Обязательно |
| R-09 | Notify sender «на согласовании» | Per-rule; default **ON** для шаблона новичков |
| R-10 | Notify на Approve | Default **OFF** (письмо просто уходит) |

**Authoring:**

| ID | Тема | Решение |
|----|------|---------|
| A-01 | MVP authoring | API + YAML/JSON declarative + простой Admin UI (CRUD) |
| A-02 | GitOps | Export/import YAML (air-gap) |
| A-03 | Шаблоны | «New hires external», «VIP domain», «Keyword legal» |

**Интеграция / security:**

| ID | Тема | Решение |
|----|------|---------|
| I-01 | Врезка | SMTP proxy (primary для «любой почты») + milter adapter |
| I-02 | IceWarp | Submission/MX path → Moderation → IceWarp |
| I-03 | Audit | hold / approve / reject / expire → ClickHouse |
| I-04 | Action links | Short-lived signed token, one-time, bind message+moderator |
| I-05 | PII в audit | Метаданные + hash; тело — retention/TTL policy |

### 4.2. P1

| ID | Тема |
|----|------|
| P1-01 | Moderated recipient / DL («всё на finance@» → approve) — inbound-to-address mode |
| P1-02 | HR/onboarding API: add to Novices + set curator |
| P1-03 | Richer Admin UI (wizard, rule test/simulate) |

### 4.3. P2 / out of MVP

| ID | Тема |
|----|------|
| P2-01 | DLP sensitive-info triggers (handoff Perimeter) |
| P2-02 | All-of / кворум модераторов |
| P2-03 | Multi-level approval (L1 → L2) |
| P2-04 | Outlook native Approve/Reject buttons |
| P2-05 | Full content NLP / smart classification |

---

## 5. Decision log (принятый тренд vs рынок)

Краткая фиксация: **Decision = ERA (A)**. Колонка «Рынок» — типичная практика, не обязательство копировать.

| Тема | ERA (принято) | Как у большинства | Зачем у большинства иначе |
|------|---------------|-------------------|---------------------------|
| Форма продукта | Отдельный SKU, SMTP edge | Фича внутри Exchange/M365 | MS контролирует весь стек |
| DLP vs moderation | Разделить | Часто один «mail security» bundle | Один бюджет ИБ |
| Approve UX | Action links + mail | Native Outlook buttons | Свой клиент у MS |
| Несколько модераторов | any-of | Обычно any-of | All-of тормозит бизнес |
| Multi-level | P2 | Есть в M365, реже on-prem | Сложность SLA |
| TTL | Auto-reject 48–72h | ~2 days / vary | Не копить quarantine |
| Notify on hold | Default ON (новички) | Часто тихо | Меньше шума; у ERA важнее онбординг |
| Каталог без AD | PG/CSV MVP | Ждут AD | Enterprise AD-centric |
| Врезка | SMTP proxy (+ milter) | Внутри своего transport | Чужой MTA нельзя патчить |
| Authoring MVP | YAML/API + simple UI | EAC + PowerShell | Cloud = клики; нам нужен air-gap |

Полный чеклист условий/резолверов — §4; открытых продуктовых развилок после этой редакции **нет**.

---

## 6. Критерии приёмки (MVP)

| ID | Критерий | Доказательство |
|----|----------|----------------|
| AC-MM-1 | Правило: группа «Новички» + external → hold + notify manager из LDAP | Integration test + golden policy |
| AC-MM-2 | Approve → письмо доставлено upstream; Reject + comment → sender notified | E2E SMTP fixture |
| AC-MM-3 | Custom attr / static override побеждает `manager` при высшем priority | Unit + golden |
| AC-MM-4 | Keywords + VIP domain conditions | Golden corpus `testdata/` |
| AC-MM-5 | TTL expire → auto-reject + audit row | Integration test |
| AC-MM-6 | Bypass group не попадает в hold | Unit test |
| AC-MM-7 | 0 client installs; action-link approve работает без Outlook plugin | Deployment checklist |
| AC-MM-8 | Audit hold/approve/reject в ClickHouse | CH rows > 0 |
| AC-MM-9 | IceWarp path: SMTP → Moderation → IceWarp deliver | Field smoke / lab |
| AC-MM-10 | YAML import/export правила (air-gap) | Golden round-trip |

---

## 7. Лицензирование и цена

| Параметр | Значение |
|----------|----------|
| Edition | `era-mail-moderation` |
| License claim | `comms-mail-moderation` (`exists: false` до реализации) |
| Модель | per-user/year **или** deal-desk project (как Bridge) |
| EU list (indicative) | **€2 / user / year** (CIS ×0.5) |
| Tier | upsell (partner + greenfield) |
| Не требует | ERA Core |

Partner narrative: рядом с Migration + Bridge — «IceWarp coexistence / hardening» (модерация без смены MTA).

---

## 8. Архитектура (planned)

```
services/comms/mail-moderation/
  ├── smtpproxy/       # primary edge for any-MTA
  ├── milter/          # adapter (Postfix-class upstreams)
  ├── policy/          # conditions, priority, stop-processing
  ├── resolve/         # LDAP manager/attr + PG/CSV map
  ├── hold/            # store + TTL worker
  ├── notify/          # moderator mail + signed action links
  ├── adminapi/        # CRUD + YAML import/export
  └── audit/           # ClickHouse
```

Переиспользование: LDAP patterns из platform/identity; audit pattern из mail/migration; licensegate `comms-mail-moderation`.

---

## 9. Доноры (patterns only)

См. [`ERA-Communications-Donors.md`](ERA-Communications-Donors.md) §3.7.

Ключевые ссылки поведения: Exchange Message Moderation / transport rules / arbitration mailbox
(документация MS — **behavior**, не код). OSS: Postfix Milter quarantine, Stalwart MTA Hooks,
Haraka quarantine+reinject, milter-manager orchestration — **идеи**, не строки.

---

## 10. Связанные документы

- [`editions-comms.yaml`](../../editions-comms.yaml)
- [`ERA-Communications-Vision.md`](ERA-Communications-Vision.md)
- [`Comms-Partner-Edition-Bundle.md`](../Comms-Partner-Edition-Bundle.md)
- [`PRD-Outlook-Bridge.md`](PRD-Outlook-Bridge.md)
- [`PRD-Comms-Migration.md`](PRD-Comms-Migration.md)
- [`PRD-ERA-Perimeter.md`](PRD-ERA-Perimeter.md) — граница DLP (P2 handoff)
