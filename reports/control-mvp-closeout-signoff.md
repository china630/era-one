# Control MVP closeout — proof (2026-07-30)

Code DoD for Service · Provision · Manage (monitor) · PAM (RDP TCP) · Observe.

## Smoke / tests

| Cluster | Evidence | Result |
|---------|----------|--------|
| Service + Provision | `reports/itops-smoke-20260730-001741.log` | PASS |
| Observe Path B | `reports/observe-smoke-20260730-001751.log` | PASS |
| PAM RDP TCP | `reports/pam-rdp-closeout-20260729.log` + `go test ./services/pam/...` | PASS |
| Manage goldens | `cargo test` appcontrol/devicecontrol/bitlocker + `virtual_patch_monitor_emits_detection` | PASS (prior wave) |

## Spec / matrix updates

- `docs/Service-Provision-Spec.md` — server MVP ✅ / field ⏸
- `docs/Observe-Spec.md` — PollReal / HOST-RESOURCES / NetFlow UDP
- `docs/Enforcement-Spec.md` + `docs/ERA-Manage-WHQL-Program.md` — monitor-complete
- `docs/PAM-Spec.md` + `docs/PAM-RDP-Security-Review-Checklist.md` — RDP TCP MVP
- `docs/Pre-Field-Code-Backlog.md` — M-02 90%, P-03 TCP MVP, O-03 90%
- `docs/distributor/ERA-Product-Line.md` §2–§4
- `docs/ADR-Implementation-Matrix.md` — 0012/0013/0016/0020

## Explicit remaining gates (non-code)

- WHQL kernel enforce (Manage)
- Field PXE on customer iron (Provision)
- Field Service rollout
- RDP inject + graphical recording + HSM audit (PAM)
- Observe field lab / full NMS
