# ADR-0016: Объём UEM/IT-Ops относительно Ivanti — что строим, что отклоняем

**Статус:** Accepted (§3 Provision, §4 Service — Implemented monitor/stage-7; field rollout [gate])
**Дата:** 29 июня 2026 г.
**Контекст:** Лобовое сравнение продуктовой линейки ERA One с Ivanti (Unified
Endpoint Security & IT-Ops) выявило функциональные разрывы. Решаем, какие из них
закрываем, а какие сознательно **не** берём — фильтр: инварианты air-gap и
«лёгкий агент» ([`security-and-air-gap`](../../.cursor/rules/security-and-air-gap.mdc)),
донорская стратегия ([`ADR-0003`](0003-repository-structure-and-donor-strategy.md)),
монорепо-композиция ([`ADR-0014`](0014-multi-product-monorepo.md)).

**Связано:** [`ADR-0005`](0005-module-independence-and-packaging.md) ·
[`ADR-0010`](0010-licensing-and-activation.md) ·
[`ADR-0011`](0011-cmdb-itam-data-model.md) ·
[`ADR-0012`](0012-agent-enforcement-mode.md) · `editions-control.yaml`

---

## Решение (триаж разрывов с Ivanti)

| Разрыв (Ivanti) | Вердикт | Куда в ERA One |
|---|---|---|
| Financial ITAM (контракты, лицензии, стоимость владения) | **BUILD** | издание **ERA Manage** (расширение CMDB) |
| OS Provisioning (bare-metal: PXE/imaging) | **BUILD** | новое издание **ERA Provision** |
| Device Control (USB/периферия, блокировка) | **BUILD** | издание **ERA Manage** (enforcement-плагин) |
| ITSM / Service Desk | **BUILD narrow** | новое издание **ERA Service** (ITSM-lite, на вырост) |
| MDM / Mobile UEM (iOS/Android) | **DECLINE** | вне продукта |
| VPN / ZTNA (secure remote access) | **INTEGRATE-ONLY** | вне core; интеграция при необходимости |

### 1. ERA Manage: Financial ITAM (BUILD)

Расширяем модель CMDB ([`ADR-0011`](0011-cmdb-itam-data-model.md)): к текущему
состоянию активов добавляем **финансово-договорной слой** — контракты, лицензии ПО
(счётчики/комплаенс), стоимость владения, гарантии/амортизация. Хранение —
Postgres (текущее состояние) + ClickHouse (история), как и остальной ITAM. Это
данные/модель, не новый агент. Донор модели — GLPI/iTop (ITIL/ITAM как данные,
[`ADR-0003`](0003-repository-structure-and-donor-strategy.md)).

### 2. ERA Manage: Device Control / USB (BUILD)

Контроль периферии (USB-накопители, классы устройств) — это **enforcement** на
агенте, тот же механизм, что Application Control ([`ADR-0012`](0012-agent-enforcement-mode.md)):
политика разрешённых/запрещённых классов, fail-safe, аудит подключений. Плагин
`era-plugin-devicecontrol` под лицензией `manage`. Гейты те же: подпись драйвера,
security-review. На Linux — udev/eBPF-LSM; на Windows — устройства через
фильтр/политику.

### 3. ERA Provision (BUILD, новое издание)

Развёртывание ОС на «голое железо»: PXE/TFTP + образы + автоустановка
(unattended/kickstart/preseed) + первичная регистрация агента. Полностью **в контуре**
(локальный репозиторий образов, MinIO), без облака — соответствует air-gap.
Серверная часть `services/provision`; агент не нужен на этапе bare-metal (загрузка по
сети), агент ставится в post-install. Лицензия `provision`.

### 4. ERA Service (BUILD narrow, новое издание)

ITSM-lite: заявки/инциденты сервис-деска, портал, привязка к CMDB/активам, SLA.
**Закладываем на вырост:** полная ITIL-модель данных с первого дня (incident /
request / problem / change + связь с CMDB + SLA), но в MVP включаем узкий контур
(incident/request + портал). Рост до полноценного ITSM = включение workflow и
экранов, **не** миграция схемы. Отдельный сервис `services/service-desk` (не врастает
в `control-plane`: SOC-кейсы остаются в control-plane, сервис-деск релизится
независимо). Донор модели — GLPI/iTop. Лицензия `service`.

### 5. MDM / Mobile UEM — DECLINE

iOS/Android MDM требует постоянной связи с облачными push-шлюзами вендоров
(APNs/FCM) и не вписывается в air-gap и модель «один лёгкий агent на endpoint».
Это отдельный рынок и стек. **Не берём.** При запросе — позиционируем как зону
интеграции со специализированным MDM, не как функцию ERA One.

### 6. VPN / ZTNA — INTEGRATE-ONLY

VPN/ZTNA по своей сути — безопасный доступ **через внешнюю/недоверенную сеть**
(обычно интернет) к внутренним ресурсам. В истинном air-gap внешнего удалённого
доступа нет по определению; внутренняя микросегментация (zero-trust между
сегментами) — задача сетевой безопасности (будущее `era-perimeter`: waf/ngfw), а не
endpoint-агента/UEM. Это ортогональная продуктовая ось. В core ERA One **не входит**;
при необходимости — интеграция.

## Последствия

- `editions-control.yaml`: добавляются `era-service`, `era-provision`; `era-manage`
  расширяется (device control, financial ITAM).
- Лицензионные флаги `service`, `provision` (claims.modules, ADR-0010; `exists=false`
  до реализации).
- Roadmap (Vision §13) пополняется фазами Service/Provision/Device Control; MDM/VPN
  фиксируются как **out of scope** (чтобы продажи не обещали лишнего).
- Граница «UEM, но в air-gap» теперь явная и проверяемая: всё, что требует внешней
  сети или мобильных облачных шлюзов, — вне продукта.
