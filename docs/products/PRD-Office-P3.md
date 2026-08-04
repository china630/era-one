# PRD: ERA Office — P3 (ERA Presentations)

**Статус:** Scaffold rollup **🟡** (Matrix AC-P1/path; O-P-H unit) · Wave **O-P** · не ga  
**Дата:** 30 июля 2026 г.  
**Native:** `.erap` = `DOCUMENT_FORMAT_ERAP`

## Scope MVP

- Create/open `.erap` deck (title/body slides)
- Minimal pptx import/export golden
- UI `/presentations`
- License `office-presentations`
- JWT AuthZ

## AC-P1…P5

| ID | Критерий | Proof |
|----|----------|-------|
| AC-P1 | create `.erap` → Drive → reopen | presentations-engine Drive bind |
| AC-P2 | pptx golden subset | `golden_pptx` |
| AC-P3 | UI `/presentations` | `go test -C ui/presentations` + Playwright `presentations.spec.ts` |
| AC-P4 | без license → 403 | HTTP negative |
| AC-P5 | без JWT → 401 | HTTP negative |

## Out

Full PowerPoint fidelity, animations, co-edit.
