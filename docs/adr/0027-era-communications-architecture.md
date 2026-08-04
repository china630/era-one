# ADR-0027: ERA Communications — архитектура и границы

**Статус:** Implemented  
**Дата:** 7 июля 2026 г. · §5c Mail Moderation **2026-07-29**  
**Контекст:** Vision из инвест-тизера требует ADR: стек, протоколы, граница с ERA Office,
standalone-модель, hybrid/migration tiers.

**Связано:** [`ADR-0025`](0025-era-one-shared-platform.md) · [`ADR-0026`](0026-sovereign-office-engine.md) ·
[`editions-comms.yaml`](../../editions-comms.yaml) · [`docs/products/ERA-Communications-Vision.md`](../products/ERA-Communications-Vision.md)

---

## 1. Решение (одной фразой)

**ERA Communications** — standalone продукт (per-user, **без ERA Core**): native mail/chat/VCS
на **Rust + Go**, с опциями **ERA Mail Connect** (hybrid BFF к внешнему IMAP/JMAP)
и **ERA Comms Migration** (bulk import в Mail Server).
Co-editing **не входит** в Comms; интеграция с ERA Office по лицензии.

---

## 2. Стек

| Слой | Технология |
|------|------------|
| Mail core | **Rust** (performance, memory safety) |
| API / calendar / adapters | **Go** |
| VCS | **LiveKit** on-prem (upstream deploy + ERA adapter; patterns — [`ERA-Communications-Donors.md`](../products/ERA-Communications-Donors.md)) |
| Audit | ClickHouse (**обязателен** в MVP, PRD AC-C7) |
| Shared | `platform/identity`, `platform/tenant`, `platform/drive` (API) |

**Доноры Comms:** patterns only — [`ERA-Communications-Donors.md`](../products/ERA-Communications-Donors.md). Office/co-editing — ADR-0026.

---

## 3. Издания

| Издание | Описание |
|---------|----------|
| **ERA Mail Server** | SMTP/IMAP, CalDAV, EWS subset, Autodiscover; ActiveSync — Phase 2 |
| **ERA Mail Client** | Webmail (свой SPA); desktop/mobile — сторонние клиенты через серверные протоколы ([`0028`](0028-era-mail-client-strategy.md)) |
| **ERA Mail Connect** | **Hybrid tier:** Client + BFF → внешний IMAP/JMAP/EWS; не Full Suite GA |
| **ERA Comms Migration** | **Migration upsell:** bulk import в Mail Server (mail/calendar/archive), one-time |
| **ERA Outlook Bridge** | **Upsell:** server EWS/Autodiscover façade для Outlook без desktop plugin ([`0030`](0030-era-outlook-bridge.md)) |
| **ERA Mail Moderation** | **Upsell:** outbound SMTP moderation (hold → Approve/Reject); standalone перед любым MTA — [`PRD-Mail-Moderation.md`](../products/PRD-Mail-Moderation.md) |
| **ERA Conference** | LiveKit, 1000+ участников |
| **ERA Chat** | Мессенджер; интеграция с Conference и почтой |
| **ERA Comms AI** | Air-Gap LLM: аудит почты, саммари (**≠ ERA Control AI**) |

Документы/таблицы/co-editing — **только ERA Office** (ADR-0026).

---

## 4. Граница с ERA Office

| Сценарий | Comms | Office + Drive |
|----------|-------|----------------|
| Вложение preview / download | ✓ | — |
| Inline storage (MVP, без Drive) | ✓ limited quota | — |
| Attach from Drive / save to Drive | ✓ API | ERA Drive license |
| Co-editing | deep link only | ✓ ERA Documents license |
| Подпись вложения | hook → platform/signing | ERA Sign license |

**Upsell:** клиент с Comms-only покупает ERA Office для co-editing (модель IceWarp/M365 split).

---

## 5. ERA Mail Connect (Hybrid)

- Отдельное издание; **не** заменяет Mail Server в Full Suite narrative.
- Scope: IMAP/JMAP (+ EWS read), webmail, autodiscover к **существующему** серверу.
- Ограничения: нет native calendar/ActiveSync на уровне Server; Comms AI ограничен.
- Cross-sell: Connect → Mail Server.

---

## 5b. ERA Comms Migration

- Отдельное издание/upsell; **не** заменяет Mail Connect hybrid-паттерн.
- Scope: bulk migration в Mail Server (IMAP, EWS calendar subset, archive import).
- Air-gap by design: без внешних migration API/phone-home.
- Cross-sell: Mail Connect (transition) → Mail Server + Migration (cutover).

---

## 5c. ERA Mail Moderation

- Отдельное издание/upsell; SMTP edge перед **любым** MTA (IceWarp first) или native на ERA Mail Server.
- Scope: policy (group/external/keywords/VIP/…) → hold → manager/curator Approve/Reject; zero desktop plugin.
- **Не** DLP-детектор PII (граница — Perimeter / Phase P2).
- PRD и decision log: [`PRD-Mail-Moderation.md`](../products/PRD-Mail-Moderation.md); доноры — [`ERA-Communications-Donors.md`](../products/ERA-Communications-Donors.md) §3.7.

---

## 6. Standalone и hybrid

- **Не требует** ERA Core, endpoint agent, Kafka для базовой работы.
- **Optional:** audit events → ERA Control ingest (SIEM envelope) при наличии Control.
- **Hybrid:** lease/updates через ADR-0018 Portal/Relay — **без** передачи содержимого писем.

---

## 7. Структура сервисов (planned)

```
services/comms/
├── mail/           # Rust core + Go API
├── mail-connect/   # BFF adapter (hybrid)
├── migration/      # bulk migration service (planned)
├── calendar/
├── chat/
├── vcs/            # LiveKit adapter
└── ai/             # Comms AI
```

---

## 8. Последствия

**Плюсы:** чёткий MVP (mail first); hybrid + migration upsell; Rust narrative для тизера.

**Обязательства:** ADR-0025 Drive API для вложений; убрать co-editing из Comms killer features;
PRD Comms MVP.

---

## 9. Артеfactы

- [`docs/products/PRD-Comms-MVP.md`](../products/PRD-Comms-MVP.md)
- [`docs/adr/0028-era-mail-client-strategy.md`](0028-era-mail-client-strategy.md)
- [`docs/products/ERA-Communications-Donors.md`](../products/ERA-Communications-Donors.md)
- [`deploy/profiles/comms.yaml`](../../deploy/profiles/comms.yaml)
