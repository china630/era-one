# Control scaffold-gate signoff (не product-ready)

**Дата:** 30 июля 2026 г.  
**Уровень:** **scaffold-gate-pass** only (канон v1.2 §4 G6) — **не** product green / Pilot-ready  
**План:** AuthZ · License · Enforce honesty · Collectors PII · AC matrix · Field prep  
**Матрица:** [`docs/Control-Implementation-Matrix.md`](../docs/Control-Implementation-Matrix.md)  
**Индекс:** [`docs/Control-Sprint-Index.md`](../docs/Control-Sprint-Index.md)

## Verdict

**Scaffold-gate cycle: PASS** for in-scope non-field AC (negative paths + honesty labels).  
F-GA-5/8/15 Scaffold = **🟡** (field-intent), not ✅.  
**Pilot-ready:** F-GA-5/8/15, Manage OS-block (AC-E4), WHQL, HSM, Guacamole video — **remain blocked / ⏸**. No false Pilot `[x]`.

## Waves

| Wave | Scope | Status |
|------|-------|--------|
| 1 | `ERA_RBAC_TRUST` + requireManage admin + nginx Trusted-Proxy + spoof→403 | ✅ |
| 2 | `GateFromEnv` fail-closed (ai-core/soar/perimeter/resolve/pam) | ✅ |
| 3 | `BlockResult.effect=telemetry_only` + plugin labels + Enforcement/Product-Line | ✅ |
| 4 | Collectors BYO/DNS redact-or-false + `mode=stub` + enforce envelope flag | ✅ |
| 5 | Control-Implementation-Matrix + ADR-0022/0006/0009/0010/0012 + Roadmap PP-1 | ✅ |
| 6 | Field prep (sizing/coverage/WHQL/PAM rows); Pilot columns stay blocked | ✅ |

## Acceptance closed (Scaffold)

| AC | Result |
|----|--------|
| AC-E1 monitor deny → detection + would_block | ✅ |
| AC-E2 enforce deny → blocked + effect=telemetry_only | ✅ |
| AC-E3 spoof/unlicensed write → 403 | ✅ |
| AC-E4 kernel OS block | ⏸ WHQL |
| AuthZ spoof admin / era-agent | ✅ |
| License module missing → 403 | ✅ |
| PII BYO/DNS redact-then-flag | ✅ |
| PP-1 Sigma→MITRE on alert | ✅ |
| F-GA-5/8/15 soft scripts/API/template | **🟡** field-intent · Pilot **[blocked: field]** |

## Gate commands (evidence)

```powershell
go test ./services/control-plane/internal/rbac/... ./services/control-plane/internal/api/...
go test ./services/platform/licensegate/...
cargo test -p era-agent-core enforce::
cargo test -p era-plugin-appcontrol -p era-plugin-devicecontrol
cargo test -p era-collectors
go test ./services/detection-engine/internal/processor/... -run Mitre
```

Targeted packages PASS in Scaffold-Green session (2026-07-30). Full `ci-gates-stage10.ps1` recommended before merge.

## Explicit non-claims

- Do **not** mark F-GA-5/8/15 Pilot `[x]` without field evidence / customer signature.
- Do **not** claim Manage OS kill / USB hardware deny / WHQL kernel as ✅.
- Enforce `blocked=true` is a **decision/telemetry** flag with **`effect=telemetry_only`**.

## Artifacts updated

- `docs/Control-Implementation-Matrix.md` (new)
- `docs/Enforcement-Spec.md` (AC-E1…E4)
- `docs/Control-Sprint-Index.md` §3
- `docs/ADR-Implementation-Matrix.md` · `docs/adr/0022-...`
- `docs/distributor/ERA-Product-Line.md` §4 Manage honesty
- `docs/Field-Server-Sizing.md` · `docs/Resolve-Spec.md` · `docs/PAM-Spec.md`
- `reports/control-scaffold-green-signoff.md` (this file)
