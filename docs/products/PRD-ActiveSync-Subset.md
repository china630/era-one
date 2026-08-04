# PRD slice: ActiveSync subset — Summer S4-2

**Статус:** Lab subset implemented  
**Дата:** 30 июля 2026 г.  
**Модуль:** `services/comms/mail/internal/activesync`

## In scope (lab)

- Provision, FolderSync, Sync (message count), Ping  
- WBXML encode/decode  
- Staging RT-06 Provision smoke  

## Out of scope until field

- Full iOS/Android calendar+contacts policy parity  
- Device wipe / remote wipe  

## Acceptance

Unit WBXML + staging Provision PASS → gap CM5-9 moves from `[blocked]` to **subset lab**.
