# ERA Office desktop (`apps/era-office-desktop`)

**Targets: Browser ❌ · Solo ✅ · Corporate ✅**

Tauri 2 shell: one binary, Solo hub **or** per-product Store SKU, plus Corporate WebView.

| Mode | Behavior |
|------|----------|
| **Solo · Suite** (`--sku=suite`, default) | Hub → Docs / Tables / Presentations / Projects |
| **Solo · Documents** (`--sku=docs`) | Direct `/docs/solo` · `.erad` / docx |
| **Solo · Tables** (`--sku=tables`) | Direct `/tables/solo` · `.erat` / xlsx / ods |
| **Solo · Presentations** (`--sku=presentations`) | Direct `/presentations/solo` · `.erap` / pptx |
| **Solo · Projects** (`--sku=projects`) | Direct `/projects/solo` · `.eraj` |
| **Corporate** | WebView → tenant Workspace URL + SSO (S4) |

## Prerequisites (Windows)

- Rust toolchain
- [WebView2](https://developer.microsoft.com/microsoft-edge/webview2/)
- Optional for installer: `cargo install tauri-cli --version "^2"` (+ NSIS)

## Run (dev)

```powershell
cargo run -p era-office-desktop
# SKU single product:
cargo run -p era-office-desktop -- --sku=docs
$env:ERA_OFFICE_SKU = "tables"; cargo run -p era-office-desktop
```

First run (suite only) shows Solo / Corporate chooser. SKU builds skip hub and open the product.

### Env overrides

```powershell
$env:ERA_OFFICE_PROFILE = "corporate"
$env:ERA_OFFICE_SERVER_URL = "https://office.example.gov"
cargo run -p era-office-desktop
```

Config: `{config_dir}/era-office-desktop/config.json`

### Deep links / file argv

```powershell
cargo run -p era-office-desktop -- "era-office://open?path=/tables/solo"
cargo run -p era-office-desktop -- "D:\work\deck.erap"
```

Second instance forwards argv to the running window (`tauri-plugin-single-instance`).

## License / demo

| Product | Module claim | Demo save gate |
|---------|--------------|----------------|
| Documents | `office-docs-solo` | ≤ 5 blocks |
| Tables | `office-tables-solo` | ≤ 25 nonempty cells |
| Presentations | `office-pres-solo` | ≤ 5 slides |
| Projects | `office-projects-solo` | ≤ 15 tasks |

Set `ERA_SOLO_LICENSE` to an `ERA1.…` token with the needed module(s).

## Lab portable pack

```powershell
.\scripts\smoke-office-solo-lab.ps1   # headless unit smoke -> reports/office-solo-lab-smoke-*.log
.\scripts\pack-office-solo-lab.ps1 -Zip
# -> dist/office-solo-lab/  (exe + assets + 5 launchers)
# -> dist/office-solo-lab.zip
```

GUI checklist: [Office-Stage-Solo-Lab-Demo.md](../../docs/Office-Stage-Solo-Lab-Demo.md).  
Corporate tenant demo: [Office-Stage-Corp-Lab-Demo.md](../../docs/Office-Stage-Corp-Lab-Demo.md).

## Release / SKU NSIS

```powershell
# Shared release exe + assets
cargo build --release -p era-office-desktop
# Copy assets (or use build script post-step):
Copy-Item -Recurse apps\era-office-desktop\src-tauri\assets target\release\assets

# Per-SKU overlay → tauri build (identifier / associations / productName)
.\scripts\build-office-sku.ps1 -Sku docs
.\scripts\build-office-sku.ps1 -Sku tables
.\scripts\build-office-sku.ps1 -Sku presentations
.\scripts\build-office-sku.ps1 -Sku projects
.\scripts\build-office-sku.ps1 -Sku suite
# → target/release/bundle/nsis/  +  era-office-<sku>.cmd
```

Overlays: `src-tauri/tauri.conf.sku-*.json`. Base identifier suite: `az.era.office.desktop`.

Ship **exe + `assets/`** together (Solo SPAs).

### Store / EV checklist (ops)

See [Office-Stage-Solo-SKU-Distro.md](../../docs/Office-Stage-Solo-SKU-Distro.md).

## Specs

- [Office-Stage-S3-Solo-Spec.md](../../docs/Office-Stage-S3-Solo-Spec.md)
- [Office-Stage-Solo-Docs-FullUI.md](../../docs/Office-Stage-Solo-Docs-FullUI.md)
- [Office-Stage-Solo-Pres-Spec.md](../../docs/Office-Stage-Solo-Pres-Spec.md)
- [Office-Stage-Solo-Projects-Spec.md](../../docs/Office-Stage-Solo-Projects-Spec.md)
- [Office-Stage-Solo-Lab-Demo.md](../../docs/Office-Stage-Solo-Lab-Demo.md)
- [Office-Stage-Corp-Lab-Demo.md](../../docs/Office-Stage-Corp-Lab-Demo.md)
- [Office-Stage-Solo-SKU-Distro.md](../../docs/Office-Stage-Solo-SKU-Distro.md)
- [Office-Roadmap.md](../../docs/Office-Roadmap.md)

## Tests

```powershell
cargo test -p era-office-desktop --lib
cargo test -p era-pres-core --lib
cargo test -p era-projects-core --lib
```
