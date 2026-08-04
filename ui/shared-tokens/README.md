# ERA shared design tokens

**SSOT:** this directory (`ui/shared-tokens/`).  
**Canon:** [`docs/ERA-UI-Shell-Theme-Matrix.md`](../../docs/ERA-UI-Shell-Theme-Matrix.md)

## Load order

1. `era-tokens-base.css` — semantic names + light suite defaults + layout tokens  
2. `era-theme-{office|control|comms}.css` — line surfaces / accents  
3. `era-theme-sku.css` — SKU accent overrides  
4. Product shell CSS (`office.css` layout/components, `control.css`, `mail.css`)

## Sync

```powershell
pwsh -File scripts/sync-era-tokens.ps1
```

Copies into:

- `ui/office-shell/web/tokens/`
- `ui/control-shell/web/tokens/`
- `ui/mail/web/tokens/`

Desktop picks up office-shell copy via `apps/era-office-desktop/src-tauri/build.rs`.

Do **not** hand-edit copies under shells — edit here, then sync.
