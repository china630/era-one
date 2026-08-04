# ERA Office — vision

**Продуктовое семейство:** ERA Office (ERA One)  
**Статус:** MVP (editions Drive/Documents/Tables/Presentations/Projects/AI; field RT-O09 open → not GA)  

**ADR:** [`0025`](../adr/0025-era-one-shared-platform.md) · [`0026`](../adr/0026-sovereign-office-engine.md)  
**PRD:** [`PRD-Office-MVP.md`](PRD-Office-MVP.md)  
**Pricing:** [`ERA-Pricing-Office-Client.md`](../distributor/ERA-Pricing-Office-Client.md) · [`pricing-office-data.yaml`](../distributor/pricing-office-data.yaml)

---

## Позиционирование

Суверенные офисные приложения и co-editing **on-prem**, свой engine (без OnlyOffice/GPL).  
**Standalone** — не требует ERA Control.

### Клиенты (roadmap)

| Профиль | Где | Данные |
|---------|-----|--------|
| **Browser** | Workspace (фаза A) | ERA Drive + co-edit |
| **Solo desktop** | Tauri, локальный диск (фаза B Solo) | файлы на машине; без co-edit |
| **Corporate desktop** | Tauri → URL tenant’а (фаза B-corp) | тот же Drive/SSO, что браузер |

Канон фаз, матрица фич и шаги **S0…S5 с подтверждением:** [`Office-Roadmap.md`](../Office-Roadmap.md).

---

## Издания и цены (EU list / user / year)

| Издание | € EU | € СНГ | Фаза |
|---------|------|-------|------|
| **ERA Drive** | 4 | 2 | mvp |
| **ERA Documents** | 8 | 4 | mvp |
| **ERA Tables** | 6 | 3 | mvp |
| **ERA Presentations** | 5 | 2.5 | mvp |
| **ERA Projects** | 4 | 2 | mvp |
| **ERA Office AI** | 6 | 3 | mvp |

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
