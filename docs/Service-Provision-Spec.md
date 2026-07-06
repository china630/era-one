# ERA Service + Provision + Deploy/Patch (Stage 7)

Спецификация IT-Ops доставки: ITSM-lite, OS provisioning, software deploy/patch.

**Связано:** [ADR-0016 §3/§4](adr/0016-uem-scope-vs-ivanti.md) · лицензии `service`, `provision`, `manage`.

## Компоненты

| Компонент | Путь | Порт |
|---|---|---|
| Service Desk | `services/service-desk` | `:8122` |
| Provision | `services/provision` | `:8124` |
| Deploy/Patch API | `control-plane` `/api/v1/manage/*` | `:8090` |
| Deploy plugin | `crates/era-plugin-deploy` | on-demand |

## ERA Service (ITSM-lite)

- ITIL-модель: incident, request, problem, change
- MVP UI: `ui/service-desk/` — incidents + портал заявок
- CMDB link: `node_id` валидируется через control-plane при создании инцидента
- API: `/api/v1/incidents`, `/requests`, `/problems`, `/changes`, `/cmdb/assets`

## ERA Provision

- Каталог образов (MinIO refs): `GET /api/v1/images`
- PXE config (simulated TFTP): `GET /api/v1/pxe/config`
- Post-install enroll → CMDB: `POST /api/v1/enroll`
- UI: `ui/provision/`

## Deploy / Patch (Manage)

- `POST /api/v1/manage/deploy/jobs` — rollout подписанного пакета
- `GET /api/v1/manage/patch/plan` — CVE-дельта (inventory × patch catalog)
- `POST /api/v1/manage/patch/jobs` — patch job
- `era-plugin-deploy` — verify OTA token + simulated install

## Compose

```bash
docker compose -f deploy/docker-compose.prod.yml --profile itops up -d service-desk provision
```

## Тесты

- `go test ./services/service-desk/... ./services/provision/... ./services/control-plane/...`
- `cargo test -p era-plugin-deploy`

## Гейты

| Гейт | Статус |
|---|---|
| Полевой пилот-rollout provision/deploy | [gate: external/field] |

Код этапа закрывается на стенде без полевого rollout.
