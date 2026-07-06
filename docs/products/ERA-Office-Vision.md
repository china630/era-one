# ERA Office — vision

**Продуктовое семейство:** ERA Office (ERA One)  
**Статус:** Roadmap  
**ADR:** [`0025`](../adr/0025-era-one-shared-platform.md) · [`0026`](../adr/0026-sovereign-office-engine.md)  
**PRD:** [`PRD-Office-MVP.md`](PRD-Office-MVP.md)  
**Pricing:** [`ERA-Pricing-Office-Client.md`](../distributor/ERA-Pricing-Office-Client.md) · [`pricing-office-data.yaml`](../distributor/pricing-office-data.yaml)

---

## Позиционирование

Суверенные офисные приложения и co-editing **on-prem**, свой engine (без OnlyOffice/GPL).  
**Standalone** — не требует ERA Control.

---

## Издания и цены (EU list / user / year)

| Издание | € EU | € СНГ | Фаза |
|---------|------|-------|------|
| **ERA Drive** | 4 | 2 | P0 |
| **ERA Documents** | 8 | 4 | P1 MVP |
| **ERA Tables** | 6 | 3 | P2 |
| **ERA Presentations** | 5 | 2.5 | P3 |
| **ERA Projects** | 4 | 2 | post-MVP |
| **ERA Office AI** | 6 | 3 | post-MVP |

**Bundles:** MVP (Drive+Documents) **€10.8** · Suite **€17.25** (~**€8.6** СНГ).

Drive **всегда включён** в Office Suite. Отдельная лицензия для upsell к Mail.

---

## RFQ

[`ERA-RFQ-Office-Template.md`](../distributor/ERA-RFQ-Office-Template.md)

---

## Engine (ADR-0026)

Native `.era-doc` / `.era-sheet` · CRDT co-editing · Rust OOXML I/O · zero GPL.

---

## Интеграция

Comms → Drive (attachments) → Office (co-edit deep link).

**Workspace:** `app.customer.local` · **Связано:** [`editions-office.yaml`](../../editions-office.yaml)
