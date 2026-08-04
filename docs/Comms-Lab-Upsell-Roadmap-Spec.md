# ERA Communications — Lab / Upsell / Roadmap Spec (без партнёра)

**Дата:** 30 июля 2026 г.  
**Инвариант:** staging зелёный; PASS-лог → docs; **`ga` не ставить** до RT-09.  
**SSOT sibling:** [`Comms-Summer-Wave-Spec.md`](Comms-Summer-Wave-Spec.md) · [`Comms-Implementation-Matrix.md`](Comms-Implementation-Matrix.md) · [`editions-comms.yaml`](../editions-comms.yaml)

| ID | Track | Работа | Статус | Proof |
|----|-------|--------|--------|-------|
| L-1 | A | Lab IMAP `dovecot-lab` (ERA lab-imap) + `deploy/comms-lab-imap/` | [x] | `scripts/run-comms-lab-imap-smoke.ps1` → `reports/comms-lab-imap-smoke.log` |
| L-2 | A | Rebuild mail-api path; staging без RT-06 fallback; ProdProfile SSL on | [x] | `scripts/run-comms-pilot-staging.ps1` (no fallback); prod overlay SSL |
| L-3 | A | Chat PG via `ERA_CHAT_DATABASE_URL` / `ERA_COMMS_DATABASE_URL` | [x] | unit `chat/internal/store`; compose `era-chat` + `005_chat.sql` |
| L-4 | A | Ollama compose overlay `docker-compose.comms.ai.yml` | [x] | overlay + `TestOllamaAvailableTrue`; Heuristic if Ollama down |
| L-5 | A | Meet join UI + air-gap LiveKit stub | [x] | `scripts/run-comms-meet-smoke.ps1` → `reports/comms-meet-smoke.log` |
| L-6 | A | Scale script `-Mailboxes 1000` | [x] | `scripts/run-comms-scale-60k.ps1 -Mailboxes 1000` → `reports/comms-scale-60k-*.log` |
| U-MIG | B | Live IMAP → ERA target; PST/MBOX fixtures; CH mailbox | [x] | `scripts/run-comms-migration-live-imap.ps1` |
| U-CONN | B | Connect Autodiscover → dovecot-lab; RT-10 lab | [x] | `scripts/run-comms-connect-lab.ps1` |
| U-BR | B | Bridge 100-mailbox synthetic FindFolder | [x] | `scripts/run-comms-bridge-100mb-lab.ps1` → `reports/comms-bridge-100mb-lab.log` |
| U-MM | B | `Escalate()` + `ERA_MM_DLP_URL` stub | [x] | `TestEngine_Escalate`, `TestDLPHandoffStub` |
| R-1 | C | Chat `roadmap→mvp` (not ga) | [x] | L-3 proof |
| R-2 | C | Conference `roadmap→mvp` (not ga) | [x] | L-5 proof |
| R-3 | C | Comms AI `roadmap→mvp` (not ga) | [x] | L-4 Available() true unit |
| R-4 | C | Desktop client | [x] | `exists: false` unchanged |
| R-5 | C | ActiveSync / Drive checklist only | [x] | no edition bump |

**Запрет:** Chat / Conference / AI → `ga` без field RT-09.

## Artifacts

| Path | Role |
|------|------|
| `deploy/docker-compose.comms.dev.yml` | dovecot-lab, chat PG, connect |
| `deploy/docker-compose.comms.ai.yml` | ollama + comms-ai |
| `deploy/dockerfiles/Dockerfile.lab-imap` | L-1 image |
| `docs/Comms-Lab-IMAP-Notes.md` | operator notes |
| `ui/meet/static/livekit-stub.js` | air-gap Meet client |

## Multi-node Chat note (R-1)

With `ERA_CHAT_DATABASE_URL` shared across chat-api replicas, rooms/messages are PG-backed (`005_chat.sql`). JSON `ERA_CHAT_DATA_DIR` remains single-node fallback.
