# ERA One — Hardening & Scale (Stage 10)

Cross-cutting упрочнение платформы после этапов 2–9.

**Связано:** [Production-GA-Spec](Production-GA-Spec.md) · [ADR-0018 §11](adr/0018-hybrid-connected-operating-model.md) · [Implementation-Roadmap §Post-GA](Implementation-Roadmap.md) · [deploy/prod/README.md](../deploy/prod/README.md)

**Статус:** Implemented (soft-complete); кластерный soak/scale → [gate: field].

## 10a. Масштаб и HA

| Артефакт | Статус |
|---|---|
| Sizing (1 нода + full editions) | [`Field-Server-Sizing.md`](Field-Server-Sizing.md), `deploy/prod/README.md` |
| HA compose Kafka RF=3 | `deploy/docker-compose.prod-ha.yml` |
| Compose profile `scale` | `event-writer-b2`, `detection-engine-b2`, общий `ERA_CONSUMER_GROUP` |
| Helm platform services + probes | `deploy/helm/era-one` + `scripts/helm-template-check.ps1` |
| Loadgen `-agents` | `scripts/run-loadgen-prod.ps1`, `cmd/loadgen` |
| Кластер 10k+ ev/s, 7×24 soak | [gate: field] |

## 10b. Update Service (контент)

| Артефакт | Статус |
|---|---|
| Sigma bundles + Ed25519 подпись | `services/update-service` |
| Golden wire-format | `bundle_test.go` (`TestBundleWireFormatStable`) |
| `/metrics` + mTLS opt-in | `update-service/internal/api` |

## 10c. Edition-matrix

| Артефакт | Статус |
|---|---|
| Модули manage/service/provision/pam/observe | `licensegate.Module*` |
| Bundle tests | `edition_matrix_test.go` (it-ops, unified, full, KnownModules) |
| `editions-control.yaml` | platform editions `mvp` |

## 10d. Безопасность

| Инвариант | Покрытие |
|---|---|
| mTLS + HTTP listen | `services/platform/httpserver` + `tlsutil` на pam/observe/service-desk/provision/detection-engine |
| RBAC skeleton | control-plane, cloud-portal |
| KMS abstraction | pam vault |
| CI gates | `scripts/ci-gates-stage10.ps1` (PII golden, pam no-secret-leak, bundle, edition-matrix) |
| Driver signing / HSM audit | [gate: external] |

## 10e. Observability & backup

| Сервис | `/metrics` | Backup |
|---|---|---|
| control-plane | да | SQLite/PG volume |
| ingest-gateway | да | Kafka/CH volumes |
| pam, service-desk, provision, observe | да | `backup-prod.ps1` (volumes when present) |
| detection-engine | да (HTTP :8097) | stateless |
| cloud-portal, update-service | да | — |

Chaos: `run-chaos-smoke.ps1` (расширен stage-10 пакетами).

## 10f. Managed private cloud groundwork

- Helm: PVC для pam, liveness/readiness probes
- Tenant scoping: `X-ERA-Tenant-ID` на CP API
- Multi-tenant SaaS — вне scope

## 10g. GA-документация

- `editions-control.yaml`, Blueprint §5, Implementation-Roadmap
- Platform specs: Agent-Core … Observe

## Приёмка (доказательство)

```powershell
scripts/ci-gates-stage10.ps1
```

Крупный масштаб и полевой soak — только с [gate: field] логом.
