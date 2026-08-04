# ERA Service + Provision + Deploy/Patch (Stage 7)

Спецификация IT-Ops доставки: ITSM-lite, OS provisioning, software deploy/patch.

**Связано:** [ADR-0016 §3/§4](adr/0016-uem-scope-vs-ivanti.md) · лицензии `service`, `provision`, `manage`.  
**Статус кода:** server MVP ✅ (API + UI FileServer + tests/golden) · **GA-гейт:** field rollout PXE/TFTP на железе.

## Компоненты

| Компонент | Путь | Порт |
|---|---|---|
| Service Desk | `services/service-desk` | `:8122` |
| Provision | `services/provision` | `:8124` |
| Deploy/Patch API | `control-plane` `/api/v1/manage/*` | `:8090` |
| Deploy plugin | `crates/era-plugin-deploy` | on-demand |

## ERA Service (ITSM-lite)

- ITIL-модель: incident, request, problem, change (API + store parity)
- MVP UI: `ui/service-desk/` — served at `http://…:8122/ui/` (`ERA_UI_DIR`)
- CMDB link: `node_id` валидируется через control-plane при создании инцидента
- API: `/api/v1/incidents`, `/requests`, `/problems`, `/changes`, `/cmdb/assets`

## ERA Provision

- Каталог образов (MinIO refs): `GET /api/v1/images`, `GET /api/v1/images/{id}`
- PXE config (server catalog; TFTP appliance — field): `GET /api/v1/pxe/config` + golden `testdata/pxe_config.golden.json`
- Post-install enroll → CMDB: `POST /api/v1/enroll` (requires control-plane)
- UI: `ui/provision/` at `http://…:8124/ui/`

## Deploy / Patch (Manage)

- `POST /api/v1/manage/deploy/jobs` — rollout подписанного пакета
- `GET /api/v1/manage/patch/plan` — CVE-дельта (inventory × patch catalog)
- `POST /api/v1/manage/patch/jobs` — patch job
- `era-plugin-deploy` — verify OTA token + install path

## Compose

```bash
docker compose -f deploy/docker-compose.prod.yml --profile itops up -d service-desk provision
```

## Тесты / smoke

```powershell
.\scripts\run-itops-smoke.ps1
# go test ./services/service-desk/... ./services/provision/...
```

## Гейты

| Гейт | Статус |
|---|---|
| Server MVP (API, UI, golden, smoke unit) | ✅ |
| Полевой пилот-rollout provision/deploy (TFTP/DHCP/bare-metal) | [gate: field] |
