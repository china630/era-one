# ERA One — ценовая политика (внутренний)

> **ONE AGENT. ONE PLATFORM. ONE CONTROL.**

**Версия:** 1.0
**Дата:** 1 июля 2026 г.
**Аудитория:** отдел продаж, пресейл, дистрибьюторы, deal-desk
**Позиционирование:** EU-компания (HQ Geneva, инженерный центр Warsaw), основной рынок продаж — AZ/CIS.
**Связано:** [`editions-control.yaml`](../../editions-control.yaml) · [`ERA-Product-Line.md`](./ERA-Product-Line.md) ·
[`ERA-Naming-and-RFQ-Guide.md`](./ERA-Naming-and-RFQ-Guide.md) ·
[`ADR-0005`](../adr/0005-module-independence-and-packaging.md) (модульность) ·
[`ADR-0010`](../adr/0010-licensing-and-activation.md) (лицензирование).

> **Дисклеймер:** ориентиры для формирования КП, не публичный прайс. Финальная цена крупных сделок —
> через deal-desk. Все суммы в EUR без НДС; локальная валюта/налоги — при выпуске КП.

---

## 0. Как устроена монетизация (важно понимать перед торгом)

**Один агент — платные модули по лицензии.** Агент (`era-agent-core`, ADR-0019) — один лёгкий
бинарник-хост плагинов. Каждый модуль гейтится лицензией отдельно (`crates/era-agent-core/src/license_gate.rs`,
`services/platform/licensegate`): плагин/сервис стартует, только если флаг есть в `claims.modules` (ADR-0010).

Следствия для продаж:
- Клиент платит **за включённые модули**, не за агента. Агент и SOC-портал — в составе базы.
- Апселл = **новая лицензия с доп. флагом**, без переустановки агента (land-and-expand).
- «Один агент» — операционный аргумент (нет зоопарка из 5 клиентов, единый бюджет CPU<2%/RAM<150МБ,
  один канал обновлений), **не** «все модули бесплатно».
- Под-функции внутри модуля включены: **ERA Manage** уже содержит BitLocker, Application/Device Control
  и console-пользователей (у ManageEngine — отдельные SKU).

**Две модели поставки (клиент выбирает):**
- **Подписка** (annual) — рекомендуется, повторяемая выручка; цены ниже — прайс §1.
- **Perpetual + maintenance** — для гос/КИИ, кому подписка неудобна — §6.

---

## 1. Прайс по модулям (годовая подписка)

Базовая единица — за год. **AZ regional = EU list − 50%** (региональный прайс-лист, не «скидка в КП»).
**Сервер = ×3** к цене workstation (критичнее, больше телеметрии), если не указано иное.

> **Как читать цену за агента:** каждый модуль лицензируется и тарифицируется **отдельно** (свой флаг в
> лицензии), цена — **за endpoint с агентом**, и на одном агенте модули **суммируются**
> (напр. Core €6 + Manage €6 + AI €4 = €16/ws AZ). Не куплен модуль — плагин не стартует, в цену не входит.
> **Серверная платформа** (ingest, ClickHouse-lake, control-plane, SOC-портал) ставится **один раз** в контуре
> и **входит в поставку** — отдельно за неё и «за каждый агент» не платят.

| Издание | Единица | EU list /год | AZ regional /год | Статус |
|---|---|---|---|---|
| **ERA Core** (XDR-база: агент-телеметрия + детекция на endpoint; даёт доступ к кейсам, SOC-порталу, Workbench) | endpoint | €12 | €6 | GA |
| **ERA Control AI** (ИИ-аналитик, LLM в контуре) | endpoint | €8 | €4 | GA |
| **ERA Response** (SOAR) | endpoint | €4 | €2 | GA |
| **ERA Vuln** (CVE-скан) | endpoint | €4 | €2 | GA-опция |
| **ERA Exposure** (risk score; треб. Core+Vuln) | endpoint | €4 | €2 | MVP |
| **ERA Manage** (UEM: CMDB/ITAM, deploy/patch, **BitLocker + App/Device Control + console — включены**) | endpoint | €12 | €6 | MVP |
| **ERA Observe** (SNMP/NetFlow/discovery) | monitored device | €6 | €3 | MVP |
| **ERA BYO-EDR Hub** (приём стороннего EDR) | source endpoint | €4 | €2 | MVP |
| **ERA Service** (ITSM) | **technician** (портал-юзеры €0) | €900 | €450 | MVP |
| **ERA Provision** (PXE/imaging) | node | €8 | €4 | MVP |
| **ERA PAM** — сейф/checkout/SSH-запись | **admin** | €50 | €25 | MVP |
| **ERA PAM** — управляемая цель (target) | target | €30 | €15 | MVP |
| **ERA Federated** (обмен IoC внутри орг.) | site | €24,000 | €12,000 | GA-опция |
| **ERA National** (STIX/TAXII хаб) | hub | €36,000 | €18,000 | GA-опция |
| Console / portal-пользователи, ERA Workbench | — | €0 | €0 | вкл. |

> **MVP-издания** (см. [`ERA-Product-Line.md`](./ERA-Product-Line.md) §1/§4): код готов и в тестах,
> но продаём **как пилот с GA-гейтами** (WHQL-подпись драйвера для боевого enforce, RDP/HSM для PAM),
> не как GA. Для пилота MVP-издания — доп. скидка §5 (launch/reference).

### ERA Control AI — альтернатива flat (для крупных парков)

ИИ-инференс — на сервере (AI Core), не на агенте, поэтому per-endpoint при 10k+ узлах душит сделку.
Предлагай **min(per-endpoint, flat)** — что клиенту выгоднее:

| Парк endpoints | EU list flat /год | AZ regional /год |
|---|---|---|
| ≤ 1,000 | €18,000 | €9,000 |
| 1,001 – 5,000 | €45,000 | €22,500 |
| 5,001 – 15,000 | €90,000 | €45,000 |
| 15,000+ | deal-desk | deal-desk |

---

## 2. Скидки за оптовость (volume)

Применяется к per-endpoint позициям (Core + per-endpoint модули), к **AZ regional** цене.
Скидка на **весь объём** (не маржинально по «полкам»). Flat-издания (AI-flat, Federated/National) объёмом не дисконтируются.

| Кол-во endpoints | Скидка |
|---|---|
| 1 – 250 | 0% |
| 251 – 1,000 | 10% |
| 1,001 – 5,000 | 20% |
| 5,001 – 10,000 | 30% |
| 10,001 – 25,000 | 40% |
| 25,000+ | 45–55% (deal-desk) |

---

## 3. Скидки за срок лицензии (term)

| Условие | Скидка (эфф./год) | Комментарий |
|---|---|---|
| **1 год** | 0% | база |
| **3 года, оплата ежегодно** | 10% | commit без предоплаты |
| **3 года, предоплата целиком** | 20% | лучший кэшфлоу → макс. скидка |
| **5 лет, предоплата** | 25% | длинные гос/КИИ-контракты |

---

## 4. Bundle-скидки (стимул брать платформу)

Пакеты из [`editions-control.yaml`](../../editions-control.yaml) `bundles` — скидка к сумме à la carte:

| Bundle | Состав | Скидка |
|---|---|---|
| **Старт SecOps** | Core + AI + Response | 20% |
| **SecOps + Vuln** | + Vuln | 22% |
| **ERA IT-Ops** | Core/Manage + Service + Provision | 25% |
| **ERA Unified / Sovereign Stack** | XDR + Manage + Service + Observe | 30% |
| **Full** | вся линейка (+ PAM по спец-цене) | 35% |

---

## 5. Правила комбинирования, пол и спец-скидки

**Формула:** `цена = EU list → AZ regional (−50%) → × (1−bundle) × (1−volume) × (1−term)`.
Скидки bundle/volume/term **перемножаются** (не складываются).

**Пол (floor):**
- Совокупная скидка от **EU list** — не глубже **75%** на коммерции (т.е. AZ regional −50% + доп. стек до ~75%).
- Гос/нацмасштаб — глубже только через deal-desk.
- Всегда резервировать **20–30% партнёрской маржи** дистрибьютору/интегратору.
- **Минимальная сделка:** €5,000 или 250 endpoints.

**Спец-скидки (в пределах floor):**

| Тип | Скидка | Когда |
|---|---|---|
| Gov / edu | −10…15% | госзаказчик, вуз |
| Competitive displacement | до −15% | rip-replace ManageEngine / Ivanti / Trend Micro |
| Launch / reference (первые 3–5 клиентов AZ) | договорная | референс-кейс, право на публикацию |
| MVP-пилот | −10…20% | издание в статусе MVP, поэтапный rollout |

**Renewal cap:** рост при продлении ≤ +7–10%/год (защита от ценовой атаки конкурента на перезаключении).

---

## 6. Perpetual + maintenance (для гос/КИИ)

- **Perpetual (вечная лицензия)** = **3× годовой подписки** (после применимых скидок), разовый платёж.
- **Maintenance 20%/год — обязателен**: обновления Sigma/CVE/AI-паков + поддержка.
- **Air-gap:** без активного maintenance нет offline-бандлов обновлений (детект устаревает) — включать в каждую сделку.
- Sovereign Hybrid (lease + Update Service + Managed View) — доп. подписка сопровождения **+10…15%** к лицензии.

---

## 7. Пример расчёта (по BOM ManageEngine)

Реальный запрос: банк, **7,500 workstation + 400 servers**, UEM + Vulnerability Mgmt + BitLocker +
Application Control + 50 console users + PAM на 200 админов, **3 года**. Ivanti предложил **$150K/год (~€139K)**.

**Маппинг на ERA** (BitLocker, App Control, console — включены в Manage → €0 отдельно):

| Позиция | Кол-во | AZ regional /ед. | Сумма /год |
|---|---|---|---|
| ERA Manage (ws) | 7,500 | €6 | €45,000 |
| ERA Manage (server ×3) | 400 | €18 | €7,200 |
| ERA Vuln | 7,500 | €2 | €15,000 |
| ERA PAM (admins) | 200 | €25 | €5,000 |
| BitLocker + App Control + 50 console users | — | €0 | €0 |
| **AZ regional list** | | | **€72,200/год** |

*(для справки: EU list той же корзины ≈ €144K/год — паритет с ценой Ivanti €139K)*

**Скидки:** volume (7,900 → 30%) × 3-year prepaid (20%) = ×0.56 → **€40,432/год**.
При competitive displacement (−13%, в пределах floor) → **≈ €35,000/год**, **≈ €105K за 3 года**.

**Итог в КП:**
- ERA: **€35K/год** (€105K/3 года).
- Ivanti: €139K/год (€417K/3 года) → **экономия ~75% (~€312K за контракт)**.
- vs ManageEngine: та же цена, но **один агент**, air-gap штатно, BitLocker/App Control/console **в базе**,
  и бонусом **XDR-фундамент** на том же агенте (дозажигается лицензией).

---

## 8. Связанные документы

- [`ERA-Pricing-Client.md`](./ERA-Pricing-Client.md) — клиентская версия прайса (регион СНГ, без floor/маржи)
- [`pricing-data.yaml`](./pricing-data.yaml) — SSOT цен для калькулятора портала (ADR-0021)
- [`ERA-Product-Line.md`](./ERA-Product-Line.md) — карта изданий и статусы готовности
- [`editions-control.yaml`](../../editions-control.yaml) — модули, лицензионные флаги, bundle
- [`head-to-head/`](./head-to-head/) — сравнения с ManageEngine / Ivanti / Trend Micro
- ADR: [`0005`](../adr/0005-module-independence-and-packaging.md) · [`0010`](../adr/0010-licensing-and-activation.md) · [`0021`](../adr/0021-product-portal-and-pricing-calculator.md)

---

*Внутренний материал. Не публиковать как открытый прайс. MVP-издания — только как пилот с GA-гейтами.*
