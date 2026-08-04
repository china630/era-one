# Perimeter + Resolve MVP — proof (2026-07-30)

Code DoD for ERA Perimeter and ERA Resolve (ADR-0031).

## Smoke / tests

| Cluster | Evidence | Result |
|---------|----------|--------|
| Perimeter | `reports/perimeter-smoke-20260730-002908.log` + `go test` waf/ngfw/dlp | PASS |
| Resolve | `reports/resolve-smoke-20260730-002925.log` + `go test ./services/resolve/...` | PASS |

## Artifacts

- ADR-0031, PRD-ERA-Perimeter, PRD-ERA-Resolve
- Perimeter-Spec, Resolve-Spec
- editions-control: `era-perimeter` / `era-resolve` → mvp
- Product Line #14/#15 + §4
- Datasheets 15/16 rewritten (Resolve = DNS DDR, not ITSM)
- Pre-Field PE-* / RS-*

## Explicit remaining gates

- Field/pen-test WAF; packet NGFW; content-DLP
- Field DNS :53 lab; DoH/DoT; live commercial TI; agent DNS collector
