# ERA Office — Pilot Readiness Checklist (honest)

> **Статус:** **INFRA / deploy** — lab staging RT-O01…O08 PASS.  
> **Product gate для дистрибьютора:** [`Office-Tech-Eval-Checklist.md`](Office-Tech-Eval-Checklist.md) (Tables TE-T — блокер).  
> `[x]` только с доказательством (тест / лог / CI).

**Заказчик:** ____________________  
**Дата пилота:** ____________________  
**Издание:** ERA Drive + Workspace (P0) / ERA Office MVP (P1)  
**Подпись ERA:** ____________________ · **Подпись заказчика:** ____________________

> **Staging:** `.\scripts\run-office-pilot-staging.ps1 -UseCompose` → `reports/office-pilot-staging-*.log`  
> **Runbook:** [`Office-Pilot-Runbook.md`](Office-Pilot-Runbook.md)  
> **Word vs ERA Documents:** [`ERA-Documents-vs-Word.md`](ERA-Documents-vs-Word.md)

---

## 1. Инфраструктура

- [x] `docker compose -f deploy/docker-compose.office.yml config` — valid (RT-O01 script)
- [x] `docker compose ... up --wait` — all healthy (lab: `office-pilot-staging-20260708-112522.log` RT-O01 PASS)
- [x] Postgres `platform` schema — `scripts/office-apply-migrations.ps1` (CI `office-pilot`)
- [x] Postgres persistent volume — `era_postgres_data` in compose
- [x] MinIO bucket `era-drive` — RT-O03/RT-O05c drive roundtrip PASS (staging log)
- [ ] Firewall: clients → HTTPS workspace (customer perimeter)
- [x] Air-gap compose scan — lab RT-O08 PASS (`office-pilot-staging-20260708-113115.log`; customer perimeter verify open)
- [x] License prod path — `docker-compose.office.prod.yml` (no `ERA_OFFICE_DEV`); lab smoke `reports/office-prod-license-smoke-20260708-112733.log` (drive/docs 403)

---

## 2. Сервисы (P0)

- [x] identity-api `GET /healthz` — PASS (`office-pilot-staging-20260708-112522.log` RT-O02)
- [x] drive-api `GET /healthz` — PASS (staging RT-O03/RT-O07)
- [x] workspace `GET /healthz` — PASS (staging RT-O05)
- [x] admin-portal `GET /healthz` — PASS (staging RT-O05b)
- [x] `platform-drive` license gate — `licensegate/platform_drive_test.go` + prod smoke 403
- [x] Upload → list → download — `pgstore_integration_test` + RT-O03 script
- [x] ACL cross-tenant deny — `pgstore_integration_test` + RT-O04 script
- [x] Drive metadata Postgres — `drive/pgstore.go` + `ERA_OFFICE_DATABASE_URL`

---

## 3. Workspace UI (P0)

- [x] OIDC login -> `/drive` - Playwright `drive.spec.ts` (JWT inject) PASS
- [x] Upload file via UI - Playwright `drive.spec.ts` PASS
- [x] Folder create + list - Playwright `drive.spec.ts` folder scenario PASS (UI); live compose optional

---

## 4. Comms integration (optional P0)

- [x] Mail attach → Drive deep link (licensed tenant) — staging RT-O06 PASS
- [ ] Comms-only tenant — no Drive attach button (full Comms+Office compose at customer)

---

## 5. ERA Documents (P1 — Office MVP)

- [x] Create `.erad` document — docs-engine API + `proto_roundtrip` + staging RT-O07b
- [x] Co-edit 2 users - `ws_coedit.rs` PASS + Playwright `docs-coedit.spec.ts` PASS
- [x] docx import/export golden - corpus 6 fixtures (`golden_docx_corpus`)
- [x] `office-documents` license gate — `office_documents_test.go` + prod smoke 403
- [x] «Edit in Documents» from mail — `documents_client_test.go`
- [x] doc_sessions Postgres persistence — `persist_tests` + compose DSN
- [x] Zero GPL SBOM - `office-sbom-gate` O-PILOT PASS (`reports/office-stage-O-PILOT-*.log`)

---

## 6. Sign-off

- [x] RT-O01…RT-O08 staging — `run-office-pilot-staging.ps1 -UseCompose` log RT-O07+RT-O07b PASS (`office-pilot-staging-20260708-112522.log`)
- [ ] RT-O09 field sign-off — customer signature
- [~] [`Office-Pilot-Gap-List.md`](Office-Pilot-Gap-List.md) — lab gaps closed; RT-O09 + customer RT-O08 field open

---

## 7. Документация

- [x] [`Office-Acceptance-System.md`](products/Office-Acceptance-System.md) — Accepted
- [x] [`Office-Implementation-Matrix.md`](Office-Implementation-Matrix.md) — Postgres hardening reflected
- [x] [`ERA-Documents-vs-Word.md`](ERA-Documents-vs-Word.md) — positioning reference
- [x] [`editions-shared.yaml`](../editions-shared.yaml) — `era-drive` `mvp` (unit/gate; **not** `ga`; RT-O09 open)
- [x] [`editions-office.yaml`](../editions-office.yaml) — `era-documents` `mvp` (unit/gate; **not** `ga`; RT-O09 open)
