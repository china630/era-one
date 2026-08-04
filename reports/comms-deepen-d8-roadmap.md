# Deepen D8 — Chat / Conference / AI (lab roadmap) — 2026-08-04

**Status:** `[x] lab` · **not `ga`** · field demos open.

## AuthZ + mode honesty

| Service | AuthZ | Mode / degraded |
|---------|-------|-----------------|
| Chat | `httpauth` (`ERA_CHAT_DEV` / JWT); spoof without DEV → 401 | `/healthz` `storage_mode`: memory \| json \| postgres |
| Conference (VCS) | `httpauth` (`ERA_VCS_DEV` / JWT) | `/healthz` `mode`: stub \| livekit (`ERA_LIVEKIT_URL`) |
| Meet UI | RBAC `vcs.user` + tenant | `/meet/healthz` `mode=stub`; client `livekit-stub.js` |
| Comms AI | `httpauth` + license `comms-ai` | `/healthz` + inference: `mode` heuristic\|ollama, `degraded=true` on heuristic |

Shared package: `services/comms/internal/httpauth`. Lab DEV keys must not ship in prod overlays.

## Docs

- AI ops: [`docs/Comms-AI-Ollama-Runbook.md`](../docs/Comms-AI-Ollama-Runbook.md) — heuristic ≠ field LLM
- Honesty G5 / G0-6: [`docs/Comms-Honesty-Green-Spec.md`](../docs/Comms-Honesty-Green-Spec.md)

## Out of scope

- Edition `ga`, partner LiveKit/Ollama field SignOff
