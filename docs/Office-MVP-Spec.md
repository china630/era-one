# ERA Office — MVP Program Specification

**Версия:** 2.1  
**Дата:** 30 июля 2026 г.  
**Статус:** Active — Product Readiness **🟡/❌** (см. [`Office-Product-Readiness-Matrix.md`](Office-Product-Readiness-Matrix.md)); TE/UI open; gates `gate[x]`; RT-O09 open · канон v1.3
**PRD:** [`PRD-Office-MVP.md`](products/PRD-Office-MVP.md)  
**Приёмка:** [`Office-Acceptance-System.md`](products/Office-Acceptance-System.md)  
**Evidence:** [`Office-Evidence-Rules.md`](Office-Evidence-Rules.md)  
**Индекс:** [`Office-Sprint-Index.md`](Office-Sprint-Index.md)  
**Matrix:** [`Office-Implementation-Matrix.md`](Office-Implementation-Matrix.md)

---

## 1. Целевая поставка

| Параметр | Значение |
|----------|----------|
| **MVP** | Drive + Documents + co-edit + docx (AC-O1…O5) |
| **Контур** | On-prem, air-gap; MinIO + Postgres |
| **Standalone** | Не требует ERA Core |
| **Native** | `.erad` / `.erat` / `.erap` / `.eraj` (Projects board) |
| **Bundle** | `office-mvp` = Drive + Documents |

---

## 2. Волны (канон)

| Волна | Фокус | PRD | Gate | AC Scaffold (Matrix) | Pilot-ready |
|-------|-------|-----|------|----------------------|-------------|
| **O-0** | Acceptance + proto SSOT | — | [x] | — | [ ] |
| **O-1** | ERA Drive runnable | AC-O3 | [x] | 🟡 header-trust | [ ] |
| **O-2** | Workspace + identity-api | AC-O4 | [x] | 🟡 soft auth | [ ] |
| **O-3** | Documents `.erad` + editor | AC-O3/O4 | [x] | 🟡 | [ ] |
| **O-4** | OpLog + insert OT lite (не full CRDT) | AC-O1 | [x] | ✅ char-safe + WS OT lite | [ ] |
| **O-5** | docx golden + SBOM | AC-O2, AC-O5 | [x] | ✅ | [ ] |
| **O-GA** | Pilot honesty | AC-O1…O8 | [x] | mixed ✅/🟡 | [ ] RT-O09 |

Post-MVP O-T / O-P / O-PR / O-AI: **Scaffold gates `[x]`**, Pilot-ready open, editions `mvp` — не «roadmap only» ([`Office-Sprint-Index.md`](Office-Sprint-Index.md) §2).

**Gate:** `.\scripts\run-office-stage-gate.ps1 -Stage <wave>`

---

## 3. Definition of Done — O-0

| ID | Критерий | Статус |
|----|----------|--------|
| F-O0-1 | Acceptance + Evidence + Index + Matrix | [x] |
| F-O0-2…5 | DriveService + gen-proto + era-proto + golden | [x] |
| F-O0-6 | Gate O-0 PASS | [x] `reports/office-stage-O-0-20260730-001327.log` |

## 4. Definition of Done — O-1

| ID | Критерий | Статус |
|----|----------|--------|
| F-O1-1…6 | migration, drive-api, compose, mail hook, license, editions scaffold | [x] |
| Gate | O-1 PASS | [x] `reports/office-stage-O-1-20260730-001413.log` |

---

## 5. Definition of Done — O-4

| ID | Критерий | Статус |
|----|----------|--------|
| F-O4-1…4 | ws_coedit peer fan-out + sync unit + ui remote apply | [x] |
| Gate | O-4 PASS | [x] `reports/office-stage-O-4-20260730-003207.log` |

---

## 6. Definition of Done — O-5

| ID | Критерий | Статус |
|----|----------|--------|
| F-O5-1…4 | golden_docx + corpus + fuzz smoke + SBOM | [x] |
| Gate | O-5 PASS | [x] `reports/office-stage-O-5-20260730-004202.log` |

---

## 7. Definition of Done — O-GA

| ID | Критерий | Статус |
|----|----------|--------|
| F-OGA-1…5 | regression + honesty docs + editions `mvp` + Pilot-ready open | [x] |
| Gate | O-GA PASS (honesty) | [x] `reports/office-stage-O-GA-20260730-004758.log` |
| Field | RT-O09 | [ ] open |

---

## 8. Связано

- ADR-0025 / ADR-0026  
- Stash recovery: [`reports/office-stash-audit-20260730.md`](../reports/office-stash-audit-20260730.md)
