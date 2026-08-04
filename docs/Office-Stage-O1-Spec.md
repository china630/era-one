# ERA Office — Stage O-1 (ERA Drive)

**Wave:** O-1  
**Версия:** 1.0  
**Дата:** 30 июля 2026 г.  
**Продукт:** ERA Drive (shared platform)  
**PRD:** [`PRD-Office-MVP.md`](products/PRD-Office-MVP.md) · AC-O3  
**Предусловие:** Wave **O-0** gate = PASS

---

## 1. Цель этапа

Довести Drive до runnable vertical: Postgres metadata + MinIO blobs + `drive-api` + compose + license `platform-drive` + Mail attachment-link contract.

## 2. Scope

### Входит

- `deploy/postgres/migrations/platform/001_drive.sql`
- `services/platform/drive` + `cmd/drive-api`
- `deploy/docker-compose.office.yml` (postgres + minio + drive-api)
- License `OfficeDevGate` / `platform-drive`
- Contract: `ui/mail` `POST /api/v1/drive/links/attachment`
- HTTP ↔ proto message alignment

### НЕ входит

- identity-api OIDC binary (→ O-2)
- Workspace UI shell (→ O-2)
- Documents / co-edit / docx (→ O-3+)

## 3. E2E-сценарий приёмки

1. `go test ./services/platform/drive/... -count=1`
2. `go test ./ui/mail/... -run Drive -count=1`
3. `docker compose -f deploy/docker-compose.office.yml config`
4. `.\scripts\run-office-stage-gate.ps1 -Stage O-1 -WriteSignoff`

## 4. Критерии приёмки

| ID | Критерий | PRD | Доказательство | Статус |
|----|----------|-----|----------------|--------|
| F-O1-1 | SQL migration era_platform Drive | AC-O3 | file + apply script | [x] |
| F-O1-2 | drive-api binary + unit/API tests | AC-O3 | `go test -C services/platform ./drive/...` | [x] |
| F-O1-3 | Compose office config valid | — | `docker compose … config` | [x] |
| F-O1-4 | Mail attach link contract | AC-O3 hook | `go test -C ui/mail ./... -run Drive` | [x] |
| F-O1-5 | License platform-drive | — | licensegate tests | [x] |
| F-O1-6 | editions-shared era-drive scaffold | — | yaml | [x] |

## 5. Backlog (OM1-*)

| ID | Задача | Статус |
|----|--------|--------|
| OM1-1 | SQL migration | [x] |
| OM1-2 | cmd/drive-api | [x] |
| OM1-3 | HTTP ↔ proto align | [x] messages + DriveService |
| OM1-4 | docker-compose.office.yml | [x] |
| OM1-5 | deploy/profiles/office.yaml | [x] |
| OM1-6 | Mail drive_client contract test | [x] |
| OM1-7 | Gate O-1 + matrix AC-O3 Scaffold | [x] |
| OM1-8 | Auth X-ERA + JWT (OIDC → O-2) | [x] |

## 6. Stage Gate

| # | Проверка | Доказательство |
|---|----------|----------------|
| G1 | `run-office-stage-gate.ps1 -Stage O-1` | PASS |
| G2 | optional e2e log | `reports/office-stage-O-1-e2e.log` |
| G3 | Matrix AC-O3 updated (rollup 🟡 if service-bind residual) | Matrix SSOT · **not** «all ✅» |
| G4 | Sprint-Index / MVP-Spec O-1 | `[x]` |
| G5 | editions-shared `era-drive` → scaffold | yaml |
| G6 | signoff | `reports/office-stage-O-1-signoff.md` |

## 7. Связано

- [`Office-Stage-O0-Spec.md`](Office-Stage-O0-Spec.md)
- Legacy detail (stash): [`Office-Stage-O2-Drive-Spec.md`](Office-Stage-O2-Drive-Spec.md)
