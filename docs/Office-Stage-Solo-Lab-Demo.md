# Office Stage — Solo Lab Demo (portable pack)

**Targets: Browser ❌ · Solo ✅ · Corporate ❌**  
**Status:** [x] 2026-08-03 — pack + headless smoke scripts; GUI checklist for humans.  
**Not:** Tech Eval sign-off · Store listing · EV publish · Pilot field.

## Purpose

One person without IDE can demo Docs / Tables / Presentations / Projects from a portable folder.

## Pack

```powershell
.\scripts\pack-office-solo-lab.ps1          # build release + dist/office-solo-lab/
.\scripts\pack-office-solo-lab.ps1 -Zip     # also dist/office-solo-lab.zip
.\scripts\pack-office-solo-lab.ps1 -SkipBuild -Zip  # reuse existing release exe
```

| Artifact | Role |
|----------|------|
| `dist/office-solo-lab/era-office-desktop.exe` | Shared binary |
| `dist/office-solo-lab/assets/` | Solo SPAs (must sit next to exe) |
| `era-office-{docs,tables,presentations,projects,suite}.cmd` | SKU launchers |
| `README-LAB.txt` | Short operator note |

Requires **WebView2** on the demo machine.

## Headless smoke (CI / pre-demo)

```powershell
.\scripts\smoke-office-solo-lab.ps1
# → reports/office-solo-lab-smoke-*.log
# cargo test -p era-office-desktop --lib
# cargo test -p era-pres-core --lib
# cargo test -p era-projects-core --lib
```

| ID | Criterion | Evidence |
|----|-----------|----------|
| AC-LAB-1 | Pack dir has exe + assets + 5 `.cmd` | `pack-office-solo-lab.ps1` |
| AC-LAB-2 | Unit smoke PASS | `smoke-office-solo-lab.ps1` log |

## GUI checklist (manual)

Run from `dist/office-solo-lab/` (or zip extract). Mark each after demo:

| # | Step | Pass? |
|---|------|-------|
| 1 | `era-office-suite.cmd` → hub with 4 product links | [ ] |
| 2 | `era-office-docs.cmd` → Docs (no hub); File Open/Save `.erad` | [ ] |
| 3 | `era-office-tables.cmd` → Tables; Open/Save `.erat` | [ ] |
| 4 | `era-office-presentations.cmd` → Pres; edit slide; Export pptx | [ ] |
| 5 | `era-office-projects.cmd` → Projects; add task; Save `.eraj` | [ ] |
| 6 | Restart Projects SKU → tasks still on disk after Open | [ ] |
| 7 | Suite running → second launch with `deck.erap` path focuses / opens Pres | [ ] |
| 8 | Argv `era-office://open?path=/tables/solo` routes to Tables | [ ] |
| 9 | Mail / Office AI nav blocked or no-op in Solo | [ ] |

Demo license caps (save blocked over limit without `ERA_SOLO_LICENSE`): Docs ≤5 blocks · Tables ≤25 cells · Pres ≤5 slides · Projects ≤15 tasks.

## Honesty

- Lab / PR demo only — **not** TE-DOC/TE-T/TE-P sign-off.
- **Not** Microsoft Store / Partner Center listing.
- Corporate profile: see [Office-Stage-Corp-Lab-Demo.md](Office-Stage-Corp-Lab-Demo.md).

## Related

- [Office-Stage-Solo-SKU-Distro.md](Office-Stage-Solo-SKU-Distro.md)
- [Office-Product-Readiness-Matrix.md](Office-Product-Readiness-Matrix.md) § Desktop
- [apps/era-office-desktop/README.md](../apps/era-office-desktop/README.md)
