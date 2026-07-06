# Agent-Core Spec — Plugin ABI MVP (ADR-0019)

**Версия:** 1.0  
**Дата:** 1 июля 2026 г.  
**Статус:** Implemented (Этап 3)

Связано: [ADR-0019](adr/0019-platform-agent-orchestrator.md) · [editions-control.yaml](../editions-control.yaml) ·
[Implementation-Roadmap](Implementation-Roadmap.md) Этап 3

---

## Backlog AC-*

| ID | Компонент | Статус |
|---|---|---|
| AC-1 | `crates/era-agent-core` — orchestrator, capture, scheduler | [x] |
| AC-2 | `crates/era-agent` — тонкая обёртка | [x] |
| AC-3 | Plugin ABI (subprocess NDJSON → Envelope) | [x] |
| AC-4 | `crates/era-plugin-sdk` | [x] |
| AC-5 | Scheduler + license-gate + budget-guard | [x] |
| AC-6 | OTA-скелет ERAAOT1 + локальное зеркало | [x] |
| AC-7 | `crates/era-plugin-inventory` | [x] |
| AC-8 | Golden PII + plugin NDJSON + e2e | [x] |

---

## Критерии приёмки

| AC | Доказательство |
|---|---|
| Поведение XDR/PII не сломано | `cargo test -p era-agent -p era-agent-core` — PASS |
| Plugin e2e | `tests/plugin_e2e.rs` — PASS |
| Golden NDJSON→Envelope | `testdata/inventory_envelope_wire.golden.hex` |
| OTA verify | `ota::tests::verify_rejects_tampered` — PASS |
| `#![deny(unsafe_code)]` | era-agent-core, era-plugin-sdk, era-plugin-inventory |

---

## Env (dev)

| Переменная | Назначение |
|---|---|
| `ERA_ENABLE_INVENTORY_PLUGIN` | `1` (default) — cron inventory |
| `ERA_PLUGIN_DIR` | каталог бинарей плагинов |
| `ERA_LICENSE_TOKEN` / `ERA_VENDOR_PUB` | license-gate prod |
| `ERA_DEV_ALLOW_PLUGINS` | `1` dev без токена |
| `ERA_OTA_MIRROR` | локальное зеркало OTA |
