# ERA Communications — доноры (patterns only)

**Версия:** 1.2  
**Дата:** 29 июля 2026 г.  
**Статус:** Accepted  
**Источник (архив):** инвест-тизер `ERA Communications.docx` — **маркетинг**; техническая стратегия — этот документ.  
**Родительское правило:** [`ADR-0003`](../adr/0003-repository-structure-and-donor-strategy.md) · [`.cursor/rules/donor-strategy.mdc`](../../.cursor/rules/donor-strategy.mdc)  
**Архитектура:** [`ADR-0027`](../adr/0027-era-communications-architecture.md)

---

## 1. Инвариант

> **Никаких форков.** Весь код Comms — свой (Rust + Go).  
> Из open source берём **паттерны, модели данных, протоколы и декларативные артефакты** — не строки чужих репозиториев.

Согласовано с ADR-0003 (AI-Driven Reverse Engineering). Golden-тесты — на **поведение**, не на diff с донором.

**Office / co-editing** — не Comms: [`ADR-0026`](../adr/0026-sovereign-office-engine.md), deep link из Comms.

---

## 2. Режимы (только два)

| Режим | Что | Пример |
|-------|-----|--------|
| **Pattern** | Алгоритм, архитектура, UX-flow, ops-модель | Stalwart: async mail pipeline; LiveKit: SFU topology |
| **Spec / data** | RFC, OASIS, Matrix spec; импорт без кода | JMAP, CalDAV, OAuth2, Autodiscover XML |

**Deploy upstream binary** (без изменения исходников) — допустимо для зрелых Apache/MIT компонентов **как отдельный процесс** с ERA-adapter (напр. LiveKit server). Это **не fork** и **не** перенос кода в монорепо.

---

## 3. Матрица OSS-доноров (patterns)

### 3.1. Mail — ERA Mail Server

| Reference | Стек | Что извлекаем (patterns) | ERA модуль |
|-----------|------|--------------------------|------------|
| **Stalwart** | Rust, AGPL | Async SMTP/IMAP/JMAP; parser layout; queue/shard **идеи**; CalDAV/CardDAV scope; audit checklist | `services/comms/mail/core` |
| **mox** | Go, MIT | DKIM/ACME/admin API **flows**; low-maintenance ops | `mail` Go API, policy |
| **JMAP / RFC 8620+** | Spec | Session, Mailbox, Email objects | Mail + Client Phase 2 |
| **CalDAV / CardDAV** | Spec | Calendar/contacts sync | `comms/calendar` |
| **EWS / Autodiscover** | MS protocols | Outlook desktop/mobile parity (subset MVP) | `mail/internal/autodiscover`, EWS adapter |

### 3.2. Chat — ERA Chat

| Reference | Что извлекаем | ERA модуль |
|-----------|---------------|------------|
| **Matrix specification** | Rooms, federation, event graph | `services/comms/chat` |
| **Dendrite / Synapse** (любая лицензия) | Microservice homeserver **layout**, federation ops — **идеи**, не код | Chat Phase 2 |

### 3.3. VCS — ERA Conference

| Reference | Что извлекаем | ERA модуль |
|-----------|---------------|------------|
| **LiveKit** (Apache-2.0) | SFU, simulcast, room lifecycle, egress **patterns**; on-prem deploy via adapter | `services/comms/vcs` |
| **WebRTC** | ICE/TURN deployment patterns | deploy profile |

### 3.4. Storage, audit, AI

| Reference | Роль |
|-----------|------|
| **ClickHouse** | Mail/chat audit (`era_comms.*`), SIEM export — **схема и query patterns** Matano-class |
| **PostgreSQL** | OLTP — стандартный deploy |
| **MinIO** | Shared platform / Drive |
| **ai-core layout** | Comms AI: on-prem LLM, **не** cloud API |

### 3.5. Web UI

| Reference | Patterns |
|-----------|----------|
| **SnappyMail / Roundcube** | IMAP webmail UX flows |
| **Next.js + Tailwind** | Stack workspace shell (`platform/workspace`) — не OSS-donor |

### 3.6. Не Comms (Office / platform)

| Reference | Куда |
|-----------|------|
| OnlyOffice, Collabora, ProseMirror, Yjs, ironcalc | ADR-0026 — ERA Office |
| oCIS | ADR-0025 — ERA Drive |
| Zitadel | ADR-0025 — `platform/identity` |

### 3.7. Mail Moderation — ERA Mail Moderation

**PRD:** [`PRD-Mail-Moderation.md`](PRD-Mail-Moderation.md) · edition `era-mail-moderation`.

| Reference | Стек / тип | Что извлекаем (patterns) | ERA модуль |
|-----------|------------|--------------------------|------------|
| **MS Exchange Message Moderation** + Transport Rules | Behavior / docs | Hold → notify → Approve/Reject (+ comment); sender's manager; bypass; priority + stop-processing | `mail-moderation/policy`, notify UX |
| **Arbitration mailbox** (Exchange) | Behavior | Hold-store до решения; TTL expiry; silent approve / notify on reject | `mail-moderation/hold` |
| **AD/Entra `manager` + custom attributes** | Directory | Резолв куратора; override линейного менеджера | `mail-moderation/resolve` |
| **Postfix Milter** (`MILTER_README`) | Protocol / ops | Before-queue inspect; quarantine / reject / accept | `mail-moderation/milter` adapter |
| **Stalwart MTA Hooks + Milters** | Pattern (уже §3.1) | External HTTP/JSON policy на SMTP stages; quarantine action | Native path на ERA Mail Server |
| **Haraka `queue/quarantine` + outbound reinject** | Pattern | Hold → отдельный процесс → reinject после approve | Hold + release pipeline |
| **milter-manager** | Pattern | Цепочка фильтров (spam → policy → moderation) | Policy pipeline orchestration |
| **pyquarantine-milter** (идеи) | Pattern | Quarantine + notify + conditional rules | Rule engine shape |
| **Sieve (RFC 5228)** | Spec | Post-release filing (опционально, не замена hold) | Secondary only |
| **Mailman / Sympa** (list moderation) | UX pattern | Approve/Reject + reject reason для модератора | Moderator notification UX |
| **Rspamd quarantine** | Adjacent | Hold по score — **не** manager-approve; граница с spam/DLP | Не смешивать с Moderation MVP |

**Анти-паттерн:** копировать GPL/AGPL milter-код или форкать Haraka/Stalwart; путать DLP-детекторы PII с moderation (см. Perimeter).

---

## 4. Feature parity (не доноры)

Список **продуктовых фишек**, которые заказчики ожидают от класса «корпоративная почта + UC».  
Источник — рынок и RFQ, **не** reverse-engineering проприетарного кода.

| Фича | MVP | Phase 2+ | Примечание |
|------|-----|----------|------------|
| Autodiscover / Zero-Touch Outlook | ✅ | | AC-C3 golden |
| SMTP / IMAP | ✅ | | |
| CalDAV full | ✅ | | |
| EWS subset (Outlook) | ✅ | | |
| ActiveSync (mobile) | | ✅ | отдельный PRD |
| JMAP (modern clients) | | ✅ | |
| Webinar / 1000+ VCS | | ✅ | LiveKit deploy |
| Federation (chat/mail) | | ✅ | Matrix patterns |
| Air-Gap AI (саммари, фишинг) | | ✅ | Comms AI |
| Co-editing вложений | | — | **ERA Office** license |
| Антисpam без cloud Cyren | ✅ | | on-prem rules + AI |
| Outbound message moderation (Approve/Reject) | | ✅ | **ERA Mail Moderation** upsell — [`PRD-Mail-Moderation.md`](PRD-Mail-Moderation.md) |

---

## 5. Анти-паттерны

- Fork/vendoring чужого движка в `services/comms/`
- Cloud LLM / phone-home для Comms AI
- OnlyOffice/Collabora **внутри** Comms bundle
- Копирование GPL/AGPL исходников (Stalwart, Element Dendrite)

---

## 6. Доказуемость

- Каждый перенесённый pattern → golden-тест или integration test на **своём** коде
- CI: grep gate — запрет `vendor/` / copied OSS trees в `services/comms/` без ADR-исключения (исключений нет)

---

## 7. Связанные документы

- [`PRD-Comms-MVP.md`](PRD-Comms-MVP.md)
- [`MVP-Comms-Mail-Sprint-1-Spec.md`](../MVP-Comms-Mail-Sprint-1-Spec.md)
- [`ERA-Communications-Vision.md`](ERA-Communications-Vision.md)
