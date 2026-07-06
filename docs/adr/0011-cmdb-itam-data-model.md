# ADR-0011: Модель данных CMDB / ITAM (издание ERA Manage)

**Статус:** Implemented (Этап 5)
**Дата:** 27 июня 2026 г.
**Контекст:** Издание **ERA Manage** (IT-Ops) требует хранения инвентаря (ITAM)
и базы конфигурационных единиц (CMDB). Сейчас в коде есть только зерно — модель
`Asset` в `control-plane` (`services/control-plane/internal/store/repository.go`):
регистрация хоста (node_id, hostname, platform, agent_id, last_seen). Полноценный
ITAM (железо, установленный софт, серийники, лицензии ПО) хранить негде.

**Связано:** [`ADR-0001`](0001-unified-event-envelope.md) ·
[`ADR-0004`](0004-storage-and-retention.md) ·
[`ADR-0005`](0005-module-independence-and-packaging.md) ·
[`ADR-0007`](0007-clickhouse-schema.md) ·
[`ERA-Platform-Vision.md`](../ERA-Platform-Vision.md) (§6 напр. 1, §13 P2)

---

## Контекст и проблема

Запросы заказчиков уровня ManageEngine Endpoint Central требуют UEM-инвентаризации
на тысячи хостов (пример: 7500 рабочих станций + 400 серверов). Данные ITAM по
своей природе **двойственны**:

1. **Текущее состояние** («что стоит на хосте *сейчас*»: HW, список ПО, серийники) —
   мутабельное, запрашивается по активу, перезаписывается при каждом snapshot.
2. **История изменений** («когда появился софт X на N хостах», тренды, форензика) —
   иммутабельный поток во времени.

По принципу ADR-0004 («разделение данных по ценности») эти два типа требуют **разных
хранилищ**. Класть мутабельный CMDB в ClickHouse (аналитическое, append-oriented)
неправильно; класть годовую историю инвентаря в реляционную БД — дорого и не нужно.

## Решение

### 1. Двухслойное хранение

| Слой | Тип данных | Хранилище | Движок/драйвер |
|---|---|---|---|
| **CMDB (current state)** | Текущий срез актива + установленный софт | БД `control-plane` | **Postgres** (prod) / SQLite (dev) — `ERA_STORE_DRIVER` |
| **Inventory history** | Snapshot-события инвентаря во времени | **ClickHouse** lake | как вся телеметрия (ADR-0007) |

**Поток данных:**

```
era-plugin-inventory (cron на агенте)
   → snapshot (JSON) → Envelope (domain=inventory)
       → ingest-gateway → Kafka
            ├─► ClickHouse: inventory-события (история, тренды, форензика)
            └─► control-plane consumer: upsert текущего среза CMDB (Postgres)
```

CMDB-срез материализуется из потока (не отдельный канал), чтобы не нарушать
контракт-first (ADR-0001/0008): агент шлёт **один** Envelope, сервер раскладывает.

### 2. Схема CMDB (control-plane, Postgres)

Расширение существующей модели `Asset` (обратносовместимо — только добавление полей):

```
assets (расширяется)
  node_id PK, tenant_id, hostname, platform, agent_id,
  last_seen, agent_version,
  -- новые поля ITAM:
  fqdn, os_name, os_version, kernel,
  cpu_model, cpu_cores, ram_mb, disk_total_gb,
  serial_number, board_serial, manufacturer, model,
  mac_addrs (json), ip_addrs (json),
  inventory_updated_at

asset_software (новая таблица)
  node_id FK, tenant_id,
  name, version, vendor, install_date,
  source (registry|dpkg|rpm|brew),
  first_seen, last_seen,
  PRIMARY KEY (node_id, name, version)

asset_software используется как вход для сверки с CVE в services/vm (ERA Vuln).
```

### 3. Inventory-события (ClickHouse)

Идут в общий конвейер событий с `domain=inventory`. Тело snapshot — в `payload`
(JSON/CBOR, как в ADR-0007), типизированная доменная таблица — отдельная задача
(Фаза 2 раскладки payload). Retention — по классу «security-significant» (ADR-0004),
т.к. история «когда появился софт» важна для форензики и аудита лицензий ПО.

### 4. Правила слияния активов (asset merge / dedup)

Один физический хост может прийти под разными идентификаторами (переустановка ОС,
смена hostname, несколько NIC). Ключи слияния — по приоритету:

1. **`agent_id`** (стабильный install-secret агента) — основной ключ.
2. **`serial_number` / `board_serial`** — для смены агента/переустановки.
3. **`mac_addrs` ∩** — вспомогательный, толерантно (N-из-M, как в ADR-0010 §9).
4. **`hostname` + `tenant_id`** — слабый, только как hint.

Слияние — на стороне control-plane (consumer), а не на агенте. Конфликты пишутся в
аудит. **Платформозависимый сбор** признаков (serial, MAC) — ответственность
плагина инвентаризации; control-plane только сопоставляет.

### 5. Лицензирование и multi-tenant

- ITAM/CMDB активируется модулем `manage` в лицензии (ADR-0005/0010), feature-flag
  в control-plane.
- Все таблицы несут `tenant_id`; запросы — в tenant-scope (как существующий
  `tenant_scope.go`).

## Последствия

**Плюсы:** переиспользуем существующий store-слой (Postgres/SQLite) и event-конвейер;
CMDB кормит `services/vm` (vuln) и assets-UI; история инвентаря бесплатно ложится в
lake; модель обратносовместима с текущим `Asset`.

**Минусы / обязательства:** нужен consumer инвентаря в control-plane; merge-логика
требует контрактных и golden-тестов (фикс. snapshot → ожидаемый CMDB-срез);
платформозависимый сбор серийников/MAC — отдельная задача плагина; раскладка
inventory `payload` в типизированные колонки ClickHouse — Фаза 2.

## Открытые вопросы

- Окончательный набор полей HW-инвентаря (донор полей — osquery/Fleet, ADR-0003).
- Хранение «usage» ПО (запускалось ли) — отдельный сигнал, возможно из XDR-телеметрии.
- Reconciliation CMDB ↔ ERA Observe (network discovery) — отложено до P5.
