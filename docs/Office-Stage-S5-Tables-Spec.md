# ERA Office — Stage S5 Solo Tables + Windows bundle scaffold

**Дата:** 3 августа 2026 г.  
**Статус:** [x]  
**Targets: Browser ❌ · Solo ✅ · Corporate ❌** (Corporate shell = S4; Tables Solo only)  
**Связано:** [Office-Roadmap](Office-Roadmap.md) §0 S5 / §3.4 B2–B3 · [ADR-0026](adr/0026-sovereign-office-engine.md) · [ADR-0010](adr/0010-licensing-and-activation.md)

## Цель

1. **B2 — Solo Tables:** open/edit/save `.erat`, import/export `.xlsx` (ODS open path), demo cell-cap, offline module license `office-tables-solo`.
2. **B3 scaffold:** `bundle.active` + NSIS target in Tauri config; release exe build; Store/EV = ops checklist only.

## Deliverable

| Путь | Роль |
|------|------|
| `apps/era-office-desktop/src-tauri/src/solo_tables.rs` | Solo Tables session |
| `license.rs` | `MODULE_TABLES_SOLO` + `DEMO_CELL_CAP=25` |
| `commands.rs` | `sheet_*` invoke |
| `ui/` | Documents \| Tables mode switch + 20×10 grid |
| `tauri.conf.json` | `bundle.active: true`, `targets: ["nsis"]` |

## Acceptance criteria

| ID | Criterion | Evidence |
|----|-----------|----------|
| AC-S5-1 | `.erat` roundtrip path | `solo_tables::tests::erat_roundtrip_path` |
| AC-S5-2 | xlsx import/export smoke | `solo_tables::tests::xlsx_export_import_smoke` |
| AC-S5-3 | Demo cell-cap blocks save | `solo_tables::tests::demo_cells_over_cap_blocks_save` |
| AC-S5-4 | Licensed unlocks over cap | `solo_tables::tests::licensed_can_save_over_cap` + `license::tests::tables_module_licenses` |
| AC-S5-5 | Thin Tables UI + mode switch | `ui/index.html` mode Docs/Tables + grid |
| AC-S5-6 | Package builds | `cargo test -p era-office-desktop --lib`; `cargo build --release -p era-office-desktop` |
| AC-S5-B3 | Bundle scaffold | `tauri.conf.json` NSIS active; README build notes; Store checklist below |

## Commands (invoke)

`sheet_new`, `sheet_get`, `sheet_apply_local`, `sheet_open_dialog` / `sheet_open_path`, `sheet_save` / `sheet_save_as_dialog` / `sheet_save_path`, `sheet_import_xlsx_dialog`, `sheet_export_xlsx_dialog`, `sheet_license_status`.

## License

- Env: `ERA_SOLO_LICENSE` with module `office-tables-solo`
- Demo: save if nonempty cells ≤ **25**

## B3 — Store / EV checklist (ops, not done here)

- [ ] Install `tauri-cli` + NSIS → `cargo tauri build` → `target/release/bundle/nsis/`
- [ ] EV code-signing certificate
- [ ] Microsoft Partner Center / Mac App Store accounts + listing
- [ ] Protocol handler `era-office://` in installer
- [ ] MDM managed `server_url` for Corporate

## Out of scope

Solo Presentations (B4), full Platform `ui/tables` port, MS Store publish, hybrid Drive sync.
