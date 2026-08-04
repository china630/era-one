# ERA Communications — Stage C-MM-P1 (Mail Moderation P1)

**Wave:** C-MM-P1  
**Версия:** 1.0  
**Дата:** 29 июля 2026 г.  
**Предусловие:** Stage **C-MM-H** = PASS  
**PRD:** [`PRD-Mail-Moderation.md`](products/PRD-Mail-Moderation.md) §4.2  
**Статус волны:** [x] 2026-07-29

## Backlog

| ID | PRD | Задача | Статус |
|----|-----|--------|--------|
| MM-P1-1 | P1-01 | Moderated recipients / DL | [x] `moderated_recipients` + engine + template `moderated-dl` |
| MM-P1-2 | P1-02 | HR API novices + curator | [x] `POST /v1/moderation/hr/novices` |
| MM-P1-3 | P1-03 | Minimal Admin UI | [x] `/ui/` rules · import · simulate · holds |

## Gate

`go test -C services/comms/mail-moderation ./...` + AC-MM-P1-* in Implementation Matrix — PASS 2026-07-29.
