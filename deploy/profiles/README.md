# Deploy profiles — ERA One product families



Профили описывают **композицию сервисов** для каждого продуктового семейства.

Single source of truth для on-prem / air-gap поставки.



| Profile | Product | Compose |

|---------|---------|---------|

| [`control.yaml`](control.yaml) | ERA Control | `docker-compose.prod.yml` (default + optional profiles) |

| [`comms.yaml`](comms.yaml) | ERA Communications | standalone; shared platform + planned comms |

| [`office.yaml`](office.yaml) | ERA Office | shared platform + docs engine (roadmap) |

| [`era-one-full.yaml`](era-one-full.yaml) | Full stack | Control + Comms + Office |



**Shared platform (ADR-0025):** identity, tenant, adminportal, drive, docs-engine, workspace.



**Workspace URL:** `https://app.customer.local` (Mail, Drive, Docs); Control SOC: `/secure`.



Запуск Control (текущий prod stack):



```powershell

docker compose -f deploy/docker-compose.prod.yml up -d --build

```



Admin portal:



```powershell

go run ./services/platform/cmd/admin-portal

# http://localhost:8140/api/v1/products

```



Office stub:



```powershell

go run ./services/docs/cmd/docs

# http://localhost:8142/api/v1/status

```



Документы: [`products.yaml`](../../products.yaml) · ADR-0024 · ADR-0025 · ADR-0026 · ADR-0027


