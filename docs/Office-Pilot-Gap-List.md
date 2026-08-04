# ERA Office — Gap-лист до реального пилота

**Версия:** 1.4  
**Дата:** 30 июля 2026 г.  
**Статус:** Active — INFRA/deploy + **AuthZ/OT honesty** (Matrix Scaffold 🟡)  
**Product backlog (Tables, UI):** [`Office-Tech-Eval-Gap-List.md`](Office-Tech-Eval-Gap-List.md) — **главный приоритет**  
**Связано:** [`Office-Pilot-Readiness-Checklist.md`](Office-Pilot-Readiness-Checklist.md) · [`Office-Implementation-Matrix.md`](Office-Implementation-Matrix.md) · [`ERA-Documents-vs-Word.md`](ERA-Documents-vs-Word.md) · [`products/PRD-Office-MVP.md`](products/PRD-Office-MVP.md) · [`adr/0026-sovereign-office-engine.md`](adr/0026-sovereign-office-engine.md) · canon [`Office-Stage-OGA-Spec.md`](Office-Stage-OGA-Spec.md)

---

## 1. Резюме

**O-0…O-5** = **gate[x]** (stage gates). **O-GA** = honesty closeout (editions `mvp`) — **не** field GA.  
**AC rollup SSOT:** [`Office-Implementation-Matrix.md`](Office-Implementation-Matrix.md) — Documents частично ✅ (O1 lite / O2/O5/O6/O7/O8); **AC-O3 service bind 🟡**; Tables/Presentations/Projects **🟡**; Pilot-ready / RT-O09 open. Шапки не пишут `all ✅` (канон v1.2 §3.4).

Это **не** эквивалент **field pilot** у заказчика (**RT-O09** остаётся открытым).

**Правило до пилота:** RT-O09 не стартует, пока не закрыты field items (**GAP-O-P0-40**, customer **RT-O08**). Lab staging log PASS — `office-pilot-staging-20260708-112522.log`.

| Контур | Издание | Код / gates | Field pilot |
|--------|---------|-------------|-------------|
| **P0** | ERA Drive + Workspace | O-0…O-5 + O-GA honesty | lab RT-O07 PASS; **RT-O09** open |
| **P1** | ERA Documents + `office-mvp` | O-0…O-5 + O-GA honesty | lab staging PASS; **RT-O09** open |
| **Не в pilot** | Tables, Presentations, Office AI | roadmap | — |

---

## 2. Текущее состояние vs «реальный продукт»

| Область | Сейчас (код) | Нужно для пилота |
|---------|--------------|------------------|
| Acceptance docs | O-GOV + P0/P1 waves ✅ | — |
| drive.proto + pgstore | ✅ `pgstore` + `OpenFromEnv` | RT-O07 field `-UseCompose` |
| platform-drive license | ✅ unit tests | prod compose без `ERA_OFFICE_DEV` |
| workspace BFF | ✅ `cmd/workspace` | OIDC Playwright e2e |
| ui/drive | ✅ embedded SPA | Playwright e2e |
| Mail→Drive / Documents | ✅ clients in `ui/mail` | full Office+Comms compose |
| office compose | ✅ volume + migrate + health | lab `up --wait` log |
| docs-engine | ✅ Rust `:8142` + `doc_sessions` PG + WS UI | field `-UseCompose` RT-O07 |
| Office MVP bundle | ✅ `editions-office.yaml` | field sign-off RT-O09 |
| SBOM zero GPL | CI job + `office-sbom-gate` | [x] O-PILOT gate PASS |

---

## 3. O-GOV — acceptance scaffold

| ID | Задача | Артефакт | Критерий |
|----|--------|----------|----------|
| GAP-O-GOV-01 | Acceptance-System | `docs/products/Office-Acceptance-System.md` · канон [`ERA-Product-Acceptance-Standard.md`](products/ERA-Product-Acceptance-Standard.md) | [x] Accepted v2.1 |
| GAP-O-GOV-02 | MVP-Spec + Sprint-Index | `Office-MVP-Spec.md`, `Office-Sprint-Index.md` | [x] |
| GAP-O-GOV-03 | Implementation-Matrix | `Office-Implementation-Matrix.md` | [x] |
| GAP-O-GOV-04 | Gap + Checklist + Runbook | `Office-Pilot-*.md` | [x] |
| GAP-O-GOV-05 | Stage specs | `Office-Stage-*.md` | [x] |
| GAP-O-GOV-06 | Gate scripts | `scripts/run-office-*.ps1` | [x] |
| GAP-O-GOV-07 | ADR-0026 wire types | `.erad`/`.erat`/`.erap`/`.eraj` | [x] |
| GAP-O-GOV-08 | ADR matrix link | `ADR-Implementation-Matrix.md` §0026 | [x] |
| GAP-O-GOV-09 | products README | `docs/products/README.md` | [x] |
| GAP-O-GOV-10 | CI office-pilot | `.github/workflows/ci.yml` | [x] migrations + go/rust integration |

---

## 4. P0 — блокеры пилота Drive + Workspace

### P0-1. Proto + schema

| ID | Задача | Модуль | Критерий готовности |
|----|--------|--------|---------------------|
| GAP-O-P0-01 | drive.proto + ACL golden | `proto/era/v1` | [x] gen + golden PASS |

### P0-2. ERA Drive service

| ID | Задача | Модуль | Критерий готовности |
|----|--------|--------|---------------------|
| GAP-O-P0-02 | drive-api persistent upload | `platform/drive/pgstore.go` | [x] `pgstore_integration_test` |
| GAP-O-P0-03 | Folders + versions | `platform/drive` | [x] integration + unit |
| GAP-O-P0-04 | ACL tenant isolation | `platform/drive` | [x] JWT-only + spoof 401 (O-AUTH) |

### P0-3. Identity + license

| ID | Задача | Модуль | Критерий готовности |
|----|--------|--------|---------------------|
| GAP-O-P0-10 | platform-drive license | `licensegate` | [x] `platform_drive_test.go` |
| GAP-O-P0-11 | tenant Postgres store | `platform/tenant/pgstore` | [x] integration + `/api/v1/tenants` |

### P0-4. Workspace + UI

| ID | Задача | Модуль | Критерий готовности |
|----|--------|--------|---------------------|
| GAP-O-P0-20 | Workspace OIDC shell | `cmd/workspace` | [x] Playwright `drive.spec.ts` + docs smoke (JWT inject) |
| GAP-O-P0-21 | admin-portal in compose | `cmd/admin-portal` | [x] healthz + tenants API |
| GAP-O-P0-22 | Drive UI upload/list | `ui/drive` | [x] Playwright `drive.spec.ts` PASS |

### P0-5. Comms integration

| ID | Задача | Модуль | Критерий готовности |
|----|--------|--------|---------------------|
| GAP-O-P0-24 | Mail attach link | `ui/mail` + drive | [~] unit tests; optional full compose |

### P0-6. Deploy и операции

| ID | Задача | Модуль | Критерий готовности |
|----|--------|--------|---------------------|
| GAP-O-P0-30 | office compose prod | `deploy/` | [x] health probes in compose |
| GAP-O-P0-31 | Postgres migrations platform | `scripts/office-apply-migrations.*` | [x] compose migrate + CI |
| GAP-O-P0-32 | Health/readiness probes | all office services | [x] compose healthchecks |
| GAP-O-P0-33 | Offline license activation | `licensegate` | [x] prod compose без `ERA_OFFICE_DEV`; lab smoke `office-prod-license-smoke-20260708-112733.log` (403) |
| GAP-O-P0-34 | Runbook пилота | `Office-Pilot-Runbook.md` | [x] backup + tenant bootstrap |
| GAP-O-P0-35 | Pilot checklist | `Office-Pilot-Readiness-Checklist.md` | [x] актуализирован |

### P0-7. Документация и статус (честность)

| ID | Задача | Файл | Критерий |
|----|--------|------|----------|
| GAP-O-P0-40 | Field pilot pending (O-GA honesty ≠ field) | `Office-Stage-OGA-Spec.md` | [ ] до RT-O09 |
| GAP-O-P0-41 | Матрица scaffold vs pilot-ready | `Office-Implementation-Matrix.md` | [x] honesty 2026-07-30 (AC 🟡 where proof ≠ PRD) |
| GAP-O-P0-42 | MVP-spec: no false field close | `Office-MVP-Spec.md` | [~] O-0…O-5 [x]; O-GA honesty; RT-O09 [ ] |

### P0-8. Postgres hardening (O-RT)

| ID | Задача | Модуль | Критерий готовности |
|----|--------|--------|---------------------|
| GAP-O-RT-PG-01 | Drive metadata в PG | `drive/pgstore.go` | [x] |
| GAP-O-RT-PG-02 | doc_sessions в PG | `docs-engine/persist.rs` | [x] |
| GAP-O-RT-PG-03 | Persistent volume | `docker-compose.office.yml` | [x] |
| GAP-O-RT-PG-04 | Tenant bootstrap dual-schema | `office-bootstrap-tenants.sql` | [x] |
| GAP-O-RT-PG-05 | RT-O07 restart roundtrip | `run-office-pilot-staging.ps1` | [x] `office-pilot-staging-20260708-112522.log` RT-O07+RT-O07b PASS |

---

## 5. P1 — блокеры Office MVP (Documents)

| ID | Задача | Модуль | Критерий |
|----|--------|--------|----------|
| GAP-O-P1-01 | office.proto erad wire | `proto/era/v1/office.proto` | [x] `proto_roundtrip` |
| GAP-O-P1-02 | docx import/export | `docs-engine/convert` | [x] `golden_docx` + corpus 6 fixtures |
| GAP-O-P1-03 | co-edit 2 users | docs-engine sync | [x] OpLog+WS JWT (AC-O1 narrowed ADR-0026) |
| GAP-O-P1-04 | Drive-only authoritative | docs-engine | [x] service token bind (O-AUTH) |
| GAP-O-P1-05 | office-documents license | licensegate | [x] HTTP 403 fail-closed |
| GAP-O-P1-06 | Comms «Edit in Documents» | ui/mail | [x] signed intent + UI button |
| GAP-O-P1-07 | fuzz docx parser | docs-engine/fuzz | [x] `fuzz_docx_import` + `fuzz_docx_smoke` CI (O-PILOT gate) |
| GAP-O-P1-08 | docs-engine in compose | deploy | [x] health + `ERA_OFFICE_DATABASE_URL` |
| GAP-O-P1-09 | doc_sessions Postgres | `docs-engine/persist.rs` | [x] `persist_tests` + CI |

**Позиционирование vs Word:** [`ERA-Documents-vs-Word.md`](ERA-Documents-vs-Word.md)

---

## 6. Field staging (RT-O*)

| ID | Сценарий | Скрипт | Статус |
|----|----------|--------|--------|
| RT-O01 | compose config valid | staging script | [x] |
| RT-O02 | identity healthz | staging | [x] script |
| RT-O03 | drive upload roundtrip | staging | [x] script |
| RT-O04 | ACL deny cross-tenant | staging | [x] script |
| RT-O05 | workspace healthz | staging | [x] `office-pilot-staging-20260708-112522.log` |
| RT-O06 | Mail attach link (licensed) | staging | [x] `office-pilot-staging-20260708-112522.log` |
| RT-O07 | restart persistence + docs smoke | `-UseCompose` | [x] `office-pilot-staging-20260708-112522.log` RT-O07+RT-O07b |
| RT-O08 | air-gap no outbound | customer verify | [x] lab compose scan PASS (`office-pilot-staging-20260708-113115.log`); field verify [ ] |
| RT-O09 | field sign-off | checklist + signature | [ ] |

**Правило:** RT-O09 — после customer sign-off; lab `run-office-pilot-staging.ps1 -UseCompose` PASS — `office-pilot-staging-20260708-112522.log`.

---

## 7. Команды

```powershell
.\scripts\office-apply-migrations.ps1
.\scripts\run-office-stage-gate.ps1 -Stage O-GOV
.\scripts\run-office-acceptance.ps1
.\scripts\run-office-pilot-staging.ps1
.\scripts\run-office-pilot-staging.ps1 -UseCompose   # RT-O07 restart
```
