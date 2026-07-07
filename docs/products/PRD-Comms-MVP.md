# PRD: ERA Communications — MVP (v1.1)

**Статус:** Accepted scope  
**Дата:** 5 июля 2026 г.  
**Продукт:** ERA Communications (ERA One)  
**ADR:** [`0025`](../adr/0025-era-one-shared-platform.md) · [`0027`](../adr/0027-era-communications-architecture.md)  
**Pricing:** [`pricing-comms-data.yaml`](../distributor/pricing-comms-data.yaml)

---

## 1. Цель MVP

On-prem **почта + клиент + autodiscover + календарь**; **ClickHouse audit обязателен**.
Phase 1.1: **ERA Mail Connect** (hybrid tier).
Phase 1.2 upsell: **ERA Comms Migration** (см. [`PRD-Comms-Migration.md`](PRD-Comms-Migration.md)).
Без chat/VCS/Comms AI в первом релизе.

---

## 2. Глоссарий протоколов

| Протокол | Простыми словами | MVP |
|----------|------------------|-----|
| **SMTP** | Отправка почты между серверами | ✅ |
| **IMAP** | Чтение почты клиентом (Thunderbird, мобильные) | ✅ |
| **Autodiscover** | Автонастройка Outlook/мобильных (XML/SRV) | ✅ |
| **CalDAV** | Открытый стандарт **календаря** (события, приглашения) — Apple Calendar, Thunderbird | ✅ **full** read/write |
| **EWS** (Exchange Web Services) | API Microsoft **Exchange** — Outlook для Windows часто требует EWS для «полного» UX | ✅ **subset** (mail + calendar sync для Outlook) |
| **ActiveSync** | Протокол Microsoft для **синхронизации с телефонами** (почта/календарь/контакты) как у Exchange | ✅ **gov pilot** (subset) |

**Рекомендация MVP:** IMAP + CalDAV full + EWS subset для Outlook desktop; **ActiveSync subset — обязателен для gov pilot** ([`PRD-Comms-Gov-Protocols.md`](PRD-Comms-Gov-Protocols.md)).

---

## 3. Inline attachments (без ERA Drive)

Все лимиты — **tenant policy** в control-plane / comms config (`services/comms/mail/policy`).
**Никакого хардкода** в коде; только defaults при первом создании tenant.

### Recommended defaults (override в admin UI)

| Параметр | Default | Описание |
|----------|---------|----------|
| `inline.max_attachment_size_mb` | **25** | Макс. размер одного вложения |
| `inline.quota_mb_per_user` | **512** | Квота inline-хранилища на ящик |
| `inline.retention_days` | **365** | Retention вложений (0 = без авто-удаления) |
| `inline.max_attachments_per_message` | **50** | Защита от abuse |

При лицензии **ERA Drive** — вложения могут сохраняться в Drive (quota Drive отдельно).

---

## 4. Scope

### In scope (MVP)

| # | Capability |
|---|------------|
| 1 | ERA Mail Server — SMTP/IMAP, Autodiscover |
| 2 | ERA Mail Client — webmail (`app.customer.local/mail`) |
| 3 | CalDAV — **full** read/write календарь |
| 4 | EWS — **subset** для Outlook (send/receive mail, calendar) |
| 5 | Inline attachments (policy выше) |
| 6 | **ClickHouse audit** — все mail events (обязательно) |
| 7 | Drive / Office — integration hooks по лицензии |

### Phase 1.1

| # | Capability |
|---|------------|
| 8 | **ERA Mail Connect** — hybrid BFF → external IMAP/JMAP |

### Phase 2 (отдельный PRD — от вас не требуется сейчас)

- ActiveSync, ERA Chat, ERA Conference, ERA Comms AI, 60k scale proof

---

## 5. ClickHouse audit (обязательно)

- Каждое значимое событие: login, send, receive, delete, admin action → `Envelope` / mail audit schema → ClickHouse.
- Deploy profile comms: ClickHouse **required** (не optional).
- AC-C7: audit row count > 0 в integration test после send/receive.

---

## 6. Лицензирование и pricing

| Издание | MVP | EU list / user / year |
|---------|-----|------------------------|
| ERA Mail Server (+ Client) | ✅ | €10 |
| **ERA Mail Connect** | Phase 1.1 | **€4** (отдельно) |
| ERA Drive / Office | optional | см. pricing-office-data.yaml |

Отдельный upsell, не входящий в MVP AC-C1…AC-C9:
- **ERA Comms Migration** — €1/mailbox one-time (см. [`pricing-comms-data.yaml`](../distributor/pricing-comms-data.yaml)).

**Не требует ERA Core.** Pricing: [`pricing-comms-data.yaml`](../distributor/pricing-comms-data.yaml).

---

## 7. Критерии приёмки

| ID | Критерий |
|----|----------|
| AC-C1 | Send/receive IMAP+SMTP air-gap |
| AC-C2 | Webmail + platform identity |
| AC-C3 | Autodiscover golden XML |
| AC-C4 | Inline quota enforced via policy API |
| AC-C5 | Drive attach (if licensed) |
| AC-C6 | Mail Connect → external IMAP (Phase 1.1) |
| AC-C7 | ClickHouse audit после send/receive |
| AC-C8 | CalDAV create/edit event |
| AC-C9 | EWS Outlook send/receive (subset) |

---

## 8. Phase 2+ — что потребуется от product owner

**Gov pilot (обязательно):** [`PRD-Comms-Gov-Protocols.md`](PRD-Comms-Gov-Protocols.md) — EWS façade, CardDAV, ActiveSync subset. См. [`Comms-Pilot-Gap-List.md`](../Comms-Pilot-Gap-List.md) P0-GOV.

При старте Phase 2 (chat/VCS) — решить:

1. ActiveSync — **must-have для gov mobile** (subset, R-GOV-5); IMAP-only app — fallback only
2. Comms AI — scope (только mail или + chat transcripts)?
3. LiveKit — self-hosted sizing / HA
4. PowerPC — подтверждение спроса

---

## 9. Связанные документы

- [`Comms-Acceptance-System.md`](Comms-Acceptance-System.md) — контроль и приёмка
- [`Comms-MVP-Spec.md`](../Comms-MVP-Spec.md)
- [`Comms-Implementation-Matrix.md`](../Comms-Implementation-Matrix.md)
- [`ERA-RFQ-Comms-Template.md`](../distributor/ERA-RFQ-Comms-Template.md)
- [`editions-comms.yaml`](../../editions-comms.yaml)
