# PRD: ERA Office — Projects

**Статус:** Scaffold rollup **🟡** (Matrix AC-PR4 soft defaults) · Wave **O-PR** · не ga  
**Дата:** 30 июля 2026 г.

## Scope MVP

- Board/task CRUD API (`docs-projects`)
- Postgres when `ERA_OFFICE_DATABASE_URL` (migration `005_projects.sql`); else memory
- UI `/projects` + deep-link `drive_object_id` to Drive docs
- License `office-projects` + JWT

## AC-PR1…PR4

| ID | Критерий | Proof |
|----|----------|-------|
| AC-PR1 | task CRUD | `TestProjectsCreateWithDeepLink` + `TestProjectsDeleteTask` + Playwright `projects.spec.ts` |
| AC-PR2 | durable store path | migration + mem fallback |
| AC-PR3 | Drive deep-link field | `drive_object_id` |
| AC-PR4 | JWT 401 / license 403 | `TestProjectsWithout*` |

## Out

MS Project / enterprise Gantt.
