# ERA Communications — vision

**Продуктовое семейство:** ERA Communications (под брендом ERA One)  
**Статус:** Roadmap — ADR/PRD приняты  
**ADR:** [`0025`](../adr/0025-era-one-shared-platform.md) · [`0027`](../adr/0027-era-communications-architecture.md) · [`0028`](../adr/0028-era-mail-client-strategy.md) · [`0030`](../adr/0030-era-outlook-bridge.md)  
**PRD:** [`PRD-Comms-MVP.md`](PRD-Comms-MVP.md) · [`PRD-Comms-Migration.md`](PRD-Comms-Migration.md) · [`PRD-Outlook-Bridge.md`](PRD-Outlook-Bridge.md) · [`PRD-Mail-Moderation.md`](PRD-Mail-Moderation.md)  
**Pricing:** [`pricing-comms-data.yaml`](../distributor/pricing-comms-data.yaml)

---

## Позиционирование

**ERA Communications** — суверенные корпоративные коммуникации on-prem.

**Standalone:** не требует ERA Core.

**Доноры:** OSS patterns only — [`ERA-Communications-Donors.md`](ERA-Communications-Donors.md).

---

## Killer features

1. **Производительность** — Rust mail core + Go; целевой масштаб 60 000+ users.
2. **Air-Gap AI** — ERA Comms AI (≠ ERA Control AI).
3. **Zero-Touch** — Autodiscover для Outlook/мобильных (серверные протоколы; см. [`0028`](../adr/0028-era-mail-client-strategy.md)).
4. **ClickHouse audit** — **обязателен** в MVP.
5. **Интеграция ERA Office** — co-editing через deep link (лицензия Office; [`ERA-Communications-Donors.md`](ERA-Communications-Donors.md)).
6. **ERA Mail Connect** — hybrid tier, отдельная цена.
7. **ERA Comms Migration** — bulk migration many-to-many (€1/mailbox one-time); [`Comms-Migration-Vendor-Matrix.md`](../Comms-Migration-Vendor-Matrix.md).
8. **ERA Outlook Bridge** — server EWS façade для Outlook без desktop plugin (IceWarp, CG, Exchange); [`ADR-0030`](../adr/0030-era-outlook-bridge.md).
9. **ERA Mail Moderation** — outbound message moderation (SMTP hold → manager/curator Approve/Reject); standalone перед IceWarp/любым SMTP; [`PRD-Mail-Moderation.md`](PRD-Mail-Moderation.md).

---

## Издания

См. [`editions-comms.yaml`](../../editions-comms.yaml). Pricing: [`pricing-comms-data.yaml`](../distributor/pricing-comms-data.yaml).

| Издание | Цена |
|---------|------|
| ERA Mail Server (+ Client) | €10 |
| ERA Mail Connect | €4 |
| ERA Comms Migration | €1/mailbox one-time |
| ERA Outlook Bridge | €3 |
| ERA Mail Moderation | €2 (roadmap) |
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
| Phase 1.2 | ERA Comms Migration |
| Phase 1.3 | ERA Outlook Bridge (partner) |
| Phase 1.4 | ERA Mail Moderation (partner / IceWarp upsell) |
| Phase 2 | ActiveSync, Chat, Conference |
| Phase 3 | Comms AI, scale proof |

**Связано:** [`products.yaml`](../../products.yaml) · [`deploy/profiles/comms.yaml`](../../deploy/profiles/comms.yaml)
