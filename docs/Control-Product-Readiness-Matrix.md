# ERA Control — Product Readiness Matrix (один экран)

**Дата:** 4 августа 2026 г. (BE+UI Depth P0–P4)  
**Канон:** [`ERA-Product-Acceptance-Standard.md`](products/ERA-Product-Acceptance-Standard.md) **v1.3** §3.4  
**Назначение:** «матрица готовности Control» / можно ли показывать / пилотировать / продавать издание.  
**Не путать с** [`Control-Implementation-Matrix.md`](Control-Implementation-Matrix.md) (= Scaffold BE / AC).  
**UI Shell:** [`Control-UI-Shell-Spec.md`](Control-UI-Shell-Spec.md) · Signoff [`reports/control-be-ui-depth-signoff.md`](../reports/control-be-ui-depth-signoff.md)

**Источники:** Control-Sprint-Index · Control-Implementation-Matrix · Pilot-Readiness-Checklist · Product-Line §4 · `editions-control.yaml` · `/ui/control/`

**Demo:** SOC walkthrough / lab smoke / WHQL — не Office Tech-Eval.  
**Rollup** = worst(применимых колонок). Core/AI/Response = software-ga carve-out; field-AC всё ещё открыты.  
**UI target после P4:** usable lab (нет none/thin). Pilot field не закрыт.

---

## Сводка линейки (SSOT готовности)

| Издание | Gate | Scaffold BE | UI | Demo / lab | Pilot lab | Pilot field | Edition | Sell / show |
|---------|------|-------------|----|------------|-----------|-------------|---------|-------------|
| **ERA Core** | ✅ soft | 🟡 (F-GA-5/8/15 field-intent) | ✅ usable lab (`/ui/control/`) | ✅ shell walkthrough | 🟡 | **[blocked]** F-GA-5/8/15 | `ga` software-ga | software GA **с оговоркой field** |
| **ERA Control AI** | ✅ | ✅ human-on-loop + list/get | ✅ usable lab | ✅ investigate queue | 🟡 | [~] field smoke | `ga` software-ga | GA; не autonomous |
| **ERA Response** | ✅ | ✅ playbook catalog + actions | ✅ usable lab | ✅ playbook run | 🟡 | [~] | `ga` software-ga | GA soft |
| **ERA Vuln** | ✅ | ✅ jobs/findings depth | ✅ usable lab | ✅ scan UI | 🟡 | [~] | `ga-option` | опция |
| **ERA Manage** | ✅ | ✅ E1–E3 + deploy detail; E4 ⏸ | ✅ usable lab | ✅ enforce honesty UI | 🟡 | ⏸ WHQL | `mvp` | MVP; kernel block не обещать |
| **ERA Service** | ✅ | ✅ detail/PATCH/comments | ✅ usable lab | ✅ ITIL ops | 🟡 | ⏸ field | `mvp` | server MVP |
| **ERA Provision** | ✅ | ✅ image/PXE/enroll jobs | ✅ usable lab | ✅ wizard lab | 🟡 | ⏸ PXE field | `mvp` | не без field PXE |
| **ERA PAM** | ✅ | ✅ sessions + rotate | ✅ usable lab | ✅ vault/sessions | 🟡 | ⏸ video/HSM | `mvp` | MVP; Guacamole ⏸ |
| **ERA Observe** | ✅ | ✅ device/alerts/pollers | ✅ usable lab | ✅ devices+alerts | 🟡 | ⏸ NMS field | `mvp` | не полный NMS |
| **ERA Perimeter** | ✅ | ✅ WAF CRUD + NGFW history | ✅ usable lab | ✅ WAF/NGFW admin | 🟡 | ⏸ pen-test | `mvp` | lab; не ASIC NGFW |
| **ERA Resolve** | ✅ | ✅ rules/packs/trace filter | ✅ usable lab | ✅ verdict UI | 🟡 | ⏸ field :53 | `mvp` | lab; live TI ⏸ |
| **Workbench / Exposure / BYO** | ✅ | ✅ case-bundle + connectors | ✅ usable lab | ✅ workbench+BYO | 🟡 | [~] | `mvp` | усиление Core |
| **Product `era-control`** | ✅ soft | depth ✅ lab | ✅ shell | ✅ | 🟡 | field open | `ga` | software GA family; MVP editions отдельно |

---

## Вердикт одной строкой

| Вопрос | Ответ |
|--------|-------|
| Soft Core/AI/Response | Да (software GA) |
| Control UI none/thin? | **Нет** — usable lab shell |
| Field F-GA-5/8/15 | **Нет** |
| Manage kernel enforce | **Нет** (WHQL) |
| MVP editions как GA | **Нет** |

---

## Как обновлять

BE → Control-Implementation-Matrix + Scaffold BE здесь.  
UI/shell → [`Control-UI-Shell-Spec.md`](Control-UI-Shell-Spec.md) + эта матрица.  
На «готовность Control» — **этот файл**.
