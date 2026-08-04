# Office Stage — Corporate Lab Demo (shell → tenant)

**Targets: Browser ❌ · Solo ❌ (chooser) · Corporate ✅**  
**Status:** [x] 2026-08-03 — runbook for B-corp v1 shell.  
**Not:** Hybrid LocalFs↔Drive · local engines on desktop · Pilot field RT-O09 close · TE sign-off for shell alone.

## Purpose

Demo Corporate desktop as “Outlook to Exchange”: Tauri WebView loads **tenant Workspace**; SSO = tenant `/login` in the same WebView. Content quality = Platform A maturity ([Product-Readiness](Office-Product-Readiness-Matrix.md) Platform rows).

## Prerequisites

- WebView2
- Reachable staging tenant Workspace URL (HTTPS preferred)
- Built desktop: `cargo build --release -p era-office-desktop` or lab pack from Solo (same exe; Corporate via profile)

## Quick start

### UI first-run

1. Launch `era-office-desktop.exe` (suite / default SKU).
2. Choose **Corporate**.
3. Enter Workspace base URL (e.g. `https://office.staging.example`).
4. Connect → WebView navigates to `{server_url}/drive/`.
5. Sign in at tenant `/login` if prompted.
6. Open Docs / Tables / Pres / Projects from Drive as in browser.

### Env (skip chooser)

```powershell
$env:ERA_OFFICE_PROFILE = "corporate"
$env:ERA_OFFICE_SERVER_URL = "https://office.staging.example"
.\era-office-desktop.exe
```

Config file: `{config_dir}/era-office-desktop/config.json` (`profile`, `server_url`).

### Deep links

```powershell
.\era-office-desktop.exe "era-office://open?path=/drive/"
.\era-office-desktop.exe "era-office://open?path=/docs/{id}"
.\era-office-desktop.exe "era-office://open?url=https%3A%2F%2Ftenant%2Fdocs%2F..."
```

Corporate profile resolves relative `path=` against `server_url`. Absolute `http(s)` args accepted.

## Checklist

| # | Step | Pass? |
|---|------|-------|
| 1 | Persist Corporate + URL; relaunch opens Workspace | [ ] |
| 2 | Bad URL shows error / does not hang forever | [ ] |
| 3 | Login → Drive list works (Platform AC-O4 path) | [ ] |
| 4 | Open `.erad` / sheet from Drive | [ ] |
| 5 | Deep link `path=/drive/` lands on Drive | [ ] |
| 6 | Switch back to Solo profile still works | [ ] |

## Dependency matrix (honesty)

| Corporate claim | Depends on Platform |
|-----------------|---------------------|
| Demo / TE | Same TE-D / TE-DOC / TE-T / TE-P / TE-PR as browser |
| Pilot field | RT-O09 (and related) — **not** closed by this shell |
| Sell “desktop Office” | Shell ✅ + Platform sell/show rows; not a second product TE |

MDM / managed install: ship config or set `ERA_OFFICE_SERVER_URL` via policy; see ops note in [SKU-Distro](Office-Stage-Solo-SKU-Distro.md) (Corporate MDM checkbox).

## Related

- [Office-Stage-S4-Corp-Spec.md](Office-Stage-S4-Corp-Spec.md) (AC scaffold)
- [Office-Product-Readiness-Matrix.md](Office-Product-Readiness-Matrix.md) § Desktop
- [Office-Roadmap.md](Office-Roadmap.md) §1.1–1.2
- [Office-Stage-Solo-Lab-Demo.md](Office-Stage-Solo-Lab-Demo.md) (Solo portable)
