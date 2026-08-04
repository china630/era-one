# ERA Communications — Evidence Rules

**Дата:** 30 июля 2026 г.  
**Статус:** Mandatory для gate/`[x]` / Scaffold / editions.  
**Канон:** [`ERA-Product-Acceptance-Standard.md`](products/ERA-Product-Acceptance-Standard.md) **v1.2** (§3.2–3.5)  
**Приёмка:** [`Comms-Acceptance-System.md`](products/Comms-Acceptance-System.md)  
**Product AC Matrix (SSOT):** [`Comms-Implementation-Matrix.md`](Comms-Implementation-Matrix.md)

## Правило

> **Нет лога / CI artifact — нет `gate[x]`.**  
> Docs claim без пакета в git и без PASS-команды = долг, не готовность.

**Scaffold ≠ Pilot-ready.** **Rollup = Matrix.**  
Запрещены шапки «all green / all check» при Matrix 🟡; запрещены partner/greenfield **ga**-ярлыки в docs при `editions-comms.yaml` = `mvp` (канон §3.3).

## Допустимые статусы

| Маркер | Значение |
|--------|----------|
| `gate[x]` / `[x]` | Proof + лог (уровень gate/задачи) |
| `[~]` | Partial / известный GAP |
| `[ ]` | Нет proof |
| `[blocked]` | Внешний/полевой гейт |
| `SKIP` | Proof недоступен; в логе |

## Edition honesty ([`editions-comms.yaml`](../editions-comms.yaml))

| status | Когда |
|--------|--------|
| `roadmap` | Нет proof / `exists: false` |
| `scaffold` / `mvp` | Unit/gate PASS; **не** field |
| `ga` | Pilot-ready + RT-09 / partner field sign-off |

Ложный `ga` без Pilot-ready **запрещён**. Partner/RFQ docs копируют yaml.

## Signoff naming

`scaffold-gate-pass` / stage signoff — OK. Не «product green», пока RT-09 / Pilot-ready open.

## Proof command patterns

```powershell
.\scripts\run-comms-stage-gate.ps1 -Stage C-1 -WriteSignoff
.\scripts\check-acceptance-consistency.ps1
go test -C services/comms/<pkg> ./... -count=1
cargo test -p era-mail-core --quiet
```

## Recovery note (2026-07-29)

Comms tree восстановлен из stash; не удалять stash до подтверждения PO.
