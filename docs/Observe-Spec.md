# ERA Observe — Network Monitoring (Stage 9)

Agentless-мониторинг сети и интеграция с NMS (PRTG/Zabbix).

**Связано:** [ADR-0020](adr/0020-network-observe-cmdb-reconciliation.md) · лицензия `observe`.  
**Статус кода:** Path A + Path B ✅ · полный NMS/Nmap ❌ · field lab SNMP ⏸

## Компоненты

| Компонент | Путь | Порт |
|---|---|---|
| Observe API | `services/observe` | `:8132` |
| NetFlow UDP | `internal/netflow.ListenUDP` | `:2055` (`ERA_NETFLOW_UDP_ADDR`) |
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

| Метод / listener | Описание |
|---|---|
| POST `/api/v1/snmp/poll?target=` | SNMP poll: `ERA_OBSERVE_SNMP_SIM=1` → sim; иначе `PollReal` (gosnmp) |
| HOST-RESOURCES-MIB | CPU (`hrProcessorLoad`) + storage → `metrics_source=host_resources`; иначе estimate от ifTable |
| POST `/api/v1/discovery/sweep?cidr=` | Real sweep; в production без silent sim (`ERA_OBSERVE_STRICT` / `ERA_PRODUCTION`) |
| POST `/api/v1/netflow/line` | CSV line или NetFlow v5 binary body |
| UDP `:2055` | NetFlow v5 listener → ingest (`ERA_NETFLOW_UDP_DISABLE=1` to skip) |
| GET `/api/v1/devices` | unmanaged devices из CMDB |
| GET `/api/v1/topology` | topology widget |

## Env

| Var | Default | Meaning |
|-----|---------|---------|
| `ERA_OBSERVE_SNMP_SIM` | unset | `1` = force sim metrics |
| `ERA_OBSERVE_STRICT` / `ERA_PRODUCTION` | unset | no silent SNMP sim fallback on error |
| `ERA_SNMP_COMMUNITY` | `public` | SNMPv2c community |
| `ERA_NETFLOW_UDP_ADDR` | `:2055` | UDP bind |
| `ERA_NETFLOW_UDP_DISABLE` | unset | `1` = skip UDP listener |

## Compose

```bash
docker compose -f deploy/docker-compose.prod.yml --profile observe up -d observe ingest-gateway control-plane
```

## Тесты / smoke

```powershell
.\scripts\run-observe-smoke.ps1
```

## Гейты

| Гейт | Статус |
|---|---|
| Path A+B code + golden/UDP/HOST-RESOURCES fallback | ✅ |
| Боевой SNMP lab на стенде заказчика | [gate: field] |
| Полный NMS / Nmap discovery | ❌ out of MVP |
