# ERA Office — Tech Eval Runbook (TE-0)

**Аудитория:** технарь, DevOps  
**Цель:** поднять демо-стенд за ~30 минут для [`Office-Tech-Eval-Checklist.md`](Office-Tech-Eval-Checklist.md)

---

## 1. Требования

- Docker Desktop / Docker Engine + Compose v2
- Порты свободны: **5433** (Postgres), **8160** (identity), **8170** (workspace), **8175** (drive), **8142–8146** (engines), **9000** (MinIO)
- Windows: PowerShell 5.1+

---

## 2. Быстрый старт (полный TE-стенд)

```powershell
cd <repo-root>
docker compose -f deploy/docker-compose.office.yml `
  --profile docs --profile office-engines up -d --wait --build

# Связность + JWT API smoke (лог в reports/)
.\scripts\office-te-connectivity.ps1 -RequireEngines
```

Без профилей поднимаются только postgres / identity / drive / workspace / admin — **Docs/Tables/Pres/Projects/AI не стартуют**.

| Сервис | Порт | Profile |
|--------|------|---------|
| Workspace BFF + UI | 8170 | (default) |
| identity-api | 8160 | (default) |
| drive-api | 8175 | (default) |
| docs-engine | 8142 | `docs` |
| tables-engine | 8143 | `office-engines` |
| presentations-engine | 8144 | `office-engines` |
| docs-projects | 8145 | `office-engines` |
| docs-ai (Office AI stub) | 8146 | `office-engines` |

**URL для показа:**

| Сервис | URL |
|--------|-----|
| Workspace | http://127.0.0.1:8170 |
| Drive UI | http://127.0.0.1:8170/drive/ |
| Documents | http://127.0.0.1:8170/docs/ |
| Tables | http://127.0.0.1:8170/tables/ |
| Presentations | http://127.0.0.1:8170/presentations/ |
| Projects | http://127.0.0.1:8170/projects/ |
| Office AI | http://127.0.0.1:8170/office-ai/ |

---

## 3. Demo tenant и auth

| Поле | Значение |
|------|----------|
| Email | `alice@mail.gov.az` |
| Password | `1234` |
| Tenant | `t-demo` |
| Login UI | http://127.0.0.1:8170/login (Google-like: email → password; Create account) |
| После входа | redirect `?next=` → Drive / Docs / …; `localStorage.era_token` |
| Token API | `POST /oauth2/staging/token` · register: `POST /oauth2/staging/register` (lab `ERA_IDENTITY_DEV=1`) |

**API через BFF:** всегда `Authorization: Bearer <access_token>`.  
Клиентские `X-ERA-Tenant` / `X-ERA-User` **отклоняются** Drive (JWT-only). Engines ходят в Drive с `ERA_DRIVE_SERVICE_TOKEN` + acting-as headers.

---

## 4. Карта прокси Workspace → backends

| Path | Backend env |
|------|-------------|
| `/oauth2/`, `/.well-known/` | `ERA_IDENTITY_API_URL` |
| `/api/v1/drive/` | `ERA_DRIVE_API_URL` |
| `/api/v1/docs/` | `ERA_DOCS_API_URL` |
| `/api/v1/tables/` | `ERA_TABLES_API_URL` |
| `/api/v1/presentations/` | `ERA_PRESENTATIONS_API_URL` |
| `/api/v1/projects/` | `ERA_PROJECTS_API_URL` |
| `/api/v1/docs-ai/` | `ERA_DOCS_AI_URL` |

Критичные операции TE:

- Drive: list/upload/`POST …/objects/{id}/versions` (PutVersion)
- Docs: create → WS sync → `POST …/snapshot` → reopen same id
- Tables: create → WS edit → debounced PutVersion → reopen
- Pres: create → `PUT …/:id` → PutVersion → reopen
- Projects: `/board`, `/tasks`
- Office AI: `/summarize`, `/rewrite` (stub без `ERA_OLLAMA_URL`)

---

## 5. Prod license check (опционально)

```powershell
docker compose -f deploy/docker-compose.office.yml -f deploy/docker-compose.office.prod.yml `
  --profile docs up -d drive-api docs-engine --force-recreate
# Ожидание: upload/create → 403 без лицензии
docker compose -f deploy/docker-compose.office.yml `
  --profile docs up -d drive-api docs-engine --force-recreate
```

---

## 6. Что сказать на демо (честно)

| Показываем | Оговорка |
|------------|----------|
| Drive — файлы, ACL, lock | TE-D* — живая подпись технаря |
| Documents — co-edit + snapshot | Не Word; см. [`ERA-Documents-vs-Word.md`](ERA-Documents-vs-Word.md) |
| Tables | Grid UI ready — TE-T sign-off открыт |
| Presentations | Thin deck, не PowerPoint |
| Projects | Internal kanban, не MS Project |
| Office AI | Air-gap stub; не cloud LLM |

---

## 7. Связано

- Checklist: [`Office-Tech-Eval-Checklist.md`](Office-Tech-Eval-Checklist.md)
- Strategy / gaps: [`Office-Tech-Eval-Strategy.md`](Office-Tech-Eval-Strategy.md), [`Office-Tech-Eval-Gap-List.md`](Office-Tech-Eval-Gap-List.md)
- Pilot ops: [`Office-Pilot-Runbook.md`](Office-Pilot-Runbook.md)
- Readiness: [`Office-Product-Readiness-Matrix.md`](Office-Product-Readiness-Matrix.md)
