# ADR-0020: ERA Observe — network monitoring + CMDB reconciliation

**Статус:** Accepted (Implemented MVP, Stage 9)
**Дата:** 1 июля 2026 г.
**Контекст:** [Vision §8](ERA-Platform-Vision.md) — два пути Observe (интеграция PRTG/Zabbix + нативный agentless). Сетевые устройства без агента должны попадать в CMDB без дублей с managed endpoint.

**Связано:** [ADR-0011](0011-cmdb-itam-data-model.md) · [ADR-0003](0003-repository-structure-and-donor-strategy.md)

## Решение

### Scope Observe (MVP)
- **Путь A:** PRTG/Zabbix webhook + syslog → `Envelope` `EVENT_CATEGORY_NETWORK` → `xdr.network`
- **Путь B:** `services/observe` — SNMP poll (sim/MIB stub), discovery (ping sweep sim), NetFlow line parse (opt)
- **Не в MVP:** полный NMS-UI, 200+ sensor types, Nmap

### CMDB reconciliation
- Сетевые узлы без агента → `asset_kind=network`, `managed=false`
- Дедуп приоритет: `agent_id` > serial > MAC > IP > hostname (как ADR-0011 inventory merge)
- Конфликт (тот же IP у managed endpoint и network device) → audit, не перезапись managed

### Доноры (идеи только)
- Prometheus snmp_exporter, Telegraf SNMP input, goflow — паттерны, не код
- LibreNMS/Zabbix — UI/alert ideas

## Последствия
- Модуль лицензии `observe`
- Топик `xdr.network` (существующий)
- Корреляция `era-observe-network-endpoint` в detection-engine
