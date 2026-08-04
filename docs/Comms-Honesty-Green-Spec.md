# ERA Comms — Honesty → Green Spec (G0–G7)

**Дата:** 30 июля 2026 г.  
**Инвариант:** Scaffold ✅ = PRD AC + negative path; **не `ga`** до RT-09.  
**SSOT:** [`Comms-Implementation-Matrix.md`](Comms-Implementation-Matrix.md) · [`Comms-Pilot-Gap-List.md`](Comms-Pilot-Gap-List.md)

| ID | Wave | Статус | Proof |
|----|------|--------|-------|
| G0-1 | AuthZ mail-api REST | [x] | `httpauth` + `TestMailSendUnauthorizedWithoutDev` / `oidc_jwt_test` |
| G0-2 | Internal `/internal/v1/*` token | [x] | `RequireInternal` + remote_store `X-ERA-Internal-Token` |
| G0-3 | Connect AuthZ | [x] | `TestRegisterAndSyncUnauthorized` |
| G0-4 | Migration AuthZ | [x] | Register wraps jobs; DEV lab |
| G0-5 | MM force-release AuthZ | [x] | `handleForceRelease` Authenticate |
| G0-6 | Chat/AI JWT-or-DEV | [x] | spoof without DEV → 401 |
| G0-7 | Explicit `mode=` | [x] | Connect/Bridge/Meet/AI/Mig/DLP |
| G1-1 | PG default store | [x] | `ERA_MAIL_STORE=memory` opt-in; else DSN required |
| G1-2 | Mailbox durable | [x] | compose PG + `run-comms-restart-persist.ps1` |
| G1-3 | CalDAV/EWS persist | [x] | `caladapter` + repo; `TestCalendarSurvivesMemoryRepoRoundTrip` |
| G1-4 | Policy → SMTP/REST | [x] | `/internal/v1/mail/policy`; SMTP 552; REST 413 |
| G1-5 | CH audit required | [x] | `ERA_MAIL_AUDIT_REQUIRE=1` → `NewFromEnvStrict` |
| G1-6 | TLS honesty | [x] | IMAP `ERA_IMAP_INSECURE=1`; smtp_tls_e2e |
| G2-* | OIDC webmail | [x] lab | PKCE app.js + staging token RT-05 + Bearer forward |
| G3-* | Gov protocols | [x] lab | [`Comms-Gov-Lab-Checklist.md`](Comms-Gov-Lab-Checklist.md); field ⏸ |
| G4-* | Upsell Scaffold | [x] | Connect/Mig/Bridge/MM honesty |
| G5-* | Chat/Conf/AI | [x] | AuthZ + mode/degraded |
| G6-* | Staging Pilot-ready lab | [x] | `run-comms-pilot-staging.ps1` RT-01…08 (lab); AuthZ stack |
| G7-* | Partner RT-09 / ga | ⏸ | [`reports/comms-rt09-skip.md`](../reports/comms-rt09-skip.md) |

## Shared AuthZ

Package: [`services/comms/internal/httpauth`](../services/comms/internal/httpauth/)

- Prod: Bearer JWT (`ERA_IDENTITY_JWT_SECRET`) or `ERA_INTERNAL_TOKEN`
- Lab: `ERA_*_DEV=1` → mode=dev (logged)
- Spoof `X-ERA-Role` without JWT/DEV → 401

## Partner gate (G7)

RT-09 SignOff, edition `ga`, IceWarp/Exchange field — **blocked until partner**. Do not bump editions to ga from this wave.
