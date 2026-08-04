# ERA Office — Stage O-5 (docx I/O + SBOM)

**Wave:** O-5  
**Версия:** 1.1  
**Дата:** 30 июля 2026 г.  
**Продукт:** ERA Documents  
**PRD:** [`PRD-Office-MVP.md`](products/PRD-Office-MVP.md) · AC-O2, AC-O5  
**Предусловие:** Wave **O-4** gate = PASS  
**Статус:** `[x]` — gate PASS `reports/office-stage-O-5-20260730-004202.log`

---

## 1. Цель этапа

> docx import → `.erad` → export golden (AC-O2); zero-GPL SBOM MVP gate (AC-O5); fuzz smoke.

Pilot-ready / edition `mvp` → **O-GA**.

## 2. Scope

### Входит

- Harden golden_docx + corpus (missing = FAIL)
- Memo structure-equiv (ignore volatile ids)
- SBOM gate: FAIL without cargo metadata; deny-list file
- Fuzz smoke Required
- Gate O-5 Required

### НЕ входит

- Syft / CycloneDX / grype
- Full `cargo fuzz` soak in CI
- OOXML macro fidelity beyond corpus
- Pilot-ready / `mvp` / `ga`

## 3. E2E-сценарий приёмки

1. `cargo test -p era-docs-engine --test golden_docx --quiet`
2. `cargo test -p era-docs-engine --test golden_docx_corpus --quiet`
3. `cargo test -p era-docs-engine fuzz_docx_smoke --quiet`
4. `pwsh -NoProfile -File ./scripts/office-sbom-gate.ps1`
5. `pwsh -NoProfile -File ./scripts/run-office-stage-gate.ps1 -Stage O-5 -WriteSignoff`

## 4. Критерии приёмки

| ID | Критерий | PRD | Доказательство | Статус |
|----|----------|-----|----------------|--------|
| F-O5-1 | Memo docx golden + roundtrip | AC-O2 | `--test golden_docx` | [x] |
| F-O5-2 | Corpus 6 fixtures hard golden | AC-O2 | `--test golden_docx_corpus` | [x] |
| F-O5-3 | Fuzz smoke | AC-O6 smoke | `fuzz_docx_smoke` | [x] |
| F-O5-4 | Zero GPL SBOM MVP | AC-O5 | `office-sbom-gate.ps1` | [x] |
| F-O5-5 | Gate O-5 PASS | — | `reports/office-stage-O-5-20260730-004202.log` | [x] |

## 5. Backlog (OM5-*)

| ID | Задача | Статус |
|----|--------|--------|
| OM5-1 | Stage O-5 Spec | [x] |
| OM5-2 | Harden golden corpus | [x] |
| OM5-3 | Memo structure-equiv | [x] |
| OM5-4 | Remove orphan corpus `*.golden.erad.json` | [x] |
| OM5-5 | Harden SBOM gate | [x] |
| OM5-6 | Fuzz smoke in gate | [x] |
| OM5-7 | Gate Required | [x] |
| OM5-8 | Matrix / Index / MVP-Spec / editions | [x] |

## 6. Stage Gate

| # | Проверка | Доказательство |
|---|----------|----------------|
| G1 | `run-office-stage-gate.ps1 -Stage O-5` | PASS |
| G3 | Matrix AC-O2 / AC-O5 Scaffold | ✅ |
| G4 | Sprint-Index O-5 `[x]` | docs |
| G6 | signoff | `reports/office-stage-O-5-signoff.md` |

## 7. Связано

- Legacy: [`Office-Stage-O1-2-Docx-Spec.md`](Office-Stage-O1-2-Docx-Spec.md)
- Не путать с [`Office-Stage-O5-CommsHook-Spec.md`](Office-Stage-O5-CommsHook-Spec.md)
- [`Office-Sprint-Index.md`](Office-Sprint-Index.md)
