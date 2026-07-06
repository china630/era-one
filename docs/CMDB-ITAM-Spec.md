# ERA Manage — CMDB / ITAM (ядро)

**Статус:** Implemented (Этап 5)  
**Дата:** 1 июля 2026 г.  
**Источник:** [ADR-0011](adr/0011-cmdb-itam-data-model.md)  
**Roadmap:** [Implementation-Roadmap](Implementation-Roadmap.md) Этап 5

## Поток

```
era-plugin-inventory → Envelope (era.inventory.*) → ingest → Kafka xdr.inventory
  ├─ event-writer → ClickHouse (история)
  └─ control-plane consumer → CMDB (assets + asset_software)
```

## Компоненты

| Компонент | Путь | Статус |
|---|---|---|
| Inventory plugin (HW/OS/SW) | `crates/era-plugin-inventory` | [x] |
| CMDB schema | `control-plane/internal/store` | [x] |
| Merge/dedup | `control-plane/internal/inventory` | [x] golden PASS |
| Kafka topic | `xdr.inventory` | [x] |
| CMDB API | `/api/v1/cmdb/*` | [x] gate `manage` |
| Financial ITAM | contracts, licenses, reconcile | [x] |
| VM CVE input | `GET /api/v1/vm/software?product=` | [x] |
| UI | `ui/assets/index.html` | [x] |

## Доказательство

```text
go test ./services/control-plane/... ./services/ingest-gateway/... ./services/vm/...
cargo test -p era-plugin-inventory -p era-plugin-sdk
```
