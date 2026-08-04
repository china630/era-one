# ERA Control — индекс исполняемых этапов

**Версия:** 1.0  
**Дата:** 30 июля 2026 г.  
**Приёмка:** [`Control-Acceptance-System.md`](products/Control-Acceptance-System.md)  
**Evidence:** [`Control-Evidence-Rules.md`](Control-Evidence-Rules.md)  
**Канон:** [`ERA-Product-Acceptance-Standard.md`](products/ERA-Product-Acceptance-Standard.md)

Обёртка над legacy GA/Sprint docs. Новые волны Control заводятся здесь.

---

## 1. Волны Production GA (Core / AI / Response)

| # | Wave | Spec / backlog | Gate | Scaffold | Pilot-ready |
|---|------|----------------|------|----------|-------------|
| 1 | **GA-1** | [`Production-GA-Spec.md`](Production-GA-Spec.md) S5 · F-GA-* | `run-ga-full.ps1` / `run-ga1-smoke.ps1` | [x] | **[blocked]** F-GA-5/8/15 field |
| 2 | **GA-1.1** | Production-GA-Spec S8 | `run-ga-full.ps1` | [x] | [~] loadgen prod proof |
| 3 | **GA-2** | Production-GA-Spec S6 · [`GA-2-Signoff.md`](GA-2-Signoff.md) | `run-chaos-smoke.ps1` + CI | [x] | [~] soak / pen-test |
| 4 | **GA-3** | Production-GA-Spec S7 · [`GA-3-Signoff.md`](GA-3-Signoff.md) | helm + edition matrix | [x] | [~] pilot 2-org |

**Программа:** [`GA-Master-Execution-Plan.md`](GA-Master-Execution-Plan.md) · soft backlog закрыт; editions Core/AI/Response = **software GA** (carve-out Evidence); полевые F-GA-5/8/15 **не** `[x]`.  
**Audit:** [`Acceptance-Honesty-Audit-20260730.md`](Acceptance-Honesty-Audit-20260730.md)

---

## 2. Исторические MVP-спринты (S1–S4)

| Wave | Spec | Статус |
|------|------|--------|
| S1 | [`MVP-Sprint-1-Spec.md`](MVP-Sprint-1-Spec.md) | [x] soft |
| S2 | [`MVP-Sprint-2-Spec.md`](MVP-Sprint-2-Spec.md) | [x] soft |
| S3 | [`MVP-Sprint-3-Spec.md`](MVP-Sprint-3-Spec.md) | [x] soft |
| S4 | [`MVP-Sprint-4-Spec.md`](MVP-Sprint-4-Spec.md) | [x] soft |

Новые задачи Core **не** добавлять только в S1: либо GA/Index, либо ADR-matrix + этот Index.

---

## 3. MVP-издания (post-Core)

**AC-матрица SSOT (Scaffold / Pilot):** [`Control-Implementation-Matrix.md`](Control-Implementation-Matrix.md) · UI Shell [`Control-UI-Shell-Spec.md`](Control-UI-Shell-Spec.md) · depth signoff [`reports/control-be-ui-depth-signoff.md`](../reports/control-be-ui-depth-signoff.md) · scaffold-gate [`reports/control-scaffold-green-signoff.md`](../reports/control-scaffold-green-signoff.md) (**не** product-ready field)

| Edition | Spec / PRD | ADR matrix | Gate (Scaffold) | Pilot-ready / GA-гейт |
|---------|------------|------------|-----------------|------------------------|
| **Manage** | [`CMDB-ITAM-Spec.md`](CMDB-ITAM-Spec.md) · [`Enforcement-Spec.md`](Enforcement-Spec.md) · WHQL | 0011, 0012 | `go test` CP + `cargo test` plugins · AC-E1…E3 ✅ | field · **AC-E4 ⏸ WHQL** |
| **Service** | [`Service-Provision-Spec.md`](Service-Provision-Spec.md) | 0016 §4 | service package tests | **[ ]** field rollout |
| **Provision** | Service-Provision-Spec | 0016 §3 | provision tests | **[ ]** field PXE |
| **PAM** | [`PAM-Spec.md`](PAM-Spec.md) · RDP checklist | 0013 | PAM package tests | **[ ]** Guacamole/HSM review |
| **Observe** | [`Observe-Spec.md`](Observe-Spec.md) | 0020 | observe tests | **[ ]** field NMS lab |
| **Perimeter** | [`PRD-ERA-Perimeter.md`](products/PRD-ERA-Perimeter.md) · [`Perimeter-Spec.md`](Perimeter-Spec.md) | 0031 | `go test ./services/waf/... ./services/ngfw/...` | **[ ]** field/pen-test |
| **Resolve** | [`PRD-ERA-Resolve.md`](products/PRD-ERA-Resolve.md) · [`Resolve-Spec.md`](Resolve-Spec.md) | 0031 | `go test ./services/resolve/...` · DnsStub | **[ ]** field DNS |
| **Workbench / Exposure / BYO-EDR** | ADR-0017 | 0017 | workbench/exposure/collectors tests | — / connectors |

Сводка продаж: [`ERA-Product-Line.md`](distributor/ERA-Product-Line.md) §4.  
Matrix: [`ADR-Implementation-Matrix.md`](ADR-Implementation-Matrix.md) · Control AC: [`Control-Implementation-Matrix.md`](Control-Implementation-Matrix.md).

---

## 4. Stage Gate (G1…G6)

Как в каноне. Для Control G1 = `ci-gates-stage10.ps1` / `run-ga-full.ps1`; G6 = GA Signoff или `reports/control-*-signoff.md`.

```powershell
.\scripts\ci-gates-stage10.ps1
.\scripts\run-ga-full.ps1
.\scripts\run-pilot-local.cmd
```

---

## 5. Быстрые команды

| Действие | Команда |
|----------|---------|
| CI gates | `.\scripts\ci-gates-stage10.ps1` |
| Soft GA proof | `.\scripts\run-ga-full.ps1` |
| PII gate | `.\scripts\run-pii-gate.ps1` |
| Local pilot | `.\scripts\run-pilot-local.cmd` |
| Matrix | [`Control-Implementation-Matrix.md`](Control-Implementation-Matrix.md) · [`ADR-Implementation-Matrix.md`](ADR-Implementation-Matrix.md) |
