# ERA XDR — Install Guide (Production GA)

**Издания:** ERA Core + ERA Control AI + ERA Response  
**Платформы агента:** Windows 10/11, Windows Server 2019+; Linux RHEL 8+/Ubuntu 20.04+; macOS 12+ (unified log export)  
**Контур:** on-prem, air-gap

## 1. Требования

| Слой | Минимум | HA (опция) |
|---|---|---|
| Сервер платформы | 8 vCPU, 16 GiB RAM, 600 GiB SSD | 3× broker Kafka — `docker-compose.prod-ha.yml` |
| Сеть | 9092, 50051, 8090–8092, 8089 | mTLS: `deploy/tls/`, `scripts/gen-dev-tls.ps1` |
| SOC UI | `http://cp:8090/ui/portal/` | SSO: reverse proxy → `X-ERA-Role`, `X-ERA-Actor` |

Sizing: [`deploy/prod/README.md`](../deploy/prod/README.md). **Field-прогон AC2 10k:** [`Field-Server-Sizing.md`](Field-Server-Sizing.md).

## 2. Развёртывание

```powershell
docker compose -f deploy/docker-compose.prod.yml up -d --build
# опции: --profile perimeter --profile ctem
# HA Kafka: docker compose -f deploy/docker-compose.prod-ha.yml up -d
.\scripts\run-ga1-smoke.ps1
.\scripts\run-ga-full.ps1
```

## 3. mTLS (агент → gateway)

```powershell
.\scripts\gen-dev-tls.ps1
$env:ERA_TLS_CA = "deploy/tls/ca.pem"
$env:ERA_TLS_CLIENT_CERT = "deploy/tls/agent.pem"
$env:ERA_TLS_CLIENT_KEY = "deploy/tls/agent-key.pem"
# gateway: ERA_TLS_CERT / ERA_TLS_KEY
```

Control-plane TLS: `ERA_TLS_CERT`, `ERA_TLS_KEY` на `:8090`.

## 4. Лицензия

```powershell
go run ./services/license/cmd/era-keygen issue -priv ./keys/vendor.key `
  -customer "Pilot Bank" -tenant pilot-bank -bundle core-ai-response -years 1 -nodes 50000
```

Prod / strict (C-06): задайте `ERA_LICENSE_TOKEN` или `ERA_LICENSE_PATH`, `ERA_VENDOR_PUB`, `ERA_LICENSE_STRICT=1` (или `ERA_PRODUCTION=1`). Без токена CP/ingest **не стартуют** (fail-closed). Sealed clock: `ERA_SEALED_CLOCK_PATH`, `ERA_SEALED_CLOCK_SECRET`.

## 5. Агенты

**Windows:** Sysmon + `ERA_PRODUCTION=1`, `ERA_SYSMON_JSONL` или `ERA_SYSMON_EVtx`.

**Linux:** auditd execve + `ERA_AUDIT_LOG`.

**macOS:** `ERA_MACOS_UNIFIED_JSONL` (NDJSON export unified log / ES).

## 6. SOC portal

Откройте `http://<cp>:8090/ui/portal/` — cases, assets, events, detections.  
LDAP/SAML: on-prem IdP → reverse proxy добавляет `X-ERA-Role`, `X-ERA-Actor` ([SSO-Setup-GA.md](SSO-Setup-GA.md)).

**SQLite vs Postgres:** пилот — SQLite + volume; production банк — Postgres после sign-off пилота ([deploy/prod/README.md](../deploy/prod/README.md)).

## 7. Нагрузочная приёмка (F-GA-5)

См. [`Field-Server-Sizing.md`](Field-Server-Sizing.md).

```powershell
docker compose -f deploy/docker-compose.prod.yml --profile scale --profile pg up -d --build
.\scripts\run-loadgen-prod.ps1   # >=10k ev/s, 5 min x 3 -> reports/loadgen-prod.log
```

## 8. Backup / restore

```powershell
.\scripts\backup-prod.ps1
.\scripts\restore-prod.ps1 -ArchivePath reports/backup-YYYYMMDD.tar.gz
```

Postgres store (опция): `ERA_STORE_DRIVER=postgres`, `ERA_STORE_DSN=postgres://...`

## Связанные документы

- [`Production-GA-Spec.md`](Production-GA-Spec.md)
- [`Pilot-Readiness-Checklist.md`](Pilot-Readiness-Checklist.md)
