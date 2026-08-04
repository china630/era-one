# ERA Manage — WHQL / driver signing program

**Статус:** external gate (не блокирует monitor-complete кода)  
**Связано:** [Enforcement-Spec.md](Enforcement-Spec.md) · ADR-0012 · ADR-0017 §4

## Зачем

Боевой Application / Device Control / Virtual Patching на Windows требует
kernel minifilter (и/или eBPF на Linux) с **подписанным** драйвером. Без WHQL/нотаризации
режим `enforce` остаётся недоступен для продакшена (fail-open monitor только).

## Scope гейта

| Компонент | Monitor (код) | Enforce (gate) |
|-----------|---------------|----------------|
| App Control | ✅ | WHQL + security-review |
| Device Control (USB) | ✅ | WHQL + security-review |
| Virtual Patching | ✅ | WHQL + security-review |
| BitLocker mgmt | status/escrow ✅ | field key custody |

## Чеклист (вне кода агента)

- [ ] EV code-signing certificate заказан
- [ ] Minifilter / eBPF prototype в lab
- [ ] WHQL submission / attestation
- [ ] Security-review хуков (fail-open, tamper)
- [ ] Pilot: monitor soak → enforce canary

Пока пункты открыты — в RFQ/Product Line писать **lab decision + `effect=telemetry_only` ✅ · OS block ⏸ WHQL**, не GA enforce / OS kill.

**Связь с Scaffold-Green:** AC-E1…E3 закрыты без WHQL; **AC-E4** остаётся ⏸ до этого program.
