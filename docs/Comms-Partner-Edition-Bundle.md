# ERA Communications — Partner Edition Bundle (Phase 4)

RFQ-ready package: **ERA Comms Migration** + **ERA Outlook Bridge** + cutover playbooks.  
Опционально (roadmap): **ERA Mail Moderation** — outbound Approve/Reject без смены MTA.

## SKUs

| Edition | License module | Status (SSOT = [`editions-comms.yaml`](../editions-comms.yaml)) |
|---------|----------------|--------|
| era-comms-migration | comms-migration | **mvp** (Pilot-ready / partner field open — not ga) |
| era-outlook-bridge | comms-outlook-bridge | **mvp** (not ga) |
| era-mail-moderation | comms-mail-moderation | **mvp** in yaml · PRD upsell — [`PRD-Mail-Moderation.md`](products/PRD-Mail-Moderation.md) |
| era-mail-server | comms-mail-server | **mvp** (RT-09 SKIP — not ga; канон v1.2 §3.3) |

Pricing SSOT: `docs/distributor/pricing-comms-data.yaml`  
Bundles: `comms-partner-migration` · `comms-partner-hardening` (+ Moderation) в [`editions-comms.yaml`](../editions-comms.yaml).

## Bundle contents

1. Migration API + worker farm (`ERA_MIG_SCALE=1`, 200 workers)
2. Outlook Bridge with IceWarp / CG / Exchange adapters
3. *(roadmap)* Mail Moderation SMTP edge — hold → manager/curator Approve/Reject ([PRD](products/PRD-Mail-Moderation.md))
4. Runbooks:
   - `Comms-Bridge-100-Mailbox-Runbook.md`
   - `Comms-Cutover-Rehearsal-Runbook.md`
   - `Comms-Migration-1k-Pilot.md`
   - `Comms-Upsell-IceWarp-to-ERA-Runbook.md`
5. Distributor cutover HTML: `docs/distributor/Comms-Partner-Cutover-Playbook.html`

## CI gates

- `go test ./services/comms/mail-bridge/... ./services/comms/migration/...`
- `.\scripts\comms-sbom-gate.ps1`
- `.\scripts\run-comms-scale-40k.ps1 -DryRun`

## Evidence

- `reports/comms-scale-40k-*.log` (partner soak)
- ADR-0030 **Implemented**
