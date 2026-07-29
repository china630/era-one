# ERA Communications — Evidence Rules

**Дата:** 29 июля 2026 г.  
**Статус:** Mandatory для `[x]` в Sprint-Index / Implementation-Matrix / editions.

## Правило

> **Нет лога / CI artifact — нет `[x]`.**  
> Docs claim без пакета в git и без PASS-команды = долг, не готовность.

## Допустимые статусы

| Маркер | Значение |
|--------|----------|
| `[x]` | Пакет в git + proof command exit 0 + лог в `reports/` или CI |
| `[~]` | Код есть; gate partial / не прогнан / FAIL с известным GAP |
| `[ ]` | Нет пакета или нет proof |
| `SKIP` | Proof недоступен (нет CH/host); документирован в логе |

## Edition honesty ([`editions-comms.yaml`](../editions-comms.yaml))

| status | Когда |
|--------|--------|
| `roadmap` | Нет product-ready proof / `exists: false` |
| `scaffold` / `mvp` | Unit/gate PASS на restored code; **не** field |
| `ga` | Pilot-ready + RT-09 / partner field sign-off |

Ложный `ga` без Pilot-ready **запрещён**.

## Proof command patterns

```powershell
.\scripts\run-comms-stage-gate.ps1 -Stage C-1 -WriteSignoff   # → reports/comms-stage-C-1-*.log
go test -C services/comms/<pkg> ./... -count=1
cargo test -p era-mail-core --quiet
```

## Recovery note (2026-07-29)

Comms tree (кроме `mail-moderation`) восстановлен из `stash@{0}^3` → commit `chore(comms): restore…`.  
Backup: branch `comms-stash-backup`, tag `comms-pre-restore`.  
Не удалять stash до подтверждения PO.
