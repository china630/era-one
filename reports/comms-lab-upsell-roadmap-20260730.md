# Comms Lab / Upsell / Roadmap — evidence (2026-07-30)

Invariant: **no edition `ga`**. Desktop `exists: false`.

| ID | Result | Log / proof |
|----|--------|-------------|
| L-1 | PASS | `reports/comms-lab-imap-smoke.log` |
| L-2 | PASS | `reports/comms-pilot-staging.log` (no RT-06 fallback; ProdProfile SSL on) |
| L-3 | PASS | chat store unit + `era-chat` compose with PG DSN |
| L-4 | PASS | `TestOllamaAvailableTrue` + `deploy/docker-compose.comms.ai.yml` |
| L-5 | PASS | `reports/comms-meet-smoke.log` |
| L-6 | PASS | `reports/comms-scale-60k-*.log` |
| U-MIG | PASS | `reports/comms-migration-live-imap-*.log` (CH mailbox > 0) |
| U-CONN | PASS | `reports/comms-connect-lab.log` (RT-10 lab items_ok=1) |
| U-BR | PASS | `reports/comms-bridge-100mb-lab.log` (100/100) |
| U-MM | PASS | `TestEngine_Escalate`, `TestDLPHandoffStub` |
| R-1..3 | mvp | `editions-comms.yaml` Chat/Conf/AI → mvp (not ga) |
| R-4 | keep | desktop `exists: false` |

SSOT: `docs/Comms-Lab-Upsell-Roadmap-Spec.md`
