# PRD: ERA Office — P0 (Drive + Workspace)

**Статус:** Draft (scope accepted)  
**Дата:** 7 июля 2026 г.  
**Продукт:** ERA Office (ERA One)  
**ADR:** [`0025`](../adr/0025-era-one-shared-platform.md) · [`0026`](../adr/0026-sovereign-office-engine.md)  
**Приёмка:** [`Office-Acceptance-System.md`](Office-Acceptance-System.md)

---

## 1. Цель P0

Доказать суверенный контур **файлы (ERA Drive) + единая оболочка (ERA Workspace)** без облачных SaaS.

**Не в P0:** ERA Documents, co-editing, `.erad` body, docx I/O (→ P1).

---

## 2. Scope

| # | Capability | Компонент |
|---|------------|-----------|
| 1 | Shared identity (SSO) | `platform/identity`, `platform/tenant` |
| 2 | **ERA Drive** — upload, folders, ACL, versions | `platform/drive` |
| 3 | **ERA Workspace** — `app.customer.local`, `/drive` | `platform/workspace` |
| 4 | Integration hook: Mail attach → Drive | ADR-0027 |

---

## 3. Критерии приёмки (AC-O0)

| ID | Критерий | Доказательство |
|----|----------|----------------|
| AC-O0-1 | Upload → list → download файла в tenant с ACL | integration test drive-api |
| AC-O0-2 | Папки + версионирование объекта | unit + integration |
| AC-O0-3 | OIDC login → Workspace `/drive` — список файлов | e2e / Playwright |
| AC-O0-4 | Без лицензии `platform-drive` — Drive API 403 | licensegate test |
| AC-O0-5 | Mail attach hook → deep link на объект Drive (при лицензии) | ui/mail integration |
| AC-O0-6 | Deploy profile `office.yaml` → implemented + health green | compose smoke |
| AC-O0-7 | Zero GPL в P0 runtime | SBOM CI gate |

---

## 4. Волны

См. [`Office-Sprint-Index.md`](../Office-Sprint-Index.md): O-0…O-6 → O-GA.

---

## 5. Связано

- [`PRD-Office-MVP.md`](PRD-Office-MVP.md) — полный MVP (P0+P1)
- [`PRD-Office-P1.md`](PRD-Office-P1.md) — Documents
- [`Office-Pilot-Gap-List.md`](../Office-Pilot-Gap-List.md)
