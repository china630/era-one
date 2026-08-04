# ERA One — продуктовая документация (index)

**Обновлено:** 30 июля 2026 г.

## Манифесты изданий

| Слой | Файл |
|------|------|
| ERA Control | [`editions-control.yaml`](../../editions-control.yaml) |
| ERA Communications | [`editions-comms.yaml`](../../editions-comms.yaml) |
| ERA Office | [`editions-office.yaml`](../../editions-office.yaml) |
| Shared platform | [`editions-shared.yaml`](../../editions-shared.yaml) |

## Pricing SSOT

| Продукт | Файл |
|---------|------|
| ERA Control | [`pricing-data.yaml`](../distributor/pricing-data.yaml) | [`ERA-Pricing-Client.md`](../distributor/ERA-Pricing-Client.md) |
| ERA Communications | [`pricing-comms-data.yaml`](../distributor/pricing-comms-data.yaml) | [`ERA-Pricing-Comms-Client.md`](../distributor/ERA-Pricing-Comms-Client.md) |
| ERA Office | [`pricing-office-data.yaml`](../distributor/pricing-office-data.yaml) | [`ERA-Pricing-Office-Client.md`](../distributor/ERA-Pricing-Office-Client.md) |

Пересборка сайта: `python scripts/build_portal.py`

## RFQ

- [`ERA-RFQ-Template.md`](../distributor/ERA-RFQ-Template.md) — Control
- [`ERA-RFQ-Comms-Template.md`](../distributor/ERA-RFQ-Comms-Template.md)
- [`ERA-RFQ-Office-Template.md`](../distributor/ERA-RFQ-Office-Template.md)

## Продуктовые семейства

| Продукт | Vision | PRD | ADR |
|---------|--------|-----|-----|
| **ERA Control** | [`ERA-Platform-Vision.md`](../ERA-Platform-Vision.md) | GA specs · [`PRD-ERA-Perimeter.md`](PRD-ERA-Perimeter.md) · [`PRD-ERA-Resolve.md`](PRD-ERA-Resolve.md) | ADR-0005, 0018, [`0031`](../adr/0031-era-resolve-and-perimeter-editions.md) · [`Control-Acceptance-System.md`](Control-Acceptance-System.md) |
| **ERA Communications** | [`ERA-Communications-Vision.md`](ERA-Communications-Vision.md) | [`PRD-Comms-MVP.md`](PRD-Comms-MVP.md) · [`PRD-Outlook-Bridge.md`](PRD-Outlook-Bridge.md) · [`PRD-Comms-Migration.md`](PRD-Comms-Migration.md) · [`PRD-Mail-Moderation.md`](PRD-Mail-Moderation.md) | [`0027`](../adr/0027-era-communications-architecture.md) · [`0030`](../adr/0030-era-outlook-bridge.md) · [`Comms-Acceptance-System.md`](Comms-Acceptance-System.md) |
| **ERA Office** | [`ERA-Office-Vision.md`](ERA-Office-Vision.md) | [`PRD-Office-MVP.md`](PRD-Office-MVP.md) · [`PRD-Office-P0.md`](PRD-Office-P0.md) · [`PRD-Office-P1.md`](PRD-Office-P1.md) · [`PRD-Office-P2.md`](PRD-Office-P2.md) | [`0026`](../adr/0026-sovereign-office-engine.md) · [`Office-Sprint-Index.md`](../Office-Sprint-Index.md) · [`Office-Acceptance-System.md`](Office-Acceptance-System.md) |
| **Shared platform** | — | [`Shared-Acceptance-System.md`](Shared-Acceptance-System.md) | [`0025`](../adr/0025-era-one-shared-platform.md) |

## Приёмка (единый стандарт)

**Канон v1.3:** [`ERA-Product-Acceptance-Standard.md`](ERA-Product-Acceptance-Standard.md) · агент: `.cursor/rules/task-acceptance.mdc` · check: `.\scripts\check-acceptance-consistency.ps1`

| Продукт | Acceptance-System | **Product Readiness (готовность)** | **AC Matrix (BE)** | Gate |
|---------|-------------------|------------------------------------|--------------------|------|
| **ERA Control** | [`Control-Acceptance-System.md`](Control-Acceptance-System.md) | [`Control-Product-Readiness-Matrix.md`](../Control-Product-Readiness-Matrix.md) | [`Control-Implementation-Matrix.md`](../Control-Implementation-Matrix.md) | `ci-gates` · `run-ga-full` |
| **ERA Communications** | [`Comms-Acceptance-System.md`](Comms-Acceptance-System.md) | [`Comms-Product-Readiness-Matrix.md`](../Comms-Product-Readiness-Matrix.md) | [`Comms-Implementation-Matrix.md`](../Comms-Implementation-Matrix.md) | `run-comms-stage-gate` |
| **ERA Office** | [`Office-Acceptance-System.md`](Office-Acceptance-System.md) | [`Office-Product-Readiness-Matrix.md`](../Office-Product-Readiness-Matrix.md) | [`Office-Implementation-Matrix.md`](../Office-Implementation-Matrix.md) | `run-office-stage-gate` |
| **Shared** | [`Shared-Acceptance-System.md`](Shared-Acceptance-System.md) | Drive UI → Office Readiness | ADR §0025 · consumer AC | package tests |

**«Матрица готовности» → колонка Product Readiness**, не AC Matrix.  
Инварианты: gate ≠ BE ✅ ≠ UI/TE ✅ ≠ Pilot; `ga` только из editions.  
Honesty: [`Acceptance-Honesty-Audit-20260730.md`](../Acceptance-Honesty-Audit-20260730.md).

## Ключевые решения

| Тема | Решение |
|------|---------|
| Identity | Включена; не продаётся отдельно |
| ERA Drive | Отдельный SKU; в Office Suite всегда |
| Inline attachments | Tenant policy (defaults в PRD-Comms) |
| ClickHouse (Comms) | **Обязателен** |
| ERA Mail Connect | €4 EU / user / year (отдельно) |
| ERA Outlook Bridge | Server EWS façade — [ADR-0030](../adr/0030-era-outlook-bridge.md) |
| ERA Comms Migration | Many-to-many — [vendor matrix](../Comms-Migration-Vendor-Matrix.md) |
| ERA Mail Moderation | Outbound SMTP Approve/Reject — [PRD-Mail-Moderation.md](PRD-Mail-Moderation.md) |
| ERA Manage WHQL | [ERA-Manage-WHQL-Program.md](../ERA-Manage-WHQL-Program.md) |
| ERA Perimeter | [PRD-ERA-Perimeter.md](PRD-ERA-Perimeter.md) · [Perimeter-Spec.md](../Perimeter-Spec.md) · ADR-0031 |
| ERA Resolve (DNS DDR) | [PRD-ERA-Resolve.md](PRD-ERA-Resolve.md) · [Resolve-Spec.md](../Resolve-Spec.md) · ADR-0031 — **не** ITSM |
| Office MVP waves | O-0…O-GA — [Office-Sprint-Index.md](../Office-Sprint-Index.md) · [Office-MVP-Spec.md](../Office-MVP-Spec.md) |
| Office UI / O-FMT | [Office-UI-Controls-Catalog.md](../Office-UI-Controls-Catalog.md) · [Menu-Map](../Office-UI-Menu-Map.md) · [Feature-Inventory](../Office-UI-Feature-Inventory.md) |
| Office Roadmap | Фазы A/B/C (legacy) — [Office-Roadmap.md](../Office-Roadmap.md) |
| Comms/Office | Standalone, без ERA Core |
