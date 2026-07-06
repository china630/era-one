# Hybrid-0 Spec — Sovereign Hybrid MVP (ADR-0018 §12)

**Версия:** 1.0  
**Дата:** 2 июля 2026 г.  
**Статус:** Implemented (MVP Hybrid-0)

Связано: [ADR-0018](adr/0018-hybrid-connected-operating-model.md) · [editions-control.yaml](../editions-control.yaml) ·
[Implementation-Roadmap](Implementation-Roadmap.md) Этап 2

---

## Backlog H0-*

| ID | Компонент | Описание | Статус |
|---|---|---|---|
| H0-1 | `services/license` | Lease-слой ERAL1: Sign/Verify/Evaluate, `era-keygen issue-lease`, golden | [x] |
| H0-2 | `control-plane/internal/hybrid` | Hybrid Relay: outbound worker, egress allowlist, health A, audit | [x] |
| H0-3 | `control-plane` API | `/api/v1/hybrid/status`, `/api/v1/hybrid/policy` (GET/PUT) | [x] |
| H0-4 | `control-plane` store | HybridPolicy/Runtime/EgressAudit/LeaseCache — memory/sqlite/postgres parity | [x] |
| H0-5 | `services/update-service` | Подписанные бандлы ERABNDL1 (Sigma corpus), pull + offline | [x] |
| H0-6 | `services/cloud-portal` | Installations, lease renew, CRL, health A ingest, Managed View RBAC | [x] |
| H0-7 | `deploy/docker-compose.prod.yml` | Profile `connected`: portal + update-service; relay = env CP | [x] |
| H0-8 | AZ docs | DPA-шаблон + схема потоков (az/ru/en) | [x] |

---

## Критерии приёмки (AC)

| AC | Критерий | Доказательство |
|---|---|---|
| AC-H0-1 | Connected OFF по умолчанию (`DefaultHybridPolicy`) | `go test` hybrid + api — PASS |
| AC-H0-2 | Lease/CRL/bundle — проверка Ed25519 перед применением | `services/license` + relay e2e — PASS |
| AC-H0-3 | Health A без сырья/PII наружу | `health_test`, portal reject raw_event — PASS |
| AC-H0-4 | Egress audit журнал | `ListEgressAudit` + relay e2e ≥2 записей — PASS |
| AC-H0-5 | Air-gap путь не сломан | без `ERA_HYBRID_CONNECTED` relay idle — PASS |
| AC-H0-6 | Store parity hybrid ops | `parity_test` sqlite — PASS |
| AC-H0-7 | Connected compose profile | `docker-compose.prod.yml` profile `connected` | [x] конфиг |
| AC-H0-8 | editions-control.yaml согласован | `deployment_modes.connected` / `hybrid_components` | сверено |

---

## Запуск connected (dev)

```powershell
# Вендор (profile connected)
docker compose -f deploy/docker-compose.prod.yml --profile connected up -d cloud-portal update-service

# Control-plane с Relay (после era-keygen genkey + issue)
$env:ERA_HYBRID_CONNECTED="1"
$env:ERA_VENDOR_PUB="<vendor.pub base64>"
$env:ERA_PORTAL_URL="http://localhost:8120"
$env:ERA_UPDATE_URL="http://localhost:8110"
```

Air-gap (default): без profile и без `ERA_HYBRID_CONNECTED` — только offline-бандл Update Service.

**Типы контента и каналы** (Sigma vs IoC vs CVE) — [`ADR-0018 §3.2.1`](adr/0018-hybrid-connected-operating-model.md);
governance процесса — [`ADR-0022`](adr/0022-detection-content-governance.md).

---

## Не в Hybrid-0 (ADR §12)

TI-outbound, Health B/C, white-label Managed View, отдельный relay-контейнер DMZ, multi-tenant SaaS.
