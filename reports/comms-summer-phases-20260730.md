==> ERA Comms Summer Phases 1-4 evidence 2026-07-30

S1-1 REST audit RecordSend on mailSend + internal deliver — unit PASS
S1-2 docker-compose.comms.prod.yml + deploy/comms-tls/
S1-3 EWS Notes/Tasks SyncNotes/SyncTasks — unit PASS
S1-4 ActiveSync Calendar+Contacts folders — golden updated PASS
S1-5 multi-rcpt + MaxRecipients + AttachmentExtDeny — unit PASS
S1-6 webmail compose/read already present
S1-7 ERA_LICENSE_MODULES in prod overlay (no ERA_MAIL_DEV)
S1-8 Comms-Partner-Summer-DryRun.md

S2-B synthetic upstream — unit PASS; ERA_BRIDGE_SYNTHETIC in dev overlay
S2-C PG quorum migration 004 + NLP/headers/level — unit PASS
S2-D Connect multi-folder + cursors — unit PASS
S2-A CH mailbox column + lab IMAP notes (live IMAP still field)

S3-1 Chat Matrix-layout IDs + 005_chat.sql
S3-2 LiveKit HS256 JWT + ui/meet /meet/api/room — unit PASS
S3-3 Ollama runbook (existing)
S3-4 run-comms-scale-60k.ps1
S3-5 docker-compose.comms.ha.yml

S4-1 PRD-Mail-Client-Desktop + editions honesty
S4-2 PRD-ActiveSync-Subset; gap unblocked to subset lab
S4-3 Comms-Drive-Hook-Note.md
S4-4 partner dry-run docs

Edition ga: NOT set (RT-09 SKIP)

Unit log: reports/comms-summer-unit-20260730-003846.log (+ activesync golden fix)

SUMMER LAB PASS (code/docs); partner field deferred
