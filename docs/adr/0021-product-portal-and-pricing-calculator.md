# ADR-0021: Публичный продуктовый портал ERA One + калькулятор цен

**Статус:** Accepted (реализован статический скелет + рабочий калькулятор; контент — в развитии)
**Дата:** 1 июля 2026 г. (обновлено 2 июля 2026 г.)
**Контекст:** Нужен публичный сайт продуктовой линейки **ERA One** на домене
`www.era-one.solutions` (`brand.domain` из [`editions-control.yaml`](../../editions-control.yaml)): маркетинг,
датащиты, сравнения с конкурентами, запрос КП. Обязательное требование — **онлайн-калькулятор
цен** по правилам уже зафиксированной ценовой политики
([`ERA-Pricing.md`](../distributor/ERA-Pricing.md)), с ценой по умолчанию для региона **СНГ**.
Ключевой риск — не размыть инвариант air-gap: публичный сайт не должен путаться с продуктом в
контуре заказчика и с лицензионным control-plane.

**Связано:** [`ADR-0005`](0005-module-independence-and-packaging.md) (модульность) ·
[`ADR-0010`](0010-licensing-and-activation.md) (лицензирование) ·
[`ADR-0018`](0018-hybrid-connected-operating-model.md) (ERA Cloud Portal ≠ этот сайт) ·
[`editions-control.yaml`](../../editions-control.yaml) · [`ERA-Pricing.md`](../distributor/ERA-Pricing.md) ·
[`pricing-data.yaml`](../distributor/pricing-data.yaml) (SSOT калькулятора).

---

## Решение

### 1. Назначение и аудитория
Публичный **vendor-сайт** (presales/маркетинг/лидогенерация): продуктовая линейка, издания,
модель развёртывания (Sovereign / Sovereign Hybrid), сравнения, **калькулятор цен**, форма
«запросить КП». Аудитория — потенциальные заказчики и партнёры региона СНГ.

### 2. Граница air-gap (инвариант)
- Сайт — **публичный периметр вендора**, полностью **отдельный** от контура заказчика и от
  продуктовых сервисов. Он **не имеет** доступа к данным, событиям, кейсам, PII заказчика.
- **Не путать** с `ERA Cloud Portal` (ADR-0018): тот — лицензии/lease/CRL control-plane для
  connected-режима; этот — публичный маркетинговый сайт. Разные системы, разные периметры.
- Калькулятор считает **на стороне клиента** (client-side), персональные данные не собирает;
  единственный сбор — явная форма лида (имя/компания/контакт) с согласием, уходит в почту/CRM
  вендора, **не** в продуктовые сервисы.

### 3. Контент (переиспользование существующих материалов)
- Датащиты `docs/distributor/datasheets/*` (клиентские, без статусов GA/Roadmap).
- Клиентская версия линейки и **клиентский прайс** [`ERA-Pricing-Client.md`](../distributor/ERA-Pricing-Client.md).
- Сравнения `docs/distributor/head-to-head/*` (ManageEngine / Ivanti / Trend Micro).
- Sovereign Hybrid (клиентская версия).

### 4. Калькулятор цен — спецификация
- **SSOT:** [`pricing-data.yaml`](../distributor/pricing-data.yaml) — единый источник (сайт и
  markdown-справочники не расходятся). Правки цен — только здесь.
- **Регион по умолчанию — СНГ** (множитель 0.5 к EU list). Переключатель EU/СНГ; EU-режим
  публично можно скрыть (оставить для partner-режима).
- **Вход:** число workstation / server endpoints; **модель лицензии** (Subscription / Perpetual);
  для Subscription — срок (1/3/5 лет); для Perpetual — договор maintenance (1/3/5 лет);
  выбор модулей или bundle; спец-единицы (PAM: admins + targets; Service: technicians; Observe: devices;
  Federated/National: sites/hubs).
- **Формула** (из SSOT): `line = eu_year × qty × (server?×3) × region.multiplier`;
  Subscription: `total = Σ line × (1 − volume) × (1 − term)`; для **ERA Control AI** — `min(per-endpoint, flat)`;
  на bundle-модули — bundle-скидка. Perpetual: `onetime = 3 × Σ line`, `maintenance = 20% × Σ line / год`.
- **Выход:** индикативная цена по выбранной модели + кнопка «запросить КП».
  Office и Communications — те же правила Subscription/Perpetual (per-user).
- **Guardrail (публичность):** показываются **только клиентские** цены. Внутренние **floor**,
  партнёрская маржа, competitive-displacement в SSOT отсутствуют и на сайт не попадают —
  остаются в `ERA-Pricing.md` и решаются deal-desk. Итог помечается как индикативный, **не оферта**.
- Модули со статусом `availability: project` помечаются «проектное внедрение» (пилот), без
  раскрытия внутренних GA-гейтов.

### 5. Техническое решение (реализовано)
- **Статический сайт** в каталоге **`site/`** (путь `ui/portal` уже занят продуктовым
  SOC-порталом — иная система). Никакой серверной логики цен: калькулятор — **client-side JS**.
  - `site/index.html` — лендинг (hero, линейка, «почему ERA», калькулятор, контакты);
  - `site/assets/portal.css` — стили в фирменной палитре (из `datasheet-common.css`);
  - `site/assets/app.js` — калькулятор, **data-driven** (контролы и цены строятся из SSOT);
  - `site/pricing-data.js` — **автогенерация** из SSOT (`window.ERA_PRICING = {...}`).
- **Сборка:** `python scripts/build_portal.py` читает `pricing-data.yaml` → пишет
  `pricing-data.js` и копирует логотип. Данные встроены как JS (а не `fetch` JSON), чтобы
  сайт открывался и в air-gap/`file://` без CORS-ограничений.
- **Тесты:** `site/test/calculator.test.js` — golden-проверки `compute()` (объём/срок/bundle,
  AI `min(per-endpoint, flat)`, спец-единицы PAM, объём 25 000+ → «по запросу»). `node site/test/calculator.test.js`.
- Единственный backend-контур — приём формы лида (почта/CRM вендора); пока — `mailto:` в КП-кнопке.
- i18n RU/EN (AZ — опционально) — в развитии. Хостинг — публичный, **вне** продуктовых сред; отдельный CI.
- Доноры (идеи, не код): типовые SSG (Astro/Hugo/Next static) — паттерн, не привязка (ADR-0003).

### 6. Управление ценами (процесс)
Изменение цены = правка `pricing-data.yaml` → (а) пересборка сайта; (б) пересборка PDF-справочников
`scripts/pricing_to_pdf.py` (внутренний и клиентский). Ревью — deal-desk. Так цифры на сайте, в
калькуляторе и в датащитах всегда синхронны.

## Последствия
- Новый публичный ассет в `site/` (маркетинговый сайт, отдельный от продуктового `ui/`); SSOT
  `docs/distributor/pricing-data.yaml` — единый источник цен для сайта и датащитов.
- **Осознанная публикация цен:** публикуем клиентские региональные цены (СНГ), не floor/маржу.
- Обязательство поддерживать актуальность SSOT при изменении прайса (иначе расхождение
  сайт ↔ датащиты ↔ КП).
- Домен/бренд — из `editions-control.yaml` (`era-one.solutions`), единый tagline/логотип.
- **Вне scope:** биллинг/оплата онлайн, личный кабинет, самообслуживание лицензий — это
  control-plane (ADR-0018), не публичный сайт.
