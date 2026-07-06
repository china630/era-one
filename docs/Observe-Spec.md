# ERA Observe — Network Monitoring (Stage 9)

Agentless-мониторинг сети и интеграция с NMS (PRTG/Zabbix).

**Связано:** [ADR-0020](adr/0020-network-observe-cmdb-reconciliation.md) · лицензия `observe`.

## Компоненты

| Компонент | Путь | Порт |
|---|---|---|
| Observe API | `services/observe` | `:8132` |
| Ingest feed | `services/ingest-gateway` `/v1/ingest` | `:8089` |
| CMDB reconcile | `services/control-plane` `/api/v1/cmdb/network/assets` | `:8090` |
| Корреляция | `services/detection-engine` `era-observe-network-endpoint` | — |

## Path A — интеграция NMS

| Метод | Путь | Описание |
|---|---|---|
| POST | `/api/v1/webhooks/prtg` | JSON PRTG notification |
| POST | `/api/v1/webhooks/zabbix` | JSON Zabbix trigger |
| POST | `/api/v1/webhooks/syslog` | `host\|message` syslog line |

События → `Envelope` `EVENT_CATEGORY_NETWORK` → Kafka `xdr.network`.

## Path B — нативный MVP

| Метод | Путь | Описание |
|---|---|---|
| POST | `/api/v1/snmp/poll?target=` | SNMP poll (sim в dev) |
| POST | `/api/v1/discovery/sweep?cidr=` | ping/ARP sweep (sim) |
| POST | `/api/v1/netflow/line` | CSV flow line |
| GET | `/api/v1/devices` | unmanaged devices из CMDB |

## CMDB reconciliation

- `platform=network`, без `agent_id` — unmanaged asset
- Дедуп: node_id → hostname → MAC → IP
- Конфликт с managed endpoint по IP → audit `cmdb.observe_conflict`, без перезаписи

## Compose

```bash
docker compose -f deploy/docker-compose.prod.yml --profile observe up -d observe ingest-gateway control-plane
```

## Тесты

- `go test ./services/observe/...` — golden PRTG/syslog, netflow parse, license gate
- `go test ./services/control-plane/internal/networkreconcile/...` — dedup + conflict
- `go test ./services/detection-engine/internal/correlator/...` — `era-observe-network-endpoint`

## Гейты

| Гейт | Статус |
|---|---|
| Полный SNMP exporter / NetFlow в проде | [gate: field] |
| Nmap discovery в закрытых сетях | [gate: policy] |

Код этапа: simulated SNMP/discovery; боевой poll — за гейтом политики сети.
