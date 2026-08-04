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
| M-02 enforce lab hooks | 90 | [x] | monitor-complete: app/USB/BitLocker goldens + VP detection; kernel enforce [blocked: WHQL] |
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
| P-03 RDP gateway | 85 | [x] | TCP relay MVP (`RDPProxy` + `TestRDPProxySession`); inject/video [gate: security-review] |
| P-04 LDAP approver | 85 | [x] | SSO profile + CP headers |
| P-05 KMS file-sealed | 85 | [x] | `kms` + StubHSM interface |
| P2-01 RDP broker inject | 90 | [x] | `pam/internal/broker` — inject_token, no password in JSON |
| P2-02 session idle/max | 90 | [x] | `sessionpolicy` + custody `proxy.rdp.timeout` |
| P2-03 metadata recording | 85 | [x] | `recording` artifact+hash (not Guacamole video) |

### Observe

| ID | % | Статус | Доказательство |
|----|---|--------|----------------|
| O-01 SNMP real | 90 | [x] | `PollReal` + HOST-RESOURCES; `poll_prod_test.go`; compose `ERA_OBSERVE_SNMP_SIM=0` |
| O-02 discovery no sim fallback | 90 | [x] | `sweep_prod_test.go` |
| O-03 NetFlow | 90 | [x] | UDP listener + golden; `run-observe-smoke.ps1` |
| O-04 CMDB + topology UI | 90 | [x] | `/api/v1/topology`; `topology.html` |

### Perimeter (ADR-0031)

| ID | % | Статус | Доказательство |
|----|---|--------|----------------|
| PE-01 WAF proxy + rules | 90 | [x] | `waf` golden + reverse-proxy tests; `run-perimeter-smoke.ps1` |
| PE-02 NGFW policy API | 90 | [x] | evaluate golden + persist; licensegate |
| PE-03 DLP session (pam\|perimeter) | 90 | [x] | dlp gate pam or perimeter |
| P2-PE-01 WAF body + CRS-lite | 90 | [x] | `EvaluateWithBody` + `era-waf-crs-*` |
| P2-PE-02 NGFW host apply opt-in | 85 | [x] | `ERA_NGFW_APPLY=1` nft/iptables; default noop |
| P2-PE-03 content-DLP inspect | 90 | [x] | `POST /api/v1/dlp/inspect` + golden |

### Resolve (ADR-0031)

| ID | % | Статус | Доказательство |
|----|---|--------|----------------|
| RS-01 Guard verdict + DNS | 90 | [x] | `guard` golden + `dnsx` UDP NXDOMAIN/sinkhole |
| RS-02 Trace DnsEvent | 90 | [x] | `trace.Buffer` + envelope DNS topic |
| RS-03 Atlas offline pack | 90 | [x] | `atlas_pack.golden.json` hit → deny |
| RS-04 compose profile | 85 | [x] | `deploy` profile `resolve`; `run-resolve-smoke.ps1` |
| P2-RS-01 DoH | 90 | [x] | `/dns-query` RFC 8484; doh golden NXDOMAIN |
| P2-RS-02 Atlas update packs | 90 | [x] | `KindAtlasPack` + `packs/reload` USB path |
| P2-RS-03 agent DnsEvent | 85 | [x] | `era-collectors::emit_dns_event` stub |

### Manage Phase 2 lab-enforce

| ID | % | Статус | Доказательство |
|----|---|--------|----------------|
| P2-M-01 lab decision enforce | 90 | [x] | `BlockResult` + `effect=telemetry_only` golden; OS block ⏸ WHQL |
| P2-M-02 kernel stub messaging | 90 | [x] | `kernel_hook=unavailable` + WHQL message |
| P2-M-03 VP monitor-only | 90 | [x] | enforce mode still allows VP would_block |

### AI Agentic (ADR-0023 Phase 3 lite)

| ID | % | Статус | Доказательство |
|----|---|--------|----------------|
| P2-AI-01 recommended_actions | 90 | [x] | golden `recommended_actions.malicious` |
| P2-AI-02 confirm/reject audit | 90 | [x] | `POST .../confirm|reject` + custody |
| P2-AI-03 SOAR draft handoff | 85 | [x] | draft-only; no auto-execute |

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
| DC-01 Sigma→MITRE runtime | 90 | [x] | `sigma.Techniques` + `DetectionRow.MitreTechniques`; `processor_mitre_test` |
| DC-02 FP suppression UI | 90 | [x] | CP `/api/v1/suppressions` + DE `suppress.Cache` + `ui/cases` |
| DC-03 MITRE heatmap | 90 | [x] | `GET /api/v1/mitre/coverage` + workbench; `mitre/coverage.golden.json` |
| DC-04 CVE feed content | 90 | [x] | `data/cve-feed/cve.json` + `vm/cvefeed` golden |
| AI-01 investigation audit log | 90 | [x] | `investigate.AuditLog` + `GET /api/v1/investigate/audit` |
| AI-02 evidence chain→custody | 90 | [x] | `SealEvidence` → `custody_root_hash` |
| AI-03 attack graph UI | 90 | [x] | `BuildAttackGraph` + workbench + `POST /investigate/graph` |

### Deferred (не в этой волне)

| Тема | Статус |
|------|--------|
| Hybrid ступени 3–4 / full TI-share | Roadmap — [Hybrid-Roadmap-3-4.md](Hybrid-Roadmap-3-4.md) |
| EPM-lite / JIT admin | Отдельная линия Manage P6; **вне** monitor-complete |

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
