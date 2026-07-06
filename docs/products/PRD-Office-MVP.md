# PRD: ERA Office — MVP

**Статус:** Draft (Accepted scope)  
**Дата:** 5 июля 2026 г.  
**Продукт:** ERA Office (ERA One)  
**ADR:** [`0025`](../adr/0025-era-one-shared-platform.md) · [`0026`](../adr/0026-sovereign-office-engine.md)

---

## 1. Цель MVP

Доказать суверенный контур **файлы + текстовые документы + co-editing** без облачных
SaaS и без сторонних office-лицензий.

**Не в MVP:** Tables, Presentations, Projects, Sign (browser-only stub), Office AI.

---

## 2. Scope

### In scope (P0 → P1)

| # | Capability | Компонент |
|---|------------|-----------|
| 1 | Shared identity (SSO) | `platform/identity`, `platform/tenant` |
| 2 | **ERA Drive** — upload, folders, ACL, versions | `platform/drive` |
| 3 | **ERA Workspace** shell — `app.customer.local` | `platform/workspace` |
| 4 | **ERA Documents** — create, edit, co-edit (2+ users) | `platform/docs-engine` + UI |
| 5 | Native format `.era-doc` | proto + golden |
| 6 | Import/export **docx** (Rust parsers, zero GPL) | `docs-engine/convert` |
| 7 | Integration hook: Mail attach → Drive (API) | ADR-0027 contract |

### Out of scope (post-MVP)

- ERA Tables, Presentations, Projects
- ERA Sign (full ASAN/SİMA)
- ERA Office AI
- Tauri desktop, Flutter mobile
- OOXML macro / legacy fidelity beyond golden corpus

---

## 3. Лицензирование

| Издание | MVP |
|---------|-----|
| **ERA Drive** | ✓ (отдельная строка RFQ; **включён** в Office Suite) |
| **ERA Documents** | ✓ |
| ERA Office Suite | Drive + Documents (MVP bundle) |
| Identity | Включена; не продаётся отдельно |

---

## 4. Критерии приёмки (AC)

| ID | Критерий | Доказательство |
|----|----------|----------------|
| AC-O1 | Два пользователя co-edit одного `.era-doc` в контуре | e2e / integration test |
| AC-O2 | Import типового docx из `testdata/` → native → export → golden match | golden test |
| AC-O3 | Файл authoritative только в Drive; engine не хранит копию | unit + ADR review |
| AC-O4 | Workspace: login → Drive → open doc | manual / e2e stub |
| AC-O5 | Zero GPL in runtime (SBOM / dependency audit) | CI gate |

---

## 5. Зависимости

- Shared platform ADR-0025 deployed
- Postgres (tenant + identity)
- MinIO (Drive objects)
- **Не требует** ERA Core

---

## 6. Риски

| Риск | Mitigation |
|------|------------|
| OOXML fidelity | Golden corpus AZ templates; disclaimer UI |
| Engine timeline | P0 Drive+shell first; Documents P1 |
| Parser security | Fuzz Rust parsers |

---

## 7. Связанные документы

- [`ERA-Office-Vision.md`](ERA-Office-Vision.md)
- [`editions-office.yaml`](../../editions-office.yaml)
- [`editions-shared.yaml`](../../editions-shared.yaml)
