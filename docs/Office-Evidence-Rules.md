# ERA Office — Evidence Rules

**Дата:** 30 июля 2026 г.  
**Статус:** Mandatory для gate/`[x]` / Scaffold / editions.  
**Канон:** [`ERA-Product-Acceptance-Standard.md`](products/ERA-Product-Acceptance-Standard.md) **v1.2** (§3.2–3.5)  
**Приёмка:** [`Office-Acceptance-System.md`](products/Office-Acceptance-System.md)  
**Product AC Matrix (SSOT):** [`Office-Implementation-Matrix.md`](Office-Implementation-Matrix.md)

## Правило

> **Нет лога / CI artifact — нет `gate[x]`.**  
> Docs claim без пакета в git и без PASS-команды = долг, не готовность.

**Scaffold ≠ Pilot-ready.** **Rollup = Matrix.**  
Запрещены шапки `Scaffold AC all ✅` / `Matrix all ✅`, если в Matrix есть 🟡 (Drive bind, Tables, Presentations, Projects, …).  
PRD «Статус: Scaffold ✅» только при edition rollup ✅.

## Допустимые статусы

| Маркер | Значение |
|--------|----------|
| `gate[x]` / `[x]` | Proof + лог (уровень gate/задачи) |
| `[~]` | Partial / известный GAP |
| `[ ]` | Нет proof |
| `[blocked]` | Внешний/полевой гейт |
| `SKIP` | Proof недоступен; в логе |

## Edition honesty ([`editions-shared.yaml`](../editions-shared.yaml) · [`editions-office.yaml`](../editions-office.yaml))

| status | Когда |
|--------|--------|
| `roadmap` | Нет proof / `exists: false` |
| `scaffold` / `mvp` | Unit/gate PASS; **не** field |
| `ga` | Pilot-ready + field sign-off (RT-O09) |

Ложный `ga` без Pilot-ready **запрещён**.

## Signoff naming

`office-stage-*-signoff` = scaffold-gate. Не product-green при RT-O09 open.

## Proof command patterns

```powershell
.\scripts\run-office-stage-gate.ps1 -Stage O-0 -WriteSignoff
.\scripts\check-acceptance-consistency.ps1
go test ./services/platform/drive/... -count=1
```

## Recovery note (2026-07-30)

Office scaffold восстановлен точечно; канон волн **O-0…O-GA**.  
Не удалять `stash@{0}` до подтверждения PO.
