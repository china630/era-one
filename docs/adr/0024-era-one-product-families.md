# ADR-0024: Продуктовые семейства ERA One (Control / Communications / Office)



**Статус:** Accepted (обновлено 5 июля 2026 г.)  

**Дата:** 4 июля 2026 г.  

**Контекст:** Бренд **ERA One** — зонтик экосystemы; три продуктовых семейства требуют

явной карты в репозитории, shared platform layer и deploy-профилей без дробления на

отдельные git-репо.



**Связано:** [`ADR-0014`](0014-multi-product-monorepo.md) · [`ADR-0025`](0025-era-one-shared-platform.md) ·

[`ADR-0026`](0026-sovereign-office-engine.md) · [`ADR-0027`](0027-era-communications-architecture.md) ·

[`products.yaml`](../../products.yaml)



---



## Контекст



- **ERA Control** — Security & IT-Ops, GA.

- **ERA Communications** — почта/чат/ВКС, roadmap (ADR-0027, PRD Comms).

- **ERA Office** — документы/таблицы, roadmap (ADR-0026, PRD Office).



**Communications и Office — standalone** (per-user, без ERA Core). Optional cross-sell и

Full Stack bundle.



---



## Решение



### 1. Один монорepo, три продуктовых семейства



| Уровень | Артеfact |

|---------|----------|

| Бренд | `products.yaml` → `brand` |

| Продукт | `products.yaml` → `products.*` |

| Издание Control | `editions-control.yaml` |

| Издание Comms | `editions-comms.yaml` |

| Издание Office | `editions-office.yaml` |

| Shared capabilities | `editions-shared.yaml` (Drive, Sign) |

| Deploy | `deploy/profiles/*.yaml` |



### 2. Shared platform (ADR-0025)



Расширение `services/platform/`: identity, tenant, adminportal, licensegate,

**drive**, **docs-engine**, **signing**, **workspace**.



- **Identity** — включена в любой продукт; **не продаётся** отдельной строкой RFQ.

- **ERA Drive** — отдельное издание; всегда в ERA Office Suite.

- **Co-editing** — platform/docs-engine; лицензия через ERA Office.

- **User shell:** `app.customer.local` (Workspace).



Детали: [`ADR-0025`](0025-era-one-shared-platform.md).



### 3. Communications и Office



| Продукт | Документы |

|---------|-----------|

| Communications | Vision, ADR-0027, PRD-Comms-MVP |

| Office | Vision, ADR-0026, PRD-Office-MVP |



### 4. Deploy profiles



- `control.yaml` — ERA Control (GA)

- `comms.yaml` — Communications (standalone)

- `office.yaml` — Office (standalone)

- `era-one-full.yaml` — Full Stack



### 5. Переименование издания Control



**ERA AI** → **ERA Control AI** (`era-control-ai`, license `control-ai`) — см. `editions-control.yaml`.



---



## Последствия



**Плюсы:** единый бренд; shared platform; чёткие границы продуктов.



**Обязательства:** sync `products.yaml`, editions-*.yaml, deploy profiles, site catalog.



---



## Артеfactы



- [`products.yaml`](../../products.yaml)

- [`editions-shared.yaml`](../../editions-shared.yaml)

- [`editions-comms.yaml`](../../editions-comms.yaml)

- [`editions-office.yaml`](../../editions-office.yaml)

- [`docs/products/`](../products/)


