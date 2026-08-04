# ERA Office — Stage S3 Solo Documents (B1 vertical)

**Дата:** 3 августа 2026 г.  
**Статус:** [x]  
**Targets: Browser ❌ · Solo ✅ · Corporate ❌** (Corporate = S4)  
**Связано:** [Office-Roadmap](Office-Roadmap.md) §0 S3 / §3.2–3.6 · [ADR-0026](adr/0026-sovereign-office-engine.md) · [ADR-0010](adr/0010-licensing-and-activation.md)

## Цель

Runnable **ERA Documents Solo** на диске: open/edit/save `.erad`, import/export `.docx`, demo save-limit, offline device license. Без Drive / WS / Identity / Postgres.

## Deliverable

| Путь | Роль |
|------|------|
| `apps/era-office-desktop/` | Tauri 2 app + thin UI |
| `apps/era-office-desktop/src-tauri` | `era-office-desktop` crate (workspace member) |
| `era-docs-core` | model / convert (S2) |
| `era-license` | verify `ERA1` token + module `office-docs-solo` |

## Acceptance criteria

| ID | Criterion | Evidence |
|----|-----------|----------|
| AC-S3-1 | New / open / save `.erad` via path I/O | `solo::tests::erad_roundtrip_path` |
| AC-S3-2 | Import / export docx via core | `solo::tests::docx_export_import_smoke` |
| AC-S3-3 | Demo blocks save gate (cap 5) | `solo::tests::demo_blocks_over_cap_blocks_save` |
| AC-S3-4 | Licensed token unlocks save over cap | `solo::tests::licensed_can_save_over_cap` + `license::tests::*` |
| AC-S3-5 | Thin Solo UI (File menu + blocks) | `apps/era-office-desktop/ui/` |
| AC-S3-6 | Package builds | `cargo test -p era-office-desktop --lib`; `cargo check -p era-office-desktop` |

## Commands (invoke)

`doc_new`, `doc_get`, `doc_apply_local`, `doc_open_dialog` / `doc_open_path`, `doc_save` / `doc_save_as_dialog` / `doc_save_path`, `doc_import_docx_dialog`, `doc_export_docx_dialog` / `doc_export_docx_path`, `license_status`, `license_set_token`.

## Out of scope

Corporate shell (S4), Solo Tables (S5), full `ui/docs` collab port, Store / EV signing.
