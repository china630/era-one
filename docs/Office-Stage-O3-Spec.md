# ERA Office — Stage O-3 (Documents foundation)

**Wave:** O-3  
**Версия:** 1.0  
**Дата:** 30 июля 2026 г.  
**Продукт:** ERA Documents  
**PRD:** [`PRD-Office-MVP.md`](products/PRD-Office-MVP.md) · AC-O3, AC-O4  
**Предусловие:** Waves **O-0…O-2** gate = PASS

---

## 1. Цель этапа

> Create / open / save `.erad` через Drive (docs-engine); basic editor `/docs/{id}`; engine без authoritative blob copy.

Co-edit multi-user → **O-4**. docx golden + SBOM → **O-5**.

## 2. Scope

### Входит

- `era-docs-engine` в Rust workspace; drive_bind + proto_roundtrip
- REST create/get/snapshot via Drive API
- ui/docs: JWT + New document + open by id
- Compose `--profile docs`
- `era-documents` edition scaffold

### НЕ входит

- 2-user CRDT e2e (O-4)
- docx golden corpus / SBOM as hard gate (O-5)

## 3. E2E-сценарий приёмки

1. `cargo test -p era-docs-engine drive_bind --quiet`
2. `cargo test -p era-docs-engine --test proto_roundtrip --quiet`
3. `go test -C ui/docs ./... -count=1`
4. `docker compose -f deploy/docker-compose.office.yml --profile docs config`
5. `.\scripts\run-office-stage-gate.ps1 -Stage O-3 -WriteSignoff`

## 4. Критерии приёмки

| ID | Критерий | PRD | Доказательство | Статус |
|----|----------|-----|----------------|--------|
| F-O3-1 | docs-engine in workspace + drive_bind | AC-O3 | cargo | [x] |
| F-O3-2 | Create/get `.erad` via Drive | AC-O3 | cargo / unit | [x] |
| F-O3-3 | ui/docs create + open by id | AC-O4 | go test | [x] |
| F-O3-4 | Compose `--profile docs` | — | docker compose | [x] |
| F-O3-5 | Gate O-3 PASS | — | `reports/office-stage-O-3-20260730-002541.log` | [x] |

## 5. Backlog (OM3-*)

| ID | Задача | Статус |
|----|--------|--------|
| OM3-1 | Stage O-3 Spec | [x] |
| OM3-2 | Cargo workspace + cargo tests | [x] |
| OM3-3 | Gate Required checks | [x] |
| OM3-4 | Compose DocsAPIURL + profile docs | [x] |
| OM3-5 | ui/docs auth + New doc | [x] |
| OM3-6 | No MinIO direct write invariant | [x] |
| OM3-7 | editions-office scaffold | [x] |
| OM3-8 | Matrix / Index / MVP-Spec | [x] |

## 6. Stage Gate

| # | Проверка | Доказательство |
|---|----------|----------------|
| G1 | `run-office-stage-gate.ps1 -Stage O-3` | PASS |
| G3 | Matrix AC-O3/O4 updated (rollup from Matrix) | Matrix SSOT · **not** false ✅ |
| G4 | Sprint-Index O-3 `[x]` | docs |
| G6 | signoff | `reports/office-stage-O-3-signoff.md` |

## 7. Связано

- Legacy: O1-1 Proto, O1-4 DriveBind, O1-5 Engine, O1-6 DocsUI
- [`Office-Sprint-Index.md`](Office-Sprint-Index.md)
