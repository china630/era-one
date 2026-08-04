# ERA Office — Stage PRJ-LITE (Projects Lite → Full MVP)

**Wave:** PRJ-LITE  
**Дата:** 2 августа 2026 г.  
**Prerequisite:** Projects floor / W2  
**Статус:** `[x]`  
**Inventory:** [Office-UI-Feature-Inventory.md](Office-UI-Feature-Inventory.md)

## Цель

Поднятие Lite Projects (**Share**, **Swimlanes**) до Full в Kanban MVP + закрытие Missing **PRJ-PRIORITY**. Не Jira / MS Project.  
`PRJ-GANTT`: ранее Never; **superseded by** [O-MS](Office-Stage-OMS-Spec.md) (due-date Lite).

## Backlog

### P0 — Share / Swimlanes / Priority

| ID | Критерий | Inventory | Статус |
|----|----------|-----------|--------|
| PRJLITE-P0-1 | Share dialog: board link + Manage ACL in Drive (`.eraj`) | PRJ-SHARE | [x] |
| PRJLITE-P0-2 | Persist viewMode; DnD across lane → assignee; Unassigned | PRJ-SWIMLANES | [x] |
| PRJLITE-P0-3 | `priority` p0\|p1\|p2\|unset; chip; create/edit; filter | PRJ-PRIORITY | [x] |

### P1 — Polish

| ID | Критерий | Inventory | Статус |
|----|----------|-----------|--------|
| PRJLITE-P1-1 | Assign picker (peers/JWT + free-text) | PRJ-ASSIGN | [x] |
| PRJLITE-P1-2 | Filter facets: assignee, label, priority, overdue | PRJ-FILTER | [x] |
| PRJLITE-P1-3 | Drive picker folder browse | PRJ-DRIVE-PICKER | [x] |
| PRJLITE-P1-4 | In-column `sort_key` reorder | PRJ-DRAG | [x] |

## Вне scope

Full MS Project Gantt (deps/critical path), sprints, epics, custom columns, Planner invites. (Gantt Lite → O-MS.)

## Proof

| Доказательство | Результат |
|----------------|-----------|
| `go test ./services/platform/cmd/docs-projects/` | PASS (incl. `TestProjectsPriorityAndSortKey`) |
| Migration `009_projects_priority.sql` | priority + sort_key |
| Playwright `projects.spec.ts` (8 tests, incl. PRJ-LITE share/priority/swimlanes) | PASS |
| Inventory + Catalog + canvas Projects Lite/Missing → Full | updated |
