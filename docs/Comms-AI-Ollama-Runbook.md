# ERA Comms AI — on-prem LLM ops (GAP-P2-03)

**Air-gap:** no external model SaaS. Use bundled/Ollama inside customer perimeter.

## Env

| Variable | Default | Purpose |
|----------|---------|---------|
| `ERA_OLLAMA_URL` | `http://127.0.0.1:11434` | Ollama HTTP |
| `ERA_OLLAMA_MODEL` | `llama3.2` | Model tag pulled offline |

`llm.FromEnv()` uses Ollama when `/api/tags` succeeds; otherwise Heuristic (scaffold / CI).

## Ops checklist

1. Install Ollama (or vendor-bundled binary) on AI host  
2. `ollama pull llama3.2` from air-gap mirror  
3. Set env on `comms-ai` service; restart  
4. Gate: `.\scripts\run-comms-stage-gate.ps1 -Stage C-5` + golden phishing fixtures  
5. Edition stays **roadmap** until field LLM smoke PASS

## Evidence

Unit/gate without Ollama = Heuristic scaffold PASS. Do not promote `era-comms-ai` to `ga` on Heuristic alone.

`/healthz` and inference responses expose `mode` (`heuristic` \| `ollama`) and `degraded=true` when heuristic — Deepen D8 honesty ([`reports/comms-deepen-d8-roadmap.md`](../reports/comms-deepen-d8-roadmap.md)).

