# ERA Communications — Stage C-MM-P2 (Mail Moderation P2)

**Wave:** C-MM-P2  
**Дата:** 30 июля 2026 г.  
**Предусловие:** C-MM-H / C-MM-P1 PASS; Perimeter DLP handoff for P2-01  
**PRD:** [`PRD-Mail-Moderation.md`](products/PRD-Mail-Moderation.md) §4.3

## Backlog

| ID | PRD | Задача | Статус |
|----|-----|--------|--------|
| MM-P2-1 | P2-01 | DLP sensitive-info trigger → Perimeter | [x] `policy.DLPTrigger` unit |
| MM-P2-2 | P2-02 | All-of / quorum moderators | [x] memory + PG RequireAll |
| MM-P2-3 | P2-03 | Multi-level L1→L2 | [x] `level` / `escalate_to` fields |
| MM-P2-4 | P2-04 | Outlook native Approve/Reject | [x] X-ERA-Moderation-* / List-Unsubscribe headers |
| MM-P2-5 | P2-05 | NLP / smart classification | [x] `nlp_suspicious` + classify heuristic |

## Gate (planned)

`go test -C services/comms/mail-moderation ./...` + AC-MM-P2-* in Implementation Matrix.

Edition remains **mvp** until P2 field; does not auto-promote `ga`.
