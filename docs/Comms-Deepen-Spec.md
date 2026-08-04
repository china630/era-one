# ERA Comms — Deepen Spec (D0–D9 + DF)

**Дата:** 4 августа 2026 г.  
**Инвариант:** Scaffold ✅ ≠ Pilot field; **не `ga`** до RT-09.  
**SSOT:** [`Comms-Implementation-Matrix.md`](Comms-Implementation-Matrix.md) · [`Comms-Product-Readiness-Matrix.md`](Comms-Product-Readiness-Matrix.md) · [`Comms-Pilot-Gap-List.md`](Comms-Pilot-Gap-List.md)

| ID | Wave | Статус | Proof |
|----|------|--------|-------|
| D0-1 | Core status non-stub | [x] | `ERA_MAIL_CORE_ADDR` + admin `0.0.0.0:8152` |
| D0-2 | Drive attach JWT + deny | [x] | `/mail/api/drive/attachment-link` + deny test |
| D0-3 | readyz CH required | [x] | `TestReadyzAuditRequiredWithoutCH` |
| D0-4 | Matrix Pilot lab vs field | [x] | Implementation-Matrix columns |
| D1-* | Deploy/ops P0-30…35 + P0-42 | [x] | Gap [x] lab · [`reports/comms-deepen-d1-deploy.md`](../reports/comms-deepen-d1-deploy.md) · profile overlays · Runbook/Checklist |
| D2-* | Webmail OIDC deepen | [x] | RT-05 + PKCE `app.js` + Bearer · [`reports/comms-deepen-d2-oidc.md`](../reports/comms-deepen-d2-oidc.md) · AC-C2 ✅ lab |
| D3-* | Gov lab checklists | [x] | [`Comms-Gov-Lab-Checklist.md`](Comms-Gov-Lab-Checklist.md) + [`reports/comms-gov-lab-deepen-20260804.log`](../reports/comms-gov-lab-deepen-20260804.log) |
| D4-* | Connect | [x] | real IMAP + stub items_ok=0 · vault env lab limit · AC-C6 Pilot lab [x] · `run-comms-connect-lab.ps1` |
| D5-* | Migration | [x] lab | [`reports/comms-deepen-d5-migration.md`](../reports/comms-deepen-d5-migration.md) · RT-11 `run-comms-migration-live-imap.ps1` |
| D6-* | Bridge | [x] lab | [`reports/comms-deepen-d6-bridge.md`](../reports/comms-deepen-d6-bridge.md) · synthetic≠field |
| D7-* | MM | [x] lab | [`reports/comms-deepen-d7-mm.md`](../reports/comms-deepen-d7-mm.md) · IceWarp ⏸ |
| D8-* | Chat/Conf/AI | [x] lab | [`reports/comms-deepen-d8-roadmap.md`](../reports/comms-deepen-d8-roadmap.md) |
| D9-* | Staging re-gate | [x] | STAGING PASS · core `0.2.0` · [`reports/comms-deepen-d9-staging-20260804.log`](../reports/comms-deepen-d9-staging-20260804.log) |
| DF-* | Partner RT-09 / ga | ⏸ | [`reports/comms-rt09-skip.md`](../reports/comms-rt09-skip.md) |

## Partner gate (DF)

RT-09 SignOff, edition `ga`, IceWarp/Exchange field — **blocked until partner**.
