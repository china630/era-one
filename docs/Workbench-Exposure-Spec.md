# ERA Workbench + Exposure + BYO-EDR Hub

**Статус:** Implemented (Этап 4)  
**Дата:** 1 июля 2026 г.  
**Источник:** [ADR-0017](adr/0017-vision-one-onprem-patterns.md) §1–§3  
**Roadmap:** [Implementation-Roadmap](Implementation-Roadmap.md) Этап 4

## 4a. ERA Workbench

- **API:** `GET /api/timeline` (`services/event-writer`) — merge events + detections по `node_id` / `correlation_id`.
- **BFF:** `GET /api/v1/workbench/timeline?case_id=` — резолв `case_id` → `node_id`, RBAC `CanReadCases`.
- **Прокси:** `/api/proxy/timeline` для same-origin UI.
- **UI:** `ui/workbench/index.html` — хронологический timeline + фильтр источника; ссылка из Cases.
- **Golden:** `services/event-writer/internal/timeline/merge_test.go` — multi-source PASS.

## 4b. ERA Exposure

- **Агрегатор:** `services/detection-engine/internal/exposure` — score = f(детекты, CVE/vm.finding, критичность платформы).
- **API:** `GET /api/v1/exposure?top=10` на `ERA_HTTP_ADDR` (`:8097`); BFF `/api/v1/exposure`.
- **Входы:** ClickHouse `detections` + `events` (payload `vm.finding`); метаданные активов из control-plane.
- **Тесты:** `exposure/score_test.go` — веса и ранжирование PASS.

## 4c. ERA BYO-EDR Hub

- **Модуль:** `crates/era-collectors/src/byo_edr.rs` — generic JSON + CEF/syslog → `Envelope` (`era.byo-edr.generic`).
- **Golden:** `crates/era-collectors/tests/golden_byo_edr.rs` — wire-формат PASS.

## Не в этом этапе

- Virtual Patching (ADR-0017 §4) → Этап 6.
- Полная asset criticality из CMDB → Этап 5.
- AI forensic evidence chain и attack graph — [`ADR-0023`](adr/0023-ai-investigation-explainability.md) Фаза 2 (Post-GA).

## Доказательство приёмки

```text
go test ./services/event-writer/... ./services/detection-engine/... ./services/control-plane/...
cargo test -p era-collectors
```
