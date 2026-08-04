# ERA One

**ONE ECOSYSTEM. ONE PERIMETER. ONE VENDOR.**

Суверенная on-premise / air-gapped платформа для enterprise и госструктур
(Азербайджан, СНГ/ЦА): безопасность и IT-Ops, коммуникации и офисный контур —
в одном вендоре, без phone-home и внешних SaaS в рантайме.

Монорепо [`china630/era-one`](https://github.com/china630/era-one) содержит три
продуктовые линейки и общий shared platform (identity, tenant, Drive, license gate).

| Линейка | Слоган | Суть | Статус (honesty) |
|---|---|---|---|
| **ERA Control** | ONE AGENT. ONE PLATFORM. ONE CONTROL. | XDR + IT-Ops + PAM + периметр + DNS Resolve; один лёгкий endpoint-агент | software **GA** (Core/AI/Response); field pilot-гейты открыты |
| **ERA Communications** | ONE IDENTITY. ONE PLATFORM. ONE CONVERSATION. | Почта, календарь, чат, ВКС; on-prem замена Exchange / IceWarp | **roadmap → MVP-волны** (см. Comms matrices) |
| **ERA Office** | ONE WORKSPACE. ONE PLATFORM. ONE TEAM. | Drive, Documents, Tables, Presentations, Projects, Office AI; Solo desktop | **mvp** (не `ga` до field RT-O09) |

SSOT продуктовой карты: [`products.yaml`](products.yaml) · продажи:
[`docs/distributor/ERA-Product-Line.md`](docs/distributor/ERA-Product-Line.md).

> Control ≠ «весь ERA One». XDR (ERA Core) — фундамент **Control**. Communications и
> Office — отдельные линейки (per-user), без обязательного агента.

## Документация

| Документ | Зачем |
|---|---|
| [`products.yaml`](products.yaml) | Карта линеек и изданий |
| [`docs/products/ERA-Product-Acceptance-Standard.md`](docs/products/ERA-Product-Acceptance-Standard.md) | Канон приёмки (Scaffold ≠ Pilot ≠ GA) |
| [`docs/Control-Product-Readiness-Matrix.md`](docs/Control-Product-Readiness-Matrix.md) | Готовность Control |
| [`docs/Comms-Product-Readiness-Matrix.md`](docs/Comms-Product-Readiness-Matrix.md) | Готовность Communications |
| [`docs/Office-Product-Readiness-Matrix.md`](docs/Office-Product-Readiness-Matrix.md) | Готовность Office |
| [`docs/adr/`](docs/adr/) | ADR (в т.ч. 0024 families, 0025 shared platform, 0026 office engine, 0027 comms) |
| [`site/`](site/) | Публичный маркетинг-сайт (отдельные коммиты; прод через `main` → `site-prod`) |

Sprint-Index / Implementation-Matrix / Evidence — по каждой линейке в `docs/`.

## Структура репозитория (упрощённо)

```
products.yaml / editions-*.yaml   продуктовая карта и издания (SSOT)
proto/era/v1/                     контракты (events, drive, office, comms, …)
crates/                           Rust: agent, collectors, license, office cores
apps/era-office-desktop/          Tauri Solo / SKU (Docs, Tables, Pres, Projects)
services/                         Go/Rust сервисы Control, Comms, platform
ui/                               web UI (SOC, mail, office apps, portal)
deploy/                           compose, profiles, migrations, dockerfiles
docs/                             ADR, PRD, readiness/acceptance
site/                             маркетинг (изолированные коммиты)
scripts/                          gates, smoke, site build, office/comms labs
```

## Быстрый старт

### Control (XDR pipeline)

```bash
docker compose -f deploy/docker-compose.dev.yml up -d
cd services/ingest-gateway && go run ./cmd/ingest-gateway
cd services/event-writer && go run ./cmd/event-writer
ERA_GATEWAY_ADDR=http://127.0.0.1:50051 cargo run -p era-agent
```

Smoke / load: `.\scripts\run-e2e.ps1` · `.\scripts\run-loadtest.ps1`

### Office / Communications

Профили: `deploy/profiles/office.yaml`, `deploy/profiles/comms.yaml`.  
Compose: `deploy/docker-compose.office.yml`, `deploy/docker-compose.comms.yml`.  
Приёмка: `.\scripts\check-acceptance-consistency.ps1` · quality: `.\scripts\run-quality-gates.ps1`.

### Лицензирование (офлайн, ADR-0010)

```bash
cd services/license
go run ./cmd/era-keygen genkey -out ./keys
go run ./cmd/era-keygen issue -priv ./keys/vendor.key \
    -customer "Bank A" -tenant t1 -modules vm,ai,response -nodes 50000 -years 3 \
    -deployment deploy-XYZ
```

Приватный ключ — только HSM/сейф; в репозиторий не коммитить.

## Ветки

| Ветка | Назначение |
|---|---|
| `dev` | Разработка продукта и staging сайта |
| `main` | Прод-источник; push путей `site/**` триггерит публикацию |
| `site-prod` | Только собранная статика сайта (CI, не править руками) |

Сайт и продуктовый код — **разные коммиты** (см. `.cursor/rules/site-commit-isolation.mdc`).

## Статус (кратко)

- **Control:** software GA по Core / Control AI / Response; дальнейшие издания и field-гейты — в Control matrices.
- **Communications / Office:** активная разработка в `dev`; честные статусы — в Product-Readiness-Matrix и `editions-*.yaml` (`mvp` ≠ `ga`).
- Исторический Sprint-1 XDR pipeline (agent → ingest → Kafka → ClickHouse) — база Control, не описание всего ERA One.

---

*Проприетарный продукт. Доноры — по принципу AI-Driven Reverse Engineering
(паттерны и модели, не копирование кода) — ADR-0003.*
