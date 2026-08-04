# Office Stage — Solo Presentations depth

**Targets: Browser ❌ · Solo ✅ · Corporate ❌**

**Status:** [x] 2026-08-03 — shim removed; local `.erap` + pptx export/import + frame-op.

## Scope

| Piece | Path |
|-------|------|
| Pure core | `crates/era-pres-core` (model + pptx/odp convert) |
| Engine thin wrap | `services/platform/presentations-engine` → depends on core |
| Solo session | `apps/era-office-desktop/src-tauri/src/solo_pres.rs` |
| Bridge | `solo_bridge` — real `frame-op`, export pptx/odp, import pptx |
| License module | `office-pres-solo` · demo ≤ 5 slides (`DEMO_SLIDE_CAP`) |
| UI | `ui/presentations` via bridge `/presentations/solo` |

## Acceptance

| ID | Criterion | Evidence |
|----|-----------|----------|
| AC-PRE-1 | `.erap` roundtrip + pptx export; frame-op not stub | `solo_pres::tests::erap_roundtrip`; `era-pres-core` pptx tests; bridge `apply_frame_op` |
| AC-PRE-2 | Open/Save via File menu (product=`presentations`) | `solo-docs-boot.js` patches menubar; `/api/v1/solo/file/*?product=presentations` |
| AC-PRE-3 | Demo license gate | `license::status_pres` |
| AC-TEST | `cargo test -p era-pres-core --lib` + desktop lib PASS | 15 + 35 tests |

## Out of scope

Drive sync, multi-user WS, ODP import, Store publish.
