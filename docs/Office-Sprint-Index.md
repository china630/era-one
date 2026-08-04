# ERA Office — индекс исполняемых этапов

**Версия:** 2.4  
**Дата:** 3 августа 2026 г.  
**Приёмка:** [`Office-Acceptance-System.md`](products/Office-Acceptance-System.md)  
**Evidence:** [`Office-Evidence-Rules.md`](Office-Evidence-Rules.md)  
**Канон:** [`ERA-Product-Acceptance-Standard.md`](products/ERA-Product-Acceptance-Standard.md)  
**Программа:** [`Office-MVP-Spec.md`](Office-MVP-Spec.md)  
**Matrix AC:** [`Office-Implementation-Matrix.md`](Office-Implementation-Matrix.md)  
**Stash audit:** [`reports/office-stash-audit-20260730.md`](../reports/office-stash-audit-20260730.md)

Канон волн MVP (план 2026-07-30): **O-0…O-GA**. Legacy fine-grained specs из stash сохранены в §4 (справочно).

> **Honesty (канон v1.3):** wave = **gate[x]** only. **Готовность продукта** → [`Office-Product-Readiness-Matrix.md`](Office-Product-Readiness-Matrix.md) (UI/TE/Pilot). AC BE → Implementation-Matrix.

---

## 1. Карта этапов — Office MVP

| # | Wave | Spec | Backlog | PRD AC | Gate | Статус |
|---|------|------|---------|--------|------|--------|
| 1 | **O-0** | [`Office-Stage-O0-Spec.md`](Office-Stage-O0-Spec.md) | OM0-* | — (acceptance+proto) | `run-office-stage-gate.ps1 -Stage O-0` | [x] scaffold gate |
| 2 | **O-1** | [`Office-Stage-O1-Spec.md`](Office-Stage-O1-Spec.md) | OM1-* | AC-O3 | `-Stage O-1` | [x] scaffold · AC-O3 Matrix 🟡 |
| 3 | **O-2** | [`Office-Stage-O2-Spec.md`](Office-Stage-O2-Spec.md) | OM2-* | AC-O4 (login→Drive) | `-Stage O-2` | [x] scaffold · AC-O4 Matrix 🟡 |
| 4 | **O-3** | [`Office-Stage-O3-Spec.md`](Office-Stage-O3-Spec.md) | OM3-* | AC-O4 full, AC-O3 | `-Stage O-3` | [x] scaffold · AC-O3/O4 Matrix 🟡 |
| 5 | **O-4** | [`Office-Stage-O4-Spec.md`](Office-Stage-O4-Spec.md) | OM4-* | AC-O1 | `-Stage O-4` | [x] scaffold · AC-O1 Matrix ✅ (char-safe + insert OT lite; not full CRDT) |
| 6 | **O-5** | [`Office-Stage-O5-Spec.md`](Office-Stage-O5-Spec.md) | OM5-* | AC-O2, AC-O5 | `-Stage O-5` | [x] scaffold · AC-O2/O5 Matrix ✅ |
| 7 | **O-GA** | [`Office-Stage-OGA-Spec.md`](Office-Stage-OGA-Spec.md) | OM-GA-* | AC-O1…O8 | `-Stage O-GA` | [x] honesty; Matrix mixed ✅/🟡; RT-O09 open |

```mermaid
flowchart LR
  O0[O-0] --> O1[O-1] --> O2[O-2] --> O3[O-3] --> O4[O-4] --> O5[O-5] --> OGA[O-GA]
```

**Фаза 1:** O-0…O-GA scaffold gates `[x]` 2026-07-30; **PRD AC не все ✅** (см. Matrix). Field RT-O09 open. **Post-MVP:** O-T…O-AI + B + O-H.

**Desktop path (подтверждение на каждый шаг):** [`Office-Roadmap.md`](Office-Roadmap.md) §0 —  
S0 docs Solo/Corp `[x]` → **S1** Lite/Missing `[x]` → **S2** core split (B0) `[x]` → **S3** Solo Documents (B1) `[x]` → **S4** Corp shell `[x]` → **S5** Tables+bundle + B4 Pres/Projects/SKU `[x]`.  
Готовность Solo/Corporate → [`Office-Product-Readiness-Matrix.md`](Office-Product-Readiness-Matrix.md) § Desktop.

---

## 2. Post-MVP waves

Все `[x]` ниже = **Scaffold / auto-gate only**; Pilot-ready / edition `ga` **не** закрыты.

| Wave | Издание | Spec / focus | Scaffold | Pilot-ready |
|------|---------|--------------|----------|-------------|
| **O-T-0…TE** | ERA Tables | OT0 Spec + tables-engine + ui/tables | [x] gates | [ ] |
| **O-P-0…4** | ERA Presentations | PRD-P3 + presentations-engine | [x] | [ ] |
| **O-PR-0…3** | ERA Projects | PRD + docs-projects | [x] | [ ] |
| **O-AI-0…3** | ERA Office AI | PRD + docs-ai air-gap | [x] | [ ] |
| **O-AUTH / O-*-H** | JWT+license harden | presentations / projects / docs-ai | [x] unit | [ ] |
| **B** | products.yaml | `era-office` → `mvp` (not ga) | [x] | n/a |
| **Desktop S0…S5** | Tauri Solo + Corporate + Store SKUs | [`Office-Roadmap.md`](Office-Roadmap.md) §0–§3 · [Lab-Demo](Office-Stage-Solo-Lab-Demo.md) · [Corp-Lab-Demo](Office-Stage-Corp-Lab-Demo.md) · [SKU-Distro](Office-Stage-Solo-SKU-Distro.md) | S0–S5 [x]; B4 + lab pack/SKU matrix [x] 2026-08-03 | [ ] Store publish / EV |
| **O-H-1…4** | Hardening | OpLog note, docx corpus, CycloneDX, e2e files | [x] | [ ] |
| **O-FMT-0** | UI canon | [Controls Catalog](Office-UI-Controls-Catalog.md) + Menu-Map/Inventory | [x] gate | n/a |
| **O-FMT-1** | Documents MS-class | styles / para / lists / painter / super-sub | [x] gate | [ ] |
| **O-FMT-2** | Tables + Docs polish | Excel cell chrome; ruler / table dialog | [x] | [x] |
| **O-FMT-3** | Presentations MS-class | duplicate / Format text / image | [x] | [x] |
| **O-MS** | Motion / Gantt / Comments / Drive New | [Office-Stage-OMS-Spec.md](Office-Stage-OMS-Spec.md) | [x] | [ ] |
| **O-SHELL** | Shell polish + Drive trash/multi + auth | [Office-Stage-OSHELL-Spec.md](Office-Stage-OSHELL-Spec.md) | [x] | [ ] |

```mermaid
flowchart LR
  OGA[O-GA] --> OT[O-T] --> OP[O-P] --> OPR[O-PR] --> OAI[O-AI] --> OB[B] --> OH[O-H]
  OH --> OFMT0[O-FMT-0] --> OFMT1[O-FMT-1] --> OFMT2[O-FMT-2] --> OFMT3[O-FMT-3]
  OFMT3 --> OMS[O-MS] --> OSHELL[O-SHELL]
```

---

## 3. Stage Gate (G1…G6)

| # | Проверка |
|---|----------|
| G1 | `run-office-stage-gate.ps1 -Stage <wave>` PASS |
| G2 | E2E → `reports/office-stage-<wave>-e2e.log` (если § в spec) |
| G3 | `Office-Implementation-Matrix.md` в том же PR |
| G4 | `Office-MVP-Spec.md` / этот Index → `[x]` только с proof |
| G5 | editions — только если license/deploy изменился |
| G6 | `-WriteSignoff` → `reports/office-stage-<wave>-signoff.md` |

```powershell
.\scripts\run-office-stage-gate.ps1 -Stage O-0 -WriteSignoff
.\scripts\run-office-stage-gate.ps1 -Stage O-1
```

---

## 4. Legacy fine-grained specs (восстановлены из stash)

Справочно; **не** канон статуса. Маппинг → канон:

| Legacy wave | Канон |
|-------------|-------|
| O-GOV + O-0 docs + O-1 proto | **O-0** |
| O-2 Drive (+ O-6 compose slice) | **O-1** |
| O-3 Identity + O-4 Workspace | **O-2** |
| O1-1…O1-6 Documents | **O-3** / **O-4** |
| O1-2 docx + SBOM | **O-5** |
| O-GA / O1-GA | **O-GA** |

Файлы: `Office-Stage-O0-Docs-Spec.md`, `O1-Proto`, `O2-Drive`, … `O1-*-Spec.md`, [`Office-Roadmap.md`](Office-Roadmap.md).
