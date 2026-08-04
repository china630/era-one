# ERA Office — Stage O1-3 (CRDT sync)

**Wave:** O1-3  
**Предусловие:** O1-1 gate = PASS  
**Gap:** GAP-O-P1-03 · **PRD:** AC-O1

## 1. Цель

op_log, merge, WebSocket sync, Postgres sessions, two-client simulation.

## 2. Backlog (OM13-*)

| ID | Задача | Статус |
|----|--------|--------|
| OM13-1 | op_log + merge | [x] |
| OM13-2 | WebSocket handler | [x] |
| OM13-3 | Postgres doc_sessions | [x] |
| OM13-4 | two-client merge test | [x] |
| OM13-5 | restart replay test | [x] `persist_tests` |

## 4. Stage Gate

| G1 | `-Stage O1-3` | [ ] |
| G6 | signoff | [ ] |
