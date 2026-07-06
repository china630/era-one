# ERA One — продуктовая документация (index)

**Обновлено:** 5 июля 2026 г.

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
| **ERA Control** | [`ERA-Platform-Vision.md`](../ERA-Platform-Vision.md) | GA specs | ADR-0005, 0018 |
| **ERA Communications** | [`ERA-Communications-Vision.md`](ERA-Communications-Vision.md) | [`PRD-Comms-MVP.md`](PRD-Comms-MVP.md) | [`0027`](../adr/0027-era-communications-architecture.md) |
| **ERA Office** | [`ERA-Office-Vision.md`](ERA-Office-Vision.md) | [`PRD-Office-MVP.md`](PRD-Office-MVP.md) | [`0026`](../adr/0026-sovereign-office-engine.md) |
| **Shared platform** | — | — | [`0025`](../adr/0025-era-one-shared-platform.md) |

## Ключевые решения

| Тема | Решение |
|------|---------|
| Identity | Включена; не продаётся отдельно |
| ERA Drive | Отдельный SKU; в Office Suite всегда |
| Inline attachments | Tenant policy (defaults в PRD-Comms) |
| ClickHouse (Comms) | **Обязателен** |
| ERA Mail Connect | €4 EU / user / year (отдельно) |
| Comms/Office | Standalone, без ERA Core |
