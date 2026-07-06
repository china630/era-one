# ADR-0019: Platform Agent Orchestrator (scheduler, plugin ABI, OTA)

**Статус:** Implemented  
**Дата:** 1 июля 2026 г.  
**Контекст:** [ERA-Platform-Vision.md](../ERA-Platform-Vision.md) §5 (core + plugins), §9 (OTA/mirror), §13 P1.
Текущий `era-agent` — монолитный capture-loop. IT-Ops модули (inventory, vuln, enforcement)
требуют планировщика, лицензированной загрузки плагинов и OTA без нарушения air-gap.

**Связано:** [ADR-0003](0003-repository-structure-and-donor-strategy.md) ·
[ADR-0005](0005-module-independence-and-packaging.md) ·
[ADR-0009](0009-pii-redaction-and-agent-budget.md) ·
[ADR-0010](0010-licensing-and-activation.md) ·
[ADR-0012](0012-agent-enforcement-mode.md) ·
[ADR-0018](0018-hybrid-connected-operating-model.md)

---

## Решение

### 1. Разделение core / plugins

| Компонент | Крейт | Роль |
|---|---|---|
| Orchestrator | `era-agent-core` | gRPC, buffer, scheduler, OTA, budget-guard, license-gate |
| Realtime XDR | `era-agent-core` (capture/tamper) | realtime — всегда в core |
| Cron/on-demand | `era-plugin-*` | subprocess, NDJSON → Envelope |
| Installer binary | `era-agent` | тонкая обёртка над core |

**Инвариант:** realtime capture и tamper остаются в core; cron/on-demand — только плагины-subprocess.

### 2. Plugin ABI (subprocess)

1. Core читает манифест плагина (`name`, `mode`, `license_module`, `schedule_secs`, `budget_hint_mb`).
2. При cron/on-demand core `exec` бинаря плагина; плагин пишет **NDJSON** в stdout (по строке — одна запись).
3. Core маппит NDJSON → `Envelope` (proto ADR-0001), sanitize, buffer.
4. Exit code = статус; stderr = логи плагина (не в lake).

Донор-референс (идеи only, ADR-0003): Telegraf / Elastic Agent — модель I/O и lifecycle, не код.

### 3. Scheduler + license-gate + budget-guard

- **Scheduler:** реестр плагинов, cron по `schedule_secs`, lazy-load (не резидентно).
- **License-gate:** плагин запускается только если `license_module` ∈ claims (`era-license`, ADR-0010).
- **Budget-guard (ADR-0009):** перед cron проверка CPU/RAM (`sysinfo`); при превышении — defer; realtime не трогаем.

### 4. OTA-модель

- Манифест артефакта: версия, SHA-256, подпись Ed25519 (`ERAAOT1` wire).
- Источник: локальное зеркало (`ERA_OTA_MIRROR`, air-gap); наполнение — Update Service (ADR-0018).
- Content-addressed кэш: `hash → blob`; verify до активации. Без P2P/WAN в рантайме.

### 5. Бюджет и безопасность

- `#![deny(unsafe_code)]` на всех agent/plugin крейтах.
- CI-gate: `criterion` bench steady-state (ADR-0009: CPU < 2%, RAM < 150 МБ ориентир).
- Envelope/proto не меняем; новые домены — только additive fields, contracts-first.

---

## Последствия

- Новые крейты: `era-agent-core`, `era-plugin-sdk`, `era-plugin-inventory`.
- `era-agent` остаётся точкой входа installer; обратная совместимость env/скриптов.
- Enforcement-плагины (ADR-0012) — отдельный этап; CMDB merge (ADR-0011) — на сервере.

## Статус реализации

Реализовано в Этапе 3 (agent-core split + inventory plugin + тесты).
