# Office Stage — Solo Store SKU distribution

**Targets: Browser ❌ · Solo ✅ · Corporate ❌** (SKU modes affect Solo entry only)

**Status:** [x] 2026-08-03 — scaffold + listing metadata + build evidence path. Store **publish** / EV signing = ops checklist (not automated).

## Product model

One binary `era-office-desktop`, four Store listings (+ suite):

| SKU | `--sku` / `ERA_OFFICE_SKU` | Identifier | Associations |
|-----|---------------------------|------------|--------------|
| Documents | `docs` | `az.era.office.documents` | `.erad`, `.docx` |
| Tables | `tables` | `az.era.office.tables` | `.erat`, `.xlsx` |
| Presentations | `presentations` | `az.era.office.presentations` | `.erap`, `.pptx` |
| Projects | `projects` | `az.era.office.projects` | `.eraj` |
| Suite (hub) | `suite` (default) | `az.era.office.desktop` | all four |

Protocol: `era-office://open?path=/docs/…` (also `/tables/…`, `/presentations/…`, `/projects/…`). File path argv opens the matching product. Second instance hands off via `tauri-plugin-single-instance`.

## Listing metadata (Partner Center draft)

Overlays: `apps/era-office-desktop/src-tauri/tauri.conf.sku-*.json`.

| SKU | productName | identifier | shortDescription (overlay) | Icons (placeholder) | Protocol |
|-----|-------------|------------|----------------------------|---------------------|----------|
| docs | ERA Documents | `az.era.office.documents` | ERA Documents Solo — local .erad editor | `icons/sku-docs.{png,ico}` | `era-office://` |
| tables | ERA Tables | `az.era.office.tables` | ERA Tables Solo — local .erat spreadsheet | `icons/sku-tables.*` | same scheme |
| presentations | ERA Presentations | `az.era.office.presentations` | ERA Presentations Solo — local .erap slides | `icons/sku-pres.*` | same |
| projects | ERA Projects | `az.era.office.projects` | ERA Projects Solo — local .eraj board | `icons/sku-projects.*` | same |
| suite | ERA Office | `az.era.office.desktop` | Solo suite hub | `icons/icon.*` | same |

File associations come from Tauri `bundle.fileAssociations` in each overlay (NSIS/deb/rpm targets). If OS protocol registration is incomplete after install, follow-up: `tauri-plugin-deep-link` — only then; do not add until verified missing.

Privacy policy URL / Store screenshots: **ops slots** (see checklist) — no secrets in repo.

## Acceptance

| ID | Criterion | Evidence |
|----|-----------|----------|
| AC-SKU-1 | `--sku=docs` opens Docs (no hub); `tables` → Tables; … | `sku.rs` unit tests; `solo_entry_go` + `ui/app.js` |
| AC-SKU-2 | Per-SKU NSIS/tauri build with distinct identifier | `scripts/build-office-sku.ps1` + overlays; evidence log below |
| AC-SKU-3 | Protocol/argv file path routes to product | `corp::file_path_from_args` + `handle_open_payload` |
| AC-TEST | `cargo test -p era-office-desktop --lib` PASS | 35 tests; also `smoke-office-solo-lab.ps1` |

## Build

```powershell
# Dev with SKU
$env:ERA_OFFICE_SKU = "presentations"
cargo run -p era-office-desktop -- --sku=presentations

# Release binary only (no NSIS) — validates overlay path + assets copy
.\scripts\build-office-sku.ps1 -Sku docs -SkipBundle
.\scripts\build-office-sku.ps1 -Sku tables -SkipBundle
.\scripts\build-office-sku.ps1 -Sku presentations -SkipBundle
.\scripts\build-office-sku.ps1 -Sku projects -SkipBundle
.\scripts\build-office-sku.ps1 -Sku suite -SkipBundle

# Full NSIS (needs tauri-cli + NSIS on PATH)
.\scripts\build-office-sku.ps1 -Sku docs
# ... tables | presentations | projects | suite
```

Artifacts: `target/release/era-office-desktop.exe`, `target/release/assets/`, `target/release/era-office-<sku>.cmd`, optional `target/release/bundle/nsis/`.

Lab portable (all SKUs, one folder): `scripts/pack-office-solo-lab.ps1` → `dist/office-solo-lab/`.

### Build evidence

| When | Command | Log |
|------|---------|-----|
| Lab smoke | `.\scripts\smoke-office-solo-lab.ps1` | `reports/office-solo-lab-smoke-*.log` |
| SKU matrix SkipBundle | `.\scripts\build-office-sku-matrix.ps1` | `reports/office-sku-build-*.log` |
| Full NSIS (optional) | `build-office-sku.ps1 -Sku <x>` | operator attaches console / CI artifact |

## EV / code signing (config only — no secrets in git)

Document in local/CI secrets store; wire when EV cert exists:

| Item | Where | Notes |
|------|-------|-------|
| Certificate thumbprint | `tauri.conf.json` → `bundle.windows.certificateThumbprint` | Or env injected at CI |
| Timestamp URL | `bundle.windows.timestampUrl` | Org TSA |
| Digest | `bundle.windows.digestAlgorithm` | e.g. `sha256` |
| Sign command override | `bundle.windows.signCommand` | Optional custom |

Never commit `.pfx`, passwords, or Partner Center API keys.

## Ops checklist (Store / EV — out of automated code)

- [ ] EV code-signing certificate + `tauri` windows sign config (table above)
- [ ] Microsoft Partner Center listings (4 SKUs + suite optional + privacy policy URL)
- [ ] Confirm file associations + `era-office://` after NSIS install; deep-link plugin only if missing
- [ ] Branded Store icons / screenshots per SKU (replace `icons/sku-*` placeholders)
- [ ] Mail / Office AI Solo — **not** in this stage (remain hidden)
- [ ] MDM managed Corporate `server_url` — see [Corp-Lab-Demo](Office-Stage-Corp-Lab-Demo.md)

**Matrix:** Store publish / EV remain `[ ]`; Sell Solo = lab / internal until EV ([Product-Readiness](Office-Product-Readiness-Matrix.md) § Desktop).

## Related

- [Office-Stage-Solo-Lab-Demo.md](Office-Stage-Solo-Lab-Demo.md)
- [Office-Stage-Solo-Pres-Spec.md](Office-Stage-Solo-Pres-Spec.md)
- [Office-Stage-Solo-Projects-Spec.md](Office-Stage-Solo-Projects-Spec.md)
- [Office-Stage-Corp-Lab-Demo.md](Office-Stage-Corp-Lab-Demo.md)
- [Office-Roadmap.md](Office-Roadmap.md)
- [Office-Product-Readiness-Matrix.md](Office-Product-Readiness-Matrix.md)
