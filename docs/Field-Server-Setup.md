# ERA XDR — Field Server Setup (пошагово)

Практическое руководство: от «нет сервера» до первого field-прогона.  
Sizing-спеки: [`Field-Server-Sizing.md`](Field-Server-Sizing.md) · Готовность: [`Production-Readiness-Assessment.md`](Production-Readiness-Assessment.md)

---

## Шаг 0 — Выбор варианта

| Вариант | Когда | Минимум |
|---------|-------|---------|
| **A. Lab VM** | Быстрый старт, Hyper-V/Proxmox | 16 vCPU, 32 GiB RAM, 600 GiB SSD |
| **B. Физический сервер в ЦОД** | Ближе к пилоту заказчика | То же + стабильная сеть |
| **C. Сервер заказчика** | Air-gap пилот | По согласованию ИБ |

**Не подходит:** ноут с Docker Desktop для AC2 (ожидайте 200–700 ev/s, не 10k).

---

## Шаг 1 — Инвентаризация (заполнить до установки)

Скопируйте [`reports/field-server-inventory.md`](../reports/field-server-inventory.md) и заполните:

- hostname, IP, ОС, CPU/RAM/disk
- кто владелец (lab / заказчик)
- доступ: SSH/RDP, VPN, air-gap

---

## Шаг 2 — ОС и Docker

### Linux (рекомендуется)

```bash
# Ubuntu 22.04/24.04 или RHEL 8+
sudo apt update && sudo apt install -y docker.io docker-compose-plugin git
sudo usermod -aG docker $USER
# перелогиниться
docker --version
docker compose version
```

### Windows Server (lab)

- Docker Desktop или WSL2 + Docker Engine
- Для AC2 предпочтительнее Linux-VM на том же железе

---

## Шаг 3 — Preflight на хосте

Из корня репозитория (клонировать репо на сервер или scp):

```powershell
# Windows на сервере
.\scripts\check-field-server.ps1

# Linux (если установлен pwsh)
pwsh ./scripts/check-field-server.ps1
```

Скрипт проверяет: Docker, CPU/RAM (предупреждения), свободное место, занятость портов 50051/8090/8089/9092.

---

## Шаг 4 — Клонирование и TLS

```bash
git clone <your-mirror-of-era-xdr> /opt/era-xdr
cd /opt/era-xdr
```

```powershell
# dev-CA для lab (не production PKI)
./scripts/gen-dev-tls.ps1
# убедиться: deploy/tls/server.crt с SAN localhost
```

---

## Шаг 5 — Поднять prod-стек

```powershell
$env:ERA_STORE_DRIVER = "postgres"
$env:ERA_STORE_DSN = "postgres://era:era_cp_pw@postgres:5432/era_cp?sslmode=disable"

docker compose -f deploy/docker-compose.prod.yml --profile scale --profile pg up -d --build
```

Первый build: 15–30 мин. Проверка:

```powershell
docker compose -f deploy/docker-compose.prod.yml ps
# health (mTLS):
cd services/platform
$env:ERA_TLS_CA = "../../deploy/tls/ca.crt"
$env:ERA_TLS_CLIENT_CERT = "../../deploy/tls/agent.crt"
$env:ERA_TLS_CLIENT_KEY = "../../deploy/tls/agent.key"
go run ./cmd/mtls-health https://127.0.0.1:8090/healthz
```

---

## Шаг 6 — Первый field-прогон (без AC2 10k)

```powershell
.\scripts\run-chaos-smoke.ps1
.\scripts\run-pilot-local.ps1 -QuickLoadgen
```

Лог: `reports/pilot-local-*.log`

---

## Шаг 7 — AC2 loadgen (на sizing-железе)

```powershell
.\scripts\run-loadgen-prod.ps1 -Rate 10000 -DurationSec 300 -Runs 3 -MinEvPerSec 10000
```

Результат → `reports/loadgen-prod.log` + копия шаблона из Field-Server-Sizing в `reports/field-server-YYYYMMDD.log`.

**PASS:** три прогона ≥10k ev/s, fail=0.

---

## Шаг 8 — Безопасность перед пилотом

- [ ] Сменить `ERA_CH_PASSWORD`, MinIO, Postgres пароли
- [ ] Firewall: только нужные порты (см. Field-Server-Sizing)
- [ ] NTP для sealed clock / лицензии
- [ ] `backup-prod.ps1` — тестовый бэкап

---

## Шаг 9 — Агенты (после стека UP)

На 1–2 тестовых хостах:

```powershell
$env:ERA_PRODUCTION = "1"
$env:ERA_TLS_CA = "deploy/tls/ca.crt"
$env:ERA_TLS_CLIENT_CERT = "deploy/tls/agent.crt"
$env:ERA_TLS_CLIENT_KEY = "deploy/tls/agent.key"
$env:ERA_GATEWAY_ADDR = "https://<ingest-host>:50051"
# Windows: Sysmon / ERA_SYSMON_JSONL
# Linux: ERA_AUDIT_LOG=/var/log/audit/audit.log
```

---

## Troubleshooting

| Симптом | Решение |
|---------|---------|
| healthz 400 на :8090 | mTLS: использовать `mtls-health`, не plain HTTP |
| loadgen ev/s=0 | `ERA_TLS_*` на клиенте; SAN localhost в server.crt |
| compose OOM | увеличить RAM или отключить profile scale временно |
| postgres CP не стартует | `--profile pg`, дождаться `era-prod-postgres` healthy |

---

## Связанные документы

- [`Install-Guide-GA.md`](Install-Guide-GA.md)
- [`Pilot-Readiness-Checklist.md`](Pilot-Readiness-Checklist.md)
