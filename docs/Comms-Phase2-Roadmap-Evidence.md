# ERA Comms — Phase 2 / Roadmap evidence note

**Дата:** 30 июля 2026 г.

## Scaffold → partial backends

| Wave | Gate / proof | Honest product state |
|------|--------------|----------------------|
| C-4 Chat | gate + `store` persist test | `ERA_CHAT_DATA_DIR` JSON — GAP-P2-01 homeserver still open |
| C-4 VCS | gate + LiveKit FromEnv unit | LiveKit when `ERA_LIVEKIT_URL`; else Stub — GAP-P2-02 field open |
| C-5 Comms AI | gate + llm FromEnv test | Ollama if up; Heuristic CI — see Ollama runbook |
| MM P2 | unit DLP/quorum | Spec [`Comms-Stage-CMM-P2-Spec.md`](Comms-Stage-CMM-P2-Spec.md); multi-level/Outlook/NLP open |
| Scale 60k / HA | quick 500 PASS + HA notes | Full 60k on sizing host; [`Comms-HA-Notes.md`](Comms-HA-Notes.md) |

## Edition

Chat / Conference / AI remain **`roadmap`** until field backends + proof.  
Do not promote on Heuristic/Stub alone.
