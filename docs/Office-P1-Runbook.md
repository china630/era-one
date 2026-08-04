# ERA Office P1 — operator runbook

## Services

| Service | Port | Health |
|---------|------|--------|
| docs-engine | 8142 | `/healthz` |
| workspace `/docs` | 8170 | BFF proxy |

## Dev license

Set `ERA_OFFICE_DEV=1` or `ERA_LICENSE_OFFICE_DOCUMENTS=1` on docs-engine.

## Smoke

```powershell
.\scripts\office-p1-smoke.ps1
cargo test -p era-docs-engine --quiet
go test ./ui/docs/... ./ui/mail/... -run Documents -count=1
```

## Gate

```powershell
.\scripts\run-office-stage-gate.ps1 -Stage O1-GA
```
