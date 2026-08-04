# ERA Office — Implementation Matrix (прослеживаемость)

**Дата:** 31 июля 2026 г. (Projects `.eraj` Drive bind + docs import unique Drive name)  
**Назначение:** PRD AC-O* → код → тест → **Scaffold BE only**.  
**Готовность продукта (UI/TE/Pilot):** [`Office-Product-Readiness-Matrix.md`](Office-Product-Readiness-Matrix.md) — канон v1.3.  
**Система:** [`Office-Acceptance-System.md`](products/Office-Acceptance-System.md)  
**Канон:** [`ERA-Product-Acceptance-Standard.md`](products/ERA-Product-Acceptance-Standard.md)  
**Evidence:** [`Office-Evidence-Rules.md`](Office-Evidence-Rules.md)  
**Индекс:** [`Office-Sprint-Index.md`](Office-Sprint-Index.md)

**Легенда:** ✅ готово (proof = PRD wording + negative path) · 🟡 частично · [ ] нет · ⏸ поле · **Scaffold** ≠ **Pilot-ready**

**Правило Scaffold ✅:** spoof → 401/403; expire JWT rejected; tenant bind; stub явный. Иначе — 🟡.

**P0 fix 2026-07-30 (later):** engines `validate_exp=true` + expired→401; REST tenant bind (docs/tables/presentations); AC-O8 `verify-intent`; UTF-8-safe OpLog + insert OT lite (`merge_concurrent`); prod JWT secret + `ERA_ENV=production` fail-closed.

**2026-07-31 engines harden:** `drive_bind` fail-closed without `ERA_DRIVE_SERVICE_TOKEN` (docs/tables/presentations); tables WS debounce `put_erat`; presentations `put_deck`→`put_erap`; compose workspace URLs + `--profile office-engines`.

---

## Сводка по изданиям (snapshot)

| Издание | PRD AC | Scaffold | Pilot-ready | Edition |
|---------|--------|----------|-------------|---------|
| **ERA Drive** | AC-O3 | ✅ JWT-only + spoof 401 (lab) | [ ] RT-O09 | `mvp` (not ga) |
| **ERA Documents** | AC-O1…O8 | **🟡** rollup (O1 lite ✅; O3 PutVersion + `edit_snapshot_reopen_same_id_content` ✅ lab; O2/O5–O8 ✅) | [ ] RT-O09 | `mvp` (not ga) |
| **ERA Tables** | AC-T1…T8 | 🟡 goldens/calc ✅; JWT/token fail-closed ✅; WS debounce PutVersion + `ws_edit_flush_reopen_same_id_content` ✅ lab | [ ] field | `mvp` (not ga) |
| **ERA Presentations** | AC-P1…P5 | 🟡 JWT/token fail-closed ✅; put_deck→PutVersion + reopen same id ✅ lab | [ ] field | `mvp` (not ga) |
| **ERA Projects** | AC-PR1…PR4 | 🟡 CRUD/JWT + `.eraj` create/flush/reopen unit; soft defaults | [ ] field | `mvp` (not ga) |
| **ERA Office AI** | AC-AI1…AI3 | ✅ stub mode + JWT/license deny (lab) | [ ] field | `mvp` (not ga) |

*Not* a full-green Scaffold rollup (канон v1.2 §3.4). Edition Drive ✅ lab; Documents/Tables/Presentations/Projects rollup 🟡. Pilot-ready / RT-O09 / `ga` remain open.

---

## PRD AC-O* → код → тест

### AC-O1 — Co-edit 2 users

| Компонент | Путь | Proof | Scaffold | Pilot-ready | Note |
|-----------|------|-------|----------|-------------|------|
| docs-engine WS + JWT | `docs-engine` | `ws_coedit` + `concurrent_inserts_same_offset_both_appear` + mid-utf8 | ✅ | [ ] | Char-safe OpLog + insert OT lite (`transform_against`/`merge_concurrent`); **не** full CRDT |
| unauth WS | `ws_coedit_unauth_rejected` | cargo | ✅ | [ ] | 401/reject |

### AC-O2 — docx golden

| Компонент | Путь | Proof | Scaffold | Pilot-ready | Note |
|-----------|------|-------|----------|-------------|------|
| golden corpus | `docs-engine` | `golden_docx` + corpus | ✅ | [ ] | |

### AC-O3 — Authoritative only in Drive + AuthZ

| Компонент | Путь | Proof | Scaffold | Pilot-ready | Note |
|-----------|------|-------|----------|-------------|------|
| drive JWT-only | `drive/api` | `TestSpoofHeadersRejected`, `TestJWTRequired` | ✅ | [ ] | Client X-ERA-* rejected |
| service bind | engines `drive_bind` | `TestServiceTokenActingAs` + `empty_service_token_fails_closed` / `authorization_bearer_set_when_ok` + `put_*_version_stable_id` + `TestPutVersionStableID` / `TestDriveAPIPutVersionStableID` | ✅ | [ ] | Fail-closed token ✅; `POST …/objects/{id}/versions` PutVersion ✅ lab |
| attachment CanRead | `handleAttachmentLink` | unit | ✅ | [ ] | |

### AC-O4 — Workspace login → Drive → open doc

| Компонент | Путь | Proof | Scaffold | Pilot-ready | Note |
|-----------|------|-------|----------|-------------|------|
| workspace edge JWT | `workspace` | `TestWorkspaceAPIAuthRequired` | ✅ | [ ] | Strips spoof headers |
| docs JWT REST | `docs-engine` | `create_without_jwt_is_401`, `create_with_expired_jwt_is_401`, `get_doc_rejects_cross_tenant` | ✅ | [ ] | `validate_exp=true`; REST tenant bind |

### AC-O5 — Zero GPL

| Компонент | Путь | Proof | Scaffold | Pilot-ready | Note |
|-----------|------|-------|----------|-------------|------|
| SBOM gate | `office-sbom-gate.ps1` | O-5 / O-AC | ✅ | [ ] | |

### AC-O6 — Fuzz docx

| Компонент | Путь | Proof | Scaffold | Pilot-ready | Note |
|-----------|------|-------|----------|-------------|------|
| fuzz smoke | docs-engine | CI | ✅ | [ ] | Smoke; expand corpus before Pilot |

### AC-O7 — Без office-documents → 403

| Компонент | Путь | Proof | Scaffold | Pilot-ready | Note |
|-----------|------|-------|----------|-------------|------|
| HTTP deny | `create_without_license_is_403`, `license_denied_*`, `jwt_secret_empty_in_production_*` | cargo | ✅ | [ ] | Fail-closed under STRICT/PRODUCTION/`ERA_ENV=production`; lab `ERA_OFFICE_DEV` only outside; default JWT secret → empty→401 in prod |

### AC-O8 — Comms deep link

| Компонент | Путь | Proof | Scaffold | Pilot-ready | Note |
|-----------|------|-------|----------|-------------|------|
| signed EditLink + BFF | `ui/mail` + `docs-engine` + `ui/docs` | `TestDocuments*` / `TestVerifyIntent`; `verify_intent_ok_and_bad_sig`; SPA `verifyCommsIntent` | ✅ | [ ] | HMAC sign (mail) + server `GET …/verify-intent` + SPA gate; Pilot field RT still open |

---

## AC-T1…T8 (ERA Tables)

| AC | Proof | Scaffold | Note |
|----|-------|----------|------|
| AC-T1 create/Drive/reopen | get_table Drive reload + ws create; `ws_edit_flushes_to_drive`; `ws_edit_flush_reopen_same_id_content`; `get_table_rejects_cross_tenant`; expired JWT 401 | ✅ | JWT/tenant ✅; WS→PutVersion flush ✅; reopen same id content ✅ lab (field RT open) |
| AC-T2 SUM | `calc_*` | ✅ | |
| AC-T3 xlsx golden | `golden_xlsx` | ✅ | |
| AC-T4 2-client cells | `ws_sheet_coedit` + JWT | 🟡 | Fan-out co-edit; not OT |
| AC-T5 Drive-only | drive_bind fail-closed + WS debounce `put_erat(…, Some(id))` → `/versions` | ✅ | Token required ✅; flush PutVersion ✅ lab |
| AC-T6 Zero GPL | SBOM | ✅ | |
| AC-T7 license 403 | `tables_create_without_license_403`, `license_denied_when_era_env_production_*` | ✅ | Fail-closed under prod-like env; lab DEV only outside |
| AC-T8 fuzz xlsx | `fuzz_xlsx_smoke` | ✅ | |

---

## AC-P1…P5 (ERA Presentations)

| AC | Criterion | Proof | Scaffold | Note |
|----|-----------|-------|----------|------|
| AC-P1 | create `.erap` → Drive → reopen | Drive bind + `put_deck_persists_to_drive` (PutVersion + reopen) + tenant reject tests | ✅ | put_deck→`/versions` ✅; reopen same id content ✅ lab (field RT open) |
| AC-P2 | pptx golden subset | `golden_pptx_*` | ✅ | |
| AC-P3 | UI `/presentations` | `go test ui/presentations` | ✅ | |
| AC-P4 | license deny 403 | `presentations_*_403`, `license_denied_when_era_env_production_*` | ✅ | Fail-closed under prod-like env |
| AC-P5 | JWT required + exp | `*_without_jwt_401`, `presentations_create_with_expired_jwt_401`, `jwt_secret_empty_in_production_*` | ✅ | `validate_exp=true`; prod default secret → 401 |

---

## AC-PR1…PR4 (ERA Projects)

| AC | Criterion | Proof | Scaffold | Note |
|----|-----------|-------|----------|------|
| AC-PR1 | task CRUD | `TestProjectsCreateWithDeepLink` | ✅ | |
| AC-PR2 | Postgres or memory + migration | store + migrate | ✅ | legacy tenant board; `.eraj` also Drive JSON |
| AC-PR3 | Drive deep-link field | `drive_object_id` on tasks | ✅ | |
| AC-PR3b | Native `.eraj` in Drive | `TestErajCreateFlushReopenSameID` + MIME `application/vnd.era.eraj` | ✅ lab | proto `DRIVE_CONTENT_FORMAT_ERAJ` / `DOCUMENT_FORMAT_ERAJ`; New project in Drive UI |
| AC-PR4 | JWT + license 401/403 | `TestProjectsWithout*` | 🟡 | Unit OK; soft defaults / staging identity |

### Docs import (Drive name)

| Item | Proof | Scaffold | Note |
|------|-------|----------|------|
| Unique `.erad` name on import (no fixed `import.erad`) | `import_erad_names_are_unique_and_not_fixed` + TE connectivity import×2 | ✅ lab | 409→mapped Conflict; UI clearer errors |

---

## AC-AI1…AI3 (ERA Office AI)

| AC | Criterion | Proof | Scaffold | Note |
|----|-----------|-------|----------|------|
| AC-AI1 | summarize `mode=stub` without Ollama | `TestDocsAIStubModeNoPhoneHome` | ✅ | |
| AC-AI2 | license/JWT 401/403 | `TestDocsAIWithout*` | ✅ | |
| AC-AI3 | optional in-contour Ollama only | `ERA_OLLAMA_URL` allowlist | ✅ | |

---

## Residual debt (не закрывать ✅ пока)

| ID | Residual | Where | Status |
|----|----------|-------|--------|
| R-O1 | `validate_exp = false` | engines `auth.rs` | **closed** — `validate_exp=true` + expired→401 |
| R-O2 | REST get_doc/get_table без tenant check | engine `server.rs` | **closed** — cross-tenant → 403 |
| R-O3 | OpLog ≠ OT; UTF-8 byte offsets | `docs-engine/src/sync.rs` | **closed (lite)** — UTF-8 char-boundary + insert OT / DeleteRange×Insert; full CRDT still out of scope (PRD AC-O1 narrowed) |
| R-O4 | Soft license / shared `dev-only-change-in-prod` JWT | engines `auth.rs` + `license_from_env` | **closed** — prod-like (`ERA_PRODUCTION`/`STRICT`/`ERA_ENV=production`/`ERA_ENV_PRODUCTION`) rejects default JWT secret (empty→401) and ignores `ERA_OFFICE_DEV` |
| R-O5 | AC-O8 VerifyIntent not in docs UI | `ui/docs` | **closed** — SPA + `/verify-intent` |

## Как обновлять

1. Gate PASS → лог в `reports/`  
2. Scaffold ✅ только с negative path **и** без residual Critical в том же AC  
3. Pilot-ready / RT-O09 / `ga` — field only
