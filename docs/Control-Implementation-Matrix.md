# ERA Control — Implementation Matrix (прослеживаемость)

**Дата:** 30 июля 2026 г. (SSOT product AC · канон v1.2)  
**Назначение:** PRD / Spec AC → код → proof → **Scaffold BE** · **Pilot-ready**.  
**Готовность продукта:** [`Control-Product-Readiness-Matrix.md`](Control-Product-Readiness-Matrix.md) — канон v1.3.  
**Система:** [`Control-Acceptance-System.md`](products/Control-Acceptance-System.md)  
**Канон:** [`ERA-Product-Acceptance-Standard.md`](products/ERA-Product-Acceptance-Standard.md) **v1.3**  
**Evidence:** [`Control-Evidence-Rules.md`](Control-Evidence-Rules.md)  
**Индекс:** [`Control-Sprint-Index.md`](Control-Sprint-Index.md)  
**Signoff (scaffold-gate only):** [`reports/control-scaffold-green-signoff.md`](../reports/control-scaffold-green-signoff.md) — **не** product-ready

**Легенда:** ✅ proof покрывает формулировку AC (вкл. negative path) · 🟡 код/тест есть, AC intent неполный · `[ ]` нет · ⏸ поле / external · **Scaffold ≠ Pilot-ready**

**Правило Scaffold ✅:** negative path обязателен (spoof → 403, deny → заявленный эффект или явный stub/effect label).

---

## Сводка по изданиям

| Издание | Spec AC | Scaffold | Pilot-ready | Note |
|---------|---------|----------|-------------|------|
| **Core / AI / Response** | F-GA | ✅ soft CI · F-GA-5/8/15 🟡 | **[blocked: field]** | UI ✅ usable lab shell |
| **Manage** | AC-E1…E3 | ✅ | [ ] field · **AC-E4 ⏸ WHQL** | UI ✅ manage module |
| **Service / Provision** | depth APIs | ✅ | ⏸ field / PXE | UI ✅ |
| **PAM** | sessions depth | ✅ | ⏸ Guacamole/HSM | UI ✅ |
| **Observe** | alerts/devices | ✅ | ⏸ NMS | UI ✅ |
| **Perimeter** | WAF CRUD | ✅ | ⏸ pen-test | UI ✅ (was none) |
| **Resolve** | rules/trace | ✅ | ⏸ field DNS | UI ✅ |
| **Vuln** | jobs/findings | ✅ | [~] | UI ✅ (was none) |
| **AuthZ / License / Shell** | trust + `/api/x` | ✅ | [ ] SSO | Control-UI-Shell-Spec |

---

## Core F-GA (soft)

| AC-ID | Criterion | Code | Proof | Scaffold | Pilot-ready |
|-------|-----------|------|-------|----------|-------------|
| F-GA-5 | ≥10k ev/s ×5 мин | `scripts/run-loadgen-prod.ps1` · [`Field-Server-Sizing.md`](Field-Server-Sizing.md) | soft script exists | **🟡** (field-intent; max soft) | **[blocked: field]** |
| F-GA-8 | Asset coverage ≥90% | CP `Store.AssetCoverage()` · `/api/v1/health` / hybrid health `coverage` | API unit | **🟡** (field-intent) | **[blocked: field]** |
| F-GA-15 | Pilot checklist signed | template in git | template | **🟡** (field-intent) | **[blocked: field]** подпись |

---

## Manage — Enforcement (AC-E*)

| AC-ID | Criterion | Code | Proof | Scaffold | Pilot-ready |
|-------|-----------|------|-------|----------|-------------|
| **AC-E1** | monitor deny → detection + `would_block` | `era-agent-core/enforce`, plugins | `cargo test -p era-agent-core enforce::` · plugin goldens | ✅ | [ ] field |
| **AC-E2** | enforce deny → detection + `blocked` + **`effect=telemetry_only`** | `BlockResult.effect`, plugins | golden `enforce_vs_monitor` · plugin `effect` | ✅ | [ ] field |
| **AC-E2b** | `ERA_ENFORCE_LIVE=1` → `allowed=false` + `user_land_block` | `enforce/engine.rs` + `user_land.rs` | golden `enforce_live_user_land` | ✅ | [ ] field |
| **AC-E3** | spoof / unlicensed write → 403; agent token ≠ admin | CP `rbac` + `enforcement.go` | `TestAgentTokenPolicyGetAllowedWritesForbidden` | ✅ | [ ] field |
| **AC-E4** | kernel OS block | WHQL minifilter | — | ⏸ WHQL | ⏸ WHQL |

Spec: [`Enforcement-Spec.md`](Enforcement-Spec.md) · WHQL: [`ERA-Manage-WHQL-Program.md`](ERA-Manage-WHQL-Program.md)

---

## AuthZ trust boundary

| AC-ID | Criterion | Code | Proof | Scaffold | Pilot-ready |
|-------|-----------|------|-------|----------|-------------|
| AuthZ-1 | `ERA_RBAC_TRUST` proxy/api_key; spoof admin → viewer/deny | `control-plane/internal/rbac` | `rbac_test.go` | ✅ | [ ] SSO field |
| AuthZ-2 | Manage writes require admin + ModuleManage | `requireManageAdmin` | `enforcement_test.go` | ✅ | [ ] |
| AuthZ-3 | Agent policy GET: `ERA_AGENT_TOKEN` / trusted proxy; **≠ RoleAdmin** | `rbac.go` + `enforcement.go` | agent GET 200 · PUT/escrow 403 | ✅ | [ ] |
| AuthZ-4 | Nginx strip + `X-ERA-Trusted-Proxy: 1` | `deploy/nginx/soc-portal.conf` | conf review | ✅ | [ ] |

---

## License fail-closed

| AC-ID | Criterion | Code | Proof | Scaffold | Pilot-ready |
|-------|-----------|------|-------|----------|-------------|
| Lic-1 | ai-core/soar/waf/ngfw/dlp/resolve/pam use `GateFromEnv` | mains + `licensegate` | `startup_test.go` · module missing → 403 | ✅ | [ ] |
| Lic-2 | `ERA_LICENSE_DEV=1` only outside production/strict (`ERA_ENV_PRODUCTION`/`ERA_ENV=production` too) | `StrictMode` + `GateFromEnv` | `TestStrictModeEnvSync` | ✅ | lab only |
| Lic-3 | HSM / prod key custody | PAM KMS abstraction | — | ⏸ external | ⏸ |

---

## Collectors PII (ADR-0009)

| AC-ID | Criterion | Code | Proof | Scaffold | Pilot-ready |
|-------|-----------|------|-------|----------|-------------|
| PII-BYO | BYO-EDR redact user/host/src_ip; no raw leak; flag after redact | `era-collectors/byo_edr.rs` | unit + golden bin | ✅ | [ ] |
| PII-DNS | DnsEvent query/answers redacted; `environment=mode=stub` | `era-collectors/dns.rs` | unit | ✅ | [ ] |
| PII-ENF | enforce detection envelope does not claim sanitize before path | `enforce/envelope.rs` | `pii_sanitized=false` until orchestrator sanitize | ✅ | [ ] |

---

## Service / Provision / PAM / Observe

| AC-ID | Criterion | Code | Proof | Scaffold | Pilot-ready |
|-------|-----------|------|-------|----------|-------------|
| Svc-1 | Service desk / tickets lab | `services/service` | package tests | ✅ | ⏸ field |
| Prov-1 | Provision PXE / image lab | `services/provision` | package tests | ✅ | ⏸ field PXE |
| PAM-1 | SSH + RDP TCP + inject/session policy | `services/pam` | package tests | ✅ | ⏸ video/HSM |
| Obs-1 | Observe Path A+B | `services/observe` | package tests | ✅ | ⏸ NMS lab |

---

## Perimeter (AC-P*)

| AC-ID | Criterion | Code | Proof | Scaffold | Pilot-ready |
|-------|-----------|------|-------|----------|-------------|
| AC-P1 | WAF body / CRS-lite | `services/waf` | `go test ./services/waf/...` | ✅ | ⏸ pen-test |
| AC-P2 | NGFW policy API | `services/ngfw` | go test | ✅ | ⏸ |
| AC-P3 | Without `ERA_NGFW_APPLY` → noop apply_backend | `ngfw/internal/apply` | `apply_test.go` · healthz `apply_backend` | ✅ | lab |
| AC-P4 | Content-DLP | `services/dlp` | go test | ✅ | ⏸ |

---

## Resolve (AC-R*)

| AC-ID | Criterion | Code | Proof | Scaffold | Pilot-ready |
|-------|-----------|------|-------|----------|-------------|
| AC-R1 | Guard verdict | `services/resolve` | go test | ✅ | ⏸ field DNS |
| AC-R2 | DoH lab | `resolve/internal/doh` | go test | ✅ | ⏸ |
| AC-R3 | Atlas packs | `resolve/internal/atlas` | go test | ✅ | ⏸ live TI |
| AC-R4 | DnsEvent stub emitter | `era-collectors` dns | unit · `mode=stub` | ✅ | ⏸ ETW/WHQL |

---

## AI recommend (human-on-loop)

| AC-ID | Criterion | Code | Proof | Scaffold | Pilot-ready |
|-------|-----------|------|-------|----------|-------------|
| AI-1 | recommended_actions + confirm/reject | `services/ai-core` | go test | ✅ | [ ] field |
| AI-2 | Unlicensed module → 403 | GateFromEnv | `license_test.go` | ✅ | [ ] |

---

## Sigma → MITRE (ADR-0022 / PP-1)

| AC-ID | Criterion | Code | Proof | Scaffold | Pilot-ready |
|-------|-----------|------|-------|----------|-------------|
| PP-1 | Sigma tags → `mitre_techniques` on alert | `detection-engine` processor + sigma.Techniques | `processor_mitre_test.go` | ✅ | [ ] heatmap UI ⏸ |

---

## UI Shell + BE depth (P0–P4)

| AC-ID | Criterion | Code | Proof | Scaffold | Pilot-ready |
|-------|-----------|------|-------|----------|-------------|
| UI-SHELL | Control app-shell + `/api/x` BFF AuthZ | `ui/control-shell`, `proxy.go` | `shell_test.go` | ✅ usable lab | [ ] field SSO |
| BE-DEPTH | Edition list/detail/actions APIs | soar/ai/vm/pam/waf/ngfw/resolve/service/provision/observe | package tests | ✅ | ⏸ field |
| UI-MODS | All edition modules under `/ui/control/` | `ui/control/*` | Control-UI-Shell-Spec | ✅ no none/thin | [ ] |

---

## Gate commands

```powershell
go test ./services/control-plane/internal/rbac/... ./services/control-plane/internal/api/...
go test ./services/platform/licensegate/...
cargo test -p era-agent-core enforce::
cargo test -p era-plugin-appcontrol -p era-plugin-devicecontrol
cargo test -p era-collectors
go test ./services/waf/... ./services/ngfw/... ./services/resolve/...
go test ./services/detection-engine/internal/processor/... -run Mitre
.\scripts\ci-gates-stage10.ps1
```

**Запрещено в этом цикле:** F-GA-5/8/15 Pilot `[x]` · Manage OS-block ✅ · WHQL/HSM as done.
