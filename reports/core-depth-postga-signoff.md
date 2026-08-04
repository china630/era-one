# Core depth Post-GA — proof (2026-07-30)

DC-01…04 + AI-01…03 code DoD; Hybrid 3–4 / EPM deferred in docs.

## Tests

| Area | Command / artifact | Result |
|------|-------------------|--------|
| DC-01 MITRE | `go test ./services/detection-engine/internal/{sigma,processor}/` | PASS |
| DC-02 suppress | CP `TestSuppressionsCRUD` + `TestSuppressBlocksEmit` | PASS |
| DC-03 heatmap | `go test ./services/detection-engine/internal/mitre/` | PASS |
| DC-04 CVE | `go test ./services/vm/internal/cvefeed/` | PASS |
| AI forensic | `go test ./services/ai-core/internal/investigate/` | PASS |

## Docs

- Pre-Field DC/AI → `[x]`
- ADR matrix 0022/0023 updated
- [`Hybrid-Roadmap-3-4.md`](../docs/Hybrid-Roadmap-3-4.md)
- EPM-lite deferred in editions + Enforcement-Spec

## Remaining (explicit)

- Hybrid stage 3/4 + full TI-share (roadmap)
- EPM-lite/JIT (Manage P6)
- Agentic SOC (ADR-0023 Phase 3)
- WHQL / field GA gates (unchanged)
