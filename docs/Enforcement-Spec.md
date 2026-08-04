# ERA Manage — Enforcement (Stage 6 + Phase 2 lab decision)

Спецификация enforcement-слоя: policy-движок, плагины, control-plane API, UI.

**Связано:** [ADR-0012](adr/0012-agent-enforcement-mode.md) · [ADR-0016 §2](adr/0016-uem-scope-vs-ivanti.md) ·
[ADR-0017 §4](adr/0017-vision-one-onprem-patterns.md) · лицензия `manage` ·
[ERA-Manage-WHQL-Program.md](ERA-Manage-WHQL-Program.md).

**Статус кода:** **monitor-complete** ✅ · **lab decision + `effect=telemetry_only`** ✅ ·  
**user-land gate** (`ERA_ENFORCE_LIVE=1` → `effect=user_land_block`) ✅ ·  
боевой kernel / WHQL minifilter — **⏸ WHQL** (не заявляется).

**Вне скоупа этой волны:** EPM-lite / JIT admin elevation — отдельная линия Manage P6.

## Инварианты

- Telemetry — дефолт; enforcement только при `manage` + policy из control-plane.
- Обязательный `monitor` перед `enforce`; дефолт **fail-open**.
- Решение офлайн на агенте; каждое deny → `Envelope` detection (включая virtual patch).
- `mode=enforce` без `ERA_ENFORCE_LIVE`: App/USB → `blocked=true` + **`effect=telemetry_only`**, `allowed=true` (honesty — процесс продолжается).
- `mode=enforce` + **`ERA_ENFORCE_LIVE=1`**: App deny (non-VP) → `allowed=false`, `effect=user_land_block`, `blocked=true`; optional Linux SIGTERM stub if pid>0 — **не** WHQL kernel.
- Virtual patch — **monitor-only** до kernel WHQL (`hook_message` явное).
- Kernel stub: `kernel_hook=unavailable` (или `user_land` probe on Linux LIVE) — unsigned minifilter не грузим.
- Боевой kernel enforce и подпись драйвера — **[gate: external / WHQL]**.

## Acceptance (AC)

| ID | Criterion | Scaffold | Pilot |
|----|-----------|----------|-------|
| **AC-E1** | monitor deny → detection + `would_block` | ✅ | [ ] field |
| **AC-E2** | enforce deny → detection + `blocked` + **`effect=telemetry_only`** (default) | ✅ | [ ] field |
| **AC-E2b** | `ERA_ENFORCE_LIVE=1` + enforce deny → `allowed=false` + `effect=user_land_block` | ✅ | [ ] field |
| **AC-E3** | spoof / unlicensed policy write → 403; `ERA_AGENT_TOKEN` ≠ admin | ✅ | [ ] field |
| **AC-E4** | kernel OS block (WHQL minifilter) | ⏸ WHQL | ⏸ WHQL |

## Компоненты

| Компонент | Путь | Роль |
|---|---|---|
| Policy engine | `crates/era-agent-core/src/enforce/` | allow/deny, `BlockResult.effect`, kernel stub |
| Kernel stub | `enforce/kernel.rs` | `probe_kernel_hook` → unavailable / user_land |
| Orchestrator hook | `orchestrator.rs` | `ERA_ENFORCEMENT=1` → load policy, check process events |
| App Control plugin | `crates/era-plugin-appcontrol` | monitor + lab decision (`blocked`, `effect`, `kernel_hook`) |
| Device Control plugin | `crates/era-plugin-devicecontrol` | USB events with `would_block` / `blocked` / `effect` |
| BitLocker plugin | `crates/era-plugin-bitlocker` | volume status + `simulated` (non-Windows); escrow via CP only |
| CP API | `services/control-plane/internal/api/enforcement.go` | policy, rollback, escrow (admin + trust) |
| UI | `ui/enforcement/` | monitor/enforce toggle, rollback, escrow |

## Modes

| Mode | App/USB deny | Virtual patch | Kernel |
|------|--------------|---------------|--------|
| `monitor` (default) | allow + `would_block` + detection · `effect=telemetry_only` | same | n/a |
| `enforce` (lab) | `blocked=true` + detection · **`effect=telemetry_only`**, `allowed=true` | still monitor-only | `kernel_hook=unavailable` |
| `enforce` + `ERA_ENFORCE_LIVE=1` | `allowed=false`, `effect=user_land_block` (+ optional SIGTERM stub) | still monitor-only | probe may report `user_land`; **WHQL kernel ⏸** |

Env: `ERA_ENFORCE_MODE=enforce` (plugins), policy JSON `mode: enforce` (agent-core).  
`ERA_ENFORCE_LIVE=1` enables user-land gate semantics (`user_land_block`); kernel WHQL remains separate and unclaimed.

## API (control-plane)

| Метод | Путь | Описание |
|---|---|---|
| GET | `/api/v1/enforcement/policy` | Policy bundle (агент: `Bearer $ERA_AGENT_TOKEN` → trusted agent **≠ admin**, или `Bearer $ERA_API_KEY` admin, или Trusted-Proxy **и** trusted hop; spoof → 401/403) |
| PUT | `/api/v1/enforcement/policy` | Обновление policy (manage + **admin**; agent token → 403) |
| POST | `/api/v1/enforcement/rollback` | Откат к предыдущей версии |
| GET | `/api/v1/enforcement/history` | Аудит изменений policy |
| GET/POST | `/api/v1/enforcement/escrow` | BitLocker escrow (ключи не в списках) |
| GET | `/api/v1/enforcement/escrow/{node}/{volume}` | Деталь (ключ — только admin) |

## Агент

```bash
ERA_ENFORCEMENT=1 ERA_CONTROL_PLANE_URL=http://127.0.0.1:8090 \
  ERA_API_KEY=… cargo run -p era-agent
```

Policy загружается при старте из CP (`Authorization: Bearer`, без forge `X-ERA-Role: admin`);
в production/strict токен обязателен. Process-события из capture проходят `check_process_envelope`.
`effect=telemetry_only` до WHQL — OS kill не заявляется.

## Тесты (доказательство приёмки)

- `cargo test -p era-agent-core enforce::` — engine + golden `enforce_vs_monitor` + `enforce_live_user_land`
- `cargo test -p era-plugin-appcontrol -p era-plugin-devicecontrol -p era-plugin-bitlocker` — goldens with `effect`
- `go test ./services/control-plane/...` — enforcement API + agent token ≠ admin + spoof→401
- `cargo test -p era-agent-core enforce::loader` — Bearer headers; no forged admin role

## Гейты

| Гейт | Статус |
|---|---|
| Monitor-complete (detections + plugin goldens) | ✅ |
| Lab decision goldens (`blocked` + `effect=telemetry_only`) | ✅ |
| User-land gate (`ERA_ENFORCE_LIVE=1` → `user_land_block`) | ✅ code path |
| Kernel OS block / USB hardware deny / WHQL driver | ⏸ WHQL |
| WHQL / нотаризация драйвера | [gate: external] — см. WHQL program |
| Security-review хуков | [gate: external] |
| Полевой monitor-soak | [gate: field] |

**Claim:** lab decision ✅ · user-land gate ✅ (`ERA_ENFORCE_LIVE`) · **Non-claim:** WHQL kernel GA ⏸
