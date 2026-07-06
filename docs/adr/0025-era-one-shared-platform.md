# ADR-0025: ERA One Shared Platform (identity, drive, co-editing, sign)

**Статус:** Accepted  
**Дата:** 5 июля 2026 г.  
**Контекст:** Продуктовые семейства ERA Control, ERA Communications и ERA Office
(ADR-0024) разделяют инфраструктурные сервисы: файлы, идентичность, co-editing,
электронная подпись. Нужна явная граница «platform layer vs product editions».

**Связано:** [`ADR-0024`](0024-era-one-product-families.md) · [`ADR-0026`](0026-sovereign-office-engine.md) ·
[`ADR-0027`](0027-era-communications-architecture.md) · [`products.yaml`](../../products.yaml) ·
[`editions-shared.yaml`](../../editions-shared.yaml)

---

## 1. Решение (одной фразой)

**Shared Platform** — технический слой ERA One (сервисы в контуре + API), **не**
отдельное продуктовое семейство в RFQ. Лицензируются **capabilities** через издания
продуктов (ERA Drive, ERA Documents, ERA Sign и т.д.). **Identity** входит в любой
продукт ERA One и **не продаётся** отдельной строкой.

---

## 2. Компоненты shared platform

| Пакет / сервис | Назначение | Лицензия |
|----------------|------------|----------|
| `platform/identity` | Users, groups, OIDC, RBAC (паттерн Zitadel; **своя** реализация) | Включена в любой продукт |
| `platform/tenant` | Tenants, orgs, domains, multi-tenancy | Включена |
| `platform/adminportal` | Admin shell, `/api/v1/products` | Включена |
| `platform/licensegate` | Проверка модулей (ADR-0010) | Включена |
| `platform/drive` | Файлы, версии, ACL, sync API (паттерн oCIS; **свой** стек MinIO + metadata) | **ERA Drive** |
| `platform/docs-engine` | CRDT co-editing, sync (см. ADR-0026) | **ERA Documents / Tables / …** |
| `platform/signing` | ASAN Imza, SİMA adapters | **ERA Sign** |
| `platform/workspace` | User shell BFF (`app.customer.local`) | Включена с любым user-facing продуктом |

**Инвариант air-gap:** все сервисы platform работают **в контуре заказчика**; phone-home
запрещён (см. workspace rules, ADR-0018 для hybrid control plane).

---

## 3. ERA Drive

- **Издание:** `era-drive` в [`editions-shared.yaml`](../../editions-shared.yaml).
- **Продаётся отдельно** — нужен и Office, и Communications (вложения, общие папки).
- **ERA Office Suite** всегда **включает** ERA Drive (bundle, не опция).
- **ERA Mail Server (MVP):** без Drive — **упрощённый inline storage** вложений (quota);
  с лицензией Drive — attach/save через Drive API.
- **Storage:** MinIO (object) + metadata/ACL на Go/Rust; donor — **идеи** oCIS, не код
  (ADR-0003).

---

## 4. Co-editing

- **Владелец:** `platform/docs-engine` (CRDT + WebSocket sync).
- **Лицензия:** издания ERA Office (Documents, Tables, Presentations).
- **ERA Communications** co-editing **не реализует**; только интеграция:
  «Редактировать в ERA Documents» (deep link), если у tenant есть лицензия Office
  (см. ADR-0027).

---

## 5. ERA Sign

- **Сервис:** `platform/signing` — adapters AZ (ASAN Imza, SİMA).
- **Каналы:** browser (WebCrypto + bridge), desktop helper (Phase 2), mobile (Phase 3).
- **Лицензия:** издание **ERA Sign** (Office line; roadmap).
- Desktop helper — **не** ERA Control agent; отдельный lightweight installer.

---

## 6. ERA Workspace (user shell)

- **URL:** `https://app.customer.local` — единый origin для Mail, Chat, Drive, Documents.
- **Маршруты:** `/mail`, `/drive`, `/docs`, `/chat`, … (lazy-loaded micro-frontends).
- **ERA Control SOC:** `app.customer.local/secure` (MVP) или `secure.customer.local` (позже).
- **Стек:** React + shared design system (Variant C, ADR-0024); desktop — Tauri (Phase 2);
  mobile — Flutter (Phase 3).
- **Admin:** `admin-portal` (:8140) — отдельная оболочка.

---

## 7. Standalone vs ERA Control

ERA Communications и ERA Office **не требуют** ERA Core / endpoint agent.

| Продукт | Зависимость от ERA Control |
|---------|---------------------------|
| ERA Communications | Нет (optional: audit → SIEM ingest) |
| ERA Office | Нет |
| ERA Control | — |
| ERA One Full Stack | Bundle всех трёх + shared platform |

Hybrid (ADR-0018): **ERA Cloud Portal + Relay** — общая **операционная** модель;
не означает установку ERA Core вместе с почтой.

---

## 8. AI runtime (infra vs editions)

| Издание | Данные | Сервис |
|---------|--------|--------|
| **ERA Control AI** | телеметрия, кейсы | `ai-core` |
| **ERA Comms AI** | почта, встречи | `comms/ai` |
| **ERA Office AI** | документы (roadmap) | `platform/docs-ai` |

**Ollama/vLLM** — shared infra в контуре; политики и prompts — per product edition.

---

## 9. Deploy

Расширение `deploy/profiles/`:

- `shared-platform.yaml` — identity, tenant, drive, docs-engine, signing (planned)
- Профили `comms.yaml`, `office.yaml`, `era-one-full.yaml` — `shared_platform.required: true`

Entrypoint workspace (planned): `services/platform/cmd/workspace`.

---

## 10. Последствия

**Плюсы:** один Drive, один co-edit engine, один SSO; cross-sell Office из Comms.

**Обязательства:** proto/API для Drive и docs-engine; license checks на platform boundary;
golden-тесты сериализации и ACL.

---

## 11. Артеfactы

- [`editions-shared.yaml`](../../editions-shared.yaml)
- [`docs/products/PRD-Office-MVP.md`](../products/PRD-Office-MVP.md)
- [`docs/products/PRD-Comms-MVP.md`](../products/PRD-Comms-MVP.md)
