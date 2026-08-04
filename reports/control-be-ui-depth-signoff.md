# Control BE + UI Depth — Signoff

**Дата:** 4 августа 2026 г.  
**План:** Control BE Depth + UI Shell (P0–P4)  
**Spec:** [`docs/Control-UI-Shell-Spec.md`](../docs/Control-UI-Shell-Spec.md)

## Verdict

**PASS (lab usable):** Control app-shell + edition modules; code-closable BE depth APIs; AuthZ on `/api/x` mutations.  
**Non-claims:** Pilot field F-GA-5/8/15, WHQL OS block, HSM, Guacamole video, live TI — remain ⏸.

## Waves

| Wave | Status |
|------|--------|
| P0 Shell + proxy AuthZ + SOC home | ✅ |
| P1 Core depth (asset/exposure/case-bundle/AI list) + workbench | ✅ |
| P2 Manage/PAM/SOAR/AI depth + UI | ✅ |
| P3 Vuln/Perimeter/Service/Provision/Observe/Resolve/BYO | ✅ |
| P4 Docs/matrix/playwright smoke + signoff | ✅ |

## Gate commands

```powershell
go test ./services/control-plane/internal/api/... ./services/control-plane/internal/rbac/...
go test ./services/soar/... ./services/ai-core/... ./services/vm/... ./services/pam/...
go test ./services/waf/... ./services/ngfw/... ./services/resolve/...
go test ./services/service-desk/... ./services/provision/... ./services/observe/...
```

## UI maturity (target)

All Control edition UIs under `/ui/control/` = **usable lab** (no none/thin).  
Readiness Pilot field columns unchanged (blocked/⏸).
