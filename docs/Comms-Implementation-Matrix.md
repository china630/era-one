# ERA Communications — Implementation Matrix (evidence-based)

**Дата:** 29 июля 2026 г.  
**Правила:** [`Comms-Evidence-Rules.md`](Comms-Evidence-Rules.md)  
**Recovery:** `b1d066f` stash restore · Re-gate batch 2026-07-29

**Легенда:** `[x]` = git + PASS + лог · `[~]` = partial · `[ ]` = нет · Pilot-ready = RT/field only

---

## Сводка изданий

| Издание | Disk | Scaffold gate | Pilot-ready | Edition |
|---------|------|---------------|-------------|---------|
| Mail Server | ✅ | C-1 PASS (CH SKIP) | [ ] RT-09 | **mvp** |
| Mail Client webmail | ✅ `ui/mail` | C-3 PASS | [ ] | roadmap (desktop false) |
| Mail Connect | ✅ | C-1.1 PASS | [ ] | roadmap |
| Migration | ✅ | C-MIG PASS | [ ] field | **mvp** |
| Outlook Bridge | ✅ | unit PASS | [ ] field | **mvp** |
| Mail Moderation | ✅ | C-MM-H PASS | [ ] IceWarp | **mvp** |
| Chat / Conf / AI | ✅ | C-4 / C-5 PASS | [ ] | roadmap (scaffold) |

---

## Stage gates (re-proof 2026-07-29)

| Wave | Result | Log |
|------|--------|-----|
| C-1 | PASS (CH SKIP) | `reports/comms-stage-C-1-20260729-235925.log` |
| C-1.1 | PASS | `reports/comms-stage-C-1.1-20260729-235840.log` |
| C-2 | PASS | `reports/comms-stage-C-2-20260729-235843.log` |
| C-3 | PASS | `reports/comms-stage-C-3-20260729-235850.log` |
| C-4 | PASS | `reports/comms-stage-C-4-20260729-235903.log` |
| C-5 | PASS | `reports/comms-stage-C-5-20260729-235914.log` |
| C-MIG | PASS | `reports/comms-stage-C-MIG-20260729-235852.log` |
| C-MM-H | PASS | `reports/comms-stage-C-MM-H-20260729-235857.log` |
| BR unit | PASS | `go test -C services/comms/mail-bridge ./...` |
| C-GA / RT-09 | [~] / SKIP | no customer field host |

---

## AC rows

| AC | Package | Proof command | Last PASS | Scaffold | Pilot-ready |
|----|---------|---------------|-----------|----------|-------------|
| AC-C1 | mail + era-mail-core | gate C-1 | 2026-07-29 | [x] | [ ] |
| AC-C2 | ui/mail | gate C-3 | 2026-07-29 | [x] | [ ] |
| AC-C3 | autodiscover | gate C-1 | 2026-07-29 | [x] | [ ] |
| AC-C4 | policy | gate C-1 | 2026-07-29 | [x] | [ ] |
| AC-C5 | drive hook | gate C-3 | 2026-07-29 | [x] | [ ] |
| AC-C6 | mail-connect | gate C-1.1 | 2026-07-29 | [x] | [ ] |
| AC-C7 | audit + CH | gate C-1 / integration | SKIP CH local | [~] | [ ] |
| AC-C8 | calendar | gate C-2 | 2026-07-29 | [x] | [ ] |
| AC-C9 | ews | gate C-2 | 2026-07-29 | [x] | [ ] |
| AC-MIG-* | migration | gate C-MIG | 2026-07-29 | [x] | [ ] |
| AC-BR-* | mail-bridge | go test | 2026-07-29 | [x] | [ ] |
| AC-MM-* + P1 | mail-moderation | C-MM-H | 2026-07-29 | [x] | [ ] field |

---

## Field / RT

| ID | Status | Evidence |
|----|--------|----------|
| RT-01…08 staging | [~] | `scripts/run-comms-pilot-staging.ps1` restored; needs compose up |
| RT-09 customer | SKIP | `reports/comms-rt09-skip.md` — no field host |
| MM IceWarp | SKIP | no `ERA_MM_ICEWARP_HOST` |

---

## Phase 2 honesty

C-4/C-5 gates PASS = **scaffold** (in-memory chat / Stub VCS / Heuristic AI). Prod backends = GAP-P2-01…03 — not `ga`.
