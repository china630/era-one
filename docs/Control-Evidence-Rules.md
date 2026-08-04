# ERA Control — Evidence Rules

**Дата:** 30 июля 2026 г.  
**Статус:** Mandatory для gate/`[x]` / Scaffold / editions.  
**Канон:** [`ERA-Product-Acceptance-Standard.md`](products/ERA-Product-Acceptance-Standard.md) **v1.2** (§3.2–3.6)  
**Приёмка:** [`Control-Acceptance-System.md`](products/Control-Acceptance-System.md)  
**Product AC Matrix (SSOT):** [`Control-Implementation-Matrix.md`](Control-Implementation-Matrix.md)

## Правило

> **Нет лога / CI artifact — нет `gate[x]`.**  
> Docs claim без пакета в git и без PASS-команды = долг, не готовность.

**Scaffold ≠ Pilot-ready.** **Rollup = Matrix** (канон §3.4).  
Запрещены шапки `all ✅` / prose `ga` вне `editions-*.yaml` при несовпадении с Matrix/yaml.

## Допустимые статусы

| Маркер | Значение |
|--------|----------|
| `gate[x]` / `[x]` | Пакет в git + proof exit 0 + лог `reports/` или CI (**только уровень gate/задачи**) |
| `[~]` | Код есть; gate partial / известный GAP |
| `[ ]` | Нет пакета или нет proof |
| `[blocked]` | Внешний/полевой гейт |
| `SKIP` | Proof недоступен; в логе |

Scaffold ✅ / 🟡 — только в Matrix по канону §3.2.

## Edition honesty ([`editions-control.yaml`](../editions-control.yaml))

| status | Когда |
|--------|--------|
| `roadmap` | Нет proof / `exists: false` |
| `scaffold` / `mvp` | Unit/CI PASS; **не** field |
| `ga` / software-ga | Carve-out § ниже **или** полный Pilot-ready |
| `ga-option` | Soft GA по лицензии; field может быть открыт |

### Carve-out: software GA (только Core / Control AI / Response)

Могут оставаться `ga` в yaml как **software GA**. Field-AC:

| Open Pilot-ready | Scaffold (max) | Pilot-ready |
|------------------|----------------|-------------|
| F-GA-5 loadgen 10k prod | **🟡** soft script | `[blocked: field]` |
| F-GA-8 coverage ≥90% | **🟡** API | `[blocked: field]` |
| F-GA-15 checklist signed | **🟡** template | `[blocked: field]` |

**Запрещено:** Scaffold ✅ на этих field-AC; Pilot-ready `[x]` без полевого лога.  
**Запрещено:** поднимать Manage/Perimeter/Resolve/… в `ga` без Pilot-ready.

## Signoff naming

Допустимо: `scaffold-gate-pass`, `control-scaffold-gate-signoff`.  
**Не** называть product-ready / «green» PASS, пока Pilot-ready open.

## Proof command patterns

```powershell
.\scripts\ci-gates-stage10.ps1
.\scripts\run-ga-full.ps1
.\scripts\check-acceptance-consistency.ps1
go test ./services/control-plane/... -count=1
```

## Связано

- [`Control-Acceptance-System.md`](products/Control-Acceptance-System.md)
- [`Control-Sprint-Index.md`](Control-Sprint-Index.md)
- [`ADR-Implementation-Matrix.md`](ADR-Implementation-Matrix.md) — ADR, не product rollup
