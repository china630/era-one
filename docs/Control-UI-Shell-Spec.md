# ERA Control — UI Shell Spec (P0–P4)

**Дата:** 4 августа 2026 г.  
**Статус:** Implemented (lab usable)  
**Канон:** [`ERA-Product-Acceptance-Standard.md`](products/ERA-Product-Acceptance-Standard.md) v1.3  
**Readiness:** [`Control-Product-Readiness-Matrix.md`](Control-Product-Readiness-Matrix.md)

## Архитектура

- Shared chrome: [`ui/control-shell/web/`](../ui/control-shell/web/) → CP `/control-assets/`
- Modules: [`ui/control/{soc,workbench,manage,...}/`](../ui/control/)
- Same-origin BFF: `/api/x/{svc}/...` → `ERA_*_URL` ([`proxy.go`](../services/control-plane/internal/api/proxy.go))
- AuthZ: viewer read; analyst+ mutate; Manage writes still admin where required
- **Design tokens:** `--era-*` canon with legacy aliases (`--bg`→`--era-bg`, `--accent`→`--era-accent`, …). Body gets `data-line="control"`. Cross-line SSOT: [`ERA-UI-Shell-Theme-Matrix.md`](ERA-UI-Shell-Theme-Matrix.md).

## Modules (usable lab)

| Module | Path | Backend |
|--------|------|---------|
| SOC Home | `/ui/control/` | CP assets/cases |
| Workbench | `/ui/control/workbench/` | case-bundle, exposure |
| Manage | `/ui/control/manage/` | enforcement, deploy, escrow |
| AI | `/ui/control/ai/` | `/api/x/ai/` |
| Response | `/ui/control/response/` | `/api/x/soar/` |
| Vuln | `/ui/control/vuln/` | `/api/x/vm/` |
| Service | `/ui/control/service/` | `/api/x/service/` |
| Provision | `/ui/control/provision/` | `/api/x/provision/` |
| PAM | `/ui/control/pam/` | `/api/x/pam/` |
| Observe | `/ui/control/observe/` | `/api/x/observe/` |
| Perimeter | `/ui/control/perimeter/` | waf+ngfw |
| Resolve | `/ui/control/resolve/` | `/api/x/resolve/` |
| BYO | `/ui/control/byo/` | CP connectors |

## Non-claims

UI usable lab ≠ Pilot field. WHQL / HSM / Guacamole video / 10k remain ⏸.

## Smoke

```powershell
go test ./services/control-plane/internal/api/... -run "Shell|EditionProxy|AssetByID|BYO|ControlRedirect"
# optional: npx playwright test ui/control/e2e
```
