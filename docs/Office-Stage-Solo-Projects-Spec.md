# Office Stage — Solo Projects depth

**Targets: Browser ❌ · Solo ✅ · Corporate ❌**

**Status:** [x] 2026-08-03 — `.eraj` on disk; board/tasks persist via SoloProjectsState.

## Scope

| Piece | Path |
|-------|------|
| Pure core | `crates/era-projects-core` — JSON board+tasks, MIME `application/vnd.era.eraj` |
| Solo session | `apps/era-office-desktop/src-tauri/src/solo_projects.rs` |
| Bridge | `/api/v1/projects/*` → `SoloProjectsState` + dirty; Save flushes file |
| License module | `office-projects-solo` · demo ≤ 15 tasks (`DEMO_TASK_CAP`) |
| UI | `ui/projects` via `/projects/solo` |

## Acceptance

| ID | Criterion | Evidence |
|----|-----------|----------|
| AC-PRJ-1 | `.eraj` roundtrip; board/tasks survive restart | `solo_projects::tests::eraj_roundtrip`; `era-projects-core` tests |
| AC-PRJ-2 | Create/list/delete tasks via bridge | `solo_bridge::tests::projects_board_and_tasks` |
| AC-PRJ-3 | Open/Save File menu (`product=projects`) | boot menubar patch + `/api/v1/solo/file/*` |
| AC-TEST | `cargo test -p era-projects-core --lib` PASS | 3 tests |

## Out of scope

Postgres `docs-projects`, Drive MIME upload, comments sync, Store publish.
