# Field Server Inventory

Заполнить **до** первого `docker compose up` на field-хосте.

## Идентификация

| Поле | Значение |
|------|----------|
| Дата | |
| Владелец контура | lab / ЦОД / заказчик |
| Hostname | |
| IP (mgmt) | |
| Роль | sizing / pilot / soak |

## Железо

| Ресурс | Факт | Минимум (sizing) |
|--------|------|------------------|
| vCPU | | 16 |
| RAM (GiB) | | 32 |
| Disk (GiB, тип) | | 600 SSD/NVMe |
| CPU model | | |

## ОС и runtime

| Поле | Значение |
|------|----------|
| ОС | |
| Kernel / build | |
| Docker version | |
| Docker Compose version | |
| Git commit era-xdr | |

## Сеть

| Порт | Открыт наружу? | Назначение |
|------|----------------|------------|
| 50051 | | ingest gRPC |
| 8090 | | control-plane |
| 8089 | | event-writer |
| 9092 | | Kafka (internal) |

## Preflight

| Проверка | Результат | Дата |
|----------|-----------|------|
| `check-field-server.ps1` | PASS / FAIL | |
| `gen-dev-tls.ps1` | | |
| compose up scale+pg | | |
| pilot-local | | |
| loadgen AC2 10k | | |

## Примечания

_Доступ по SSH, ограничения ИБ, air-gap, контакт ответственного._
