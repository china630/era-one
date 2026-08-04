# Hybrid — Roadmap ступени 3–4 и TI-share

**Статус:** Hybrid-0 ✅ · ступени 3–4 / full TI-share — **Roadmap**  
**Связано:** [ADR-0018](adr/0018-hybrid-connected-operating-model.md) · [Hybrid-0-Spec.md](Hybrid-0-Spec.md)

## Матрица зрелости

| Ступень | Смысл | Статус |
|---------|-------|--------|
| 1 Air-gap | default, без egress | GA |
| 2 Sovereign Hybrid | Hybrid-0: relay, lease, Update Service, CRL, egress audit | ✅ Implemented |
| 3 Managed private cloud | single-tenant K8s / ЦОД вендора | ⏸ Helm есть (`deploy/helm/era-one`); ops runbook / white-label — Roadmap |
| 4 Multi-tenant SaaS | demand-gated | ❌ отложено |

## TI-share

- Policy flag / helper: `services/control-plane/internal/hybrid/ti_outbound.go` (pseudonymize + audit) — **stub**, не полный Portal TI marketplace.
- Default: **OFF** (opt-in). Health B stub ≠ GA Health B/C.
- Prerequisites до GA TI-share: pen-test, customer contract, sealed TI packs, Portal ingest ACL.

## Explicit non-claims

Не продавать «Managed SaaS» или «live TI exchange» как GA до закрытия DoD ступени 3/4.
