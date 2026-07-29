# ERA Communications — Implementation Matrix (прослеживаемость)

**Дата:** 7 июля 2026 г.  
**Назначение:** PRD AC-C* → код → тест → статус. Живой документ.  
**Система:** [`Comms-Acceptance-System.md`](products/Comms-Acceptance-System.md)

**Легенда:** ✅ готово · 🟡 частично · [ ] нет · ⏸ поле/гейт · **Pilot-ready** — отдельно от scaffold

---

## Сводка по изданиям (snapshot)

| Издание | PRD AC закрыто | Auto-proof | Статус edition |
|---------|----------------|------------|----------------|
| **ERA Mail Server** | 7/7 MVP AC | ✅ C-1+C-2+C-3 gates | `ga` ([`editions-comms.yaml`](../editions-comms.yaml)) |
| ERA Mail Connect | 1/1 (AC-C6) | ✅ C-1.1 gate | roadmap |
| ERA Comms Migration | 5/5 (AC-MIG-1…5) | [x] Phase 3–4 partner ga | `go test` + 1k/40k scripts 2026-07-10 |
| ERA Outlook Bridge | 6/6 (AC-BR-*) | [x] Phase 3 Exchange + 100mb runbook | `go test ./services/comms/mail-bridge/...` 2026-07-10 |
| ERA Mail Moderation | 10/10 AC-MM + P1 | [x] C-MM-H Pilot-ready / C-MM-P1 | `go test` + gate C-MM-H 2026-07-29 |
| ERA Conference | Phase 2 | ✅ C-4 gate | roadmap |
| ERA Chat | Phase 2 | ✅ C-4 gate | roadmap |

*Последний прогон:* `scripts/run-comms-stage-gate.ps1 -Stage C-4`

---

## Backlog по волнам (snapshot)

| Wave | Backlog prefix | Spec | Статус |
|------|----------------|------|--------|
| C-1 | CM1-* | [`MVP-Comms-Mail-Sprint-1-Spec.md`](MVP-Comms-Mail-Sprint-1-Spec.md) | [x] |
| C-1.1 | CM2-* | [`Comms-Stage-C1.1-Spec.md`](Comms-Stage-C1.1-Spec.md) | [x] |
| C-2 | CM3-* | [`Comms-Stage-C2-Spec.md`](Comms-Stage-C2-Spec.md) | [x] |
| C-3 | CM4-* | [`Comms-Stage-C3-Spec.md`](Comms-Stage-C3-Spec.md) | [x] |
| C-4 | CM5-* | [`Comms-Stage-C4-Spec.md`](Comms-Stage-C4-Spec.md) | [x] |
| C-5 | CM6-* | [`Comms-Stage-C5-Spec.md`](Comms-Stage-C5-Spec.md) | [x] |
| C-MIG | CM-MIG-* | [`Comms-Stage-CMIG-Spec.md`](Comms-Stage-CMIG-Spec.md) | [x] |
| C-MM | CM-MM-* | [`Comms-Stage-CMM-Spec.md`](Comms-Stage-CMM-Spec.md) | [x] |
| C-MM-H | MM-H-* | [`Comms-Stage-CMM-H-Spec.md`](Comms-Stage-CMM-H-Spec.md) | [x] |
| C-MM-P1 | MM-P1-* | [`Comms-Stage-CMM-P1-Spec.md`](Comms-Stage-CMM-P1-Spec.md) | [x] |
| C-GA | CM-GA-* | [`Comms-Stage-CGA-Spec.md`](Comms-Stage-CGA-Spec.md) | [x] |

Полный индекс: [`Comms-Sprint-Index.md`](Comms-Sprint-Index.md).

---

## PRD → код → тест

### AC-C1 — Send/receive IMAP+SMTP air-gap

| Компонент | Путь | Тест / доказательство | Scaffold | Pilot-ready |
|-----------|------|----------------------|----------|-------------|
| SMTP receiver | `services/comms/mail/core/src/smtp.rs` | `cargo test -p era-mail-core --test smtp_tls_e2e` | ✅ | ✅ AUTH+STARTTLS |
| IMAP subset | `services/comms/mail/core/src/imap.rs` | `smtp_imap_e2e` + login deny | ✅ | ✅ STARTTLS |
| Unified store | `services/comms/mail/internal/repo` | `go test -tags=integration .../repo` | ✅ | ✅ PG+MinIO |
| Remote store | `services/comms/mail/core/src/remote_store.rs` | `go test -tags=integration .../internalapi` | ✅ | ✅ |
| Air-gap compose E2E | `deploy/docker-compose.comms.yml` | `run-comms-pilot-staging.ps1` | ✅ | ✅ readyz PG+MinIO+CH |

### AC-C2 — Webmail + platform identity

| Компонент | Путь | Тест | Scaffold | Pilot-ready |
|-----------|------|------|----------|-------------|
| Webmail SPA | `ui/mail/web/` | `go test era/ui/mail/...` | ✅ | ✅ |
| OIDC | `services/platform/cmd/identity-api` | `go test .../oidc` | — | ✅ |
| Identity RBAC | `services/platform/identity` | legacy shell | ✅ | 🟡 |

### AC-C3 — Autodiscover golden XML

| Компонент | Путь | Тест | Статус |
|-----------|------|------|--------|
| Generator | `services/comms/mail/internal/autodiscover/` | `autodiscover_test.go` + golden (IMAP/SMTP/EWS/CalDAV) | ✅ |
| Tenant resolve | `services/platform/tenant` | tenant tests | ✅ |

### AC-C4 — Inline quota via policy API

| Комponent | Путь | Тест | Статус |
|-----------|------|------|--------|
| Policy store + defaults | `services/comms/mail/internal/policy/` | `policy_test.go` | ✅ |
| HTTP GET `/api/v1/policy` | `services/comms/mail/internal/api/server.go` | `server_test.go` | ✅ |

### AC-C5 — Drive attach (if licensed)

| Комponent | Путь | Тест | Статус |
|-----------|------|------|--------|
| Drive API hook | `ui/mail/` (license-aware hook) | `go test ./ui/mail/...` | ✅ |

### AC-C6 — Mail Connect → external IMAP

| Комponent | Путь | Тест | Статус |
|-----------|------|------|--------|
| BFF | `services/comms/mail-connect/` | `go test ./services/comms/mail-connect/...` | ✅ Phase 1.1 (hybrid tier) |

### AC-MIG-1…AC-MIG-5 — ERA Comms Migration bulk import

| Комponent | Путь | Тест | Статус |
|-----------|------|------|--------|
| Migration API/workers | `services/comms/migration/` | `go test ./services/comms/migration/...` | ✅ |
| IMAP importer golden | `services/comms/migration/internal/importers/imap/testdata/` | `go test ./services/comms/migration/...` | ✅ |
| EWS calendar import | `services/comms/migration/internal/importers/ews/` | `go test ./services/comms/migration/...` | ✅ |
| PST/MBOX smoke | `services/comms/migration/internal/importers/archive/` | `go test ./services/comms/migration/...` | ✅ |
| CH migration audit | `services/comms/migration/internal/audit/` | `go test ./services/comms/migration/...` | ✅ |

### AC-MM-1…AC-MM-10 — ERA Mail Moderation

| Компонент | Путь | Тест / доказательство | Статус |
|-----------|------|----------------------|--------|
| AC-MM-1 policy novices+external | `services/comms/mail-moderation/internal/policy` | `go test ./services/comms/mail-moderation/...` | [x] Pilot-ready |
| AC-MM-2 Approve/Reject | `smtpproxy` + `notify` | same | [x] Pilot-ready |
| AC-MM-3 resolve override | `internal/resolve` | same | [x] Pilot-ready |
| AC-MM-4 keywords/VIP | `testdata/rules` golden | same | [x] Pilot-ready |
| AC-MM-5 TTL expire | `internal/hold` | same | [x] Pilot-ready |
| AC-MM-6 bypass | `internal/policy` | same | [x] Pilot-ready |
| AC-MM-7 action links | `internal/notify` | same | [x] Pilot-ready |
| AC-MM-8 audit | `internal/audit` | same | [x] Pilot-ready |
| AC-MM-9 IceWarp/SMTP path | smtpproxy e2e + lab runbook | `Comms-Mail-Moderation-IceWarp-Lab.md` + `run-comms-mm-icewarp-lab.ps1` | [x] Pilot-ready |
| AC-MM-10 YAML I/O | `internal/adminapi` / policy | same | [x] Pilot-ready |

### AC-MM-P1 — Mail Moderation P1

| Компонент | Путь | Тест / доказательство | Статус |
|-----------|------|----------------------|--------|
| AC-MM-P1-1 moderated DL | `policy.moderated_recipients` + engine | `TestEvaluate_ModeratedRecipient`, `TestEngine_ModeratedDL` | [x] |
| AC-MM-P1-2 HR API | `POST /v1/moderation/hr/novices` | `TestHRNovices` | [x] |
| AC-MM-P1-3 Admin UI | `/ui/` | `TestUI` | [x] |

### AC-C7 — ClickHouse audit после send/receive

| Комponent | Путь | Тест | Статус |
|-----------|------|------|--------|
| Proto | `proto/era/v1/comms.proto` | gen-proto | ✅ |
| DDL | `deploy/clickhouse/004_comms_mail_audit.sql` | compose init + `ApplyMailAuditDDL` | ✅ |
| Go writer | `services/comms/mail/internal/audit/writer.go` | `writer_test.go` | ✅ |
| Webhook | `core/src/audit_hook.rs` + `internal/auditapi/` | `smtp_audit_e2e` + auditapi_test | ✅ |
| CH audit path E2E | `internal/audit/audit_path_integration_test.go` | `go test -tags integration ./services/comms/mail/internal/audit/...` | ✅ (docker CH) |
| Calendar audit | `internal/calendaraudit/recorder.go` | `go test -tags integration ./services/comms/mail/internal/calendaraudit/...` | ✅ |

### AC-C8 — CalDAV create/edit event

| Комponent | Путь | Тест | Статус |
|-----------|------|------|--------|
| CalDAV | `services/comms/calendar/caldav/` | `go test ./services/comms/calendar/...` | ✅ |
| iCal golden | `services/comms/calendar/ical/` | `ical_test.go` | ✅ |

### AC-C9 — EWS Outlook subset

| Комponent | Путь | Тест | Статус |
|-----------|------|------|--------|
| EWS adapter | `services/comms/mail/internal/ews/` | `go test ./services/comms/mail/internal/ews/...` | ✅ |
| EWS golden | `internal/ews/testdata/` | `handler_test.go` | ✅ |

### Phase 2 — F-C21…F-C23 (Chat + Conference)

| Компонент | Путь | Тест | Статус |
|-----------|------|------|--------|
| ERA Chat API | `services/comms/chat/` | `go test ./services/comms/chat/...` | ✅ |
| ERA Conference adapter | `services/comms/vcs/` | `go test ./services/comms/vcs/...` | ✅ |
| Chat UI shell | `ui/chat/` | `go test ./ui/chat/...` | ✅ |
| Meet UI shell | `ui/meet/` | `go test ./ui/meet/...` | ✅ |
| Chat/VCS CH audit | `services/comms/auditch/` | `go test -tags integration ./services/comms/auditch/...` | ✅ |
| ActiveSync | `mail/activesync` | `go test era/services/comms/mail/internal/activesync/...` | ✅ | ✅ WBXML golden |

---

## ADR-0027 → код

| Решение ADR | Код | Статус |
|-------------|-----|--------|
| Rust mail core + Go API | `services/comms/mail/` | ✅ C-1 |
| CalDAV + EWS | `calendar/` + `mail/internal/ews/` | ✅ C-2 |
| ClickHouse audit обязателен | audit + DDL | ✅ |
| Office boundary (no co-edit) | ADR-0027 §4 | ✅ docs |
| Mail Connect hybrid tier | `services/comms/mail-connect/` | [x] |
| ERA Comms Migration upsell | `services/comms/migration/` | [~] Phase 2 CG connector + PG queue + MBOX golden |
| ERA Outlook Bridge | `services/comms/mail-bridge/` | [~] BR-3/4/6/7 IceWarp + IMAP-generic adapters |
| LiveKit adapter | `services/comms/vcs/` | [x] |
| ERA Chat | `services/comms/chat/` | [x] |
| Patterns only donors | [`ERA-Communications-Donors.md`](products/ERA-Communications-Donors.md) | ✅ |

---

## Контракты

| Артеfact | Путь | Тест |
|----------|------|------|
| MailAuditEvent | `proto/era/v1/comms.proto` | `go test ./gen/go/...` |

---

## Как обновлять

1. Merge PR с кодом + тестом  
2. Строка в этой таблице → ✅ или 🟡 + команда теста  
3. [`Comms-MVP-Spec.md`](Comms-MVP-Spec.md) F-C* / CM*-* + [`Comms-Sprint-Index.md`](Comms-Sprint-Index.md)  
4. `run-comms-stage-gate.ps1 -Stage C-X` — PASS в CI или локально  

См. также [`ADR-Implementation-Matrix.md`](ADR-Implementation-Matrix.md) §ADR-0027.
