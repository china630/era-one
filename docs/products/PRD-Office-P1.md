# PRD: ERA Office — P1 (ERA Documents)

**Статус:** Draft (scope accepted)  
**Дата:** 7 июля 2026 г.  
**Продукт:** ERA Office (ERA One)  
**ADR:** [`0025`](../adr/0025-era-one-shared-platform.md) · [`0026`](../adr/0026-sovereign-office-engine.md)  
**Приёмка:** [`Office-Acceptance-System.md`](Office-Acceptance-System.md)

**Предусловие:** P0 O-GA закрыт (ERA Drive + Workspace).

---

## 1. Цель P1

Доказать **текстовые документы + co-editing + docx roundtrip** в контуре, без OnlyOffice/GPL.

**Native format:** `.erad` = wire type `erad` = `DocumentFormat.ERAD`.

**Не в P1:** Tables (`.erat`/P2), Presentations (`.erap`/P3), Office AI, Tauri/Flutter.

---

## 2. Scope

| # | Capability | Компонент |
|---|------------|-----------|
| 1 | **ERA Documents** — create, edit, co-edit (2+ users) | `platform/docs-engine` + `ui/docs` |
| 2 | Native `.erad` | `proto/era/v1/office.proto` + golden |
| 3 | Import/export **docx** (Rust, zero GPL) | `docs-engine/convert` |
| 4 | Authoritative storage только в Drive | drive bind API |
| 5 | Comms deep link «Edit in Documents» | ADR-0027 |

---

## 3. Критерии приёмки (AC-O1…O8)

| ID | Критерий | Доказательство |
|----|----------|----------------|
| AC-O1 | ≥2 WS clients; OpLog append order + peer fan-out (не OT/CRDT; ADR-0026 O-H-1); WS JWT | `ws_coedit` |
| AC-O2 | docx из `testdata/` → native → export → golden | `cargo test` golden |
| AC-O3 | Authoritative blob только в Drive | unit + integration |
| AC-O4 | login → Drive → open doc → edit | e2e |
| AC-O5 | Zero GPL в P1 runtime | SBOM CI gate |
| AC-O6 | Fuzz docx import | `cargo fuzz` |
| AC-O7 | Без `office-documents` license — create/open 403 | licensegate test |
| AC-O8 | Comms deep link «Редактировать в Documents» | integration |

---

## 4. Волны

См. [`Office-Sprint-Index.md`](../Office-Sprint-Index.md): O1-GOV → O1-1…O1-8 → O1-GA.

---

## 5. Bundle

**ERA Office MVP** = Drive + Documents ([`editions-office.yaml`](../../editions-office.yaml) `office-mvp`).

---

## 6. Связано

- [`PRD-Office-MVP.md`](PRD-Office-MVP.md)
- [`PRD-Office-P0.md`](PRD-Office-P0.md)
- [`Office-Pilot-Gap-List.md`](../Office-Pilot-Gap-List.md) §5
