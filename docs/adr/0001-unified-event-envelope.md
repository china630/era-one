# ADR-0001: Unified Event Envelope (единый конверт события)

**Статус:** Accepted
**Дата:** 8 июня 2026 г.
**Контекст фазы:** Фаза 1 (MVP) — проектирование Data Contracts и Ingest Pipeline.

---

## Контекст

ERA XDR собирает телеметрию из разнородных источников (endpoint, identity, email,
cloud, network) и должна объединять её для кросс-доменной корреляции (XDR).
Нужен **единый транспортный контракт события**, который:

1. Генерируется Rust-агентом и отправляется в Ingest Gateway через Kafka.
2. Поддерживает строгую идентификацию источника и корреляцию.
3. Версионируется и эволюционирует без слома исторических данных в ClickHouse.
4. Готов к нагрузке ~4.5 ТБ/сутки от 150 000 хостов.

## Решение

### 1. Структура «Unified Event Envelope»

Конверт = транспорт + идентификация + контроль. **Семантика payload нормализуется
в OCSF** (Open Cybersecurity Schema Framework). Полная схема — в
[`proto/era/v1/envelope.proto`](../../proto/era/v1/envelope.proto).

Ключевые проектные решения:

| Решение | Обоснование |
|---|---|
| **`event_id` = ULID (16 байт)** | Лексикографически сортируемый по времени → дешёвая сортировка/дедуп в ClickHouse |
| **Два времени: `observed_at` / `ingested_at`** | Расследования + детект tampering с системными часами хоста |
| **Иерархия `tenant→environment→cluster→node→agent`** | Единая идентификация сущности по всей платформе |
| **`oneof payload` (типизированный)** | Строгая типизация событий + `RawEvent` для расширения без слома схемы |
| **`ocsf` обязателен для типизированных payload** | Корреляция XDR и LLM-маппинг на MITRE кратно надёжнее на стандарте |
| **`detection` (on-agent Sigma)** | Критичное срабатывает даже при разрыве связи с ядром (автономность) |
| **`pii_sanitized`** | Очистка PII выполняется на агенте ДО записи в Data Lake |
| **`correlation_id`** | Storyline: связь дочерних событий в один инцидент |

### 2. OCSF — нормализация с первого дня

- **На агенте:** тривиальный статический маппинг (`ProcessEvent → class_uid 1007`,
  `NetworkEvent → 4001`, `DnsEvent → 4003`). Нулевая нагрузка — это константы.
- **В `processors` (Go, кластер):** сложный маппинг, enrichment, `activity_id` по
  контексту. Тяжёлую логику не выносим на 150k тонких агентов.

OCSF — **семантическая модель**, наш Envelope — **транспортный конверт**. Мы не
заменяем одно другим: Envelope несёт идентификацию/контроль, payload несёт
OCSF-семантику.

### 3. Версионирование схемы — гибрид «registry + embedded version»

Два уровня, оба обязательны:

- **Wire (Kafka):** чистый protobuf + `schema_version` (semver-строка) внутри
  `Envelope`. **Никакого обращения к реестру в рантайме на агенте** → сохраняем
  автономность и лёгкость клиента. Не используем Confluent-wire-формат с
  magic-byte + schema-id, чтобы не создавать сетевую зависимость на горячем пути.
- **Build-time / CI:** **Apicurio Registry** (Apache 2.0, on-prem, без облака) как
  single source of truth — каталог `.proto`, проверка обратной совместимости
  (`BACKWARD`/`FULL`) в каждом PR, генерация Go/Rust-стабов из одного источника,
  хранение всех исторических версий для декодируемости старых данных.

### 4. Транспорт в Kafka

| Параметр | Значение | Обоснование |
|---|---|---|
| **Partition key** | `tenant_id \|\| node_id` | Строгий порядок событий одного хоста → корректный process-tree/storyline; равномерный шардинг |
| **Topic** | `xdr.<category>` (topic-per-domain) | Маршрутизация и независимая подписка модулей |
| **Producer** | `enable.idempotence=true`, `acks=all` | Гарантированная доставка без дублей |
| **Compression** | `zstd` | Лучший ratio для логов |
| **Батчинг** | `linger.ms=20`, `batch.size=1 MiB` | Throughput под высокую нагрузку |
| **Backpressure** | disk-backed buffer на агенте | События не теряются при недоступности Kafka (паттерн Vector) |

### 5. Сериализация на Rust

`prost` (кодек) + `prost-build` (codegen в `build.rs`) + `rdkafka` (продюсер) +
`ulid` + `prost-types` (Timestamp). Поток:

```
eBPF/ETW capture → PII sanitize → build Envelope → prost encode
  → disk buffer (backpressure) → Kafka (key = tenant|node) → ingest-gateway
```

## Последствия

**Плюсы:** единый формат для всех источников; автономность агента; историческая
декодируемость; готовность к XDR-корреляции и AI-маппингу; governance совместимости.

**Минусы / обязательства:** дисциплина proto-evolution (только добавление полей,
номера не переиспользуем); поддержка таблицы OCSF-маппинга; эксплуатация Apicurio
в контуре заказчика.

## Связано

- Master: [`ERA-XDR-Architecture-Blueprint.md`](../../reports/ERA-XDR-Architecture-Blueprint.md)
- Контракт: [`proto/era/v1/envelope.proto`](../../proto/era/v1/envelope.proto)
- [`ADR-0004`](./0004-storage-and-retention.md) — как конверт ложится в ClickHouse
