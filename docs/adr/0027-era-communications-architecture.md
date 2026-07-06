# ADR-0027: ERA Communications — архитектура и границы

**Статус:** Accepted  
**Дата:** 5 июля 2026 г.  
**Контекст:** Vision из инвест-тизера требует ADR: стек, протоколы, граница с ERA Office,
standalone-модель, migration tier.

**Связано:** [`ADR-0025`](0025-era-one-shared-platform.md) · [`ADR-0026`](0026-sovereign-office-engine.md) ·
[`editions-comms.yaml`](../../editions-comms.yaml) · [`docs/products/ERA-Communications-Vision.md`](../products/ERA-Communications-Vision.md)

---

## 1. Решение (одной фразой)

**ERA Communications** — standalone продукт (per-user, **без ERA Core**): native mail/chat/VCS
на **Rust + Go**, с опцией **ERA Mail Connect** (BFF к внешнему IMAP/JMAP) для миграции.
Co-editing **не входит** в Comms; интеграция с ERA Office по лицензии.

---

## 2. Стек

| Слой | Технология |
|------|------------|
| Mail core | **Rust** (performance, memory safety) |
| API / calendar / adapters | **Go** |
| VCS | **LiveKit** on-prem (паттерн; adapter в `comms/vcs`) |
| Audit | ClickHouse (**обязателен** в MVP, PRD AC-C7) |
| Shared | `platform/identity`, `platform/tenant`, `platform/drive` (API) |

---

## 3. Издания

| Издание | Описание |
|---------|----------|
| **ERA Mail Server** | SMTP/IMAP, CalDAV, EWS subset, Autodiscover; ActiveSync — Phase 2 |
| **ERA Mail Client** | Webmail + desktop/mobile |
| **ERA Mail Connect** | **Migration tier:** Client + BFF → внешний IMAP/JMAP/EWS; не Full Suite GA |
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

## 5. ERA Mail Connect

- Отдельное издание; **не** заменяет Mail Server в Full Suite narrative.
- Scope: IMAP/JMAP (+ EWS read), webmail, autodiscover к **существующему** серверу.
- Ограничения: нет native calendar/ActiveSync на уровне Server; Comms AI ограничен.
- Cross-sell: Connect → Mail Server.

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
├── mail-connect/   # BFF adapter (migration)
├── calendar/
├── chat/
├── vcs/            # LiveKit adapter
└── ai/             # Comms AI
```

---

## 8. Последствия

**Плюсы:** чёткий MVP (mail first); migration tier; Rust narrative для тизера.

**Обязательства:** ADR-0025 Drive API для вложений; убрать co-editing из Comms killer features;
PRD Comms MVP.

---

## 9. Артеfactы

- [`docs/products/PRD-Comms-MVP.md`](../products/PRD-Comms-MVP.md)
- [`deploy/profiles/comms.yaml`](../../deploy/profiles/comms.yaml)
