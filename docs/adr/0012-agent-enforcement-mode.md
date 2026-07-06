# ADR-0012: Enforcement-режим агента (Application Control)

**Статус:** Implemented (monitor-ready; боевой `enforce` и подпись драйвера — [gate: external])
**Дата:** 27 июня 2026 г. · обновлено 1 июля 2026 г.
**Контекст:** Издание **ERA Manage** требует **Application Control** — allow/deny
запуска приложений (white/black-list). Сегодня `crates/era-agent` — **telemetry-only**
(`capture → sanitize → buffer → gRPC`): он *наблюдает* процессы, но не *блокирует*.
Application Control превращает агента из наблюдателя в **enforcer**, что является
архитектурным сдвигом с серьёзными последствиями для безопасности и стабильности.

**Связано:** [`ADR-0005`](0005-module-independence-and-packaging.md) ·
[`ADR-0009`](0009-pii-redaction-and-agent-budget.md) ·
[`ADR-0010`](0010-licensing-and-activation.md) ·
[`ERA-Platform-Vision.md`](../ERA-Platform-Vision.md) (§6 напр. 6, §13 P2b) ·
правило `rust-agent-conventions`

---

## Контекст и проблема

Заказчики уровня ManageEngine Endpoint Central покупают Application Control отдельной
строкой. Чтобы дать альтернативу, агент должен **блокировать запуск** несанкционированного
ПО, а не только присылать алерт постфактум. Это требует перехвата на уровне ОС
(до/в момент `exec`), что несёт риски: блокировка легитимного ПО → простой бизнеса,
сбой хука → возможный bootloop/недоступность хоста.

## Решение

### 1. Enforcement — отдельный, лицензируемый, опциональный режим

- Telemetry-режим (текущий) остаётся **дефолтом**. Enforcement включается **только**
  при наличии модуля `manage` в лицензии (ADR-0005/0010) **и** явной policy из
  control-plane.
- Реализация — отдельный capture/enforce-модуль (по аналогии с `CaptureBackend`),
  не в общем горячем пути телеметрии. `#![deny(unsafe_code)]` сохраняется глобально;
  `unsafe` — только в изолированных платформенных хуках с обоснованием (как требует
  `rust-agent-conventions`).

### 2. Платформенные механизмы (не своё ядро без нужды)

| ОС | Механизм enforcement | Подход |
|---|---|---|
| **Windows** | **WDAC / AppLocker** (нативные) + minifilter для realtime | Управляем нативной policy через OS API; свой драйвер — только если WDAC недостаточно |
| **Linux** | **fapolicyd** / **eBPF-LSM** (`bpf_lsm`) | eBPF-LSM для deny на `bprm_check_security`; fapolicyd как fallback |
| **macOS** | Endpoint Security `ES_EVENT_TYPE_AUTH_EXEC` | Authorization-события (требует entitlements/нотаризации) |

**Принцип:** максимально опираться на встроенные в ОС механизмы (политика, а не
самописный драйвер). Свой kernel-компонент — последнее средство, с подписью.

### 3. Модель policy

- Policy (allow/deny списки: по пути, хэшу, издателю/signer, родителю) приходит из
  **control-plane** в policy bundle (как detection rules).
- Сопоставление и решение — на агенте, **офлайн** (air-gap): нет обращения к серверу
  в момент `exec` (требование лёгкости и air-gap).
- Каждое срабатывание (allow-violation/deny) едет в `Envelope` как **detection** →
  ingest → cases/SOAR. Enforcement и наблюдаемость не разделяются.

### 4. Fail-safe и режимы

| Режим | Поведение | Назначение |
|---|---|---|
| `monitor` (audit) | только лог/detection, **не блокирует** | пилот, обкатка policy без риска |
| `enforce` | блокирует deny | боевой |
| **fail-open** | при сбое хука — **разрешать** запуск | дефолт (не «уложить» хост) |
| fail-closed | при сбое — запрещать | только по явной policy высокого риска |

**Дефолт — fail-open + обязательная стадия `monitor`** перед `enforce` (нельзя
включать блокировку, не прогнав policy в audit). Откат policy — мгновенный из CP.

### 5. Бюджет и анти-тампер

- Бюджет ADR-0009 соблюдается: ориентир +5–15 МБ RAM к core; enforcement не должен
  нарушать лимиты (CI-gate бенчмарками).
- Модуль под защитой tamper (`crates/era-agent-core/src/tamper`): отключение enforcement —
  привилегированная операция, попытка обхода → detection.
- **Фазы tamper** (согласовано с [`ADR-0006`](0006-coverage-gaps-strategic-bets-and-practices.md)):
  Фаза 1 (MVP) — detect-and-alert; Фаза 2 — kernel prevent за гейтом WHQL.
  Не позиционировать Фазу 1 как EPP-grade self-defense.

## Критические гейты (не ускоряются AI-разработкой)

1. **Подпись драйвера** (Windows minifilter / WHQL, macOS нотаризация) — календарное
   время вендора, не «скорость кода».
2. **Security-review** enforcement-хуков (риск bootloop, обхода, DoS на хосте).
3. **Полевая обкатка** в режиме `monitor` на парке заказчика до `enforce`.

Поэтому в roadmap (§13 P2b) policy-движок и audit-режим — быстрые; боевой enforce —
за гейтами.

## Последствия

**Плюсы:** закрывает Application Control одним агентом (вместо отдельной надстройки
как у ManageEngine); переиспользует policy-bundle и Envelope/detection-конвейер;
опирается на нативные механизмы ОС.

**Минусы / обязательства:** агент перестаёт быть чисто telemetry → выше требования к
надёжности; нужны подпись драйверов и security-review; обязательная стадия `monitor`;
golden-тесты policy (фикс. набор бинарей → ожидаемые allow/deny); fuzzing парсера
policy (как для парсеров ввода).

## Открытые вопросы

- Нужен ли свой minifilter поверх WDAC или WDAC достаточно для целевых заказчиков.
- Унификация policy-формата App Control с device/USB policy (§6 напр. 5).
- Связь с EPM-lite (JIT elevation, P6) — общий policy-движок привилегий.
