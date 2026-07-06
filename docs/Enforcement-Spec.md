# ERA Manage — Enforcement (Stage 6)

Спецификация enforcement-слоя: policy-движок, плагины, control-plane API, UI.

**Связано:** [ADR-0012](adr/0012-agent-enforcement-mode.md) · [ADR-0016 §2](adr/0016-uem-scope-vs-ivanti.md) ·
[ADR-0017 §4](adr/0017-vision-one-onprem-patterns.md) · лицензия `manage`.

## Инварианты

- Telemetry — дефолт; enforcement только при `manage` + policy из control-plane.
- Обязательный `monitor` перед `enforce`; дефолт **fail-open**.
- Решение офлайн на агенте; каждое deny → `Envelope` detection.
- Боевой kernel enforce и подпись драйвера — **[gate: external]**.

## Компоненты

| Компонент | Путь | Роль |
|---|---|---|
| Policy engine | `crates/era-agent-core/src/enforce/` | allow/deny path/hash/signer/parent, virtual patches, device rules |
| Orchestrator hook | `orchestrator.rs` | `ERA_ENFORCEMENT=1` → load policy, check process events |
| App Control plugin | `crates/era-plugin-appcontrol` | monitor/simulated hook stub |
| Device Control plugin | `crates/era-plugin-devicecontrol` | USB audit stub |
| BitLocker plugin | `crates/era-plugin-bitlocker` | on-demand volume status + escrow request |
| CP API | `services/control-plane/internal/api/enforcement.go` | policy, rollback, escrow |
| UI | `ui/enforcement/` | monitor/enforce toggle, rollback, escrow |

## API (control-plane)

| Метод | Путь | Описание |
|---|---|---|
| GET | `/api/v1/enforcement/policy` | Policy bundle (агент: `X-ERA-Actor: era-agent`) |
| PUT | `/api/v1/enforcement/policy` | Обновление policy (manage + admin) |
| POST | `/api/v1/enforcement/rollback` | Откат к предыдущей версии |
| GET | `/api/v1/enforcement/history` | Аудит изменений policy |
| GET/POST | `/api/v1/enforcement/escrow` | BitLocker escrow (ключи не в списках) |
| GET | `/api/v1/enforcement/escrow/{node}/{volume}` | Деталь (ключ — только admin) |

## Агент

```bash
ERA_ENFORCEMENT=1 ERA_CONTROL_PLANE_URL=http://127.0.0.1:8090 cargo run -p era-agent
```

Policy загружается при старте из CP; process-события из capture проходят `check_process_envelope`.

## Тесты (доказательство приёмки)

- `cargo test -p era-agent-core` — engine golden, policy fuzz, orchestrator unit
- `cargo test -p era-plugin-appcontrol -p era-plugin-devicecontrol -p era-plugin-bitlocker`
- `go test ./services/control-plane/...` — enforcement API + default policy golden

## Гейты

| Гейт | Статус |
|---|---|
| WHQL / нотаризация драйвера | [gate: external] |
| Security-review хуков | [gate: external] |
| Полевой monitor-soak | [gate: external] |

Код этапа закрывается в режиме **monitor + simulated enforce** без прохождения гейтов.
