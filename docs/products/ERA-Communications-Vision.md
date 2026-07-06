# ERA Communications — vision

**Продуктовое семейство:** ERA Communications (под брендом ERA One)  
**Статус:** Roadmap — ADR/PRD приняты  
**ADR:** [`0025`](../adr/0025-era-one-shared-platform.md) · [`0027`](../adr/0027-era-communications-architecture.md)  
**PRD:** [`PRD-Comms-MVP.md`](PRD-Comms-MVP.md)  
**Pricing:** [`pricing-comms-data.yaml`](../distributor/pricing-comms-data.yaml)

---

## Позиционирование

**ERA Communications** — суверенные корпоративные коммуникации on-prem. Замена Exchange / CommuniGate / IceWarp.

**Standalone:** не требует ERA Core.

---

## Killer features

1. **Производительность** — Rust mail core + Go; целевой масштаб 60 000+ users.
2. **Air-Gap AI** — ERA Comms AI (≠ ERA Control AI).
3. **Zero-Touch** — Autodiscover для Outlook/мобильных.
4. **ClickHouse audit** — **обязателен** в MVP.
5. **Интеграция ERA Office** — co-editing через deep link (лицензия Office).
6. **ERA Mail Connect** — migration tier, отдельная цена.

---

## Издания

См. [`editions-comms.yaml`](../../editions-comms.yaml). Pricing: [`pricing-comms-data.yaml`](../distributor/pricing-comms-data.yaml).

| Издание | EU list / user / year |
|---------|----------------------|
| ERA Mail Server (+ Client) | €10 |
| ERA Mail Connect | €4 |
| ERA Conference | €6 |
| ERA Chat | €6 |
| ERA Comms AI | €8 |
| Full Suite bundle | ~€19 CIS equiv. |

---

## RFQ

[`ERA-RFQ-Comms-Template.md`](../distributor/ERA-RFQ-Comms-Template.md)

---

## Дорожная карта

| Фаза | Результат |
|------|-----------|
| MVP | Mail + CalDAV + EWS subset + ClickHouse |
| Phase 1.1 | ERA Mail Connect |
| Phase 2 | ActiveSync, Chat, Conference |
| Phase 3 | Comms AI, scale proof |

**Связано:** [`products.yaml`](../../products.yaml) · [`deploy/profiles/comms.yaml`](../../deploy/profiles/comms.yaml)
