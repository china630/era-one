# ERA Communications — Product Readiness Matrix (один экран)

**Дата:** 4 августа 2026 г. (Deepen D0–D9 lab)  
**Канон:** [`ERA-Product-Acceptance-Standard.md`](products/ERA-Product-Acceptance-Standard.md) **v1.3** §3.4  
**Deepen:** [`Comms-Deepen-Spec.md`](Comms-Deepen-Spec.md)  
**Не путать с** [`Comms-Implementation-Matrix.md`](Comms-Implementation-Matrix.md) (= Scaffold BE / AC).

**Demo:** staging RT-01…08 · partner/lab smoke · browser field.  
**Rollup** = worst(применимых колонок).

---

## Сводка линейки (SSOT готовности)

| Издание | Gate | Scaffold BE | UI | Demo (RT/lab) | Pilot lab | Pilot field | Edition | Sell / show |
|---------|------|-------------|----|---------------|-----------|-------------|---------|-------------|
| **ERA Mail Server** | ✅ C-* | ✅ AuthZ + core status + CH require | n/a | ✅ staging RT; Outlook field open | ✅ | [ ] RT-09 | `mvp` | greenfield **с оговорками**; not ga |
| **ERA Mail Client** (webmail) | 🟡 | ✅ OIDC/BFF + PKCE (D2) | 🟡 SPA + Workspace chrome tokens | ✅ PKCE+RT-05 lab | ✅ lab | [ ] | `roadmap` | не полный клиент |
| **ERA Mail Connect** | ✅ | ✅ | n/a | ✅ IMAP lab (D4) | ✅ lab | [ ] RT-10 | `mvp` | hybrid lab; vault=env |
| **ERA Comms Migration** | ✅ | ✅ | n/a | 🟡 1k/live IMAP lab | 🟡 | [ ] cutover | `mvp` | partner runbook |
| **ERA Outlook Bridge** | ✅ | ✅ | n/a | 🟡 100mb; synthetic≠field | 🟡 | [ ] | `mvp` | partner lab |
| **ERA Mail Moderation** | ✅ | ✅ | 🟡 admin thin | ⏸ IceWarp | 🟡 native | ⏸ | `mvp` | не без IceWarp |
| **ERA Chat** | ✅ | ✅ | 🟡 | [ ] field | 🟡 PG | [ ] | `mvp` | not ga |
| **ERA Conference** | ✅ | ✅ | 🟡 stub/live | [ ] | 🟡 | [ ] | `mvp` | not ga |
| **ERA Comms AI** | ✅ | ✅ | 🟡 | [ ] | 🟡 Ollama | [ ] | `mvp` | heuristic/stub |
| **Product `era-communications`** | mixed | mostly ✅ | 🟡 | 🟡 | staging ✅ | RT-09 SKIP | editions `mvp` | **не** ga suite |

---

## Вердикт

| Вопрос | Ответ |
|--------|-------|
| Server scaffold + staging lab | Да (D0/D9) |
| Webmail как Exchange client | Нет (roadmap desktop) |
| Field RT-09 | SKIP — [`reports/comms-rt09-skip.md`](../reports/comms-rt09-skip.md) |
| Edition `ga` | Запрещён до Pilot field |

## Как обновлять

BE → Implementation-Matrix. UI/RT → этот файл. Deepen waves → Comms-Deepen-Spec.md.
