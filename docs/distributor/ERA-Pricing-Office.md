# ERA Office — ценовая политика (внутренний)

**Версия:** 1.0  
**Дата:** 6 июля 2026 г.  
**Аудитория:** продажи, пресейл, deal-desk  
**SSOT:** [`pricing-office-data.yaml`](./pricing-office-data.yaml)  
**Связано:** [`editions-office.yaml`](../../editions-office.yaml) · [`editions-shared.yaml`](../../editions-shared.yaml) ·
[`ERA-RFQ-Office-Template.md`](./ERA-RFQ-Office-Template.md) · ADR-0026

> Ориентир для КП, не публичная оферта. EUR без НДС.

---

## 0. Модель

- **Per-user** (не endpoint). **Standalone** — ERA Core не нужен.
- **ERA Drive** (`platform-drive`) — shared edition; продаётся отдельно или в bundle Office/Mail upsell.
- **Office Suite** always includes Drive — не предлагать Suite без Drive в RFQ.
- Скидки volume/term/perpetual — **наследуются** из [`pricing-data.yaml`](./pricing-data.yaml).

---

## 1. EU list (€/user/year)

| SKU | EU | AZ/CIS (−50%) | Примечание |
|-----|-----|---------------|------------|
| ERA Drive | 4 | 2 | Upsell к Mail |
| ERA Documents | 8 | 4 | Requires Drive |
| ERA Tables | 6 | 3 | P2 |
| ERA Presentations | 5 | 2.5 | P3 |
| ERA Projects | 4 | 2 | post-MVP |
| ERA Office AI | 6 | 3 | ≠ Control/Comms AI |

---

## 2. Bundles

| Key | Modules | Discount | EU / user |
|-----|---------|----------|-----------|
| `office-mvp` | Drive + Documents | 10% | 10.8 |
| `office-suite` | Drive + Doc + Tables + Pres | 25% | 17.25 |
| `office-suite-ai` | Suite + Office AI | 28% | ~20.9 |

**Pitch:** Suite CIS **~€8.6/user** vs M365 **~€350+** — суверенность + свой engine.

---

## 3. Конкуренты (talk track)

| Конкурент | Слабость ERA бьёт |
|-----------|-------------------|
| M365 / Google | Cloud, data residency |
| OnlyOffice/Collabora | Лицензия вендору, GPL/AGPL narrative |
| «Свой LibreOffice» | Нет co-editing enterprise, support burden |

---

## 4. Cross-sell

1. **Comms-only** → Drive (attachments) → Office (co-edit).
2. **Full Stack ERA One** — bundle discount на уровне deal-desk (Control + Comms + Office).

---

## 5. Статус

Все издания **roadmap** — продавать как **пилот** до GA Office engine (ADR-0026 P1).
