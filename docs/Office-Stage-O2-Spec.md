# ERA Office — Stage O-2 (Workspace + Identity)

**Wave:** O-2  
**Версия:** 1.0  
**Дата:** 30 июля 2026 г.  
**Продукт:** ERA Office / Shared platform  
**PRD:** [`PRD-Office-MVP.md`](products/PRD-Office-MVP.md) · AC-O4 Scaffold  
**Предусловие:** Waves **O-0** и **O-1** gate = PASS

---

## 1. Цель этапа

> Login через identity-api (OIDC/staging token) → Workspace `/drive` → список файлов Drive.

Без open `.erad` / docs-engine (это **O-3**).

## 2. Scope

### Входит

- `services/platform/internal/oidc` + `cmd/identity-api`
- Workspace BFF: `/drive` SPA, proxy `/api/v1/drive/`, proxy `/oauth2/` → identity
- Drive UI: login form → `era_token` → list children
- Compose O-2: identity + drive + workspace; docs-engine — profile `docs`
- Gate O-2 Required checks

### НЕ входит

- Documents editor / docs-engine product (O-3)
- Co-edit, docx, SBOM (O-4/O-5)
- Pilot-ready / edition `mvp`

## 3. E2E-сценарий приёмки

1. `go test -C services/platform ./internal/oidc/...`
2. `go test -C services/platform ./workspace/... ./cmd/workspace/...`
3. `go test -C ui/drive ./...`
4. `docker compose -f deploy/docker-compose.office.yml config`
5. `.\scripts\run-office-stage-gate.ps1 -Stage O-2 -WriteSignoff`

## 4. Критерии приёмки

| ID | Критерий | PRD | Доказательство | Статус |
|----|----------|-----|----------------|--------|
| F-O2-1 | oidc + identity-api compile | AC-O4 | go test / go build | [x] |
| F-O2-2 | Workspace `/healthz` + `/drive` | AC-O4 | go test | [x] |
| F-O2-3 | Login → JWT → list Drive path | AC-O4 | ui/drive + oidc staging | [x] |
| F-O2-4 | Compose без обязательного docs-engine | — | compose config | [x] |
| F-O2-5 | Gate O-2 PASS | — | `reports/office-stage-O-2-20260730-002109.log` | [x] |

## 5. Backlog (OM2-*)

| ID | Задача | Статус |
|----|--------|--------|
| OM2-1 | Restore `internal/oidc` from stash | [x] |
| OM2-2 | Stage O-2 Spec + gate Required | [x] |
| OM2-3 | go test oidc + build identity-api | [x] |
| OM2-4 | Workspace drive + identity proxy | [x] |
| OM2-5 | Drive UI login → era_token | [x] |
| OM2-6 | Compose profile `docs` | [x] |
| OM2-7 | Dockerfiles present | [x] |
| OM2-8 | Gate O-2 PASS | [x] |
| OM2-9 | Matrix / Index / MVP-Spec | [x] |

## 6. Stage Gate

| # | Проверка | Доказательство |
|---|----------|----------------|
| G1 | `run-office-stage-gate.ps1 -Stage O-2` | PASS |
| G2 | optional e2e log | `reports/office-stage-O-2-e2e.log` |
| G3 | Matrix AC-O4 updated (copy rollup from Matrix) | Matrix SSOT · **not** false ✅ |
| G4 | Sprint-Index / MVP-Spec O-2 | `[x]` |
| G5 | — | — |
| G6 | signoff | `reports/office-stage-O-2-signoff.md` |

## 7. Связано

- [`Office-Sprint-Index.md`](Office-Sprint-Index.md)
- Legacy: [`Office-Stage-O3-Identity-Spec.md`](Office-Stage-O3-Identity-Spec.md), [`Office-Stage-O4-Workspace-Spec.md`](Office-Stage-O4-Workspace-Spec.md)
