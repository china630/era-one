# ERA Communications — Implementation Matrix (evidence-based)

**Дата:** 4 августа 2026 г. (Deepen D0–D4)  
**Приёмка:** [`Comms-Acceptance-System.md`](products/Comms-Acceptance-System.md)  
**Готовность продукта:** [`Comms-Product-Readiness-Matrix.md`](Comms-Product-Readiness-Matrix.md)  
**Deepen:** [`Comms-Deepen-Spec.md`](Comms-Deepen-Spec.md)  
**Канон:** [`ERA-Product-Acceptance-Standard.md`](products/ERA-Product-Acceptance-Standard.md)

**Легенда:** ✅ proof · 🟡 partial · `[ ]` нет · ⏸ поле · **Pilot lab** ≠ **Pilot field**

---

## Сводка изданий

| Издание | Disk | Scaffold / Lab | Pilot lab | Pilot field | Edition |
|---------|------|----------------|-----------|-------------|---------|
| Mail Server | ✅ | ✅ AuthZ + protocols; core status via ERA_MAIL_CORE_ADDR | [x] staging RT-01…08 | [ ] RT-09 ⏸ | **mvp** |
| Mail Client webmail | ✅ | ✅ OIDC/BFF + PKCE + Drive attach | [x] RT-05 lab | [ ] | roadmap |
| Mail Connect | ✅ | ✅ AuthZ + mode=stub / live IMAP | [x] dovecot-lab / unit | [ ] RT-10 | **mvp** |
| Migration | ✅ | ✅ AuthZ + metrics honesty | [~] | [ ] cutover | **mvp** |
| Outlook Bridge | ✅ | ✅ AuthZ + upstream_mode | [~] | [ ] | **mvp** |
| Mail Moderation | ✅ | ✅ admin AuthZ | [~] | ⏸ IceWarp | **mvp** |
| Chat | ✅ | ✅ AuthZ | [~] | [ ] | **mvp** |
| Conference | ✅ | ✅ AuthZ + mode | [~] | [ ] | **mvp** |
| Comms AI | ✅ | ✅ AuthZ + degraded | [~] | [ ] | **mvp** |

---

## PRD AC-C*

| AC | Proof | Scaffold | Pilot lab | Pilot field | Note |
|----|-------|----------|-----------|-------------|------|
| **AC-C1** | AuthZ; PG; core non-stub status; SMTP/IMAP TLS | ✅ | [x] | [ ] | D0-1 |
| **AC-C2** | OIDC PKCE + staging token + BFF Bearer | ✅ | [x] | [ ] | D2 lab; browser IdP field open |
| **AC-C3** | Autodiscover golden | ✅ | [x] | [ ] | staging RT-08 |
| **AC-C4** | REST 413 + SMTP 552 | ✅ | [x] | [ ] | staging RT-07 |
| **AC-C5** | Drive attach JWT + license deny | ✅ | [x] | [ ] | D0-2 |
| **AC-C6** | Connect AuthZ + live IMAP / stub=0 | ✅ | [x] | [ ] | D4 lab |
| **AC-C7** | CH require + readyz 503 if missing | ✅ | [x] | [ ] | D0-3 |
| **AC-C8** | CalDAV AuthZ + staging | ✅ | [x] | [ ] | Outlook field open |
| **AC-C9** | EWS AuthZ + staging | ✅ | [x] | [ ] | Outlook field open |

### Upsell / roadmap

| Area | Scaffold | Pilot lab | Pilot field | Note |
|------|----------|-----------|-------------|------|
| AC-MIG | ✅ | [~] | [ ] | D5 |
| Bridge | ✅ | [~] | [ ] | D6 |
| AC-MM | ✅ | [~] | ⏸ | D7 / IceWarp DF |
| Chat / Conf / AI | ✅ | [~] | [ ] | D8 |

Partner / `ga`: [`reports/comms-rt09-skip.md`](../reports/comms-rt09-skip.md).
