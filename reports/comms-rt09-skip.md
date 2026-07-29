# RT-09 Field Sign-off — SKIP (2026-07-29)

**Wave:** C-GA / P3-FIELD  
**Result:** SKIP  
**Reason:** No customer/air-gap field host available in this environment.

## Preconditions (scaffold) — PASS

- Stash restore committed (`b1d066f`)
- Stage gates C-1…C-5, C-MIG, C-MM-H PASS (see `reports/comms-stage-*-20260729-*.log`)
- Pilot scripts restored: `scripts/run-comms-pilot-field.ps1`, `run-comms-pilot-staging.ps1`

## Blocked until

1. Staging compose with PG+MinIO+CH (`run-comms-pilot-staging.ps1` → RT-01…08 logs)
2. Customer site air-gap + Outlook/Thunderbird matrix
3. Signed checklist in [`Comms-Customer-Field-RT09.md`](../docs/Comms-Customer-Field-RT09.md)

## Edition impact

Mail Server / Migration / Bridge / Moderation remain **`mvp`** until RT-09 or partner field sign-off.  
Do **not** set `ga` from this SKIP.
