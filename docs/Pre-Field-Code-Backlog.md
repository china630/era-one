# Pre-Field Code Backlog (P0 → P1 → P2)

Статусы: `[ ]` todo · `[~]` в работе · `[x]` готово · `[blocked]` заблокировано.

**Field sizing:** [`Field-Server-Sizing.md`](Field-Server-Sizing.md) · **Setup:** [`Field-Server-Setup.md`](Field-Server-Setup.md) · **Готовность:** [`Production-Readiness-Assessment.md`](Production-Readiness-Assessment.md) · **Post-GA gaps:** [`ADR-0022`](adr/0022-detection-content-governance.md) · [`ADR-0023`](adr/0023-ai-investigation-explainability.md)  
**CI:** `scripts/ci-gates-stage10.ps1` · **Proof:** `reports/prefield-proof-2026-07-01.log`

Критерий **90%** = код + автотест в CI + proof-лог. `[blocked: external]` — WHQL, HSM-аудит, RDP security-review, pen-test, подпись заказчика.

---

## P0 — перед переездом на сервер (~90%)

| ID | Задача | % | Статус | Доказательство |
|----|--------|---|--------|----------------|
| C-01 | prod capture + e2e ingest→CH | 90 | [x] | `capture::production_ignores_stub_env`; `pipeline/e2e_golden_test.go` |
| C-02 | budget CI-gate | 95 | [x] | `ci-gates-stage10.ps1` budget_guard |
| C-03 | loadgen prod script + smoke | 85 | [~] | smoke 233 ev/s; AC2 10k → [Field-Server-Sizing](Field-Server-Sizing.md) |
| C-04 | Postgres CP + parity CI | 90 | [x] | profile `pg`; `TestPostgresParity` в ci-gates (docker) |
| C-05 | scale + consumer group | 90 | [x] | profile `scale`; `consumer/group_test.go` |
| C-06 | fail-closed license | 90 | [x] | `licensegate/startup_test.go`; Install-Guide §4 strict |
| C-07 | ITDR Kerberos golden | 90 | [x] | `itdr/golden_test.go` |
| C-08 | tamper prod-guard | 90 | [x] | `tamper::tamper_sim_ignored_in_production` |
| C-09 | exposure tests + UI | 90 | [x] | `exposure/score_test.go`; `ui/workbench` top-10 |
| C-10 | workbench timeline golden | 90 | [x] | `timeline/testdata/timeline_merged.golden.json` |
| C-11 | pilot-local full stack | 90 | [x] | `run-pilot-local.ps1` scale+pg+mTLS API |
| C-12 | sealed clock e2e | 90 | [x] | `licensegate/sealed_clock_e2e_test.go` |

---

## P1 — platform-модули (~90%)

### Manage

| ID | % | Статус | Доказательство |
|----|---|--------|----------------|
| M-01 inventory CH | 90 | [x] | `inventory_history` + chwriter |
| M-02 enforce lab hooks | 75 | [~] | monitor path; kernel enforce [blocked: WHQL] |
| M-03 USB/BitLocker events | 85 | [x] | enforcement events + CP API |
| M-04 virtual patching monitor | 85 | [x] | enforcement policy API |
| M-05 PXE MinIO boot chain | 85 | [x] | `provision/TestPXEConfigGolden` |
| M-06 service-desk SQLite | 90 | [x] | `service-desk/internal/store/sqlite.go` |
| M-07 deploy install+rollback | 90 | [x] | `manage_deploy_test.go` |
| M-08 OTA bundle verify | 90 | [x] | `update-service/api/ota_e2e_test.go` |

### PAM

| ID | % | Статус | Доказательство |
|----|---|--------|----------------|
| P-01 vault persist | 90 | [x] | `vault_persist_test.go` |
| P-02 SSH transcript | 90 | [x] | `proxy/ssh_proxy_test.go` |
| P-03 RDP gateway | 40 | [blocked] | security-review external |
| P-04 LDAP approver | 85 | [x] | SSO profile + CP headers |
| P-05 KMS file-sealed | 85 | [x] | `kms` + StubHSM interface |

### Observe

| ID | % | Статус | Доказательство |
|----|---|--------|----------------|
| O-01 SNMP real | 90 | [x] | `PollReal`; `poll_prod_test.go`; compose `ERA_OBSERVE_SNMP_SIM=0` |
| O-02 discovery no sim fallback | 90 | [x] | `sweep_prod_test.go` |
| O-03 NetFlow | 85 | [x] | `netflow` parsers + tests |
| O-04 CMDB + topology UI | 90 | [x] | `/api/v1/topology`; `topology.html` |

### Hybrid + Hardening

| ID | % | Статус | Доказательство |
|----|---|--------|----------------|
| H-01 CVE bundles | 90 | [x] | `KindCVEFeed` update-service |
| H-02 connected e2e | 85 | [x] | `relay_e2e_test.go` |
| H-03 mTLS relay client | 90 | [x] | `hybrid/client.go` + `ERA_TLS_CA` |
| H-04 Health B | 85 | [x] | `health_b.go` |
| S-01 Helm | 85 | [x] | `deploy/helm/era-one` |
| S-02 backup/restore | 85 | [x] | `backup-restore-smoke.ps1` pg_dump |
| S-03 Grafana | 85 | [x] | `deploy/monitoring/grafana/` |
| S-04 ingest httpserver | 90 | [x] | `httpserver.Listen` |
| S-05 ci-gates extended | 90 | [x] | `ci-gates-stage10.ps1` |

### ADR-0006 (G-01..G-05)

| ID | % | Статус | Доказательство |
|----|---|--------|----------------|
| G-01 NDR | 90 | [x] | `ndr/golden_test.go` |
| G-02 deception | 90 | [x] | `deception/golden_test.go` |
| G-03 CTEM | 90 | [x] | `ctem/golden_test.go` |
| G-04 compliance | 90 | [x] | `compliance/golden_test.go` |
| G-05 risk escalation | 90 | [x] | `risk/golden_test.go` |

### Post-GA — Detection content & AI (ADR-0022 / ADR-0023)

| ID | % | Статус | Доказательство |
|----|---|--------|----------------|
| DC-01 Sigma→MITRE runtime | 0 | [ ] | ADR-0022 DC-6; detection row `mitre_techniques` из rule tags |
| DC-02 FP suppression UI | 0 | [ ] | ADR-0022 DC-7; CP API + UI |
| DC-03 MITRE heatmap | 0 | [ ] | ADR-0022 DC-6 |
| DC-04 CVE feed content | 30 | [~] | `KindCVEFeed`; нет `data/cve-feed/` sample |
| AI-01 investigation audit log | 0 | [ ] | ADR-0023 AI-4 |
| AI-02 evidence chain→custody | 0 | [ ] | ADR-0023 AI-5; `platform/custody` |
| AI-03 attack graph UI | 0 | [ ] | ADR-0023 AI-6; workbench |

---

## P2 (~90%)

| ID | Задача | % | Статус | Доказательство |
|----|--------|---|--------|----------------|
| L-01 federated audit | 90 | [x] | `federated/hub/zone_auth_test.go` |
| L-02 TI-outbound | 90 | [x] | `hybrid/ti_outbound_test.go` |
| L-03 PAM rotation | 90 | [x] | `pam/rotation/scheduler_test.go` |
| L-04 topology widget | 90 | [x] | `observe/api/server_test.go` TestTopologyWidget |
| L-05 era-plugin-vuln cron | 90 | [x] | `era-plugin-vuln` cargo test |
| L-06 rename guide | 100 | [x] | `docs/ERA-One-Rename-Notes.md` |

---

## Не код (вне бэклога)

WHQL, HSM audit, RDP security-review, MDM, VPN/ZTNA, multi-tenant SaaS, pen-test, покупка sizing-сервера.
