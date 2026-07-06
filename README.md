# ERA XDR

**Суверенная, On-Premise (air-gapped) Cloud-Native платформа класса Extended
Detection & Response** для крупных изолированных enterprise-сетей и госструктур
локального рынка (Азербайджан, регион СНГ/ЦА). Целевой масштаб — до 150 000 хостов
в одном кластере.

> Агрегирует подходы мировых лидеров (CrowdStrike Falcon, SentinelOne Singularity,
> **Trend Vision One**), но физически разворачивается внутри контура заказчика без
> отправки телеметрии во внешние облака.

## Три критерия + киллер-фича

- **Надёжность** · **Безопасность** · **Лёгкость клиента** (один Rust-бинарник).
- **Киллер-фича:** автономность самообучения контура — самообучается серверный
  AI Core, агент остаётся лёгким и лишь самоадаптируется; знания ретранслируются
  всем агентам («стадный иммунитет»).

## Документация

| Документ | Описание |
|---|---|
| [Architecture Blueprint](reports/ERA-XDR-Architecture-Blueprint.md) | Главный документ: концепция, стек, модули, дорожная карта |
| [AI Donors Matrix Analysis](reports/AI-Donors-Matrix-Deep-Analysis.md) | Анализ доноров и стратегии AI-Driven Reverse Engineering |
| [MVP Sprint-1 Spec](docs/MVP-Sprint-1-Spec.md) | Спека первого спринта с критериями приёмки |
| [Development Plan](docs/Development-Plan.md) | Фазы, DoD, статус разработки |
| [ADR (решения)](docs/adr/) | Architecture Decision Records (0001–0009) |

### Ключевые решения (ADR)

`0001` контракт события · `0002` топология обучения · `0003` монорепо и доноры ·
`0004` хранение/retention · `0005` независимость модулей · `0006` дыры покрытия и
стратегические ставки · `0007` ClickHouse DDL · `0008` gRPC ingest · `0009` PII и
бюджет агента · `0010` лицензирование и офлайн-активация.

### Лицензирование (вендор-сторона)

Офлайн-активация на Ed25519, помодульно, на 1/3 года (ADR-0010):

```bash
cd services/license
# пара ключей вендора (приватный — в HSM/сейф!)
go run ./cmd/era-keygen genkey -out ./keys
# выпуск лицензии (помодульно, на 1/3 года, с привязкой к развёртыванию)
go run ./cmd/era-keygen issue -priv ./keys/vendor.key \
    -customer "Bank A" -tenant t1 -modules vm,ai,response -nodes 50000 -years 3 \
    -deployment deploy-XYZ
# проверка
go run ./cmd/era-keygen verify -pub ./keys/vendor.pub -token <TOKEN> -deployment deploy-XYZ
# отзыв (CRL) для досрочного аннулирования
go run ./cmd/era-keygen revoke -priv ./keys/vendor.key -lids lic-aaa,lic-bbb -out crl.token
# вычислить deployment fingerprint (на стороне control-plane)
go run ./cmd/era-keygen fingerprint -machine MID-1 -board BRD-1 -disks d1,d2 -macs m1
```

Защита включает: офлайн-подпись Ed25519, привязку к развёртыванию (fingerprint),
отзыв (CRL) и **anti-rollback** (sealed monotonic clock — откат системных часов
детектируется и сам поднимается как security-инцидент). Подробно — ADR-0010.

## Структура репозитория

```
proto/era/v1/     контракты (envelope.proto, ingest.proto) — source of truth
gen/go/               сгенерированные Go-стабы (protoc, S1-1)
crates/era-proto/     Rust-стабы protobuf/gRPC (tonic-build, S1-1)
crates/era-agent/     Rust: сенсор — capture, PII, gRPC PushEvents
crates/era-license/   Rust: верификатор лицензий агента (defense-in-depth)
services/ingest-gateway/  Go: gRPC PushEvents + Kafka producer
services/event-writer/    Go: Kafka consumer → ClickHouse + UI API
services/vm/          Go: сканер уязвимостей (донор Nuclei)
services/license/     Go: офлайн-лицензирование + CLI era-keygen
deploy/               dev-окружение и ClickHouse DDL
docs/                 ADR + спецификации
reports/              аналитика и blueprint
```

## Быстрый старт (dev-окружение)

```bash
# 1) Фундамент
docker compose -f deploy/docker-compose.dev.yml up -d

# 2) ingest-gateway (gRPC :50051, HTTP :8082)
cd services/ingest-gateway && go run ./cmd/ingest-gateway

# 3) event-writer (Kafka→ClickHouse, UI :8089)
cd services/event-writer && go run ./cmd/event-writer

# 4) era-agent
ERA_GATEWAY_ADDR=http://127.0.0.1:50051 cargo run -p era-agent
```

E2E smoke: `.\scripts\run-e2e.ps1` · Load test: `.\scripts\run-loadtest.ps1`

Полезные адреса: ClickHouse `http://localhost:8123/play` · Kafka UI `http://localhost:8088` ·
Events UI `http://localhost:8089/ui/` · Apicurio `http://localhost:8085`

## Статус

**Sprint-1 (Фаза 1 MVP) — реализован.** Сквозной конвейер:
agent → ingest-gateway → Kafka → ClickHouse → UI/API.
AC1/3–5/7/8 PASS; AC2 = 7232 ev/s на dev (prod loadgen — на сервере); AC6 — RSS gate в CI (`ci-gates-stage10.ps1`).
План фаз — [Development Plan](docs/Development-Plan.md).

---
*Проприетарный продукт. Доноры используются по принципу AI-Driven Reverse
Engineering (переписывание паттернов, не копирование кода) — см. ADR-0003.*
