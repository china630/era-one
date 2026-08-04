# PRD: ERA Office — Office AI

**Статус:** Scaffold **✅** stub+deny (Matrix AC-AI*) · Pilot-ready [ ] · Wave **O-AI** · не ga  
**Дата:** 30 июля 2026 г.  
**ADR:** 0025 / 0026 — air-gap only

## Scope MVP

- `docs-ai` summarize: explicit `mode=stub` when Ollama unset
- Optional local `ERA_OLLAMA_URL` in-contour only
- License `office-ai` + JWT
- UI `/office-ai` + Documents handoff «Summarize with AI»

## AC-AI1…AI3

| ID | Критерий | Proof |
|----|----------|-------|
| AC-AI1 | stub mode no phone-home | `TestDocsAIStubModeNoPhoneHome` + Playwright `office-ai.spec.ts` |
| AC-AI2 | JWT 401 / license 403 | `TestDocsAIWithout*` |
| AC-AI3 | Ollama only configured host | `ERA_OLLAMA_URL` |

## Out

Cloud SaaS LLM; field model pack as Pilot-ready.
